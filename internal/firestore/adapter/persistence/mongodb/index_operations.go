package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Index operation errors
var (
	ErrIndexAlreadyExists = errors.New("index already exists")
	ErrIndexNotFound      = errors.New("index not found")
)

// --- Interfaces para testabilidad y hexagonalidad ---
type IndexCollection interface {
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
	InsertOne(ctx context.Context, doc interface{}) (interface{}, error)
	DeleteOne(ctx context.Context, filter interface{}) (DeleteResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResult, error)
	Find(ctx context.Context, filter interface{}) (Cursor, error)
	FindOne(ctx context.Context, filter interface{}) SingleResult
}

type DocumentCollection interface {
	Indexes() IndexManager
	CountDocuments(ctx context.Context, filter interface{}) (int64, error) // Añadido para estadísticas
}

type IndexManager interface {
	CreateOne(ctx context.Context, model interface{}) (interface{}, error)
	DropOne(ctx context.Context, name string) (interface{}, error)
	ListSpecifications(ctx context.Context) ([]IndexSpec, error)
}

type Cursor interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Close(ctx context.Context) error
	Err() error
}

type SingleResult interface {
	Decode(val interface{}) error
}

type DeleteResult struct{ DeletedCount int64 }
type UpdateResult struct{}
type IndexSpec struct{ Name string }

// IndexOperations refactorizado para usar interfaces
// IndexOperations handles index management operations
type IndexOperations struct {
	indexesCol   IndexCollection
	documentsCol DocumentCollection
	logger       logger.Logger
}

// NewIndexOperations crea una nueva instancia inyectando dependencias
func NewIndexOperations(indexesCol IndexCollection, documentsCol DocumentCollection, logger logger.Logger) *IndexOperations {
	return &IndexOperations{
		indexesCol:   indexesCol,
		documentsCol: documentsCol,
		logger:       logger,
	}
}

// CreateIndex creates a new index for a collection
func (i *IndexOperations) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	now := time.Now()

	// Check if index already exists
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"name":          index.Name,
	}

	count, err := i.indexesCol.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	if count > 0 {
		return ErrIndexAlreadyExists
	} // Set index metadata - using a separate Index document for MongoDB storage
	indexDoc := model.Index{
		ID:         primitive.NewObjectID().Hex(),
		ProjectID:  projectID,
		DatabaseID: databaseID,
		Collection: collectionID,
		Name:       index.Name,
		Fields:     index.Fields,
		State:      model.IndexStateCreating,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// Create index document
	_, err = i.indexesCol.InsertOne(ctx, indexDoc)
	if err != nil {
		return fmt.Errorf("failed to create index document: %w", err)
	}

	// Create the actual MongoDB index
	err = i.createMongoDBIndex(ctx, projectID, databaseID, collectionID, index)
	if err != nil {
		// Rollback: delete the index document
		i.indexesCol.DeleteOne(ctx, filter)
		return fmt.Errorf("failed to create MongoDB index: %w", err)
	}

	// Update index state to ready
	updateDoc := bson.M{
		"$set": bson.M{
			"state": model.IndexStateReady,
		},
	}
	i.indexesCol.UpdateOne(ctx, filter, updateDoc)

	return nil
}

// DeleteIndex deletes an index
func (i *IndexOperations) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	// Get the index first
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"index_id":      indexID,
	}

	var index model.CollectionIndex
	err := i.indexesCol.FindOne(ctx, filter).Decode(&index)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrIndexNotFound
		}
		return fmt.Errorf("failed to get index: %w", err)
	}

	// Delete the MongoDB index
	err = i.deleteMongoDBIndex(ctx, projectID, databaseID, collectionID, index.Name)
	if err != nil {
		return fmt.Errorf("failed to delete MongoDB index: %w", err)
	}

	// Delete the index document
	result, err := i.indexesCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete index document: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrIndexNotFound
	}

	return nil
}

// ListIndexes lists all indexes for a collection
func (i *IndexOperations) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	cursor, err := i.indexesCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []*model.CollectionIndex
	for cursor.Next(ctx) {
		var index model.CollectionIndex
		if err := cursor.Decode(&index); err != nil {
			i.logger.Errorf("Failed to decode index: %v", err)
			continue
		}
		indexes = append(indexes, &index)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return indexes, nil
}

