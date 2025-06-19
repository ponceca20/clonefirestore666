package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicIncrementHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &MockFirestoreUC{
		AtomicIncrementFn: func(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
			return &usecase.AtomicIncrementResponse{NewValue: 42}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
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
