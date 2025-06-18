package integration

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	httpadapter "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
)

// RandString genera un string aleatorio de n caracteres válidos para IDs
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Integration test for document routes
func TestDocumentRoutes_Integration(t *testing.T) {
	app := fiber.New()
	// Usar el Usecase real con el mock centralizado para cumplir la interfaz
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
	// Register only document routes for isolation
	h.RegisterRoutes(app)

	orgID := "org-ponceca"
	projectID := "project01"
	databaseID := "database01"
	collectionID := "collection01"
	documentID := "document01"

	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	// --- Create Document ---
	createBody := `{"projectId": "` + projectID + `", "databaseId": "` + databaseID + `", "collectionId": "` + collectionID + `", "data": {"field1": "value1", "field2": 42}}`
	req := httptest.NewRequest(stdhttp.MethodPost, basePath+"/documents/"+collectionID, strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, resp.StatusCode)
	// --- Get Document ---
	getReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/documents/"+collectionID+"/"+documentID, nil)
	getResp, err := app.Test(getReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, getResp.StatusCode)
	var getDoc map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&getDoc))
	require.Equal(t, documentID, getDoc["documentID"])

	// --- Update Document ---
	updateBody := `{"projectId": "` + projectID + `", "databaseId": "` + databaseID + `", "collectionId": "` + collectionID + `", "documentId": "` + documentID + `", "data": {"field1": "newValue"}}`
	updateReq := httptest.NewRequest(stdhttp.MethodPut, basePath+"/documents/"+collectionID+"/"+documentID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, updateResp.StatusCode)

	// --- List Documents in Collection ---
	listDocReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/documents/"+collectionID, nil)
	listDocResp, err := app.Test(listDocReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listDocResp.StatusCode)

	// Verify response body structure
	var listResponse map[string]interface{}
	listBody, err := io.ReadAll(listDocResp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(listBody, &listResponse))

	// Verify expected fields in response
	require.Contains(t, listResponse, "documents")
	require.Contains(t, listResponse, "count")

	// Verify documents array exists
	documents, ok := listResponse["documents"].([]interface{})
	require.True(t, ok, "documents should be an array")
	require.NotNil(t, documents)

	// --- List Documents with query parameters ---
	listDocWithParamsReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/documents/"+collectionID+"?pageSize=10&orderBy=name&showMissing=true", nil)
	listDocWithParamsResp, err := app.Test(listDocWithParamsReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listDocWithParamsResp.StatusCode)

	// Verify response structure for parameterized request
	var listResponseWithParams map[string]interface{}
	listBodyWithParams, err := io.ReadAll(listDocWithParamsResp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(listBodyWithParams, &listResponseWithParams))
	require.Contains(t, listResponseWithParams, "documents")
	require.Contains(t, listResponseWithParams, "count")

	// --- Delete Document ---
	deleteReq := httptest.NewRequest(stdhttp.MethodDelete, basePath+"/documents/"+collectionID+"/"+documentID, nil)
	deleteResp, err := app.Test(deleteReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusNoContent, deleteResp.StatusCode)

	// --- Query Documents ---
	queryBody := `{"projectId": "` + projectID + `", "databaseId": "` + databaseID + `", "structuredQuery": {"collectionId": "` + collectionID + `"}}`
	queryReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/query/"+collectionID, strings.NewReader(queryBody))
	queryReq.Header.Set("Content-Type", "application/json")
	queryResp, err := app.Test(queryReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, queryResp.StatusCode)
}

