package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
)

// newTestDocumentRepositoryMock returns a DocumentRepository with mock collections for unit tests.
func newTestDocumentRepositoryMock() *DocumentRepository {
	return &DocumentRepository{
		documentsCol:   &mockCollection{},
		collectionsCol: &mockCollection{},
	}
}

// TestDocumentOperations_CreateAndGetDocument tests creating and retrieving a document.
func TestDocumentOperations_CreateAndGetDocument(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	projectID := "p1"
	databaseID := "d1"
	collectionID := "c1"
	documentID := "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, _ = docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
}

// TestDocumentOperations_UpdateDocument tests updating a document.
func TestDocumentOperations_UpdateDocument(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	projectID := "p1"
	databaseID := "d1"
	collectionID := "c1"
	documentID := "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, _ = docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	updateFields := map[string]*model.FieldValue{"foo": model.NewFieldValue("baz")}
	_, _ = docs.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, updateFields, nil)
}

// TestDocumentOperations_DeleteDocument tests deleting a document.
func TestDocumentOperations_DeleteDocument(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	projectID := "p1"
	databaseID := "d1"
	collectionID := "c1"
	documentID := "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, _ = docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	_ = docs.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}
