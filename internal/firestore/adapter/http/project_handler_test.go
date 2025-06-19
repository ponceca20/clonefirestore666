package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors" // Standard errors package
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	sharedErrors "firestore-clone/internal/shared/errors" // Alias for shared errors

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProjectMockFirestoreUC provides dummy implementations for FirestoreUsecaseInterface
// to satisfy interface compliance in project tests. Following hexagonal architecture
// by keeping the adapter layer isolated from application logic.
type ProjectMockFirestoreUC struct{}

// Project methods - basic implementations that can be overridden by customProjectUC
func (m *ProjectMockFirestoreUC) CreateProject(context.Context, usecase.CreateProjectRequest) (*model.Project, error) {
	return &model.Project{ProjectID: "default-project"}, nil
}
func (m *ProjectMockFirestoreUC) GetProject(context.Context, usecase.GetProjectRequest) (*model.Project, error) {
	return &model.Project{ProjectID: "default-project"}, nil
}
func (m *ProjectMockFirestoreUC) UpdateProject(context.Context, usecase.UpdateProjectRequest) (*model.Project, error) {
	return &model.Project{ProjectID: "default-project"}, nil
}
func (m *ProjectMockFirestoreUC) DeleteProject(context.Context, usecase.DeleteProjectRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListProjects(context.Context, usecase.ListProjectsRequest) ([]*model.Project, error) {
	return []*model.Project{{ProjectID: "default-project"}}, nil
}

// Dummy implementations for interface compliance - not used in project tests
func (m *ProjectMockFirestoreUC) CreateDocument(context.Context, usecase.CreateDocumentRequest) (*model.Document, error) {
	return &model.Document{DocumentID: "doc1"}, nil
}
func (m *ProjectMockFirestoreUC) GetDocument(context.Context, usecase.GetDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) UpdateDocument(context.Context, usecase.UpdateDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) DeleteDocument(context.Context, usecase.DeleteDocumentRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListDocuments(context.Context, usecase.ListDocumentsRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) CreateCollection(context.Context, usecase.CreateCollectionRequest) (*model.Collection, error) {
	return &model.Collection{CollectionID: "c1"}, nil
}
func (m *ProjectMockFirestoreUC) GetCollection(context.Context, usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) UpdateCollection(context.Context, usecase.UpdateCollectionRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListCollections(context.Context, usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) DeleteCollection(context.Context, usecase.DeleteCollectionRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListSubcollections(context.Context, usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) CreateIndex(context.Context, usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) DeleteIndex(context.Context, usecase.DeleteIndexRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListIndexes(context.Context, usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) QueryDocuments(context.Context, usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) RunQuery(context.Context, usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) RunAggregationQuery(context.Context, usecase.AggregationQueryRequest) (*usecase.AggregationQueryResponse, error) {
	return &usecase.AggregationQueryResponse{}, nil
}
func (m *ProjectMockFirestoreUC) RunBatchWrite(context.Context, usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) BeginTransaction(context.Context, string) (string, error) {
	return "", nil
}
func (m *ProjectMockFirestoreUC) CommitTransaction(context.Context, string, string) error { return nil }
func (m *ProjectMockFirestoreUC) CreateDatabase(context.Context, usecase.CreateDatabaseRequest) (*model.Database, error) {
	return &model.Database{DatabaseID: "db1"}, nil
}
func (m *ProjectMockFirestoreUC) GetDatabase(context.Context, usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) UpdateDatabase(context.Context, usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) DeleteDatabase(context.Context, usecase.DeleteDatabaseRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) ListDatabases(context.Context, usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, nil
}
func (m *ProjectMockFirestoreUC) AtomicIncrement(context.Context, usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	return &usecase.AtomicIncrementResponse{}, nil
}
func (m *ProjectMockFirestoreUC) AtomicArrayUnion(context.Context, usecase.AtomicArrayUnionRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) AtomicArrayRemove(context.Context, usecase.AtomicArrayRemoveRequest) error {
	return nil
}
func (m *ProjectMockFirestoreUC) AtomicServerTimestamp(context.Context, usecase.AtomicServerTimestampRequest) error {
	return nil
}

// Local test double embedding ProjectMockFirestoreUC for custom project logic
// This follows hexagonal architecture by mocking the application layer
type customProjectUC struct {
	ProjectMockFirestoreUC
	createProjectFunc func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error)
	getProjectFunc    func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error)
	listProjectsFunc  func(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error)
	updateProjectFunc func(ctx context.Context, req usecase.UpdateProjectRequest) (*model.Project, error)
	deleteProjectFunc func(ctx context.Context, req usecase.DeleteProjectRequest) error
}

func (m *customProjectUC) CreateProject(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
	if m.createProjectFunc != nil {
		return m.createProjectFunc(ctx, req)
	}
	return m.ProjectMockFirestoreUC.CreateProject(ctx, req)
}

func (m *customProjectUC) GetProject(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(ctx, req)
	}
	return m.ProjectMockFirestoreUC.GetProject(ctx, req)
}

func (m *customProjectUC) ListProjects(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(ctx, req)
	}
	return m.ProjectMockFirestoreUC.ListProjects(ctx, req)
}

func (m *customProjectUC) UpdateProject(ctx context.Context, req usecase.UpdateProjectRequest) (*model.Project, error) {
	if m.updateProjectFunc != nil {
		return m.updateProjectFunc(ctx, req)
	}
	return m.ProjectMockFirestoreUC.UpdateProject(ctx, req)
}

func (m *customProjectUC) DeleteProject(ctx context.Context, req usecase.DeleteProjectRequest) error {
	if m.deleteProjectFunc != nil {
		return m.deleteProjectFunc(ctx, req)
	}
	return m.ProjectMockFirestoreUC.DeleteProject(ctx, req)
}

// --- CreateProject Tests ---

func TestCreateProjectHandler_Success(t *testing.T) {
	app := fiber.New()

	expectedProject := &model.Project{
		ProjectID:      "test-project-123",
		OrganizationID: "test-org-456",
		DisplayName:    "Test Project",
		LocationID:     "us-central1",
	}

	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			// Verify that organization ID was set from URL path
			assert.Equal(t, "test-org-456", req.Project.OrganizationID)
			assert.Equal(t, "test-project-123", req.Project.ProjectID)
			return expectedProject, nil
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	requestBody := map[string]interface{}{
		"project": map[string]interface{}{
			"projectID":   "test-project-123",
			"displayName": "Test Project",
			"locationId":  "us-central1",
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/organizations/test-org-456/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result model.Project
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, expectedProject.ProjectID, result.ProjectID)
	assert.Equal(t, expectedProject.OrganizationID, result.OrganizationID)
}

func TestCreateProjectHandler_MissingOrganizationID(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Route without organizationId parameter
	app.Post("/projects", h.CreateProject)

	requestBody := map[string]interface{}{
		"project": map[string]interface{}{
			"projectID":   "test-project-123",
			"displayName": "Test Project",
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "missing_organization_id", result["error"])
}

func TestCreateProjectHandler_MissingProjectInBody(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	// Request without project field
	requestBody := map[string]interface{}{
		"name": "should be ignored",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/organizations/test-org/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "missing_project", result["error"])
}

func TestCreateProjectHandler_InvalidRequestBody(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	// Invalid JSON
	req := httptest.NewRequest("POST", "/organizations/test-org/projects", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request_body", result["error"])
}

func TestCreateProjectHandler_ValidationError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			return nil, sharedErrors.NewValidationError("Project ID is required")
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	requestBody := map[string]interface{}{
		"project": map[string]interface{}{
			"displayName": "Test Project",
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/organizations/test-org/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "validation_failed", result["error"])
	assert.Contains(t, result["message"], "Project ID is required")
}

func TestCreateProjectHandler_ConflictError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			conflictErr := sharedErrors.NewConflictError("Project already exists")
			return nil, conflictErr
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	requestBody := map[string]interface{}{
		"project": map[string]interface{}{
			"projectID":   "existing-project",
			"displayName": "Test Project",
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/organizations/test-org/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "project_already_exists", result["error"])
}

func TestCreateProjectHandler_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			return nil, sharedErrors.NewInternalError("Database connection failed")
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Post("/organizations/:organizationId/projects", h.CreateProject)

	requestBody := map[string]interface{}{
		"project": map[string]interface{}{
			"projectID":   "test-project",
			"displayName": "Test Project",
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/organizations/test-org/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "create_project_failed", result["error"])
}

// --- GetProject Tests ---

func TestGetProjectHandler_Success(t *testing.T) {
	app := fiber.New()

	expectedProject := &model.Project{
		ProjectID:      "test-project-123",
		OrganizationID: "test-org-456",
		DisplayName:    "Test Project",
	}

	mockUC := &customProjectUC{
		getProjectFunc: func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
			assert.Equal(t, "test-project-123", req.ProjectID)
			return expectedProject, nil
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects/:projectID", h.GetProject)

	req := httptest.NewRequest("GET", "/organizations/test-org-456/projects/test-project-123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result model.Project
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, expectedProject.ProjectID, result.ProjectID)
	assert.Equal(t, expectedProject.OrganizationID, result.OrganizationID)
}

func TestGetProjectHandler_NotFound(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		getProjectFunc: func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
			return nil, sharedErrors.NewNotFoundError("Project 'nonexistent-project'")
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects/:projectID", h.GetProject)

	req := httptest.NewRequest("GET", "/organizations/test-org/projects/nonexistent-project", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "project_not_found", result["error"])
}

func TestGetProjectHandler_InternalError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		getProjectFunc: func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
			return nil, sharedErrors.NewInternalError("Database connection failed")
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects/:projectID", h.GetProject)

	req := httptest.NewRequest("GET", "/organizations/test-org/projects/test-project", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Handler treats all errors as not found

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "project_not_found", result["error"])
}

func TestListProjectsHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		listProjectsFunc: func(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
			return []*model.Project{
				{ProjectID: "p1", OrganizationID: req.OrganizationID},
				{ProjectID: "p2", OrganizationID: req.OrganizationID},
			}, nil
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects", h.ListProjects)

	req := httptest.NewRequest("GET", "/organizations/my-org/projects?ownerEmail=test@example.com", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), result["count"])

	projects, ok := result["projects"].([]interface{})
	require.True(t, ok)
	assert.Len(t, projects, 2)
}

func TestListProjectsHandler_MissingOrganizationID(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects", h.ListProjects)

	// Test with empty organization ID - Fiber treats this as route not found
	req := httptest.NewRequest("GET", "/organizations//projects", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode) // Fiber returns 404 for malformed routes
	// Test with missing organizationId parameter by using URL encoded space (which gets trimmed)
	req2 := httptest.NewRequest("GET", "/organizations/%20/projects", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 400, resp2.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp2.Body).Decode(&result)
	assert.Equal(t, "missing_organization_id", result["error"])
}

func TestListProjectsHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		listProjectsFunc: func(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
			return nil, errors.New("internal error")
		},
	}

	h := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}
	app.Get("/organizations/:organizationId/projects", h.ListProjects)

	req := httptest.NewRequest("GET", "/organizations/my-org/projects", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "list_projects_failed", result["error"])
}
