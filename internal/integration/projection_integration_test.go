package integration

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	httpadapter "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/usecase"
)

// TestProjectionQueryIntegration tests that projection queries work without type inference errors
func TestProjectionQueryIntegration(t *testing.T) {
	t.Run("Projection query should not cause internal server error", func(t *testing.T) {
		app := fiber.New()
		// Use mock usecase with proper dependencies
		uc := usecase.NewFirestoreUsecase(
			usecase.NewMockFirestoreRepo(),
			usecase.NewMockSecurityRulesEngine(),
			usecase.NewMockQueryEngine(),
			usecase.NewMockProjectionService(),
			&usecase.MockLogger{},
		)

		h := &httpadapter.HTTPHandler{
			FirestoreUC: uc,
			Log:         &usecase.MockLogger{},
		} // Register routes
		v1 := app.Group("/v1")
		projects := v1.Group("/projects/:projectId")
		databases := projects.Group("/databases/:databaseId")
		databases.Post("/documents:runQuery", h.RunQuery)

		// Test query with projection and boolean filter
		queryPayload := map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"select": map[string]interface{}{
					"fields": []map[string]interface{}{
						{"fieldPath": "name"},
						{"fieldPath": "available"},
					},
				},
				"from": []map[string]interface{}{
					{"collectionId": "products"},
				},
				"where": map[string]interface{}{
					"fieldFilter": map[string]interface{}{
						"field": map[string]interface{}{
							"fieldPath": "available",
						},
						"op": "EQUAL",
						"value": map[string]interface{}{
							"booleanValue": true,
						},
					},
				},
			},
		}

		jsonPayload, err := json.Marshal(queryPayload)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/v1/projects/test-project/databases/(default)/documents:runQuery", strings.NewReader(string(jsonPayload)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("organization-id", "test-org")

		resp, err := app.Test(req, 10000) // 10 second timeout
		require.NoError(t, err)
		defer resp.Body.Close()

		// The key test: it should not return a 500 error due to type inference issues
		// This is the core issue we fixed - projection queries were failing due to missing type inference
		require.NotEqual(t, 500, resp.StatusCode,
			"Query with projection should not fail with internal server error due to type inference issues")

		// Should be either 200 (success) or 400 (validation error), but not 500 (internal error)
		require.True(t, resp.StatusCode == 200 || resp.StatusCode == 400,
			"Expected 200 or 400, got %d", resp.StatusCode)
	})
}
