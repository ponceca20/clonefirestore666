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

// Local test double embedding MockFirestoreUC for custom CreateCollection
// This avoids import cycles and allows test-specific logic

type testFirestoreUC struct {
	MockFirestoreUC
	createCollectionFunc func(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error)
}

func (m *testFirestoreUC) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	if m.createCollectionFunc != nil {
		return m.createCollectionFunc(ctx, req)
	}
	return m.MockFirestoreUC.CreateCollection(ctx, req)
}

func TestCreateCollectionHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &testFirestoreUC{
		createCollectionFunc: func(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
			return &model.Collection{CollectionID: "c1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateCollection)

	body := []byte(`{"collectionId":"c1"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "c1", result["collectionID"])
}

func TestCreateCollectionHandler_MissingCollectionID(t *testing.T) {
	app := fiber.New()
	mockUC := &testFirestoreUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	// Use route with optional collectionID parameter
	app.Post("/test/:projectID/:databaseID/:collectionID?", h.CreateCollection)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_collection_id", result["error"])
}

func TestCreateCollectionHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &testFirestoreUC{
		createCollectionFunc: func(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateCollection)

	body := []byte(`{"collectionId":"c1"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_collection_failed", result["error"])
}
