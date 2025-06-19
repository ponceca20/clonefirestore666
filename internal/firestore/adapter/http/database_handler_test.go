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

// Local test double embedding MockFirestoreUC for custom CreateDatabase
// This avoids import cycles and allows test-specific logic

type customDatabaseUC struct {
	MockFirestoreUC
	createDatabaseFunc func(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error)
}

func (m *customDatabaseUC) CreateDatabase(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
	if m.createDatabaseFunc != nil {
		return m.createDatabaseFunc(ctx, req)
	}
	return m.MockFirestoreUC.CreateDatabase(ctx, req)
}

func TestCreateDatabaseHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customDatabaseUC{
		createDatabaseFunc: func(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
			return &model.Database{DatabaseID: "d1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/test/:projectID", h.CreateDatabase)

	body := []byte(`{"database":{"name":"d1"}}`)
	req := httptest.NewRequest("POST", "/test/p1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "d1", result["databaseID"])
}

func TestCreateDatabaseHandler_MissingDatabase(t *testing.T) {
	app := fiber.New()
	mockUC := &customDatabaseUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/test/:projectID", h.CreateDatabase)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/test/p1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_database", result["error"])
}

func TestCreateDatabaseHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customDatabaseUC{
		createDatabaseFunc: func(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/test/:projectID", h.CreateDatabase)

	body := []byte(`{"database":{"name":"d1"}}`)
	req := httptest.NewRequest("POST", "/test/p1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_database_failed", result["error"])
}
