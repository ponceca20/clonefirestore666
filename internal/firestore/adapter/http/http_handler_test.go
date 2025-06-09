package http

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFirestoreUsecase mocks the FirestoreUsecaseInterface for handler tests
// Only methods used in handler are mocked

type MockFirestoreUsecase struct {
	mock.Mock
}

func (m *MockFirestoreUsecase) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockFirestoreUsecase) GetDocument(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockFirestoreUsecase) UpdateDocument(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockFirestoreUsecase) DeleteDocument(ctx context.Context, req usecase.DeleteDocumentRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockFirestoreUsecase) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Document), args.Error(1)
}

// FirestoreUsecaseInterface requires these methods, but not used in handler tests
func (m *MockFirestoreUsecase) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	args := m.Called(ctx, projectID)
	return args.String(0), args.Error(1)
}

// FirestoreUsecaseInterface requires these methods, but not used in handler tests
func (m *MockFirestoreUsecase) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	args := m.Called(ctx, projectID, transactionID)
	return args.Error(0)
}

// FirestoreUsecaseInterface requires these methods, but not used in handler tests
func (m *MockFirestoreUsecase) ListDocuments(ctx context.Context, req usecase.ListDocumentsRequest) ([]*model.Document, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) RunBatchWrite(ctx context.Context, req usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) ListCollections(ctx context.Context, req usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) DeleteCollection(ctx context.Context, req usecase.DeleteCollectionRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
func (m *MockFirestoreUsecase) ListSubcollections(ctx context.Context, req usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) CreateIndex(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) DeleteIndex(ctx context.Context, req usecase.DeleteIndexRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
func (m *MockFirestoreUsecase) ListIndexes(ctx context.Context, req usecase.ListIndexesRequest) ([]model.Index, error) {
	args := m.Called(ctx, req)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) CreateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(ctx, path, data)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) GetDocumentLegacy(ctx context.Context, path string) (map[string]interface{}, error) {
	args := m.Called(ctx, path)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) UpdateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(ctx, path, data)
	return nil, args.Error(1)
}
func (m *MockFirestoreUsecase) DeleteDocumentLegacy(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}
func (m *MockFirestoreUsecase) ListDocumentsLegacy(ctx context.Context, path string, limit int, pageToken string) ([]*model.Document, string, error) {
	args := m.Called(ctx, path, limit, pageToken)
	return nil, "", args.Error(2)
}
func (m *MockFirestoreUsecase) RunQueryLegacy(ctx context.Context, projectID string, queryJSON string) ([]*model.Document, error) {
	args := m.Called(ctx, projectID, queryJSON)
	return nil, args.Error(1)
}

func TestCreateDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)

	mockUC.On("CreateDocument", mock.Anything, mock.Anything).Return(&model.Document{ID: "doc1"}, nil)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/documents/c1", strings.NewReader(`{"field":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCreateDocument_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/documents/c1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)

	mockUC.On("CreateDocument", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/documents/c1", strings.NewReader(`{"field":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestGetDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Get("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.GetDocument)

	mockUC.On("GetDocument", mock.Anything, mock.Anything).Return(&model.Document{ID: "doc1"}, nil)

	req := httptest.NewRequest("GET", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetDocument_NotFound(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Get("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.GetDocument)

	mockUC.On("GetDocument", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Put("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)

	mockUC.On("UpdateDocument", mock.Anything, mock.Anything).Return(&model.Document{ID: "doc1", Fields: map[string]interface{}{"field": "updated"}}, nil)

	req := httptest.NewRequest("PUT", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", strings.NewReader(`{"field":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUpdateDocument_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil}
	app.Put("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)

	req := httptest.NewRequest("PUT", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Put("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)

	mockUC.On("UpdateDocument", mock.Anything, mock.Anything).Return(nil, errors.New("update error"))

	req := httptest.NewRequest("PUT", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", strings.NewReader(`{"field":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestDeleteDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Delete("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.DeleteDocument)

	mockUC.On("DeleteDocument", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest("DELETE", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Delete("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.DeleteDocument)

	mockUC.On("DeleteDocument", mock.Anything, mock.Anything).Return(errors.New("delete error"))

	req := httptest.NewRequest("DELETE", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestQueryDocuments_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	mockUC.On("RunQuery", mock.Anything, mock.Anything).Return([]*model.Document{{ID: "doc1"}}, nil)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`{"filters":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestQueryDocuments_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestQueryDocuments_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	mockUC.On("RunQuery", mock.Anything, mock.Anything).Return(nil, errors.New("query error"))

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`{"filters":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
