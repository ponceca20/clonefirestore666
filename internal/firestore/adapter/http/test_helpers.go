package http

import (
	"context"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"
)

// MockFirestoreUC implementa FirestoreUsecaseInterface para los tests HTTP
// Centralizada para evitar duplicación y mantener consistencia
type MockFirestoreUC struct {
	AtomicIncrementFn       func(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error)
	AtomicArrayUnionFn      func(ctx context.Context, req usecase.AtomicArrayUnionRequest) error
	AtomicArrayRemoveFn     func(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error
	AtomicServerTimestampFn func(ctx context.Context, req usecase.AtomicServerTimestampRequest) error
	RunAggregationQueryFn   func(ctx context.Context, req usecase.AggregationQueryRequest) (*usecase.AggregationQueryResponse, error)
}

// Métodos funcionales que pueden ser customizados por test
func (m *MockFirestoreUC) AtomicIncrement(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	if m.AtomicIncrementFn != nil {
		return m.AtomicIncrementFn(ctx, req)
	}
	return &usecase.AtomicIncrementResponse{}, nil
}

func (m *MockFirestoreUC) AtomicArrayUnion(ctx context.Context, req usecase.AtomicArrayUnionRequest) error {
	if m.AtomicArrayUnionFn != nil {
		return m.AtomicArrayUnionFn(ctx, req)
	}
	return nil
}

func (m *MockFirestoreUC) AtomicArrayRemove(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error {
	if m.AtomicArrayRemoveFn != nil {
		return m.AtomicArrayRemoveFn(ctx, req)
	}
	return nil
}

func (m *MockFirestoreUC) AtomicServerTimestamp(ctx context.Context, req usecase.AtomicServerTimestampRequest) error {
	if m.AtomicServerTimestampFn != nil {
		return m.AtomicServerTimestampFn(ctx, req)
	}
	return nil
}

func (m *MockFirestoreUC) RunAggregationQuery(ctx context.Context, req usecase.AggregationQueryRequest) (*usecase.AggregationQueryResponse, error) {
	if m.RunAggregationQueryFn != nil {
		return m.RunAggregationQueryFn(ctx, req)
	}
	return &usecase.AggregationQueryResponse{}, nil
}

// Métodos dummy para cumplir con la interfaz FirestoreUsecaseInterface
func (m *MockFirestoreUC) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	return &model.Document{DocumentID: "doc1"}, nil
}

func (m *MockFirestoreUC) GetDocument(context.Context, usecase.GetDocumentRequest) (*model.Document, error) {
	return nil, nil
}

func (m *MockFirestoreUC) UpdateDocument(context.Context, usecase.UpdateDocumentRequest) (*model.Document, error) {
	return nil, nil
}

func (m *MockFirestoreUC) DeleteDocument(context.Context, usecase.DeleteDocumentRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListDocuments(context.Context, usecase.ListDocumentsRequest) ([]*model.Document, error) {
	return nil, nil
}

func (m *MockFirestoreUC) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	return &model.Collection{CollectionID: "c1"}, nil
}

func (m *MockFirestoreUC) GetCollection(context.Context, usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, nil
}

func (m *MockFirestoreUC) UpdateCollection(context.Context, usecase.UpdateCollectionRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListCollections(context.Context, usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, nil
}

func (m *MockFirestoreUC) DeleteCollection(context.Context, usecase.DeleteCollectionRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListSubcollections(context.Context, usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, nil
}

func (m *MockFirestoreUC) CreateIndex(context.Context, usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, nil
}

func (m *MockFirestoreUC) DeleteIndex(context.Context, usecase.DeleteIndexRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListIndexes(context.Context, usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, nil
}

func (m *MockFirestoreUC) QueryDocuments(context.Context, usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}

func (m *MockFirestoreUC) RunQuery(context.Context, usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}

func (m *MockFirestoreUC) RunBatchWrite(context.Context, usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, nil
}

func (m *MockFirestoreUC) BeginTransaction(context.Context, string) (string, error) {
	return "", nil
}

func (m *MockFirestoreUC) CommitTransaction(context.Context, string, string) error {
	return nil
}

func (m *MockFirestoreUC) CreateProject(context.Context, usecase.CreateProjectRequest) (*model.Project, error) {
	return &model.Project{ProjectID: "p1"}, nil
}

func (m *MockFirestoreUC) GetProject(context.Context, usecase.GetProjectRequest) (*model.Project, error) {
	return &model.Project{ProjectID: "p1"}, nil
}

func (m *MockFirestoreUC) UpdateProject(context.Context, usecase.UpdateProjectRequest) (*model.Project, error) {
	return nil, nil
}

func (m *MockFirestoreUC) DeleteProject(context.Context, usecase.DeleteProjectRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListProjects(context.Context, usecase.ListProjectsRequest) ([]*model.Project, error) {
	return nil, nil
}

func (m *MockFirestoreUC) CreateDatabase(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
	return &model.Database{DatabaseID: "d1"}, nil
}

func (m *MockFirestoreUC) GetDatabase(context.Context, usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, nil
}

func (m *MockFirestoreUC) UpdateDatabase(context.Context, usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}

func (m *MockFirestoreUC) DeleteDatabase(context.Context, usecase.DeleteDatabaseRequest) error {
	return nil
}

func (m *MockFirestoreUC) ListDatabases(context.Context, usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, nil
}

// TestLogger implementa logger.Logger para los tests
// Centralizada para evitar duplicación y mantener consistencia
type TestLogger struct{}

func (TestLogger) Debug(args ...interface{})                              {}
func (TestLogger) Info(args ...interface{})                               {}
func (TestLogger) Error(args ...interface{})                              {}
func (TestLogger) Warn(args ...interface{})                               {}
func (TestLogger) Debugf(format string, args ...interface{})              {}
func (TestLogger) Infof(format string, args ...interface{})               {}
func (TestLogger) Errorf(format string, args ...interface{})              {}
func (TestLogger) Warnf(format string, args ...interface{})               {}
func (TestLogger) Fatal(args ...interface{})                              {}
func (TestLogger) Fatalf(format string, args ...interface{})              {}
func (TestLogger) WithFields(fields map[string]interface{}) logger.Logger { return TestLogger{} }
func (TestLogger) WithContext(ctx context.Context) logger.Logger          { return TestLogger{} }
func (TestLogger) WithComponent(component string) logger.Logger           { return TestLogger{} }
