package mongodb

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CollectionOperations handles CRUD for collections in the optimized architecture.
type CollectionOperations struct {
	repo *DocumentRepository
}

// NewCollectionOperations creates a new CollectionOperations instance.
func NewCollectionOperations(repo *DocumentRepository) *CollectionOperations {
	return &CollectionOperations{repo: repo}
}

// MongoCollection represents the MongoDB collection structure
type MongoCollection struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	ProjectID     string             `bson:"projectID"`
	DatabaseID    string             `bson:"databaseID"`
	CollectionID  string             `bson:"collectionID"`
	Path          string             `bson:"path"`
	ParentPath    string             `bson:"parentPath"`
	DocumentCount int64              `bson:"documentCount"`
	StorageSize   int64              `bson:"storageSize"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
	IsActive      bool               `bson:"isActive"`
}

// Helper: convert MongoCollection to model.Collection
func mongoToModelCollection(mongoCol *MongoCollection) *model.Collection {
	return &model.Collection{
		ID:            mongoCol.ID,
		ProjectID:     mongoCol.ProjectID,
		DatabaseID:    mongoCol.DatabaseID,
		CollectionID:  mongoCol.CollectionID,
		Path:          mongoCol.Path,
		ParentPath:    mongoCol.ParentPath,
		DocumentCount: mongoCol.DocumentCount,
		StorageSize:   mongoCol.StorageSize,
		CreatedAt:     mongoCol.CreatedAt,
		UpdatedAt:     mongoCol.UpdatedAt,
		IsActive:      mongoCol.IsActive,
	}
}

func (ops *CollectionOperations) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	now := time.Now()

	// Build the collection path
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collection.CollectionID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents", projectID, databaseID)

	mongoCol := &MongoCollection{
		ID:            primitive.NewObjectID(),
		ProjectID:     projectID,
		DatabaseID:    databaseID,
		CollectionID:  collection.CollectionID,
		Path:          path,
		ParentPath:    parentPath,
		DocumentCount: 0,
		StorageSize:   0,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsActive:      true,
	}

	// Insert into MongoDB
	_, err := ops.repo.collectionsCol.InsertOne(ctx, mongoCol)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Log the creation
	ops.repo.logger.Info(fmt.Sprintf("Collection created successfully in MongoDB - collectionID: %s", collection.CollectionID))

	return nil
}

func (ops *CollectionOperations) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
	}

	var mongoCol MongoCollection
	err := ops.repo.collectionsCol.FindOne(ctx, filter).Decode(&mongoCol)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	return mongoToModelCollection(&mongoCol), nil
}

func (ops *CollectionOperations) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collection.CollectionID,
	}

	update := bson.M{
		"$set": bson.M{
			"documentCount": collection.DocumentCount,
			"storageSize":   collection.StorageSize,
			"updatedAt":     time.Now(),
			"isActive":      collection.IsActive,
		},
	}

	_, err := ops.repo.collectionsCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update collection: %w", err)
	}

	return nil
}

func (ops *CollectionOperations) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
	}

	result, err := ops.repo.collectionsCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	if result.Deleted() == 0 {
		return ErrCollectionNotFound
	}

	return nil
}

func (ops *CollectionOperations) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	filter := bson.M{
		"projectID":  projectID,
		"databaseID": databaseID,
		"isActive":   true,
	}

	cursor, err := ops.repo.collectionsCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer cursor.Close(ctx)

	var collections []*model.Collection
	for cursor.Next(ctx) {
		var mongoCol MongoCollection
		if err := cursor.Decode(&mongoCol); err != nil {
			return nil, fmt.Errorf("failed to decode collection: %w", err)
		}
		collections = append(collections, mongoToModelCollection(&mongoCol))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return collections, nil
}

func (ops *CollectionOperations) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	// For subcollections, we need to find collections that have the parent path matching the document path
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)

	filter := bson.M{
		"parentPath": parentPath,
		"isActive":   true,
	}

	cursor, err := ops.repo.collectionsCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list subcollections: %w", err)
	}
	defer cursor.Close(ctx)

	var subcollectionIDs []string
	for cursor.Next(ctx) {
		var mongoCol MongoCollection
		if err := cursor.Decode(&mongoCol); err != nil {
			return nil, fmt.Errorf("failed to decode subcollection: %w", err)
		}
		subcollectionIDs = append(subcollectionIDs, mongoCol.CollectionID)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return subcollectionIDs, nil
}
