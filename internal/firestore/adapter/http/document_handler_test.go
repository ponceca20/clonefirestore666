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

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Local test double embedding MockFirestoreUC for custom CreateDocument
// This avoids import cycles and allows test-specific logic

type customDocumentUC struct {
	MockFirestoreUC
	createDocumentFunc func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error)
}

func (m *customDocumentUC) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	if m.createDocumentFunc != nil {
		return m.createDocumentFunc(ctx, req)
	}
	return m.MockFirestoreUC.CreateDocument(ctx, req)
}

func TestCreateDocumentHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{
		createDocumentFunc: func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
			return &model.Document{DocumentID: "doc1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{"field":"value"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "doc1", result["documentID"])
}

func TestCreateDocumentHandler_MissingData(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_data", result["error"])
}

func TestCreateDocumentHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{
		createDocumentFunc: func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{"field":"value"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_document_failed", result["error"])
}