// Integration test for collection routes
func TestCollectionRoutes_Integration(t *testing.T) {
	app := fiber.New()
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
	h.RegisterRoutes(app)

	orgID := "org-ponceca"
	projectID := "project01"
	databaseID := "database01"
	collectionID := "collection01"
	documentID := "document01"

	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	// --- List Collections ---
	listReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/collections", nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listResp.StatusCode)

	// --- Create Collection ---
	createBody := `{"collectionId": "` + collectionID + `"}`
	createReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/collections", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, createResp.StatusCode)

	// --- Get Collection ---
	getReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/collections/"+collectionID, nil)
	getResp, err := app.Test(getReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, getResp.StatusCode)
	var getCol map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&getCol))
	require.Equal(t, collectionID, getCol["collectionID"])

	// --- Update Collection ---
	updateBody := `{"collection": {"displayName": "Updated Collection"}}`
	updateReq := httptest.NewRequest(stdhttp.MethodPut, basePath+"/collections/"+collectionID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, updateResp.StatusCode)

	// --- Delete Collection ---
	deleteReq := httptest.NewRequest(stdhttp.MethodDelete, basePath+"/collections/"+collectionID, nil)
	deleteResp, err := app.Test(deleteReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusNoContent, deleteResp.StatusCode)

	// --- List Subcollections ---
	subcolReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/documents/"+collectionID+"/"+documentID+"/subcollections", nil)
	subcolResp, err := app.Test(subcolReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, subcolResp.StatusCode)
}

// Integration test for index routes
func TestIndexRoutes_Integration(t *testing.T) {
	app := fiber.New()
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), nil, nil, nil, &usecase.MockLogger{},
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)

	orgID, projectID, databaseID, collectionID, indexID := "org-ponceca", "project01", "database01", "collection01", "idx1"
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	// --- Create Index ---
	createBody := `{"index": {"name": "` + indexID + `", "fields": [{"path": "f1", "order": "ASCENDING"}], "state": "READY", "collection": "` + collectionID + `"}}`
	createReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/collections/"+collectionID+"/indexes", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, createResp.StatusCode)

	// --- List Indexes ---
	listReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/collections/"+collectionID+"/indexes", nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listResp.StatusCode)

	// --- Delete Index ---
	deleteReq := httptest.NewRequest(stdhttp.MethodDelete, basePath+"/collections/"+collectionID+"/indexes/"+indexID, nil)
	deleteResp, err := app.Test(deleteReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusNoContent, deleteResp.StatusCode)
}

// Integration test for batch write route
func TestBatchWriteRoute_Integration(t *testing.T) {
	app := fiber.New()
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), nil, nil, nil, &usecase.MockLogger{},
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)

	orgID, projectID, databaseID := "org-ponceca", "project01", "database01"
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	batchBody := `{"projectId": "` + projectID + `", "databaseId": "` + databaseID + `", "writes": [{"type": "create", "documentId": "doc1", "path": "/organizations/` + orgID + `/projects/` + projectID + `/databases/` + databaseID + `/documents/c1/doc1", "data": {"field": "value"}}]}`
	batchReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/batchWrite", strings.NewReader(batchBody))
	batchReq.Header.Set("Content-Type", "application/json")
	batchResp, err := app.Test(batchReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, batchResp.StatusCode)
}

// Integration test for transaction routes
func TestTransactionRoutes_Integration(t *testing.T) {
	app := fiber.New()
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), nil, nil, nil, &usecase.MockLogger{},
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)
	orgID, projectID, databaseID := "org-ponceca", "project01", "database01"
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	// --- Begin Transaction ---
	beginReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/beginTransaction", nil)
	beginResp, err := app.Test(beginReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, beginResp.StatusCode)
	// Parse response to get transaction ID
	var beginResponse struct {
		TransactionID string `json:"transactionId"`
	}
	bodyBytes, err := io.ReadAll(beginResp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(bodyBytes, &beginResponse)
	require.NoError(t, err)
	require.NotEmpty(t, beginResponse.TransactionID)

	// --- Commit Transaction ---
	commitBody := `{"transactionId": "` + beginResponse.TransactionID + `"}`
	commitReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/commit", strings.NewReader(commitBody))
	commitReq.Header.Set("Content-Type", "application/json")
	commitResp, err := app.Test(commitReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, commitResp.StatusCode)
}

