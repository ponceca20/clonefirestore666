package mongodb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DocumentOperations handles basic CRUD operations for documents in Firestore clone.
type DocumentOperations struct {
	repo *DocumentRepository
}

// NewDocumentOperations creates a new DocumentOperations instance.
func NewDocumentOperations(repo *DocumentRepository) *DocumentOperations {
	return &DocumentOperations{repo: repo}
}

// GetDocument retrieves a document by ID.
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

// CreateDocument creates a new document.
func (d *DocumentOperations) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

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

// UpdateDocument updates a document.
func (d *DocumentOperations) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"update_time": now,
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	if len(updateMask) > 0 {
		for _, fieldPath := range updateMask {
			if fieldValue, exists := data[fieldPath]; exists {
				updateDoc["$set"].(bson.M)[fmt.Sprintf("fields.%s", fieldPath)] = fieldValue
			}
		}
	} else {
		for fieldPath, fieldValue := range data {
			updateDoc["$set"].(bson.M)[fmt.Sprintf("fields.%s", fieldPath)] = fieldValue
		}
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedDoc model.Document
	err := d.repo.documentsCol.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&updatedDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &updatedDoc, nil
}

// SetDocument sets (creates or updates) a document.
func (d *DocumentOperations) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	now := time.Now()

	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
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

	opts := options.Replace().SetUpsert(true)
	result, err := d.repo.documentsCol.ReplaceOne(ctx, filter, doc, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to set document: %w", err)
	}

	if result.Matched() == 0 {
		doc.CreateTime = now
		doc.Version = 1
	} else {
		updateDoc := bson.M{"$inc": bson.M{"version": 1}}
		_, _ = d.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	}

	return d.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// DeleteDocument deletes a document by ID.
func (d *DocumentOperations) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	result, err := d.repo.documentsCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.Deleted() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// GetDocumentByPath retrieves a document by path.
func (d *DocumentOperations) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	parsedPath, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}
	return d.GetDocument(ctx, parsedPath.ProjectID, parsedPath.DatabaseID, parsedPath.CollectionID, parsedPath.DocumentID)
}

// CreateDocumentByPath creates a document by path.
func (d *DocumentOperations) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	parsedPath, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}
	return d.CreateDocument(ctx, parsedPath.ProjectID, parsedPath.DatabaseID, parsedPath.CollectionID, parsedPath.DocumentID, data)
}

// UpdateDocumentByPath updates a document by path.
func (d *DocumentOperations) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	parsedPath, err := parseDocumentPath(path)
	if err != nil {
		return nil, err
	}
	return d.UpdateDocument(ctx, parsedPath.ProjectID, parsedPath.DatabaseID, parsedPath.CollectionID, parsedPath.DocumentID, data, updateMask)
}

// DeleteDocumentByPath deletes a document by path.
func (d *DocumentOperations) DeleteDocumentByPath(ctx context.Context, path string) error {
	parsedPath, err := parseDocumentPath(path)
	if err != nil {
		return err
	}
	return d.DeleteDocument(ctx, parsedPath.ProjectID, parsedPath.DatabaseID, parsedPath.CollectionID, parsedPath.DocumentID)
}

// ListDocuments lists documents in a collection with pagination and ordering.
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
		opts.SetLimit(int64(pageSize))
	}

	if orderBy != "" {
		direction := 1
		if strings.HasPrefix(orderBy, "-") {
			direction = -1
			orderBy = strings.TrimPrefix(orderBy, "-")
		}
		opts.SetSort(bson.D{{Key: orderBy, Value: direction}})
	} else {
		opts.SetSort(bson.D{{Key: "document_id", Value: 1}})
	}

	if pageToken != "" {
		opts.SetSkip(int64(parsePageToken(pageToken)))
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
			return nil, "", fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, "", fmt.Errorf("cursor error: %w", err)
	}

	nextPageToken := ""
	if int32(len(documents)) == pageSize {
		nextPageToken = generatePageToken(len(documents))
	}

	return documents, nextPageToken, nil
}

// DocumentPath represents parsed document path components.
type DocumentPath struct {
	ProjectID    string
	DatabaseID   string
	CollectionID string
	DocumentID   string
}

// parseDocumentPath parses a Firestore document path.
func parseDocumentPath(path string) (*DocumentPath, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid document path format: %s", path)
	}
	if parts[0] != "projects" || parts[2] != "databases" || parts[4] != "documents" {
		return nil, fmt.Errorf("invalid document path format: %s", path)
	}
	return &DocumentPath{
		ProjectID:    parts[1],
		DatabaseID:   parts[3],
		CollectionID: parts[5],
		DocumentID:   parts[6],
	}, nil
}

// parsePageToken decodes a page token (simplified).
func parsePageToken(token string) int {
	if offset, err := strconv.Atoi(token); err == nil {
		return offset
	}
	return 0
}

// generatePageToken encodes a page token (simplified).
func generatePageToken(offset int) string {
	return strconv.Itoa(offset)
}
