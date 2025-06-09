package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DocumentOperations handles basic CRUD operations for documents
type DocumentOperations struct {
	repo *DocumentRepository
}

// NewDocumentOperations creates a new DocumentOperations instance
func NewDocumentOperations(repo *DocumentRepository) *DocumentOperations {
	return &DocumentOperations{repo: repo}
}

// GetDocument retrieves a document by ID
func (d *DocumentOperations) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	var doc model.Document
	err := d.repo.documentsCol.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// CreateDocument creates a new document
func (d *DocumentOperations) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Check if document already exists
	count, err := d.repo.documentsCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to check document existence: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("document already exists")
	}

	doc := &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Path:         fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID),
		Fields:       data,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}

	_, err = d.repo.documentsCol.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

// UpdateDocument updates a document
func (d *DocumentOperations) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Build update document
	updateDoc := bson.M{
		"$set": bson.M{
			"update_time": now,
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	// Apply field updates based on mask
	if len(updateMask) > 0 {
		setFields := bson.M{}
		for _, field := range updateMask {
			if value, exists := data[field]; exists {
				setFields["fields."+field] = value
			}
		}
		if len(setFields) > 0 {
			for k, v := range setFields {
				updateDoc["$set"].(bson.M)[k] = v
			}
		}
	} else {
		// Update entire fields if no mask specified
		updateDoc["$set"].(bson.M)["fields"] = data
	}

	updateResult, err := d.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	if updateResult.MatchedCount == 0 {
		return nil, ErrDocumentNotFound
	}

	// Retrieve and return updated document
	return d.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// SetDocument sets (creates or updates) a document
func (d *DocumentOperations) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	if merge {
		// Merge mode: update existing fields or create new document
		updateDoc := bson.M{
			"$set": bson.M{
				"update_time": now,
			},
			"$inc": bson.M{
				"version": 1,
			},
			"$setOnInsert": bson.M{
				"_id":           primitive.NewObjectID(),
				"project_id":    projectID,
				"database_id":   databaseID,
				"collection_id": collectionID,
				"document_id":   documentID,
				"path":          fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID),
				"create_time":   now,
				"exists":        true,
			},
		}

		// Merge individual fields
		for field, value := range data {
			updateDoc["$set"].(bson.M)["fields."+field] = value
		}

		opts := options.Update().SetUpsert(true)
		_, err := d.repo.documentsCol.UpdateOne(ctx, filter, updateDoc, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to set document (merge): %w", err)
		}
	} else {
		// Replace mode: replace entire document or create new
		doc := &model.Document{
			ProjectID:    projectID,
			DatabaseID:   databaseID,
			CollectionID: collectionID,
			DocumentID:   documentID,
			Path:         fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID),
			Fields:       data,
			UpdateTime:   now,
			Version:      1,
			Exists:       true,
		}

		update := bson.M{
			"$set": doc,
			"$setOnInsert": bson.M{
				"_id":         primitive.NewObjectID(),
				"create_time": now,
			},
		}

		opts := options.Update().SetUpsert(true)
		_, err := d.repo.documentsCol.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to set document (replace): %w", err)
		}
	}

	// Retrieve and return the document
	return d.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// DeleteDocument deletes a document by ID
func (d *DocumentOperations) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	deleteResult, err := d.repo.documentsCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if deleteResult.DeletedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// GetDocumentByPath retrieves a document by path
func (d *DocumentOperations) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	pathParts, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}

	return d.GetDocument(ctx, pathParts.ProjectID, pathParts.DatabaseID, pathParts.CollectionID, pathParts.DocumentID)
}

// CreateDocumentByPath creates a document by path
func (d *DocumentOperations) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	pathParts, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}

	return d.CreateDocument(ctx, pathParts.ProjectID, pathParts.DatabaseID, pathParts.CollectionID, pathParts.DocumentID, data)
}

// UpdateDocumentByPath updates a document by path
func (d *DocumentOperations) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	pathParts, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}

	return d.UpdateDocument(ctx, pathParts.ProjectID, pathParts.DatabaseID, pathParts.CollectionID, pathParts.DocumentID, data, updateMask)
}

// DeleteDocumentByPath deletes a document by path
func (d *DocumentOperations) DeleteDocumentByPath(ctx context.Context, path string) error {
	pathParts, err := parseDocumentPath(path)
	if err != nil {
		return err
	}

	return d.DeleteDocument(ctx, pathParts.ProjectID, pathParts.DatabaseID, pathParts.CollectionID, pathParts.DocumentID)
}

// ListDocuments lists documents in a collection
func (d *DocumentOperations) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}

	if !showMissing {
		filter["exists"] = true
	}

	opts := options.Find()
	if pageSize > 0 {
		opts.SetLimit(int64(pageSize) + 1) // Get one extra to check if there's a next page
	}

	// Handle pagination
	var skipCount int64 = 0
	if pageToken != "" {
		// Simple base64 decode for skip count
		if decoded, err := primitive.ObjectIDFromHex(pageToken); err == nil {
			filter["_id"] = bson.M{"$gt": decoded}
		}
	}

	// Handle ordering
	sort := bson.D{{Key: "document_id", Value: 1}} // Default sort
	if orderBy != "" {
		switch orderBy {
		case "document_id":
			sort = bson.D{{Key: "document_id", Value: 1}}
		case "document_id desc":
			sort = bson.D{{Key: "document_id", Value: -1}}
		case "create_time":
			sort = bson.D{{Key: "create_time", Value: 1}}
		case "create_time desc":
			sort = bson.D{{Key: "create_time", Value: -1}}
		case "update_time":
			sort = bson.D{{Key: "update_time", Value: 1}}
		case "update_time desc":
			sort = bson.D{{Key: "update_time", Value: -1}}
		}
	}
	opts.SetSort(sort)

	if skipCount > 0 {
		opts.SetSkip(skipCount)
	}

	cursor, err := d.repo.documentsCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*model.Document
	for cursor.Next(ctx) {
		var doc model.Document
		if err := cursor.Decode(&doc); err != nil {
			d.repo.logger.Error("Failed to decode document: %v", err)
			continue
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, "", fmt.Errorf("cursor error: %w", err)
	}

	// Generate next page token
	nextPageToken := ""
	if pageSize > 0 && len(documents) > int(pageSize) {
		// Remove the extra document and use its ID as next page token
		documents = documents[:pageSize]
		if len(documents) > 0 {
			lastDoc := documents[len(documents)-1]
			nextPageToken = lastDoc.ID.Hex()
		}
	}

	return documents, nextPageToken, nil
}

// DocumentPath represents parsed document path components
type DocumentPath struct {
	ProjectID    string
	DatabaseID   string
	CollectionID string
	DocumentID   string
}

// parseDocumentPath parses a Firestore document path
func parseDocumentPath(path string) (*DocumentPath, error) {
	// Expected format: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
	// Example: projects/my-project/databases/(default)/documents/users/user1

	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 6 {
		return nil, ErrInvalidPath
	}

	if parts[0] != "projects" || parts[2] != "databases" || parts[4] != "documents" {
		return nil, ErrInvalidPath
	}

	return &DocumentPath{
		ProjectID:    parts[1],
		DatabaseID:   parts[3],
		CollectionID: parts[5],
		DocumentID:   strings.Join(parts[6:], "/"), // Handle nested document IDs
	}, nil
}