// Integration test for atomic operation routes
func TestAtomicRoutes_Integration(t *testing.T) {
	app := fiber.New()
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), nil, nil, nil, &usecase.MockLogger{},
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)

	orgID, projectID, databaseID, collectionID, documentID := "org-ponceca", "project01", "database01", "collection01", "document01"
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases/" + databaseID

	// --- Atomic Increment ---
	incBody := `{"field": "counter", "incrementBy": 1}`
	incReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/documents/"+collectionID+"/"+documentID+"/increment", strings.NewReader(incBody))
	incReq.Header.Set("Content-Type", "application/json")
	incResp, err := app.Test(incReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, incResp.StatusCode)

	// --- Array Union ---
	unionBody := `{"field": "tags", "elements": ["a"]}`
	unionReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/documents/"+collectionID+"/"+documentID+"/arrayUnion", strings.NewReader(unionBody))
	unionReq.Header.Set("Content-Type", "application/json")
	unionResp, err := app.Test(unionReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, unionResp.StatusCode)

	// --- Array Remove ---
	removeBody := `{"field": "tags", "elements": ["a"]}`
	removeReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/documents/"+collectionID+"/"+documentID+"/arrayRemove", strings.NewReader(removeBody))
	removeReq.Header.Set("Content-Type", "application/json")
	removeResp, err := app.Test(removeReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, removeResp.StatusCode)

	// --- Server Timestamp ---
	tsBody := `{"field": "updatedAt"}`
	tsReq := httptest.NewRequest(stdhttp.MethodPost, basePath+"/documents/"+collectionID+"/"+documentID+"/serverTimestamp", strings.NewReader(tsBody))
	tsReq.Header.Set("Content-Type", "application/json")
	tsResp, err := app.Test(tsReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, tsResp.StatusCode)
}

// Integration test for project routes
// Mock in-memory project repository for robust integration
// Embeds the original mock and only overrides project methods

type InMemoryProjectRepo struct {
	usecase.MockFirestoreRepo
	mu       sync.Mutex
	projects map[string]*model.Project
}

func NewInMemoryProjectRepo() *InMemoryProjectRepo {
	return &InMemoryProjectRepo{projects: make(map[string]*model.Project)}
}

func (r *InMemoryProjectRepo) CreateProject(_ context.Context, p *model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.projects[p.ProjectID]; exists {
		return model.ErrInvalidProjectID // Simula error de duplicado
	}
	r.projects[p.ProjectID] = p
	return nil
}
func (r *InMemoryProjectRepo) GetProject(_ context.Context, id string) (*model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.projects[id]
	if !ok {
		return nil, model.ErrProjectNotFound
	}
	return p, nil
}
func (r *InMemoryProjectRepo) UpdateProject(_ context.Context, p *model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[p.ProjectID]; !ok {
		return model.ErrProjectNotFound
	}
	r.projects[p.ProjectID] = p
	return nil
}
func (r *InMemoryProjectRepo) DeleteProject(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[id]; !ok {
		return model.ErrProjectNotFound
	}
	delete(r.projects, id)
	return nil
}
func (r *InMemoryProjectRepo) ListProjects(_ context.Context, _ string) ([]*model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Project
	for _, p := range r.projects {
		out = append(out, p)
	}
	return out, nil
}

func TestProjectRoutes_Integration(t *testing.T) {
	app := fiber.New()
	mockRepo := usecase.NewMockFirestoreRepo() // Use the correct mock constructor with package prefix
	uc := usecase.NewFirestoreUsecase(
		mockRepo, nil, nil, nil, &usecase.MockLogger{},
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)

	orgID, projectID := "org-ponceca", "project-ponceca"
	basePath := "/organizations/" + orgID + "/projects"

	// --- Create Project ---
	createBody := `{ "project": { "projectId": "` + projectID + `", "organizationId": "` + orgID + `", "displayName": "Test Project" } }`
	createReq := httptest.NewRequest(stdhttp.MethodPost, basePath, strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, createResp.StatusCode)

	// --- List Projects ---
	listReq := httptest.NewRequest(stdhttp.MethodGet, basePath, nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listResp.StatusCode)

	// --- Get Project ---
	getReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/"+projectID, nil)
	getResp, err := app.Test(getReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, getResp.StatusCode)

	// --- Update Project ---
	updateBody := `{"project": {"displayName": "Updated Project", "organizationId": "` + orgID + `"}}`
	updateReq := httptest.NewRequest(stdhttp.MethodPut, basePath+"/"+projectID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, updateResp.StatusCode)

	// --- Delete Project ---
	deleteReq := httptest.NewRequest(stdhttp.MethodDelete, basePath+"/"+projectID, nil)
	deleteResp, err := app.Test(deleteReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusNoContent, deleteResp.StatusCode)
}

