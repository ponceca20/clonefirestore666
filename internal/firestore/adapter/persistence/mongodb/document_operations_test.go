package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Use mocks from collection_operations_test.go ---
// No need to redefine mockCollection, mockSingleResult, mockUpdateResult, mockDeleteResult, mockCursor here.
// Ensure collection_operations_test.go is built first or move mocks to a shared test file if needed.

// mockCollectionWithOptions implements CollectionInterface with correct method signatures for options
// (for test only)
type mockCollectionWithOptions struct{}

func (m *mockCollectionWithOptions) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}
func (m *mockCollectionWithOptions) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockCollectionWithOptions) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &mockSingleResult{}
}
func (m *mockCollectionWithOptions) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *mockCollectionWithOptions) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *mockCollectionWithOptions) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	return &mockDeleteResult{DeletedCount: 1}, nil
}
func (m *mockCollectionWithOptions) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *mockCollectionWithOptions) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *mockCollectionWithOptions) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &mockSingleResult{}
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
	_, err := docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	// Test GetDocument
	doc, err := docs.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}
	if doc == nil || doc.DocumentID != documentID {
		t.Errorf("Expected documentID %s, got %+v", documentID, doc)
	}
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
	_, err := docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	updateFields := map[string]*model.FieldValue{"foo": model.NewFieldValue("baz")}
	updatedDoc, err := docs.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, updateFields, nil)
	if err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}
	if updatedDoc == nil || updatedDoc.DocumentID != documentID {
		t.Errorf("Expected updated documentID %s, got %+v", documentID, updatedDoc)
	}
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
	_, err := docs.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	err = docs.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
	// Optionally, try to get the document and expect a not found or nil result
}

// TestDocumentOperations_GetDocumentByPath tests retrieving a document by path.
func TestDocumentOperations_GetDocumentByPath(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	path := "projects/p1/databases/d1/documents/c1/doc1"

	// First create a document
	data := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, err := docs.CreateDocumentByPath(ctx, path, data)
	if err != nil {
		t.Fatalf("CreateDocumentByPath failed: %v", err)
	}

	// Then try to get it by path
	doc, err := docs.GetDocumentByPath(ctx, path)
	if err != nil {
		t.Fatalf("GetDocumentByPath failed: %v", err)
	}
	if doc == nil || doc.Path != path {
		t.Errorf("Expected path %s, got %+v", path, doc)
	}
	if doc.Fields["foo"].Value != "bar" {
		t.Errorf("Expected field foo=bar, got %+v", doc.Fields["foo"])
	}
}

// TestDocumentOperations_ListDocuments tests listing documents in a collection.
func TestDocumentOperations_ListDocuments(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	projectID := "p1"
	databaseID := "d1"
	collectionID := "c1"
	docs.CreateDocument(ctx, projectID, databaseID, collectionID, "doc1", map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")})
	docs.CreateDocument(ctx, projectID, databaseID, collectionID, "doc2", map[string]*model.FieldValue{"foo": model.NewFieldValue("baz")})
	docsList, _, err := docs.ListDocuments(ctx, projectID, databaseID, collectionID, 10, "", "", false)
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}
	if len(docsList) < 2 {
		t.Errorf("Expected at least 2 documents, got %d", len(docsList))
	}
}

// TestDocumentOperations_SetDocument tests setting (create or update) a document.
func TestDocumentOperations_SetDocument(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	projectID := "p1"
	databaseID := "d1"
	collectionID := "c1"
	documentID := "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	// Set as create
	doc, err := docs.SetDocument(ctx, projectID, databaseID, collectionID, documentID, fields, false)
	if err != nil {
		t.Fatalf("SetDocument (create) failed: %v", err)
	}
	if doc == nil || doc.DocumentID != documentID {
		t.Errorf("Expected documentID %s, got %+v", documentID, doc)
	}
	// Set as update (merge)
	fields["foo"] = model.NewFieldValue("baz")
	doc, err = docs.SetDocument(ctx, projectID, databaseID, collectionID, documentID, fields, true)
	if err != nil {
		t.Fatalf("SetDocument (update) failed: %v", err)
	}
}

// TestDocumentOperations_CreateDocumentByPath tests creating a document by path.
func TestDocumentOperations_CreateDocumentByPath(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	path := "projects/p1/databases/d1/documents/c1/doc2"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	doc, err := docs.CreateDocumentByPath(ctx, path, fields)
	if err != nil {
		t.Fatalf("CreateDocumentByPath failed: %v", err)
	}
	if doc == nil || doc.Path != path {
		t.Errorf("Expected path %s, got %+v", path, doc)
	}
}

// TestDocumentOperations_UpdateDocumentByPath tests updating a document by path.
func TestDocumentOperations_UpdateDocumentByPath(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	path := "projects/p1/databases/d1/documents/c1/doc2"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, _ = docs.CreateDocumentByPath(ctx, path, fields)
	updateFields := map[string]*model.FieldValue{"foo": model.NewFieldValue("baz")}
	doc, err := docs.UpdateDocumentByPath(ctx, path, updateFields, nil)
	if err != nil {
		t.Fatalf("UpdateDocumentByPath failed: %v", err)
	}
	if doc == nil || doc.Path != path {
		t.Errorf("Expected path %s, got %+v", path, doc)
	}
}

// TestDocumentOperations_DeleteDocumentByPath tests deleting a document by path.
func TestDocumentOperations_DeleteDocumentByPath(t *testing.T) {
	repo := newTestDocumentRepositoryMock()
	docs := NewDocumentOperations(repo)
	ctx := context.Background()
	path := "projects/p1/databases/d1/documents/c1/doc2"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	_, _ = docs.CreateDocumentByPath(ctx, path, fields)
	err := docs.DeleteDocumentByPath(ctx, path)
	if err != nil {
		t.Fatalf("DeleteDocumentByPath failed: %v", err)
	}
}
