package http

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteMiddlewareChain tests the complete middleware chain
// simulating exactly how it's used in the real application
func TestCompleteMiddlewareChain(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedOrg  string
		expectedProj string
		expectedDB   string
		expectedColl string
		shouldPass   bool
	}{
		{
			name:         "Complete Firestore Path",
			path:         "/api/v1/organizations/myorg103/projects/project1/databases/database1/documents/items",
			expectedOrg:  "myorg103",
			expectedProj: "project1",
			expectedDB:   "database1",
			expectedColl: "items",
			shouldPass:   true,
		},
		{
			name:         "Another Organization",
			path:         "/api/v1/organizations/tenant123/projects/webapp1/databases/production/documents/users",
			expectedOrg:  "tenant123",
			expectedProj: "webapp1",
			expectedDB:   "production",
			expectedColl: "users",
			shouldPass:   true,
		},
		{
			name:       "Missing Organization",
			path:       "/api/v1/projects/project1/databases/database1/documents/items",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New() // Set up the exact same middleware chain as in the real app
			v1 := app.Group("/api/v1")

			// Organization-scoped Firestore API with middleware chain in correct order
			orgAPI := v1.Group("/organizations/:organizationId", TenantMiddleware())
			projectAPI := orgAPI.Group("/projects/:projectID")
			dbAPI := projectAPI.Group("/databases/:databaseID", ProjectMiddleware(), ValidateFirestoreHierarchy())

			// Test handler that checks if all context values are available
			dbAPI.Get("/documents/:collectionID", func(c *fiber.Ctx) error {
				ctx := c.UserContext()

				// Extract all values from context
				orgID, orgErr := utils.GetOrganizationIDFromContext(ctx)
				projectID, projErr := utils.GetProjectIDFromContext(ctx)
				databaseID, dbErr := utils.GetDatabaseIDFromContext(ctx)

				// Get URL parameters too
				collectionID := c.Params("collectionID")

				return c.JSON(fiber.Map{
					"success":        true,
					"organizationId": orgID,
					"projectId":      projectID,
					"databaseId":     databaseID,
					"collectionId":   collectionID,
					"errors": fiber.Map{
						"org":  orgErr != nil,
						"proj": projErr != nil,
						"db":   dbErr != nil,
					},
				})
			})

			// Make request
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Read response
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			t.Logf("Response Status: %d", resp.StatusCode)
			t.Logf("Response Body: %s", string(body))

			if tt.shouldPass {
				assert.Equal(t, 200, resp.StatusCode, "Response body: %s", string(body))

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				require.NoError(t, err) // Verify all expected values are present
				if success, ok := response["success"].(bool); ok {
					assert.True(t, success)
				}
				assert.Equal(t, tt.expectedOrg, response["organizationId"])
				assert.Equal(t, tt.expectedProj, response["projectId"])
				assert.Equal(t, tt.expectedDB, response["databaseId"])
				assert.Equal(t, tt.expectedColl, response["collectionId"])

				// Verify no errors in context extraction
				if errors, ok := response["errors"].(map[string]interface{}); ok {
					if orgErr, ok := errors["org"].(bool); ok {
						assert.False(t, orgErr, "Organization should be found in context")
					}
					if projErr, ok := errors["proj"].(bool); ok {
						assert.False(t, projErr, "Project should be found in context")
					}
					if dbErr, ok := errors["db"].(bool); ok {
						assert.False(t, dbErr, "Database should be found in context")
					}
				}
			} else {
				assert.Equal(t, 404, resp.StatusCode, "Should return 404 for invalid paths")
			}
		})
	}
}

// TestHeaderFallbacks tests middleware fallback to headers
func TestHeaderFallbacks(t *testing.T) {
	app := fiber.New()

	// Simple route with just TenantMiddleware
	app.Get("/test", TenantMiddleware(), func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		orgID, err := utils.GetOrganizationIDFromContext(ctx)

		return c.JSON(fiber.Map{
			"organizationId": orgID,
			"error":          err != nil,
			"errorMsg":       err,
		})
	})

	t.Run("Header X-Organization-ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Organization-ID", "header-org")

		resp, err := app.Test(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
		assert.False(t, response["error"].(bool))
		assert.Equal(t, "header-org", response["organizationId"])
	})

	t.Run("Query Parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?organization_id=query-org", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
		assert.False(t, response["error"].(bool))
		assert.Equal(t, "query-org", response["organizationId"])
	})

	t.Run("Authorization Bearer with suffix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer some-token@bearer-org")

		resp, err := app.Test(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)
		assert.False(t, response["error"].(bool))
		assert.Equal(t, "bearer-org", response["organizationId"])
	})
}

// TestValidationErrorCases tests various error scenarios
func TestValidationErrorCases(t *testing.T) {
	app := fiber.New()

	app.Get("/test", TenantMiddleware(), ValidateFirestoreHierarchy(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	t.Run("No Organization ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, 400, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), "organization_id_missing")
	})

	t.Run("Invalid Organization ID Format", func(t *testing.T) {
		app := fiber.New()
		app.Get("/organizations/:organizationId/test", TenantMiddleware(), ValidateFirestoreHierarchy(), func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"success": true})
		})

		// Test with invalid org ID (too short)
		req := httptest.NewRequest("GET", "/organizations/ab/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, 400, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "invalid_organization_id")
	})
}
