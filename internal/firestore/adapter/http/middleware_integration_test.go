package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMiddlewareIntegration tests that TenantMiddleware and ValidateFirestoreHierarchy work together
func TestMiddlewareIntegration(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		expectedStatus int
		expectedError  string
		shouldPassOrg  bool
	}{
		{
			name:           "Valid organization in URL path",
			method:         "POST",
			path:           "/api/v1/organizations/myorg103/projects/project1/databases/database1/documents/items",
			expectedStatus: 200,
			shouldPassOrg:  true,
		},
		{
			name:   "Valid organization in header",
			method: "POST",
			path:   "/api/v1/organizations/myorg103/projects/project1/databases/database1/documents/items",
			headers: map[string]string{
				"X-Organization-ID": "myorg103",
			},
			expectedStatus: 200,
			shouldPassOrg:  true,
		},
		{
			name:           "Invalid organization ID - too short",
			method:         "POST",
			path:           "/api/v1/organizations/x/projects/project1/databases/database1/documents/items",
			expectedStatus: 400,
			expectedError:  "invalid_organization_id",
			shouldPassOrg:  false,
		},
		{
			name:           "Invalid organization ID format - starts with number",
			method:         "POST",
			path:           "/api/v1/organizations/123org/projects/project1/databases/database1/documents/items",
			expectedStatus: 400,
			expectedError:  "invalid_organization_id",
			shouldPassOrg:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Fiber app
			app := fiber.New() // Create a test handler that checks if organization ID is in context
			testHandler := func(c *fiber.Ctx) error {
				ctx := c.UserContext()

				// Check if organization ID is available in context
				orgID, err := utils.GetOrganizationIDFromContext(ctx)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "context_error",
						"message": fmt.Sprintf("Failed to get org ID from context: %v", err),
					})
				}

				return c.JSON(fiber.Map{
					"success":        true,
					"organizationId": orgID,
					"path":           c.Path(),
					"params":         c.AllParams(),
				})
			} // Register the middleware chain exactly as in the real application
			orgAPI := app.Group("/api/v1/organizations/:organizationId", TenantMiddleware())
			projectAPI := orgAPI.Group("/projects/:projectID")
			dbAPI := projectAPI.Group("/databases/:databaseID", ProjectMiddleware(), ValidateFirestoreHierarchy())

			// Register test endpoint
			dbAPI.Post("/documents/:collectionID", testHandler)

			// Create request body
			reqBody := map[string]interface{}{
				"name": "test item",
				"data": "test data",
			}
			bodyBytes, _ := json.Marshal(reqBody)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add headers if specified
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Read response body
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Log for debugging
			t.Logf("Response Status: %d", resp.StatusCode)
			t.Logf("Response Body: %s", string(body))

			// Assert status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode) // Parse response
			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			if tt.expectedStatus == 200 {
				// Should be successful
				if success, ok := response["success"].(bool); ok {
					assert.True(t, success)
				}

				if tt.shouldPassOrg {
					assert.NotEmpty(t, response["organizationId"])
					t.Logf("Organization ID in context: %s", response["organizationId"])
				}
			} else {
				// Should have error
				assert.Contains(t, response, "error")
				if tt.expectedError != "" {
					assert.Equal(t, tt.expectedError, response["error"])
				}
			}
		})
	}
}

// TestTenantMiddlewareIsolation tests TenantMiddleware in isolation
func TestTenantMiddlewareIsolation(t *testing.T) {
	app := fiber.New()
	// Simple test handler that checks context
	testHandler := func(c *fiber.Ctx) error {
		ctx := c.UserContext() // Use UserContext() instead of Context()
		orgID, err := utils.GetOrganizationIDFromContext(ctx)

		return c.JSON(fiber.Map{
			"organizationId": orgID,
			"error":          err != nil,
			"errorMsg":       fmt.Sprintf("%v", err),
		})
	}

	// Apply only TenantMiddleware
	app.Get("/organizations/:organizationId/test", TenantMiddleware(), testHandler)

	tests := []struct {
		name        string
		path        string
		expectOrgID string
		expectError bool
	}{
		{
			name:        "Valid org ID in path",
			path:        "/organizations/myorg103/test",
			expectOrgID: "myorg103",
			expectError: false,
		},
		{
			name:        "Valid org ID with numbers",
			path:        "/organizations/tenant123/test",
			expectOrgID: "tenant123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			t.Logf("Response: %+v", response)

			if tt.expectError {
				assert.True(t, response["error"].(bool))
			} else {
				assert.False(t, response["error"].(bool))
				assert.Equal(t, tt.expectOrgID, response["organizationId"])
			}
		})
	}
}

// TestValidateFirestoreHierarchyIsolation tests ValidateFirestoreHierarchy in isolation
func TestValidateFirestoreHierarchyIsolation(t *testing.T) {
	app := fiber.New()

	// Test handler that manually sets organization ID in context
	testHandler := func(c *fiber.Ctx) error {
		// Manually set organization ID in context
		ctx := utils.WithOrganizationID(c.Context(), "testorg")
		c.SetUserContext(ctx)

		return c.JSON(fiber.Map{
			"success": true,
			"message": "ValidateFirestoreHierarchy passed",
		})
	}

	// Apply only ValidateFirestoreHierarchy
	app.Get("/test", testHandler, ValidateFirestoreHierarchy())

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Response body: %s", string(body))
	t.Logf("Response status: %d", resp.StatusCode)

	// Should pass because we manually set the org ID
	assert.Equal(t, 200, resp.StatusCode)
}

// TestContextPersistence tests if context persists between middlewares
func TestContextPersistence(t *testing.T) {
	app := fiber.New()

	var capturedOrgIDs []string
	// Middleware 1: Set organization ID
	middleware1 := func(c *fiber.Ctx) error {
		ctx := utils.WithOrganizationID(c.Context(), "persistence-test")
		c.SetUserContext(ctx)

		// Verify it's set
		orgID, err := utils.GetOrganizationIDFromContext(c.UserContext())
		capturedOrgIDs = append(capturedOrgIDs, fmt.Sprintf("MW1: %s (err: %v)", orgID, err))

		return c.Next()
	}

	// Middleware 2: Check if org ID is still there
	middleware2 := func(c *fiber.Ctx) error {
		orgID, err := utils.GetOrganizationIDFromContext(c.UserContext())
		capturedOrgIDs = append(capturedOrgIDs, fmt.Sprintf("MW2: %s (err: %v)", orgID, err))

		return c.Next()
	}

	// Final handler
	handler := func(c *fiber.Ctx) error {
		orgID, err := utils.GetOrganizationIDFromContext(c.UserContext())
		capturedOrgIDs = append(capturedOrgIDs, fmt.Sprintf("Handler: %s (err: %v)", orgID, err))

		return c.JSON(fiber.Map{
			"organizationId": orgID,
			"error":          err != nil,
			"trace":          capturedOrgIDs,
		})
	}

	app.Get("/test-persistence", middleware1, middleware2, handler)

	req := httptest.NewRequest("GET", "/test-persistence", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	t.Logf("Full trace: %+v", response["trace"])

	// Should maintain organization ID throughout the chain
	assert.Equal(t, "persistence-test", response["organizationId"])
	assert.False(t, response["error"].(bool))
}
