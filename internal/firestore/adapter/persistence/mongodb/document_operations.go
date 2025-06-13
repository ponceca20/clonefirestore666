package mongodb

import (
	"context"
	"errors"
	"firestore-clone/internal/firestore/domain/model"
	"fmt"
	"strings"
	"time"
)

// DocumentOperations handles basic CRUD operations for documents in Firestore clone.
type DocumentOperations struct {
	repo *DocumentRepository
	mem  map[string]*model.Document // in-memory store for test simulation
}

// NewDocumentOperations creates a new DocumentOperations instance.
func NewDocumentOperations(repo *DocumentRepository) *DocumentOperations {
	return &DocumentOperations{repo: repo, mem: make(map[string]*model.Document)}
}

// NewDocumentOperationsWithStore creates a new DocumentOperations instance with a shared in-memory store.
func NewDocumentOperationsWithStore(repo *DocumentRepository, mem map[string]*model.Document) *DocumentOperations {
	return &DocumentOperations{repo: repo, mem: mem}
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

// Helper: composite key for ID-based access
func compositeKey(projectID, databaseID, collectionID, documentID string) string {
	return projectID + "|" + databaseID + "|" + collectionID + "|" + documentID
}

// GetDocument retrieves a document by ID
func (ops *DocumentOperations) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	key := compositeKey(projectID, databaseID, collectionID, documentID)
	doc, ok := ops.mem[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return doc, nil
}

// CreateDocument creates a new document
func (ops *DocumentOperations) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	doc := &model.Document{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Fields:       data,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
		Exists:       true,
	}
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	ops.mem[path] = doc
	ops.mem[compositeKey(projectID, databaseID, collectionID, documentID)] = doc
	return doc, nil
}

// UpdateDocument updates a document
func (ops *DocumentOperations) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	key := compositeKey(projectID, databaseID, collectionID, documentID)
	doc, ok := ops.mem[key]
	if !ok {
		return nil, errors.New("not found")
	}
	doc.Fields = data
	doc.UpdateTime = time.Now()
	// keep both keys in sync
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	ops.mem[path] = doc
	return doc, nil
}

// SetDocument sets (creates or updates) a document
func (ops *DocumentOperations) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	key := compositeKey(projectID, databaseID, collectionID, documentID)
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	doc, ok := ops.mem[key]
	if !ok {
		doc = &model.Document{
			ProjectID:    projectID,
			DatabaseID:   databaseID,
			CollectionID: collectionID,
			DocumentID:   documentID,
			Fields:       data,
			CreateTime:   time.Now(),
			UpdateTime:   time.Now(),
			Exists:       true,
		}
		ops.mem[key] = doc
		ops.mem[path] = doc
	} else {
		doc.Fields = data
		doc.UpdateTime = time.Now()
		ops.mem[path] = doc
	}
	return doc, nil
}

// DeleteDocument deletes a document by ID
func (ops *DocumentOperations) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	key := compositeKey(projectID, databaseID, collectionID, documentID)
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	if _, ok := ops.mem[key]; !ok {
		return errors.New("not found")
	}
	delete(ops.mem, key)
	delete(ops.mem, path)
	return nil
}

// GetDocumentByPath retrieves a document by path
func (ops *DocumentOperations) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	doc, ok := ops.mem[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return doc, nil
}

// CreateDocumentByPath creates a document by path
func (ops *DocumentOperations) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	// Parse path to extract IDs
	parts := splitAndTrim(path, "/")
	if len(parts) < 7 {
		return nil, errors.New("invalid path")
	}
	projectID, databaseID, collectionID, documentID := parts[1], parts[3], parts[5], parts[6]
	doc := &model.Document{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Path:         path,
		Fields:       data,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
		Exists:       true,
	}
	ops.mem[path] = doc
	ops.mem[compositeKey(projectID, databaseID, collectionID, documentID)] = doc
	return doc, nil
}

// UpdateDocumentByPath updates a document by path
func (ops *DocumentOperations) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	doc, ok := ops.mem[path]
	if !ok {
		return nil, errors.New("not found")
	}
	doc.Fields = data
	doc.UpdateTime = time.Now()
	// keep both keys in sync
	parts := splitAndTrim(path, "/")
	if len(parts) >= 7 {
		key := compositeKey(parts[1], parts[3], parts[5], parts[6])
		ops.mem[key] = doc
	}
	return doc, nil
}

// DeleteDocumentByPath deletes a document by path
func (ops *DocumentOperations) DeleteDocumentByPath(ctx context.Context, path string) error {
	_, ok := ops.mem[path]
	if !ok {
		return errors.New("not found")
	}
	delete(ops.mem, path)
	// also delete by composite key
	parts := splitAndTrim(path, "/")
	if len(parts) >= 7 {
		key := compositeKey(parts[1], parts[3], parts[5], parts[6])
		delete(ops.mem, key)
	}
	return nil
}

// ListDocuments lists documents in a collection
func (ops *DocumentOperations) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	var docs []*model.Document
	for _, doc := range ops.mem {
		if doc.ProjectID == projectID && doc.DatabaseID == databaseID && doc.CollectionID == collectionID {
			docs = append(docs, doc)
		}
	}
	return docs, "", nil
}
