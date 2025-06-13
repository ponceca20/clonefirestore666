package mongodb

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Shared test mocks for mongodb package ---
// These mocks implement the correct CollectionInterface signatures for Firestore clone tests.
//
// NOTE: Do NOT redeclare mockCollection if it is already declared in another test file in this package.
// To avoid redeclaration errors, only define these mocks in one place and import/use them in other test files.
// If you need to share mocks, move them to a single file (e.g., mongodb_test_helpers.go) and remove from other test files.
//
// Remove this file's mockCollection if it is already defined in collection_operations_test.go.
//
// If you want to keep all mocks here, delete the duplicate definition from collection_operations_test.go.

// These mocks implement the correct CollectionInterface signatures for Firestore clone tests.
type mockCollection struct{}

func (m *mockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}
func (m *mockCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return nil, nil
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

type mockSingleResult struct{}

func (m *mockSingleResult) Decode(v interface{}) error { return nil }

type mockUpdateResult struct{ MatchedCount int64 }

func (m *mockUpdateResult) Matched() int64 { return m.MatchedCount }

type mockDeleteResult struct{ DeletedCount int64 }

func (m *mockDeleteResult) Deleted() int64 { return m.DeletedCount }

type mockCursor struct{}

func (m *mockCursor) Next(ctx context.Context) bool   { return false }
func (m *mockCursor) Decode(val interface{}) error    { return nil }
func (m *mockCursor) Close(ctx context.Context) error { return nil }
func (m *mockCursor) Err() error                      { return nil }

// newTestDocumentRepositoryMockWithStore returns a DocumentRepository with a provided shared in-memory store for DocumentOperations.
func newTestDocumentRepositoryMockWithStore(store map[string]*model.Document) *DocumentRepository {
	repo := &DocumentRepository{}
	docOps := NewDocumentOperationsWithStore(repo, store)
	repo.documentOps = docOps
	repo.documentsCol = &mockCollection{}
	repo.collectionsCol = &mockCollection{}
	return repo
}

// newTestDocumentRepositoryMock returns a DocumentRepository with a new in-memory store for DocumentOperations.
func newTestDocumentRepositoryMock() *DocumentRepository {
	return newTestDocumentRepositoryMockWithStore(make(map[string]*model.Document))
}
