// Centralized test helpers for Firestore usecase tests
// Place all shared mocks and helpers here to avoid redeclaration errors.
package usecase

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"
	"time"
)

type MockFirestoreRepo struct{}

func (m *MockFirestoreRepo) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields map[string]*model.FieldValue) (*model.Document, error) {
	return &model.Document{DocumentID: documentID}, nil
}
func (m *MockFirestoreRepo) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return &model.Document{DocumentID: documentID, Fields: map[string]*model.FieldValue{"count": model.NewFieldValue(int64(42))}}, nil
}
func (m *MockFirestoreRepo) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields map[string]*model.FieldValue, mask []string) (*model.Document, error) {
	return &model.Document{DocumentID: documentID}, nil
}
func (m *MockFirestoreRepo) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return []*model.Document{{DocumentID: "doc1"}}, "", nil
}
func (m *MockFirestoreRepo) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	return nil
}

// Add missing methods to MockFirestoreRepo to fully implement FirestoreRepository
func (m *MockFirestoreRepo) CreateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockFirestoreRepo) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{
		ProjectID: projectID,
	}, nil
}
func (m *MockFirestoreRepo) UpdateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteProject(ctx context.Context, projectID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return []*model.Project{
		{
			ProjectID: "p1",
		},
	}, nil
}
func (m *MockFirestoreRepo) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *MockFirestoreRepo) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return &model.Database{
		ProjectID:  projectID,
		DatabaseID: databaseID,
	}, nil
}
func (m *MockFirestoreRepo) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return []*model.Database{
		{
			ProjectID:  projectID,
			DatabaseID: "d1",
		},
	}, nil
}
func (m *MockFirestoreRepo) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return &model.Collection{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
	}, nil
}

func (m *MockFirestoreRepo) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return []*model.Collection{
		{
			ProjectID:    projectID,
			DatabaseID:   databaseID,
			CollectionID: "c1",
		},
	}, nil
}
func (m *MockFirestoreRepo) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) DeleteDocumentByPath(ctx context.Context, path string) error {
	return nil
}
func (m *MockFirestoreRepo) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunTransaction(ctx context.Context, fn func(repository.Transaction) error) error {
	return nil
}
func (m *MockFirestoreRepo) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return []*model.WriteResult{{UpdateTime: time.Now()}}, nil
}

// Add missing CreateIndex, DeleteIndex, ListIndexes to MockFirestoreRepo
func (m *MockFirestoreRepo) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, idx *model.CollectionIndex) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexName string) error {
	return nil
}

// Fix ListIndexes to return []*model.CollectionIndex
func (m *MockFirestoreRepo) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return []*model.CollectionIndex{
		{
			Name:   "idx1",
			Fields: []model.IndexField{{Path: "f1", Order: model.IndexFieldOrderAscending}},
			State:  "READY",
		},
	}, nil
}
func (m *MockFirestoreRepo) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return []string{"sub1"}, nil
}

// Add other required methods for other usecases as needed

// Update MockLogger to return Logger interface for WithFields, WithContext, WithComponent
type MockLogger struct{}

func (m *MockLogger) Info(args ...interface{})                               {}
func (m *MockLogger) Error(args ...interface{})                              {}
func (m *MockLogger) Debug(args ...interface{})                              {}
func (m *MockLogger) Warn(args ...interface{})                               {}
func (m *MockLogger) Fatal(args ...interface{})                              {}
func (m *MockLogger) Debugf(format string, args ...interface{})              {}
func (m *MockLogger) Infof(format string, args ...interface{})               {}
func (m *MockLogger) Warnf(format string, args ...interface{})               {}
func (m *MockLogger) Errorf(format string, args ...interface{})              {}
func (m *MockLogger) Fatalf(format string, args ...interface{})              {}
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *MockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *MockLogger) WithComponent(component string) logger.Logger           { return m }
