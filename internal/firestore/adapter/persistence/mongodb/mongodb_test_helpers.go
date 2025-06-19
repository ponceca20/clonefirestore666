package mongodb

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"firestore-clone/internal/firestore/usecase"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Nota: ErrDocumentNotFound se define en document_repo.go

// --- Shared test mocks for mongodb package ---
// These mocks implement the correct CollectionInterface signatures for Firestore clone tests.

// MockDocumentStore simulates an in-memory MongoDB document store for testing
type MockDocumentStore struct {
	documents map[string]*MongoDocumentFlat
	mutex     sync.RWMutex
}

// Clear removes all documents from the store
func (store *MockDocumentStore) Clear() {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.documents = make(map[string]*MongoDocumentFlat)
}

// mockSingleResultWithData implementa SingleResultInterface con datos
type mockSingleResultWithData struct {
	data interface{}
	err  error
}

func (m *mockSingleResultWithData) Decode(v interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.data == nil {
		return ErrDocumentNotFound
	}

	// Si el destino es un puntero a MongoDocumentFlat
	if doc, ok := v.(*MongoDocumentFlat); ok {
		if flatDoc, ok := m.data.(*MongoDocumentFlat); ok {
			*doc = *flatDoc
			return nil
		}
	}

	return fmt.Errorf("decode type mismatch: cannot decode %T to %T", m.data, v)
}

// mockCursorWithData implementa CursorInterface con datos
type mockCursorWithData struct {
	docs    []*MongoDocumentFlat
	current int
	closed  bool
}

func (m *mockCursorWithData) Next(ctx context.Context) bool {
	if m.closed || m.current >= len(m.docs) {
		return false
	}
	m.current++
	return true
}

func (m *mockCursorWithData) Decode(v interface{}) error {
	if m.closed {
		return fmt.Errorf("cursor is closed")
	}
	if m.current == 0 || m.current > len(m.docs) {
		return ErrDocumentNotFound
	}

	doc := m.docs[m.current-1]
	if ptr, ok := v.(*MongoDocumentFlat); ok {
		*ptr = *doc
		return nil
	}

	return fmt.Errorf("decode type mismatch: cannot decode %T to %T", doc, v)
}

func (m *mockCursorWithData) Close(ctx context.Context) error {
	m.closed = true
	return nil
}

func (m *mockCursorWithData) All(ctx context.Context, results interface{}) error {
	return nil
}

func (m *mockCursorWithData) Err() error {
	return nil
}

func (m *mockCursorWithData) ID() int64 {
	return 0
}

// mockUpdateResult implementa UpdateResultInterface
type mockUpdateResult struct {
	MatchedCount  int64
	ModifiedCount int64
	UpsertedCount int64
	UpsertedID    interface{}
}

func (m *mockUpdateResult) Matched() int64 { return m.MatchedCount }

// mockDeleteResult implementa DeleteResultInterface
type mockDeleteResult struct {
	DeletedCount int64
}

func (m *mockDeleteResult) Deleted() int64 { return m.DeletedCount }

// NewMockDocumentStore creates a new MockDocumentStore for testing
func NewMockDocumentStore() *MockDocumentStore {
	return &MockDocumentStore{
		documents: make(map[string]*MongoDocumentFlat),
	}
}

func (m *MockDocumentStore) generateKey(projectID, databaseID, collectionID, documentID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", projectID, databaseID, collectionID, documentID)
}

// MockCollectionWithStore implements CollectionInterface with state management
type MockCollectionWithStore struct {
	store *MockDocumentStore
}

// NewMockCollectionWithStore creates a new MockCollectionWithStore for testing
func NewMockCollectionWithStore() *MockCollectionWithStore {
	return &MockCollectionWithStore{
		store: NewMockDocumentStore(),
	}
}

func (m *MockCollectionWithStore) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()
	return int64(len(m.store.documents)), nil
}