// GetIndex retrieves a specific index
func (i *IndexOperations) GetIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) (*model.CollectionIndex, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"index_id":      indexID,
	}

	var index model.CollectionIndex
	err := i.indexesCol.FindOne(ctx, filter).Decode(&index)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrIndexNotFound
		}
		return nil, fmt.Errorf("failed to get index: %w", err)
	}

	return &index, nil
}

// createMongoDBIndex creates the actual MongoDB index
func (i *IndexOperations) createMongoDBIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	// Build MongoDB index model
	indexKeys := bson.D{}
	for _, field := range index.Fields {
		direction := 1
		if field.Order == model.IndexFieldOrderDescending {
			direction = -1
		}
		indexKeys = append(indexKeys, bson.E{Key: "fields." + field.Path + ".value", Value: direction})
	}

	// Create index options
	indexOptions := options.Index().SetName(index.Name)

	// Create the index on the documents collection
	_, err := i.documentsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    indexKeys,
		Options: indexOptions,
	})

	return err
}

// deleteMongoDBIndex deletes the actual MongoDB index
func (i *IndexOperations) deleteMongoDBIndex(ctx context.Context, projectID, databaseID, collectionID, indexName string) error {
	_, err := i.documentsCol.Indexes().DropOne(ctx, indexName)
	return err
}

// RebuildIndex rebuilds an existing index
func (i *IndexOperations) RebuildIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	// Get the index
	index, err := i.GetIndex(ctx, projectID, databaseID, collectionID, indexID)
	if err != nil {
		return err
	}

	// Set state to building
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"index_id":      indexID,
	}
	updateDoc := bson.M{
		"$set": bson.M{
			"state": model.IndexStateCreating,
		},
	}

	_, err = i.indexesCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update index state: %w", err)
	}

	// Drop and recreate the MongoDB index
	err = i.deleteMongoDBIndex(ctx, projectID, databaseID, collectionID, index.Name)
	if err != nil {
		i.logger.Warnf("Failed to drop existing index during rebuild: %v", err)
	}

	err = i.createMongoDBIndex(ctx, projectID, databaseID, collectionID, index)
	if err != nil {
		// Set state to error
		updateDoc["$set"] = bson.M{"state": model.IndexStateError}
		i.indexesCol.UpdateOne(ctx, filter, updateDoc)
		return fmt.Errorf("failed to recreate index: %w", err)
	}

	// Set state to ready
	updateDoc["$set"] = bson.M{"state": model.IndexStateReady}
	_, err = i.indexesCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update index state to ready: %w", err)
	}

	return nil
}

// GetIndexStatistics returns statistics for an index
func (i *IndexOperations) GetIndexStatistics(ctx context.Context, projectID, databaseID, collectionID, indexID string) (*model.IndexStatistics, error) {
	// Get the index first
	index, err := i.GetIndex(ctx, projectID, databaseID, collectionID, indexID)
	if err != nil {
		return nil, err
	}

	// Get MongoDB index statistics
	indexStats, err := i.documentsCol.Indexes().ListSpecifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get index specifications: %w", err)
	}
	// Find our index and calculate statistics
	for _, spec := range indexStats {
		if spec.Name == index.Name {
			// Calculate document count for this collection using injected interface
			filter := bson.M{
				"project_id":    projectID,
				"database_id":   databaseID,
				"collection_id": collectionID,
			}

			docCount, err := i.indexesCol.CountDocuments(ctx, filter)
			if err != nil {
				docCount = 0 // Log error but don't fail completely
			}

			// Get storage size estimate (simplified calculation)
			storageSize := docCount * 1024 // Estimate 1KB per document
			lastUsed := time.Now()

			stats := &model.IndexStatistics{
				IndexID:       indexID,
				IndexName:     index.Name,
				DocumentCount: docCount,
				StorageSize:   storageSize,
				LastUsed:      lastUsed,
			}
			return stats, nil
		}
	}

	return nil, fmt.Errorf("index statistics not found")
}
