package mongodb

import (
	"context"
	"errors"
	"firestore-clone/internal/firestore/domain/model"
	"fmt"
	"strings"
	"time"

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

// NewDocumentOperationsWithStore creates a new DocumentOperations instance with shared store.
// This method is kept for compatibility but now uses MongoDB instead of in-memory store.
func NewDocumentOperationsWithStore(repo *DocumentRepository, mem map[string]*model.Document) *DocumentOperations {
	return &DocumentOperations{repo: repo}
}

// MongoDocument represents the MongoDB document structure
type MongoDocument struct {
	ID                primitive.ObjectID           `bson:"_id,omitempty"`
	ProjectID         string                       `bson:"projectID"`
	DatabaseID        string                       `bson:"databaseID"`
	CollectionID      string                       `bson:"collectionID"`
	DocumentID        string                       `bson:"documentID"`
	Path              string                       `bson:"path"`
	ParentPath        string                       `bson:"parentPath"`
	Fields            map[string]*model.FieldValue `bson:"fields"`
	CreateTime        time.Time                    `bson:"createTime"`
	UpdateTime        time.Time                    `bson:"updateTime"`
	ReadTime          time.Time                    `bson:"readTime"`
	Version           int64                        `bson:"version"`
	Exists            bool                         `bson:"exists"`
	HasSubcollections bool                         `bson:"hasSubcollections"`
}

// Helper: parse Firestore path
func parseFirestorePath(path string) (projectID, databaseID, collectionID, documentID string, err error) {
	// projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
	parts := make([]string, 0)
	for _, p := range splitAndTrim(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) < 6 {
		return "", "", "", "", errors.New("invalid path")
	}
	return parts[1], parts[3], parts[5], parts[6], nil
}

func splitAndTrim(s, sep string) []string {
	var out []string
	for _, p := range strings.Split(s, sep) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Helper: convert MongoDocument to model.Document
func mongoToModelDocument(mongoDoc *MongoDocument) *model.Document {
	return &model.Document{
		ID:                mongoDoc.ID,
		ProjectID:         mongoDoc.ProjectID,
		DatabaseID:        mongoDoc.DatabaseID,
		CollectionID:      mongoDoc.CollectionID,
		DocumentID:        mongoDoc.DocumentID,
		Path:              mongoDoc.Path,
		ParentPath:        mongoDoc.ParentPath,
		Fields:            mongoDoc.Fields,
		CreateTime:        mongoDoc.CreateTime,
		UpdateTime:        mongoDoc.UpdateTime,
		ReadTime:          mongoDoc.ReadTime,
		Version:           mongoDoc.Version,
		Exists:            mongoDoc.Exists,
		HasSubcollections: mongoDoc.HasSubcollections,
	}
}

// Helper: convert model.Document to MongoDocument
func modelToMongoDocument(doc *model.Document) *MongoDocument {
	return &MongoDocument{
		ID:                doc.ID,
		ProjectID:         doc.ProjectID,
		DatabaseID:        doc.DatabaseID,
		CollectionID:      doc.CollectionID,
		DocumentID:        doc.DocumentID,
		Path:              doc.Path,
		ParentPath:        doc.ParentPath,
		Fields:            doc.Fields,
		CreateTime:        doc.CreateTime,
		UpdateTime:        doc.UpdateTime,
		ReadTime:          doc.ReadTime,
		Version:           doc.Version,
		Exists:            doc.Exists,
		HasSubcollections: doc.HasSubcollections,
	}
}

// GetDocument retrieves a document by ID
func (ops *DocumentOperations) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	var mongoDoc MongoDocument
	err := targetCollection.FindOne(ctx, filter).Decode(&mongoDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return mongoToModelDocument(&mongoDoc), nil
}

// CreateDocument creates a new document
func (ops *DocumentOperations) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	now := time.Now()

	// Generate a new document ID if not provided
	if documentID == "" {
		documentID = primitive.NewObjectID().Hex()
	}

	// Build the document path
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID)

	doc := &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Path:         path,
		ParentPath:   parentPath,
		Fields:       data,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}

	// Convert to MongoDB document
	mongoDoc := modelToMongoDocument(doc)

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	// Insert into MongoDB
	_, err := targetCollection.InsertOne(ctx, mongoDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Log the creation
	ops.repo.logger.Info(fmt.Sprintf("Document created successfully in MongoDB - documentID: %s", documentID))

	return doc, nil
}

// UpdateDocument updates a document
func (ops *DocumentOperations) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	update := bson.M{
		"$set": bson.M{
			"fields":     data,
			"updateTime": time.Now(),
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var mongoDoc MongoDocument
	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	err := targetCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&mongoDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return mongoToModelDocument(&mongoDoc), nil
}

// SetDocument sets (creates or updates) a document
func (ops *DocumentOperations) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	now := time.Now()

	// Generate a new document ID if not provided
	if documentID == "" {
		documentID = primitive.NewObjectID().Hex()
	}

	// Build the document path
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID)

	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	update := bson.M{
		"$set": bson.M{
			"projectID":    projectID,
			"databaseID":   databaseID,
			"collectionID": collectionID,
			"documentID":   documentID,
			"path":         path,
			"parentPath":   parentPath,
			"fields":       data,
			"updateTime":   now,
			"exists":       true,
		},
		"$setOnInsert": bson.M{
			"createTime": now,
			"version":    1,
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var mongoDoc MongoDocument
	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	err := targetCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&mongoDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to set document: %w", err)
	}

	return mongoToModelDocument(&mongoDoc), nil
}

// DeleteDocument deletes a document by ID
func (ops *DocumentOperations) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	result, err := targetCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.Deleted() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// GetDocumentByPath retrieves a document by path
func (ops *DocumentOperations) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// CreateDocumentByPath creates a document by path
func (ops *DocumentOperations) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
}

// UpdateDocumentByPath updates a document by path
func (ops *DocumentOperations) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
}

// DeleteDocumentByPath deletes a document by path
func (ops *DocumentOperations) DeleteDocumentByPath(ctx context.Context, path string) error {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid path format: %w", err)
	}

	return ops.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// ListDocuments lists documents in a collection
func (ops *DocumentOperations) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"exists":       true,
	}

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	cursor, err := targetCollection.Find(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*model.Document
	for cursor.Next(ctx) {
		var mongoDoc MongoDocument
		if err := cursor.Decode(&mongoDoc); err != nil {
			return nil, "", fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, mongoToModelDocument(&mongoDoc))
	}

	if err := cursor.Err(); err != nil {
		return nil, "", fmt.Errorf("cursor error: %w", err)
	}

	return documents, "", nil
}
