package http

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func newTestDocument() *model.Document {
	return &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Fields:       map[string]*model.FieldValue{"field": model.NewFieldValue("value")},
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
		Exists:       true,
	}
}

func newTestLogger() logger.Logger {
	return logger.NewLogger()
}

func TestCreateDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)

	mockUC.On("CreateDocument", mock.Anything, mock.Anything).Return(newTestDocument(), nil)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/documents/c1", strings.NewReader(`{"field":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCreateDocument_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil, Log: newTestLogger()}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/documents/c1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Get("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.GetDocument)

	mockUC.On("GetDocument", mock.Anything, mock.Anything).Return(newTestDocument(), nil)

	req := httptest.NewRequest("GET", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetDocument_NotFound(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Get("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.GetDocument)

	// Cambia el mensaje de error a 'document not found' para que el handler devuelva 404
	mockUC.On("GetDocument", mock.Anything, mock.Anything).Return(nil, errors.New("document not found"))

	req := httptest.NewRequest("GET", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateDocument_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Put("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)

	updatedDoc := newTestDocument()
	updatedDoc.Fields = map[string]*model.FieldValue{"field": model.NewFieldValue("updated")}
	mockUC.On("UpdateDocument", mock.Anything, mock.Anything).Return(updatedDoc, nil)

	req := httptest.NewRequest("PUT", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", strings.NewReader(`{"field":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUpdateDocument_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil, Log: newTestLogger()}
	app.Put("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)

	req := httptest.NewRequest("PUT", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Delete("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.DeleteDocument)

	mockUC.On("DeleteDocument", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest("DELETE", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteDocument_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Delete("/api/v1/firestore/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.DeleteDocument)

	mockUC.On("DeleteDocument", mock.Anything, mock.Anything).Return(errors.New("delete error"))

	req := httptest.NewRequest("DELETE", "/api/v1/firestore/projects/p1/databases/d1/documents/c1/doc1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestQueryDocuments_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	mockUC.On("RunQuery", mock.Anything, mock.Anything).Return([]*model.Document{newTestDocument()}, nil)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`{"filters":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestQueryDocuments_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &HTTPHandler{FirestoreUC: nil, Log: newTestLogger()}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`invalid-json`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestQueryDocuments_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := new(MockFirestoreUsecase)
	h := &HTTPHandler{FirestoreUC: mockUC, Log: newTestLogger()}
	app.Post("/api/v1/firestore/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	mockUC.On("RunQuery", mock.Anything, mock.Anything).Return(nil, errors.New("query error"))

	req := httptest.NewRequest("POST", "/api/v1/firestore/projects/p1/databases/d1/query/c1", strings.NewReader(`{"filters":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
