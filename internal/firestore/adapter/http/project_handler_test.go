package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Local test double embedding MockFirestoreUC for custom project logic
// This avoids redeclaration and ensures interface compliance

type customProjectUC struct {
	MockFirestoreUC
	createProjectFunc func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error)
	getProjectFunc    func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error)
	listProjectsFunc  func(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error)
}

func (m *customProjectUC) CreateProject(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
	if m.createProjectFunc != nil {
		return m.createProjectFunc(ctx, req)
	}
	return m.MockFirestoreUC.CreateProject(ctx, req)
}
func (m *customProjectUC) GetProject(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(ctx, req)
	}
	return m.MockFirestoreUC.GetProject(ctx, req)
}
func (m *customProjectUC) ListProjects(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(ctx, req)
	}
	return m.MockFirestoreUC.ListProjects(ctx, req)
}

func TestCreateProjectHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			return &model.Project{ProjectID: "p1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test", h.CreateProject)

	body := []byte(`{"name":"p1"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "p1", result["projectID"])
}

func TestCreateProjectHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		createProjectFunc: func(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test", h.CreateProject)

	body := []byte(`{"name":"p1"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_project_failed", result["error"])
}

func TestGetProjectHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		getProjectFunc: func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
			return &model.Project{ProjectID: "p1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Get("/test/:projectID", h.GetProject)

	req := httptest.NewRequest("GET", "/test/p1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "p1", result["projectID"])
}

func TestGetProjectHandler_NotFound(t *testing.T) {
	app := fiber.New()
	mockUC := &customProjectUC{
		getProjectFunc: func(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Get("/test/:projectID", h.GetProject)

	req := httptest.NewRequest("GET", "/test/p1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
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

	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
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
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
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

	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Get("/organizations/:organizationId/projects", h.ListProjects)

	req := httptest.NewRequest("GET", "/organizations/my-org/projects", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "list_projects_failed", result["error"])
}