func (m *MockCollectionWithStore) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	// Handle MongoDocumentFlat insertion
	if mongoDoc, ok := doc.(*MongoDocumentFlat); ok {
		key := m.store.generateKey(mongoDoc.ProjectID, mongoDoc.DatabaseID, mongoDoc.CollectionID, mongoDoc.DocumentID)
		mongoDoc.ID = primitive.NewObjectID()
		mongoDoc.CreateTime = time.Now()
		mongoDoc.UpdateTime = time.Now()
		mongoDoc.Exists = true
		m.store.documents[key] = mongoDoc
		return mongoDoc.ID, nil
	}

	return nil, fmt.Errorf("unsupported document type: %T", doc)
}

func (m *MockCollectionWithStore) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()

	if bsonFilter, ok := filter.(bson.M); ok {
		// Si el filtro está vacío, devolver cualquier documento
		if len(bsonFilter) == 0 {
			for _, doc := range m.store.documents {
				return &mockSingleResultWithData{data: doc}
			}
		}

		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		if projectID != "" && databaseID != "" && collectionID != "" && documentID != "" {
			key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
			if doc, exists := m.store.documents[key]; exists {
				return &mockSingleResultWithData{data: doc}
			}
		}
	}

	return &mockSingleResultWithData{err: ErrDocumentNotFound}
}

func (m *MockCollectionWithStore) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
		if doc, exists := m.store.documents[key]; exists {
			if updateDoc, ok := update.(bson.M); ok {
				if setDoc, ok := updateDoc["$set"].(bson.M); ok {
					// Actualizar campos
					if fields, ok := setDoc["fields"].(map[string]interface{}); ok {
						doc.Fields = fields
					}
					doc.UpdateTime = time.Now()
					return &mockUpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
				}
			}
		}
	}

	return &mockUpdateResult{MatchedCount: 0, ModifiedCount: 0}, ErrDocumentNotFound
}

func (m *MockCollectionWithStore) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
		if _, exists := m.store.documents[key]; exists {
			if mongoDoc, ok := replacement.(*MongoDocumentFlat); ok {
				mongoDoc.UpdateTime = time.Now()
				m.store.documents[key] = mongoDoc
				return &mockUpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
			}
		}
	}

	return &mockUpdateResult{MatchedCount: 0, ModifiedCount: 0}, ErrDocumentNotFound
}

func (m *MockCollectionWithStore) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
		if _, exists := m.store.documents[key]; exists {
			delete(m.store.documents, key)
			return &mockDeleteResult{DeletedCount: 1}, nil
		}
	}

	return &mockDeleteResult{DeletedCount: 0}, ErrDocumentNotFound
}

func (m *MockCollectionWithStore) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()

	var matchingDocs []*MongoDocumentFlat

	if bsonFilter, ok := filter.(bson.M); ok {
		// Si el filtro está vacío, devolver todos los documentos
		if len(bsonFilter) == 0 {
			for _, doc := range m.store.documents {
				matchingDocs = append(matchingDocs, doc)
			}
		} else {
			projectID, _ := bsonFilter["projectID"].(string)
			databaseID, _ := bsonFilter["databaseID"].(string)
			collectionID, _ := bsonFilter["collectionID"].(string)

			// Handle pagination filter: documentID: {$gt: "value"}
			var pageTokenFilter string
			if docIDFilter, ok := bsonFilter["documentID"].(bson.M); ok {
				if gtValue, ok := docIDFilter["$gt"].(string); ok {
					pageTokenFilter = gtValue
				}
			}

			for _, doc := range m.store.documents {
				// Apply basic filters if they exist
				if projectID != "" && doc.ProjectID != projectID {
					continue
				}
				if databaseID != "" && doc.DatabaseID != databaseID {
					continue
				}
				if collectionID != "" && doc.CollectionID != collectionID {
					continue
				}

				// Apply pagination filter if it exists
				if pageTokenFilter != "" && doc.DocumentID <= pageTokenFilter {
					continue
				}

				matchingDocs = append(matchingDocs, doc)
			}
		}
	}
	// Apply FindOptions: sort and limit
	if len(opts) > 0 && opts[0] != nil {
		opt := opts[0]

		// Apply sort - simple implementation for documentID sorting
		if opt.Sort != nil {
			// For simplicity, we'll implement basic documentID sorting
			// This is sufficient for the pagination test which sorts by documentID
			sort.Slice(matchingDocs, func(i, j int) bool {
				return matchingDocs[i].DocumentID < matchingDocs[j].DocumentID
			})
		}

		// Apply limit
		if opt.Limit != nil && *opt.Limit > 0 {
			limit := int(*opt.Limit)
			if len(matchingDocs) > limit {
				matchingDocs = matchingDocs[:limit]
			}
		}
	}

	return &mockCursorWithData{
		docs:    matchingDocs,
		current: 0,
	}, nil
}

