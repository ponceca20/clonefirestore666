package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Shared test mocks for mongodb package ---
// These mocks implement the correct CollectionInterface signatures for Firestore clone tests.

// mockDocumentStore simulates an in-memory MongoDB document store for testing
type mockDocumentStore struct {
	documents map[string]interface{}
	mutex     sync.RWMutex
}

func newMockDocumentStore() *mockDocumentStore {
	return &mockDocumentStore{
		documents: make(map[string]interface{}),
	}
}

func (m *mockDocumentStore) generateKey(projectID, databaseID, collectionID, documentID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", projectID, databaseID, collectionID, documentID)
}

// mockCollectionWithStore implements CollectionInterface with state management
type mockCollectionWithStore struct {
	store *mockDocumentStore
}

func (m *mockCollectionWithStore) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()
	return int64(len(m.store.documents)), nil
}

func (m *mockCollectionWithStore) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	// Handle MongoDocument insertion
	if mongoDoc, ok := doc.(MongoDocument); ok {
		key := m.store.generateKey(mongoDoc.ProjectID, mongoDoc.DatabaseID, mongoDoc.CollectionID, mongoDoc.DocumentID)
		mongoDoc.ID = primitive.NewObjectID()
		mongoDoc.CreateTime = time.Now()
		mongoDoc.UpdateTime = time.Now()
		mongoDoc.Exists = true
		m.store.documents[key] = mongoDoc
		return primitive.NewObjectID(), nil
	}

	// Try with pointer to MongoDocument
	if mongoDocPtr, ok := doc.(*MongoDocument); ok {
		mongoDoc := *mongoDocPtr
		key := m.store.generateKey(mongoDoc.ProjectID, mongoDoc.DatabaseID, mongoDoc.CollectionID, mongoDoc.DocumentID)
		mongoDoc.ID = primitive.NewObjectID()
		mongoDoc.CreateTime = time.Now()
		mongoDoc.UpdateTime = time.Now()
		mongoDoc.Exists = true
		m.store.documents[key] = mongoDoc
		return primitive.NewObjectID(), nil
	}

	// Handle generic document insertion
	m.store.documents[fmt.Sprintf("doc_%d", len(m.store.documents))] = doc
	return primitive.NewObjectID(), nil
}

func (m *mockCollectionWithStore) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()
	// Handle BSON filter for finding documents
	if bsonFilter, ok := filter.(bson.M); ok {
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

	return &mockSingleResultWithData{err: fmt.Errorf("document not found")}
}

func (m *mockCollectionWithStore) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	// Handle BSON filter for updating documents
	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		if projectID != "" && databaseID != "" && collectionID != "" && documentID != "" {
			key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
			if doc, exists := m.store.documents[key]; exists {
				if mongoDoc, ok := doc.(MongoDocument); ok {
					mongoDoc.UpdateTime = time.Now()
					mongoDoc.Version++

					// Handle update operations
					if updateDoc, ok := update.(bson.M); ok {
						if setFields, ok := updateDoc["$set"].(bson.M); ok {
							if fields, ok := setFields["fields"].(map[string]*model.FieldValue); ok {
								mongoDoc.Fields = fields
							}
							if updateTime, ok := setFields["updateTime"].(time.Time); ok {
								mongoDoc.UpdateTime = updateTime
							}
						}
						if incFields, ok := updateDoc["$inc"].(bson.M); ok {
							if versionInc, ok := incFields["version"].(int); ok {
								mongoDoc.Version += int64(versionInc)
							}
						}
					}

					m.store.documents[key] = mongoDoc
					return &mockUpdateResult{MatchedCount: 1}, nil
				}
			}
		}
	}

	return &mockUpdateResult{MatchedCount: 0}, nil
}

func (m *mockCollectionWithStore) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return m.UpdateOne(ctx, filter, replacement)
}

func (m *mockCollectionWithStore) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	// Handle BSON filter for deleting documents
	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		if projectID != "" && databaseID != "" && collectionID != "" && documentID != "" {
			key := m.store.generateKey(projectID, databaseID, collectionID, documentID)
			if _, exists := m.store.documents[key]; exists {
				delete(m.store.documents, key)
				return &mockDeleteResult{DeletedCount: 1}, nil
			}
		}
	}

	return &mockDeleteResult{DeletedCount: 0}, nil
}

func (m *mockCollectionWithStore) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	m.store.mutex.RLock()
	defer m.store.mutex.RUnlock()

	var docs []interface{}
	for _, doc := range m.store.documents {
		docs = append(docs, doc)
	}

	return &mockCursorWithData{docs: docs}, nil
}

func (m *mockCollectionWithStore) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return m.Find(ctx, bson.M{})
}