// Integration test for database routes
func TestDatabaseRoutes_Integration(t *testing.T) {
	app := fiber.New()
	uc := usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), nil, nil, nil, &usecase.MockLogger{}, // Use the correct mock constructor with package prefix
	)
	h := &httpadapter.HTTPHandler{FirestoreUC: uc, Log: &usecase.MockLogger{}}
	h.RegisterRoutes(app)

	orgID, projectID := "org-ponceca", "project-ponceca"
	databaseID := "dbponceca" + RandString(8) // Genera un ID único y válido
	basePath := "/organizations/" + orgID + "/projects/" + projectID + "/databases"

	// --- Crear el proyecto necesario antes de la base de datos ---
	createProjectBody := `{ "project": { "projectId": "` + projectID + `", "organizationId": "` + orgID + `", "displayName": "Test Project" } }`
	createProjectReq := httptest.NewRequest(stdhttp.MethodPost, "/organizations/"+orgID+"/projects", strings.NewReader(createProjectBody))
	createProjectReq.Header.Set("Content-Type", "application/json")
	createProjectResp, err := app.Test(createProjectReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, createProjectResp.StatusCode)

	// --- Create Database ---
	createBody := `{ "projectId": "` + projectID + `", "database": { "databaseId": "` + databaseID + `", "name": "Test DB" } }`
	createReq := httptest.NewRequest(stdhttp.MethodPost, basePath, strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusCreated, createResp.StatusCode)

	// --- List Databases ---
	listReq := httptest.NewRequest(stdhttp.MethodGet, basePath, nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, listResp.StatusCode)

	// --- Get Database ---
	getReq := httptest.NewRequest(stdhttp.MethodGet, basePath+"/"+databaseID, nil)
	getResp, err := app.Test(getReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, getResp.StatusCode)

	// --- Update Database ---
	updateBody := `{ "projectId": "` + projectID + `", "database": { "databaseId": "` + databaseID + `", "name": "Updated DB" } }`
	updateReq := httptest.NewRequest(stdhttp.MethodPut, basePath+"/"+databaseID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusOK, updateResp.StatusCode)
	// --- Delete Database ---
	deleteReq := httptest.NewRequest(stdhttp.MethodDelete, basePath+"/"+databaseID, nil)
	deleteResp, err := app.Test(deleteReq)
	require.NoError(t, err)
	require.Equal(t, stdhttp.StatusNoContent, deleteResp.StatusCode)
}

// TestQueryWithProjectionIntegration tests that projection queries don't crash
func TestQueryWithProjectionIntegration(t *testing.T) {
	t.Run("Verify projection query API structure", func(t *testing.T) {
		app := fiber.New()

		// Use mock usecase
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

		// Register query routes
		setupQueryRoutes(app, h)

		// Test that a query with projection is accepted and doesn't crash
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
		require.NotEqual(t, stdhttp.StatusInternalServerError, resp.StatusCode,
			"Query with projection should not fail with internal server error")
	})
}

// Helper function to setup query routes for testing
func setupQueryRoutes(app *fiber.App, h *httpadapter.HTTPHandler) {
	v1 := app.Group("/v1")
	projects := v1.Group("/projects/:projectId")
	databases := projects.Group("/databases/:databaseId")
	documents := databases.Group("/documents")

	// Query endpoints
	documents.Post(":runQuery", h.RunQuery)
	documents.Post("*:runQuery", h.RunQuery)
}

// Helper functions for creating pointer values
func StringPtr(s string) *string    { return &s }
func BoolPtr(b bool) *bool          { return &b }
func Int64Ptr(i int64) *int64       { return &i }
func Float64Ptr(f float64) *float64 { return &f }
