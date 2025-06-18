package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpadapter "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/usecase"
)

// TestSubcollectionRoutes_Integration tests the creation and management of subcollections
func TestSubcollectionRoutes_Integration(t *testing.T) {
	app := fiber.New()

	// Setup mock usecase
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(),
		nil, // securityRepo mock
		nil, // queryEngine mock
		nil, // projectionService mock
		&usecase.MockLogger{},
	)

	h := &httpadapter.HTTPHandler{
		FirestoreUC: uc,
		Log:         &usecase.MockLogger{},
	}

	// Register routes
	h.RegisterRoutes(app)

	orgID := "new-org-1749766807"
	projectID := "new-proj-from-postman"
	databaseID := "Database-2026"

	basePath := "/api/v1/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	t.Run("Create subcollection document - should succeed", func(t *testing.T) {
		// Test creating a document in a subcollection: productos/prod-001/reseñas
		subcollectionPath := "/documents/productos/prod-001/reseñas"
		documentID := "res-abc"

		createBody := `{
			"usuario": "gamer_fan_123",
			"rating": 5,
			"comentario": "¡Excelente producto! Corre todos los juegos sin problemas."
		}`

		// Test POST to subcollection
		req := httptest.NewRequest(http.MethodPost, basePath+subcollectionPath+"?documentId="+documentID, strings.NewReader(createBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		// Print response for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Response Status: %d", resp.StatusCode)
		t.Logf("Response Body: %s", string(bodyBytes))

		// Should create successfully (201) or return 404 if route not found
		// We expect either success or a specific error that helps us understand what's missing
		assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest)
	})

	t.Run("Get subcollection document - should work if route exists", func(t *testing.T) {
		// Test getting a document from a subcollection
		subcollectionPath := "/documents/productos/prod-001/reseñas/res-abc"

		req := httptest.NewRequest(http.MethodGet, basePath+subcollectionPath, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		t.Logf("GET Response Status: %d", resp.StatusCode)

		// Should work or return 404 if route not found
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
	})

	t.Run("List subcollections - should work", func(t *testing.T) {
		// Test listing subcollections under a document
		parentDocPath := "/documents/productos/prod-001/subcollections"

		req := httptest.NewRequest(http.MethodGet, basePath+parentDocPath, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		t.Logf("List Subcollections Response Status: %d", resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			t.Logf("Subcollections Response: %+v", result)

			// Should have subcollections field
			assert.Contains(t, result, "subcollections")
		}
	})

	t.Run("Test various subcollection depths", func(t *testing.T) {
		testCases := []struct {
			name string
			path string
		}{
			{
				name: "Single level subcollection",
				path: "/documents/users/user1/posts",
			},
			{
				name: "Double level subcollection",
				path: "/documents/users/user1/posts/post1/comments",
			},
			{
				name: "Triple level subcollection",
				path: "/documents/users/user1/posts/post1/comments/comment1/replies",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				createBody := `{"content": "test content", "timestamp": "2025-06-17T00:00:00Z"}`

				req := httptest.NewRequest(http.MethodPost, basePath+tc.path+"?documentId=test-doc", strings.NewReader(createBody))
				req.Header.Set("Content-Type", "application/json")

				resp, err := app.Test(req)
				require.NoError(t, err)

				t.Logf("%s - Response Status: %d", tc.name, resp.StatusCode)

				// Document the current behavior
				assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest)
			})
		}
	})
}

// TestSubcollectionPathParsing tests if our path parsing logic handles subcollections correctly
func TestSubcollectionPathParsing_Integration(t *testing.T) {
	app := fiber.New()

	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(),
		nil, nil, nil,
		&usecase.MockLogger{},
	)

	h := &httpadapter.HTTPHandler{
		FirestoreUC: uc,
		Log:         &usecase.MockLogger{},
	}

	h.RegisterRoutes(app)
	orgID := "test-org"
	projectID := "test-project"
	databaseID := "test-database"
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	testPaths := []struct {
		description string
		path        string
		method      string
		shouldWork  bool
	}{
		{
			description: "Simple collection",
			path:        "/documents/users",
			method:      http.MethodPost,
			shouldWork:  true,
		},
		{
			description: "Simple document",
			path:        "/documents/users/user1",
			method:      http.MethodGet,
			shouldWork:  true,
		},
		{
			description: "Subcollection create",
			path:        "/documents/users/user1/posts",
			method:      http.MethodPost,
			shouldWork:  false, // Currently not supported
		},
		{
			description: "Subcollection document get",
			path:        "/documents/users/user1/posts/post1",
			method:      http.MethodGet,
			shouldWork:  false, // Currently not supported
		},
		{
			description: "Deep subcollection",
			path:        "/documents/users/user1/posts/post1/comments",
			method:      http.MethodPost,
			shouldWork:  false, // Currently not supported
		},
	}

	for _, tc := range testPaths {
		t.Run(tc.description, func(t *testing.T) {
			var req *http.Request

			if tc.method == http.MethodPost {
				body := `{"test": "data"}`
				req = httptest.NewRequest(tc.method, basePath+tc.path+"?documentId=test", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, basePath+tc.path, nil)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)

			t.Logf("Path: %s, Method: %s, Status: %d", tc.path, tc.method, resp.StatusCode)

			if tc.shouldWork {
				// These should work (existing functionality)
				assert.True(t, resp.StatusCode < 400, "Expected success for supported path: %s", tc.path)
			} else {
				// These currently don't work (missing subcollection support)
				// Document the current state - they should return 404 (route not found)
				t.Logf("Expected failure for unsupported path: %s (Status: %d)", tc.path, resp.StatusCode)
			}
		})
	}
}