func (m *mockCollectionWithStore) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	m.store.mutex.Lock()
	defer m.store.mutex.Unlock()

	// Check if upsert is enabled
	isUpsert := false
	if opts != nil && len(opts) > 0 && opts[0] != nil && opts[0].Upsert != nil {
		isUpsert = *opts[0].Upsert
	}

	// Handle BSON filter for finding and updating documents
	if bsonFilter, ok := filter.(bson.M); ok {
		projectID, _ := bsonFilter["projectID"].(string)
		databaseID, _ := bsonFilter["databaseID"].(string)
		collectionID, _ := bsonFilter["collectionID"].(string)
		documentID, _ := bsonFilter["documentID"].(string)

		if projectID != "" && databaseID != "" && collectionID != "" && documentID != "" {
			key := m.store.generateKey(projectID, databaseID, collectionID, documentID)

			// Try to find existing document
			if doc, exists := m.store.documents[key]; exists {
				if mongoDoc, ok := doc.(MongoDocument); ok {
					originalDoc := mongoDoc

					// Handle update operations
					if updateDoc, ok := update.(bson.M); ok {
						if setFields, ok := updateDoc["$set"].(bson.M); ok {
							if fields, ok := setFields["fields"].(map[string]*model.FieldValue); ok {
								mongoDoc.Fields = fields
							}
							if updateTime, ok := setFields["updateTime"].(time.Time); ok {
								mongoDoc.UpdateTime = updateTime
							}
						}
						if incFields, ok := updateDoc["$inc"].(bson.M); ok {
							if versionInc, ok := incFields["version"].(int); ok {
								mongoDoc.Version += int64(versionInc)
							}
						}
					}

					m.store.documents[key] = mongoDoc

					// Return the updated document if ReturnDocument is After
					if opts != nil && len(opts) > 0 && opts[0] != nil && opts[0].ReturnDocument != nil && *opts[0].ReturnDocument == options.After {
						return &mockSingleResultWithData{data: mongoDoc}
					}
					return &mockSingleResultWithData{data: originalDoc}
				}
			} else if isUpsert {
				// Document doesn't exist but upsert is enabled - create new document
				newDoc := MongoDocument{
					ID:           primitive.NewObjectID(),
					ProjectID:    projectID,
					DatabaseID:   databaseID,
					CollectionID: collectionID,
					DocumentID:   documentID,
					CreateTime:   time.Now(),
					UpdateTime:   time.Now(),
					Version:      1,
					Exists:       true,
				}

				// Apply update operations to new document
				if updateDoc, ok := update.(bson.M); ok {
					if setFields, ok := updateDoc["$set"].(bson.M); ok {
						if path, ok := setFields["path"].(string); ok {
							newDoc.Path = path
						}
						if parentPath, ok := setFields["parentPath"].(string); ok {
							newDoc.ParentPath = parentPath
						}
						if fields, ok := setFields["fields"].(map[string]*model.FieldValue); ok {
							newDoc.Fields = fields
						}
						if updateTime, ok := setFields["updateTime"].(time.Time); ok {
							newDoc.UpdateTime = updateTime
						}
						if exists, ok := setFields["exists"].(bool); ok {
							newDoc.Exists = exists
						}
					}
					if setOnInsertFields, ok := updateDoc["$setOnInsert"].(bson.M); ok {
						if createTime, ok := setOnInsertFields["createTime"].(time.Time); ok {
							newDoc.CreateTime = createTime
						}
						if version, ok := setOnInsertFields["version"].(int); ok {
							newDoc.Version = int64(version)
						}
					}
				}

				m.store.documents[key] = newDoc
				return &mockSingleResultWithData{data: newDoc}
			}
		}
	}

	return &mockSingleResultWithData{err: fmt.Errorf("document not found")}
}

// mockSingleResultWithData implements SingleResultInterface with actual data
type mockSingleResultWithData struct {
	data interface{}
	err  error
}

func (m *mockSingleResultWithData) Decode(result interface{}) error {
	if m.err != nil {
		return m.err
	}

	if m.data == nil {
		return fmt.Errorf("no data available")
	}

	// Handle MongoDocument decoding
	if mongoDoc, ok := m.data.(MongoDocument); ok {
		if target, ok := result.(*MongoDocument); ok {
			*target = mongoDoc
			return nil
		}

		// Handle conversion from MongoDocument to model.Document
		if target, ok := result.(*model.Document); ok {
			*target = model.Document{
				ID:           mongoDoc.ID,
				ProjectID:    mongoDoc.ProjectID,
				DatabaseID:   mongoDoc.DatabaseID,
				CollectionID: mongoDoc.CollectionID,
				DocumentID:   mongoDoc.DocumentID,
				Path:         mongoDoc.Path,
				ParentPath:   mongoDoc.ParentPath,
				Fields:       mongoDoc.Fields,
				CreateTime:   mongoDoc.CreateTime,
				UpdateTime:   mongoDoc.UpdateTime,
				ReadTime:     mongoDoc.ReadTime,
				Version:      mongoDoc.Version,
				Exists:       mongoDoc.Exists,
			}
			return nil
		}
	}

	// Handle generic interface{} decoding
	if target, ok := result.(*interface{}); ok {
		*target = m.data
		return nil
	}

	return fmt.Errorf("decode type mismatch: cannot decode %T to %T", m.data, result)
}

