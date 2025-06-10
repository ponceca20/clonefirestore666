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
	if index == nil {
		return fmt.Errorf("index cannot be nil")
	}

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
	}

	// Set index metadata - using a separate Index document for MongoDB storage
	indexDoc := model.Index{
		ID:         primitive.NewObjectID().Hex(),
		ProjectID:  projectID,
		DatabaseID: databaseID,
		Collection: collectionID,
		Name:       index.Name,
		Fields:     convertToIndexFields(index.Fields),
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
		// Cleanup index document if MongoDB index creation fails
		i.indexesCol.DeleteOne(ctx, filter)
		return fmt.Errorf("failed to create MongoDB index: %w", err)
	}

	// Update index state to ready
	updateFilter := bson.M{"_id": indexDoc.ID}
	updateDoc := bson.M{
		"$set": bson.M{
			"state":      model.IndexStateReady,
			"updated_at": time.Now(),
		},
	}
	i.indexesCol.UpdateOne(ctx, updateFilter, updateDoc)

	i.logger.Info("Index created successfully", "name", index.Name, "collection", collectionID)
	return nil
}

// DeleteIndex deletes an index
func (i *IndexOperations) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	// Get the index first
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"_id":           indexID,
	}

	var indexDoc model.Index
	err := i.indexesCol.FindOne(ctx, filter).Decode(&indexDoc)
	if err != nil {
		return ErrIndexNotFound
	}

	// Delete the MongoDB index
	err = i.deleteMongoDBIndex(ctx, projectID, databaseID, collectionID, indexDoc.Name)
	if err != nil {
		i.logger.Error("Failed to delete MongoDB index", "error", err, "indexName", indexDoc.Name)
	}

	// Delete the index document
	result, err := i.indexesCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete index document: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrIndexNotFound
	}

	i.logger.Info("Index deleted successfully", "name", indexDoc.Name, "collection", collectionID)
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
		var indexDoc model.Index
		if err := cursor.Decode(&indexDoc); err != nil {
			continue // Skip invalid indexes
		}

		index := &model.CollectionIndex{
			Name:   indexDoc.Name,
			Fields: convertFromIndexFields(indexDoc.Fields),
			State:  indexDoc.State,
		}
		indexes = append(indexes, index)
	}

	return indexes, nil
}

// GetIndex retrieves a specific index
func (i *IndexOperations) GetIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) (*model.CollectionIndex, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"_id":           indexID,
	}

	var indexDoc model.Index
	err := i.indexesCol.FindOne(ctx, filter).Decode(&indexDoc)
	if err != nil {
		return nil, ErrIndexNotFound
	}

	index := &model.CollectionIndex{
		Name:   indexDoc.Name,
		Fields: convertFromIndexFields(indexDoc.Fields),
		State:  indexDoc.State,
	}

	return index, nil
}

// createMongoDBIndex creates the actual MongoDB index
func (i *IndexOperations) createMongoDBIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	// Build MongoDB index model
	keys := bson.D{}
	for _, field := range index.Fields {
		direction := 1
		if field.Order == model.IndexFieldOrderDescending {
			direction = -1
		}
		keys = append(keys, bson.E{Key: "fields." + field.Path + ".value", Value: direction})
	}

	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(index.Name),
	}

	// Create the index on the documents collection
	_, err := i.documentsCol.Indexes().CreateOne(ctx, indexModel)
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

	// Delete and recreate the MongoDB index
	err = i.deleteMongoDBIndex(ctx, projectID, databaseID, collectionID, index.Name)
	if err != nil {
		i.logger.Warn("Failed to delete index during rebuild", "error", err)
	}

	// Recreate the index
	err = i.createMongoDBIndex(ctx, projectID, databaseID, collectionID, index)
	if err != nil {
		return fmt.Errorf("failed to recreate index: %w", err)
	}

	// Update index state
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"_id":           indexID,
	}
	updateDoc := bson.M{
		"$set": bson.M{
			"state":      model.IndexStateReady,
			"updated_at": time.Now(),
		},
	}
	i.indexesCol.UpdateOne(ctx, filter, updateDoc)

	return nil
}

// GetIndexStatistics returns statistics for an index
func (i *IndexOperations) GetIndexStatistics(ctx context.Context, projectID, databaseID, collectionID, indexID string) (*model.IndexStatistics, error) {
	// Get document count for the collection
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	count, err := i.documentsCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	// For simplicity, return basic statistics
	// In a real implementation, this would include more detailed MongoDB index statistics
	stats := &model.IndexStatistics{
		IndexID:       indexID,
		IndexName:     "", // Would need to lookup index name
		DocumentCount: count,
		StorageSize:   0, // Would need to query MongoDB index statistics
		LastUsed:      time.Now(),
	}

	return stats, nil
}

// Helper functions for field conversion
func convertToIndexFields(fields []model.IndexField) []model.IndexField {
	// Direct conversion since the types are compatible
	return fields
}

func convertFromIndexFields(fields []model.IndexField) []model.IndexField {
	// Direct conversion since the types are compatible
	return fields
}
