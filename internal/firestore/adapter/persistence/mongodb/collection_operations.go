package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CollectionOperations handles collection-related operations
type CollectionOperations struct {
	repo *DocumentRepository
}

// NewCollectionOperations creates a new CollectionOperations instance
func NewCollectionOperations(repo *DocumentRepository) *CollectionOperations {
	return &CollectionOperations{repo: repo}
}

// CreateCollection creates a new collection
func (c *CollectionOperations) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	now := time.Now()

	// Check if collection already exists
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collection.ID,
	}

	count, err := c.repo.collectionsCol.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if count > 0 {
		return ErrCollectionAlreadyExists
	}
	// Create collection document
	collectionDoc := &model.Collection{
		ID:            collection.ID,
		ProjectID:     projectID,
		DatabaseID:    databaseID,
		CollectionID:  collection.CollectionID,
		CreatedAt:     now,
		UpdatedAt:     now,
		DocumentCount: 0,
		Indexes:       []model.CollectionIndex{},
	}

	_, err = c.repo.collectionsCol.InsertOne(ctx, collectionDoc)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// GetCollection retrieves a collection by ID
func (c *CollectionOperations) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	var collection model.Collection
	err := c.repo.collectionsCol.FindOne(ctx, filter).Decode(&collection)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	return &collection, nil
}

// UpdateCollection updates a collection
func (c *CollectionOperations) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collection.CollectionID,
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"display_name": collection.DisplayName,
			"description":  collection.Description,
			"updated_at":   time.Now(),
		},
	}

	updateResult, err := c.repo.collectionsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update collection: %w", err)
	}

	if updateResult.MatchedCount == 0 {
		return ErrCollectionNotFound
	}

	return nil
}

// DeleteCollection deletes a collection by ID
func (c *CollectionOperations) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	// First check if collection has any documents
	docFilter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	docCount, err := c.repo.documentsCol.CountDocuments(ctx, docFilter)
	if err != nil {
		return fmt.Errorf("failed to check collection documents: %w", err)
	}

	if docCount > 0 {
		return fmt.Errorf("cannot delete collection with existing documents")
	}

	// Delete the collection
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	deleteResult, err := c.repo.collectionsCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	if deleteResult.DeletedCount == 0 {
		return ErrCollectionNotFound
	}

	return nil
}

// ListCollections lists all collections in a database
func (c *CollectionOperations) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	filter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	cursor, err := c.repo.collectionsCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer cursor.Close(ctx)

	var collections []*model.Collection
	for cursor.Next(ctx) {
		var collection model.Collection
		if err := cursor.Decode(&collection); err != nil {
			c.repo.logger.Error("Failed to decode collection: %v", err)
			continue
		}
		collections = append(collections, &collection)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return collections, nil
}

// ListSubcollections lists all subcollections under a document
func (c *CollectionOperations) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	// Build the parent path
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)

	// Aggregation pipeline to find subcollections
	pipeline := mongo.Pipeline{
		{
			{Key: "$match", Value: bson.D{
				{Key: "project_id", Value: projectID},
				{Key: "database_id", Value: databaseID},
				{Key: "path", Value: bson.D{{Key: "$regex", Value: "^" + parentPath + "/"}}},
			}},
		},
		{
			{Key: "$addFields", Value: bson.D{
				{Key: "subcollectionPath", Value: bson.D{
					{Key: "$arrayElemAt", Value: bson.A{
						bson.D{{Key: "$split", Value: bson.A{"$path", parentPath + "/"}}},
						1,
					}},
				}},
			}},
		},
		{
			{Key: "$addFields", Value: bson.D{
				{Key: "subcollectionId", Value: bson.D{
					{Key: "$arrayElemAt", Value: bson.A{
						bson.D{{Key: "$split", Value: bson.A{"$subcollectionPath", "/"}}},
						0,
					}},
				}},
			}},
		},
		{
			{Key: "$match", Value: bson.D{
				{Key: "subcollectionId", Value: bson.D{{Key: "$ne", Value: ""}}},
			}},
		},
		{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$subcollectionId"},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			}},
		},
		{
			{Key: "$project", Value: bson.D{
				{Key: "subcollectionId", Value: "$_id"},
				{Key: "_id", Value: 0},
			}},
		},
	}

	cursor, err := c.repo.documentsCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate subcollections: %w", err)
	}
	defer cursor.Close(ctx)

	var subcollectionNames []string
	for cursor.Next(ctx) {
		var result struct {
			SubcollectionID string `bson:"subcollectionId"`
		}
		if err := cursor.Decode(&result); err != nil {
			c.repo.logger.Error("Failed to decode subcollection: %v", err)
			continue
		}
		subcollectionNames = append(subcollectionNames, result.SubcollectionID)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return subcollectionNames, nil
}

// Additional collection-related errors
var (
	ErrCollectionAlreadyExists = fmt.Errorf("collection already exists")
)
