package http

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHTTPHandler implementa una versión mock de HTTPHandler para testing
// Implementa métodos que pueden ser interceptados para validar el comportamiento
type MockHTTPHandler struct {
	RunQueryCalled       bool
	RunAggregationCalled bool
	LastCalledHandler    string
	ShouldReturnError    bool
	ErrorMessage         string
}

// RunQuery mock handler que simula el comportamiento del handler real
func (m *MockHTTPHandler) RunQuery(c *fiber.Ctx) error {
	m.RunQueryCalled = true
	m.LastCalledHandler = "runQuery"

	if m.ShouldReturnError {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": m.ErrorMessage,
		})
	}

	return c.JSON(fiber.Map{"handler": "runQuery", "status": "success"})
}

// RunAggregationQuery mock handler que simula el comportamiento del handler real
func (m *MockHTTPHandler) RunAggregationQuery(c *fiber.Ctx) error {
	m.RunAggregationCalled = true
	m.LastCalledHandler = "runAggregationQuery"

	if m.ShouldReturnError {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": m.ErrorMessage,
		})
	}

	return c.JSON(fiber.Map{"handler": "runAggregationQuery", "status": "success"})
}

// Reset reinicia el estado del mock para reutilización en tests
func (m *MockHTTPHandler) Reset() {
	m.RunQueryCalled = false
	m.RunAggregationCalled = false
	m.LastCalledHandler = ""
	m.ShouldReturnError = false
	m.ErrorMessage = ""
}

func TestFirestoreQueryRouter_RouteQueryEndpoints(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		body           map[string]interface{}
		expectedStatus int
		shouldRoute    bool
		expectedCall   string // "runQuery" or "runAggregationQuery"
	}{
		{
			name:   "runQuery with structuredQuery should route to RunQuery",
			path:   "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			method: "POST",
			body: map[string]interface{}{
				"structuredQuery": map[string]interface{}{
					"from": []map[string]interface{}{
						{"collectionId": "test-collection"},
					},
				},
			},
			expectedStatus: 200,
			shouldRoute:    true,
			expectedCall:   "runQuery",
		},
		{
			name:   "runAggregationQuery with structuredAggregationQuery should route to RunAggregationQuery",
			path:   "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runAggregationQuery",
			method: "POST",
			body: map[string]interface{}{
				"structuredAggregationQuery": map[string]interface{}{
					"structuredQuery": map[string]interface{}{
						"from": []map[string]interface{}{
							{"collectionId": "test-collection"},
						},
					},
					"aggregations": []map[string]interface{}{
						{"alias": "count", "count": map[string]interface{}{}},
					},
				},
			},
			expectedStatus: 200,
			shouldRoute:    true,
			expectedCall:   "runAggregationQuery",
		},
		{
			name:   "runQuery with wrong body should return error",
			path:   "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			method: "POST",
			body: map[string]interface{}{
				"structuredAggregationQuery": map[string]interface{}{
					"aggregations": []map[string]interface{}{},
				},
			},
			expectedStatus: 400,
			shouldRoute:    false,
		},
		{
			name:   "runAggregationQuery with wrong body should return error",
			path:   "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runAggregationQuery",
			method: "POST",
			body: map[string]interface{}{
				"structuredQuery": map[string]interface{}{
					"from": []map[string]interface{}{},
				},
			},
			expectedStatus: 400,
			shouldRoute:    false,
		},
		{
			name:           "GET request should not be routed",
			path:           "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			method:         "GET",
			body:           map[string]interface{}{},
			expectedStatus: 404, // Fiber returns 404 for unhandled routes
			shouldRoute:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Crear mock handler
			mockHandler := &MockHTTPHandler{}

			// Crear app de Fiber para testing
			app := fiber.New()

			// Configurar rutas
			router := app.Group("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID")

			// Crear router con el mock handler
			queryRouter := NewSecurityAwareQueryRouter(mockHandler)
			queryRouter.RegisterSecureRoutes(router)

			// Preparar request body
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			// Crear request
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Cookie", "fs_auth_token=valid_token") // Para pasar validación de seguridad

			// Ejecutar request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verificar status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verificar que se llamó al handler correcto si se esperaba routing exitoso
			if tt.shouldRoute {
				assert.Equal(t, tt.expectedCall, mockHandler.LastCalledHandler, "Should call correct handler")

				// Verificar que solo se llamó al handler esperado
				if tt.expectedCall == "runQuery" {
					assert.True(t, mockHandler.RunQueryCalled, "RunQuery should have been called")
					assert.False(t, mockHandler.RunAggregationCalled, "RunAggregationQuery should NOT have been called")
				} else if tt.expectedCall == "runAggregationQuery" {
					assert.False(t, mockHandler.RunQueryCalled, "RunQuery should NOT have been called")
					assert.True(t, mockHandler.RunAggregationCalled, "RunAggregationQuery should have been called")
				}
			} else {
				// Si no debería enrutarse, ningún handler debería haber sido llamado
				assert.False(t, mockHandler.RunQueryCalled, "RunQuery should NOT have been called")
				assert.False(t, mockHandler.RunAggregationCalled, "RunAggregationQuery should NOT have been called")
			}
		})
	}
}

