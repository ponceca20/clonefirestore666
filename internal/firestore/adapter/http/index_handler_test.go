package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Local test double embedding MockFirestoreUC for custom index logic
// This avoids redeclaration and ensures interface compliance
type customIndexUC struct {
	MockFirestoreUC
	createIndexFunc func(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error)
}

func (m *customIndexUC) CreateIndex(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
	if m.createIndexFunc != nil {
		return m.createIndexFunc(ctx, req)
	}
	return m.MockFirestoreUC.CreateIndex(ctx, req)
}

func TestCreateIndexHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customIndexUC{
		createIndexFunc: func(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
			return &model.Index{Name: "idx1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateIndex)

	body := []byte(`{"index":{"collection":"c1"}}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "idx1", result["name"])
}

func TestCreateIndexHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customIndexUC{
		createIndexFunc: func(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateIndex)

	body := []byte(`{"index":{"collection":"c1"}}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_index_failed", result["error"])
}
