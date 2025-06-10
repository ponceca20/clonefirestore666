package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollection implements the minimal mongo.Collection interface for testing
// Only the methods used in CollectionOperations are mocked

type mockCollection struct{}

func (m *mockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil // Always return 0 collections (does not exist)
}
func (m *mockCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return nil, nil // Simulate successful insert
}
func (m *mockCollection) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &mockSingleResult{}
}
func (m *mockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *mockCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *mockCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	return &mockDeleteResult{DeletedCount: 1}, nil
}
func (m *mockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *mockCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *mockCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &mockSingleResult{}
}

// Mock types for FindOne, UpdateOne, DeleteOne, Find

type mockSingleResult struct{}

func (m *mockSingleResult) Decode(v interface{}) error { return nil }

// Implementación de métodos de interfaz para los mocks

type mockUpdateResult struct{ MatchedCount int64 }

func (m *mockUpdateResult) Matched() int64 { return m.MatchedCount }

type mockDeleteResult struct{ DeletedCount int64 }

func (m *mockDeleteResult) Deleted() int64 { return m.DeletedCount }

type mockCursor struct{}

func (m *mockCursor) Next(ctx context.Context) bool   { return false }
func (m *mockCursor) Decode(val interface{}) error    { return nil }
func (m *mockCursor) Close(ctx context.Context) error { return nil }
func (m *mockCursor) Err() error                      { return nil }

// Usa la definición real de DocumentRepository y CollectionOperations del código de producción, no las redeclares aquí.

// Crea un DocumentRepository con mocks para los tests
func newTestDocumentRepositoryForCollections() *DocumentRepository {
	return &DocumentRepository{
		collectionsCol: &mockCollection{},
		documentsCol:   &mockCollection{},
	}
}

// En los tests, usa el adaptador para crear el repo compatible
func TestCollectionOperations_CreateCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	col := &model.Collection{ID: primitive.NewObjectID(), CollectionID: "c1"}
	_ = ops.CreateCollection(ctx, "p1", "d1", col)
}

func TestCollectionOperations_GetCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_, _ = ops.GetCollection(ctx, "p1", "d1", "c1")
}

func TestCollectionOperations_UpdateCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	col := &model.Collection{ID: primitive.NewObjectID(), CollectionID: "c1"}
	_ = ops.UpdateCollection(ctx, "p1", "d1", col)
}

func TestCollectionOperations_DeleteCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_ = ops.DeleteCollection(ctx, "p1", "d1", "c1")
}

func TestCollectionOperations_ListCollections(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_, _ = ops.ListCollections(ctx, "p1", "d1")
}

// En los tests, reemplaza NewCollectionOperations(repo) por NewCollectionOperations((*DocumentRepository)(repo)) si es necesario,
// o adapta el constructor para aceptar la interfaz en vez de *DocumentRepository.