func TestFirestoreQueryRouter_SecurityValidation(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		authToken      string
		cookie         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid cookie token should pass",
			path:           "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			cookie:         "fs_auth_token=valid_token",
			expectedStatus: 200,
		},
		{
			name:           "Valid Authorization header should pass",
			path:           "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			authToken:      "Bearer valid_token",
			expectedStatus: 200,
		},
		{
			name:           "No authentication should fail",
			path:           "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			expectedStatus: 401,
			expectedError:  "authentication_required",
		}, {
			name:           "Valid databaseID should pass",
			path:           "/api/v1/organizations/test-org/projects/test-proj/databases/test-db/documents:runQuery",
			cookie:         "fs_auth_token=valid_token",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Crear mock handler que siempre devuelve éxito
			mockHandler := &MockHTTPHandler{}

			// Crear app de Fiber para testing
			app := fiber.New()

			// Configurar rutas
			router := app.Group("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID")
			queryRouter := NewSecurityAwareQueryRouter(mockHandler)
			queryRouter.RegisterSecureRoutes(router)

			// Preparar request body válido
			body := map[string]interface{}{
				"structuredQuery": map[string]interface{}{
					"from": []map[string]interface{}{
						{"collectionId": "test-collection"},
					},
				},
			}
			bodyBytes, err := json.Marshal(body)
			require.NoError(t, err)

			// Crear request
			req := httptest.NewRequest("POST", tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tt.authToken != "" {
				req.Header.Set("Authorization", tt.authToken)
			}
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}

			// Ejecutar request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verificar status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verificar error message si se espera un error
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}
		})
	}
}

func TestFirestoreQueryRouter_Integration(t *testing.T) {
	t.Run("Complete integration test with realistic scenario", func(t *testing.T) {
		// Este test simula el escenario real del problema reportado

		// Crear mock handler
		mockHandler := &MockHTTPHandler{}

		// Crear app de Fiber
		app := fiber.New()

		// Configurar rutas exactamente como en el sistema real
		router := app.Group("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID")
		queryRouter := NewSecurityAwareQueryRouter(mockHandler)
		queryRouter.RegisterSecureRoutes(router)

		// Test 1: runQuery debe ir a RunQuery
		queryBody := map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"from": []map[string]interface{}{
					{"collectionId": "reseñas", "allDescendants": true},
				},
				"where": map[string]interface{}{
					"fieldFilter": map[string]interface{}{
						"field": map[string]interface{}{"fieldPath": "rating"},
						"op":    "EQUAL",
						"value": map[string]interface{}{"doubleValue": 3},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(queryBody)
		req := httptest.NewRequest("POST",
			"/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/documents:runQuery",
			bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "fs_auth_token=test_token")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.True(t, mockHandler.RunQueryCalled, "RunQuery should have been called")
		assert.False(t, mockHandler.RunAggregationCalled, "RunAggregationQuery should NOT have been called")
		assert.Equal(t, "runQuery", mockHandler.LastCalledHandler)

		// Reset mock state
		mockHandler.Reset()

		// Test 2: runAggregationQuery debe ir a RunAggregationQuery
		aggBody := map[string]interface{}{
			"structuredAggregationQuery": map[string]interface{}{
				"structuredQuery": map[string]interface{}{
					"from": []map[string]interface{}{
						{"collectionId": "productos2"},
					},
				},
				"aggregations": []map[string]interface{}{
					{"alias": "count", "count": map[string]interface{}{}},
				},
			},
		}

		bodyBytes, _ = json.Marshal(aggBody)
		req = httptest.NewRequest("POST",
			"/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/documents:runAggregationQuery",
			bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "fs_auth_token=test_token")

		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.False(t, mockHandler.RunQueryCalled, "RunQuery should NOT have been called")
		assert.True(t, mockHandler.RunAggregationCalled, "RunAggregationQuery should have been called")
		assert.Equal(t, "runAggregationQuery", mockHandler.LastCalledHandler)
	})
}