func (m *mockSingleResultWithData) Err() error {
	return m.err
}

// mockCursorWithData implements CursorInterface with actual data
type mockCursorWithData struct {
	docs    []interface{}
	current int
}

func (m *mockCursorWithData) Next(ctx context.Context) bool {
	return m.current < len(m.docs)
}

func (m *mockCursorWithData) Decode(result interface{}) error {
	if m.current >= len(m.docs) {
		return fmt.Errorf("no more documents")
	}

	currentDoc := m.docs[m.current]

	if mongoDoc, ok := currentDoc.(MongoDocument); ok {
		if target, ok := result.(*MongoDocument); ok {
			*target = mongoDoc
			m.current++
			return nil
		}

		// Handle conversion from MongoDocument to model.Document
		if target, ok := result.(*model.Document); ok {
			*target = model.Document{
				ID:           mongoDoc.ID,
				ProjectID:    mongoDoc.ProjectID,
				DatabaseID:   mongoDoc.DatabaseID,
				CollectionID: mongoDoc.CollectionID,
				DocumentID:   mongoDoc.DocumentID,
				Path:         mongoDoc.Path,
				ParentPath:   mongoDoc.ParentPath,
				Fields:       mongoDoc.Fields,
				CreateTime:   mongoDoc.CreateTime,
				UpdateTime:   mongoDoc.UpdateTime,
				ReadTime:     mongoDoc.ReadTime,
				Version:      mongoDoc.Version,
				Exists:       mongoDoc.Exists,
			}
			m.current++
			return nil
		}
	}

	// Handle generic interface{} decoding
	if target, ok := result.(*interface{}); ok {
		*target = currentDoc
		m.current++
		return nil
	}

	m.current++
	return fmt.Errorf("decode type mismatch: cannot decode %T to %T", currentDoc, result)
}

func (m *mockCursorWithData) Close(ctx context.Context) error {
	return nil
}

func (m *mockCursorWithData) Err() error {
	return nil
}

func (m *mockCursorWithData) All(ctx context.Context, results interface{}) error {
	return nil
}

// Basic mocks for simple cases where store functionality is not needed
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
func (m *mockSingleResult) Err() error                 { return nil }

type mockUpdateResult struct{ MatchedCount int64 }

func (m *mockUpdateResult) Matched() int64 { return m.MatchedCount }

type mockDeleteResult struct{ DeletedCount int64 }

func (m *mockDeleteResult) Deleted() int64 { return m.DeletedCount }

type mockCursor struct{}

func (m *mockCursor) Next(ctx context.Context) bool                      { return false }
func (m *mockCursor) Decode(val interface{}) error                       { return nil }
func (m *mockCursor) Close(ctx context.Context) error                    { return nil }
func (m *mockCursor) Err() error                                         { return nil }
func (m *mockCursor) All(ctx context.Context, results interface{}) error { return nil }

// newTestDocumentRepositoryMockWithStore returns a DocumentRepository with a provided shared in-memory store for DocumentOperations.
func newTestDocumentRepositoryMockWithStore(store map[string]*model.Document) *DocumentRepository {
	mockStore := newMockDocumentStore()
	mockCol := &mockCollectionWithStore{store: mockStore}
	mockDB := &mockDatabaseProviderForOps{store: mockStore}

	repo := &DocumentRepository{
		db:             mockDB,
		logger:         &usecase.MockLogger{}, // Initialize the logger to prevent nil pointer dereference
		documentsCol:   mockCol,
		collectionsCol: mockCol,
	}
	docOps := NewDocumentOperationsWithStore(repo, store)
	repo.documentOps = docOps
	return repo
}

// newTestDocumentRepositoryMock returns a DocumentRepository with a new in-memory store for DocumentOperations.
func newTestDocumentRepositoryMock() *DocumentRepository {
	return newTestDocumentRepositoryMockWithStore(make(map[string]*model.Document))
}

// mockDatabaseProviderForOps implements DatabaseProvider for document operations tests
type mockDatabaseProviderForOps struct {
	store *mockDocumentStore
}

func (m *mockDatabaseProviderForOps) Collection(name string) CollectionInterface {
	return &mockCollectionWithStore{store: m.store}
}

func (m *mockDatabaseProviderForOps) Client() interface{} {
	return nil
}

// newTestDocumentRepositoryForOps creates a DocumentRepository with functional mocks for document operations tests
func newTestDocumentRepositoryForOps() *DocumentRepository {
	mockStore := newMockDocumentStore()
	mockCol := &mockCollectionWithStore{store: mockStore}
	mockDB := &mockDatabaseProviderForOps{store: mockStore}

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