func (m *MockCollectionWithStore) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}

func (m *MockCollectionWithStore) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
		if doc, exists := m.store.documents[key]; exists {
			if updateDoc, ok := update.(bson.M); ok {
				if setDoc, ok := updateDoc["$set"].(bson.M); ok {
					if fields, ok := setDoc["fields"].(map[string]interface{}); ok {
						doc.Fields = fields
					}
					doc.UpdateTime = time.Now()
					return &mockSingleResultWithData{data: doc}
				}
			}
		}
	}

	return &mockSingleResultWithData{err: ErrDocumentNotFound}
}

// Basic mocks for simple cases where store functionality is not needed
type MockCollection struct{}

func (m *MockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}
func (m *MockCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return nil, nil
}
func (m *MockCollection) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &mockSingleResult{}
}
func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *MockCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return &mockUpdateResult{MatchedCount: 1}, nil
}
func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	return &mockDeleteResult{DeletedCount: 1}, nil
}
func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *MockCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &mockCursor{}, nil
}
func (m *MockCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &mockSingleResult{}
}

type mockSingleResult struct{}

func (m *mockSingleResult) Decode(v interface{}) error { return nil }
func (m *mockSingleResult) Err() error                 { return nil }

type mockCursor struct{}

func (m *mockCursor) Next(ctx context.Context) bool                      { return false }
func (m *mockCursor) Decode(val interface{}) error                       { return nil }
func (m *mockCursor) Close(ctx context.Context) error                    { return nil }
func (m *mockCursor) Err() error                                         { return nil }
func (m *mockCursor) All(ctx context.Context, results interface{}) error { return nil }

// MockDatabaseProviderForOps implements DatabaseProvider for document operations tests
type MockDatabaseProviderForOps struct {
	store *MockDocumentStore
}

func (m *MockDatabaseProviderForOps) Collection(name string) CollectionInterface {
	return &MockCollectionWithStore{store: m.store}
}

func (m *MockDatabaseProviderForOps) Client() interface{} {
	return nil
}

// NewTestDocumentRepositoryForOps creates a DocumentRepository with functional mocks for document operations tests
func NewTestDocumentRepositoryForOps() *DocumentRepository {
	mockStore := NewMockDocumentStore()
	mockCol := &MockCollectionWithStore{store: mockStore}
	mockDB := &MockDatabaseProviderForOps{store: mockStore}

	repo := &DocumentRepository{
		db:             mockDB,
		logger:         &usecase.MockLogger{},
		documentsCol:   mockCol,
		collectionsCol: mockCol,
		indexesCol:     mockCol,
	}

	// Initialize document operations to prevent nil pointer dereference
	repo.documentOps = NewDocumentOperations(repo)

	return repo
}

// NewTestDocumentRepositoryForOpsWithCleanup creates a DocumentRepository and returns a cleanup function
func NewTestDocumentRepositoryForOpsWithCleanup() (*DocumentRepository, func()) {
	mockStore := NewMockDocumentStore()
	mockCol := &MockCollectionWithStore{store: mockStore}
	mockDB := &MockDatabaseProviderForOps{store: mockStore}

	repo := &DocumentRepository{
		db:             mockDB,
		logger:         &usecase.MockLogger{},
		documentsCol:   mockCol,
		collectionsCol: mockCol,
		indexesCol:     mockCol,
	}

	// Initialize document operations to prevent nil pointer dereference
	repo.documentOps = NewDocumentOperations(repo)

	cleanup := func() {
		mockStore.Clear()
	}

	return repo, cleanup
}
