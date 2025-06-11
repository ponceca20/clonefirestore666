package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFirestoreUC implements FirestoreUsecaseInterface for atomic handler tests
// Only atomic methods are functional, others are dummies for interface compliance

type MockFirestoreUC struct {
	AtomicIncrementFn       func(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error)
	AtomicArrayUnionFn      func(ctx context.Context, req usecase.AtomicArrayUnionRequest) error
	AtomicArrayRemoveFn     func(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error
	AtomicServerTimestampFn func(ctx context.Context, req usecase.AtomicServerTimestampRequest) error
}

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

// --- Dummies for interface compliance ---
func (m *MockFirestoreUC) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	// Para los tests de éxito, retorna un documento válido
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
func (m *MockFirestoreUC) DeleteIndex(context.Context, usecase.DeleteIndexRequest) error { return nil }
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
func (m *MockFirestoreUC) BeginTransaction(context.Context, string) (string, error) { return "", nil }
func (m *MockFirestoreUC) CommitTransaction(context.Context, string, string) error  { return nil }
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

// Minimal logger for test (implements logger.Logger interface)
type testLogger struct{}

func (testLogger) Debug(args ...interface{})                              {}
func (testLogger) Info(args ...interface{})                               {}
func (testLogger) Error(args ...interface{})                              {}
func (testLogger) Warn(args ...interface{})                               {}
func (testLogger) Debugf(format string, args ...interface{})              {}
func (testLogger) Infof(format string, args ...interface{})               {}
func (testLogger) Errorf(format string, args ...interface{})              {}
func (testLogger) Warnf(format string, args ...interface{})               {}
func (testLogger) Fatal(args ...interface{})                              {}
func (testLogger) Fatalf(format string, args ...interface{})              {}
func (testLogger) WithFields(fields map[string]interface{}) logger.Logger { return testLogger{} }
func (testLogger) WithContext(ctx context.Context) logger.Logger          { return testLogger{} }
func (testLogger) WithComponent(component string) logger.Logger           { return testLogger{} }

func TestAtomicIncrementHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{
		AtomicIncrementFn: func(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
			return &usecase.AtomicIncrementResponse{NewValue: 42}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicIncrement)

	body := []byte(`{"field":"count","incrementBy":2}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, float64(42), result["newValue"])
}

func TestAtomicIncrementHandler_MissingField(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicIncrement)

	body := []byte(`{"incrementBy":2}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_field", result["error"])
}

func TestAtomicIncrementHandler_MissingIncrementBy(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicIncrement)

	body := []byte(`{"field":"count"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_increment_by", result["error"])
}

func TestAtomicIncrementHandler_InvalidBody(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicIncrement)

	body := []byte(`not a json`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid_request_body", result["error"])
}

func TestAtomicIncrementHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{
		AtomicIncrementFn: func(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicIncrement)

	body := []byte(`{"field":"count","incrementBy":2}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "atomic_increment_failed", result["error"])
}

// Similar tests can be written for AtomicArrayUnion, AtomicArrayRemove, AtomicServerTimestamp
// For brevity, only one example for each is shown below

func TestAtomicArrayUnionHandler_MissingField(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayUnion)

	body := []byte(`{"values":[]}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_field", result["error"])
}

func TestAtomicArrayUnionHandler_MissingElements(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayUnion)

	body := []byte(`{"field":"tags"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_elements", result["error"])
}

func TestAtomicArrayUnionHandler_InvalidBody(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayUnion)

	body := []byte(`not a json`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid_request_body", result["error"])
}

func TestAtomicArrayRemoveHandler_MissingField(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayRemove)

	body := []byte(`{"elements":[1,2]}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_field", result["error"])
}

func TestAtomicArrayRemoveHandler_MissingElements(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayRemove)

	body := []byte(`{"field":"tags"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_elements", result["error"])
}

func TestAtomicArrayRemoveHandler_InvalidBody(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicArrayRemove)

	body := []byte(`not a json`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid_request_body", result["error"])
}

func TestAtomicServerTimestampHandler_MissingField(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicServerTimestamp)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_field", result["error"])
}

func TestAtomicServerTimestampHandler_InvalidBody(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID/:documentID", h.AtomicServerTimestamp)

	body := []byte(`not a json`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1/doc1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid_request_body", result["error"])
}
