package mongodb

import (
	"context"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// ResourceAccessor implements repository.ResourceAccessor for MongoDB
type ResourceAccessor struct {
	db  *mongo.Database
	log logger.Logger
}

// NewResourceAccessor creates a new ResourceAccessor
func NewResourceAccessor(db *mongo.Database, log logger.Logger) repository.ResourceAccessor {
	return &ResourceAccessor{
		db:  db,
		log: log,
	}
}

// GetDocument retrieves a document by path for use in get() function
func (r *ResourceAccessor) GetDocument(ctx context.Context, projectID, databaseID, path string) (map[string]interface{}, error) {
	// Parse the path to extract collection and document ID
	collection, docID, err := r.parsePath(path)
	if err != nil {
		r.log.Error("Failed to parse document path",
			zap.String("path", path),
			zap.Error(err))
		return nil, err
	}

	// Build the collection name with tenant prefix
	collectionName := r.buildCollectionName(projectID, databaseID, collection)

	// Query the document
	filter := bson.M{"_id": docID}
	var result bson.M
	err = r.db.Collection(collectionName).FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.log.Debug("Document not found for get() function",
				zap.String("path", path),
				zap.String("collection", collectionName))
			return nil, nil
		}
		r.log.Error("Failed to get document",
			zap.String("path", path),
			zap.String("collection", collectionName),
			zap.Error(err))
		return nil, err
	}

	// Convert to map[string]interface{} and remove MongoDB-specific fields
	doc := make(map[string]interface{})
	for key, value := range result {
		if key != "_id" && key != "project_id" && key != "database_id" && key != "created_at" && key != "updated_at" {
			doc[key] = value
		}
	}

	r.log.Debug("Retrieved document for get() function",
		zap.String("path", path),
		zap.String("collection", collectionName))

	return doc, nil
}

// ExistsDocument checks if a document exists for use in exists() function
func (r *ResourceAccessor) ExistsDocument(ctx context.Context, projectID, databaseID, path string) (bool, error) {
	// Parse the path to extract collection and document ID
	collection, docID, err := r.parsePath(path)
	if err != nil {
		r.log.Error("Failed to parse document path for exists check",
			zap.String("path", path),
			zap.Error(err))
		return false, err
	}

	// Build the collection name with tenant prefix
	collectionName := r.buildCollectionName(projectID, databaseID, collection)

	// Check if document exists
	filter := bson.M{"_id": docID}
	count, err := r.db.Collection(collectionName).CountDocuments(ctx, filter)
	if err != nil {
		r.log.Error("Failed to check document existence",
			zap.String("path", path),
			zap.String("collection", collectionName),
			zap.Error(err))
		return false, err
	}

	exists := count > 0
	r.log.Debug("Checked document existence for exists() function",
		zap.String("path", path),
		zap.String("collection", collectionName),
		zap.Bool("exists", exists))

	return exists, nil
}

// ParsePath parses a Firestore path to extract collection and document ID (exposed for testing)
func (r *ResourceAccessor) ParsePath(path string) (string, string, error) {
	return r.parsePath(path)
}

// parsePath parses a Firestore path to extract collection and document ID
// Example: "/users/123" -> collection: "users", docID: "123"
// Example: "/users/123/posts/456" -> collection: "users.posts", docID: "456"
func (r *ResourceAccessor) parsePath(path string) (string, string, error) {
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Split path into segments
	segments := make([]string, 0)
	if path != "" {
		for _, segment := range strings.Split(path, "/") {
			if segment != "" {
				segments = append(segments, segment)
			}
		}
	}

	if len(segments) < 2 {
		return "", "", fmt.Errorf("invalid document path, must have at least collection and document ID")
	}

	// For subcollections, join collection segments with dots
	// users/123/posts/456 -> collection: "users.posts", docID: "456"
	collectionSegments := make([]string, 0)
	for i := 0; i < len(segments)-1; i += 2 {
		collectionSegments = append(collectionSegments, segments[i])
	}

	collection := strings.Join(collectionSegments, ".")
	docID := segments[len(segments)-1]

	return collection, docID, nil
}

// buildCollectionName builds the full collection name with tenant prefix
func (r *ResourceAccessor) buildCollectionName(projectID, databaseID, collection string) string {
	return fmt.Sprintf("%s_%s_%s", projectID, databaseID, collection)
}
