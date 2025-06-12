package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	firestorehttp "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/contextkeys"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OrganizationHandlerTestSuite struct {
	suite.Suite
	app      *fiber.App
	mockRepo *firestorehttp.MockOrganizationRepo
	handler  *firestorehttp.OrganizationHandler
	testOrg  *model.Organization
	testTime time.Time
}

func (suite *OrganizationHandlerTestSuite) SetupTest() {
	suite.app = fiber.New()
	suite.mockRepo = new(firestorehttp.MockOrganizationRepo)
	suite.handler = firestorehttp.NewOrganizationHandler(suite.mockRepo)
	suite.testTime = time.Now()

	// Create test organization
	var err error
	suite.testOrg, err = model.NewOrganization("test-org-123", "Test Organization", "billing@test.com")
	require.NoError(suite.T(), err)
	suite.testOrg.Description = "Test organization description"
	suite.testOrg.DefaultLocation = "us-central1"
	suite.testOrg.AdminEmails = []string{"admin@test.com", "admin2@test.com"}
	suite.testOrg.CreatedAt = suite.testTime
	suite.testOrg.UpdatedAt = suite.testTime
	suite.testOrg.ProjectCount = 5
	suite.testOrg.Usage = &model.OrganizationUsage{
		ProjectCount:  5,
		DatabaseCount: 15,
		StorageBytes:  1024000,
	}
	suite.testOrg.Quotas = &model.OrganizationQuotas{
		MaxProjects:     100,
		MaxDatabases:    500,
		MaxStorageBytes: 1073741824, // 1GB
	}
	// Register routes
	suite.handler.RegisterRoutes(suite.app)
}

func (suite *OrganizationHandlerTestSuite) TearDownTest() {
	// Clear mock expectations after each test
	suite.mockRepo.ExpectedCalls = nil
	suite.mockRepo.Calls = nil
}

func TestOrganizationHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationHandlerTestSuite))
}

// ============== CREATE ORGANIZATION TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_Success() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID:  "new-org-456",
		DisplayName:     "New Organization",
		Description:     "A new test organization",
		BillingEmail:    "billing@neworg.com",
		AdminEmails:     []string{"admin@neworg.com"},
		DefaultLocation: "us-west1",
	}

	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.MatchedBy(func(org *model.Organization) bool {
		return org.OrganizationID == req.OrganizationID &&
			org.DisplayName == req.DisplayName &&
			org.BillingEmail == req.BillingEmail
	})).Return(nil)

	body, err := json.Marshal(req)
	require.NoError(suite.T(), err)

	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response firestorehttp.OrganizationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "organizations/"+req.OrganizationID, response.Name)
	assert.Equal(suite.T(), req.OrganizationID, response.OrganizationID)
	assert.Equal(suite.T(), req.DisplayName, response.DisplayName)
	assert.Equal(suite.T(), req.BillingEmail, response.BillingEmail)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_MissingOrganizationID() {
	req := firestorehttp.CreateOrganizationRequest{
		DisplayName:  "Test Org",
		BillingEmail: "billing@test.com",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "missing_organization_id", errorResp["error"])

	suite.mockRepo.AssertNotCalled(suite.T(), "CreateOrganization")
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_MissingDisplayName() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID: "test-org",
		BillingEmail:   "billing@test.com",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "missing_display_name", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_MissingBillingEmail() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID: "test-org",
		DisplayName:    "Test Org",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "missing_billing_email", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_InvalidJSON() {
	invalidJSON := `{"organizationId": "test", "displayName": incomplete`

	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader([]byte(invalidJSON)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "invalid_request_body", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_AlreadyExists() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID: "existing-org",
		DisplayName:    "Existing Org",
		BillingEmail:   "billing@existing.com",
	}

	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.Anything).Return(model.ErrOrganizationExists)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusConflict, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "organization_already_exists", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_RepositoryError() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID: "test-org",
		DisplayName:    "Test Org",
		BillingEmail:   "billing@test.com",
	}

	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.Anything).Return(errors.New("database connection failed"))

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "create_organization_failed", errorResp["error"])
}

// ============== GET ORGANIZATION TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestGetOrganization_Success() {
	orgID := "test-org-123"
	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(suite.testOrg, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.OrganizationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "organizations/"+orgID, response.Name)
	assert.Equal(suite.T(), suite.testOrg.OrganizationID, response.OrganizationID)
	assert.Equal(suite.T(), suite.testOrg.DisplayName, response.DisplayName)
	assert.Equal(suite.T(), suite.testOrg.BillingEmail, response.BillingEmail)
	assert.Equal(suite.T(), suite.testOrg.ProjectCount, response.ProjectCount)
	assert.NotNil(suite.T(), response.Usage)
	assert.NotNil(suite.T(), response.Quotas)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestGetOrganization_NotFound() {
	orgID := "non-existent-org"
	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(nil, model.ErrOrganizationNotFound)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "organization_not_found", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestGetOrganization_RepositoryError() {
	orgID := "test-org-123"
	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(nil, errors.New("database error"))

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "get_organization_failed", errorResp["error"])
}

// ============== LIST ORGANIZATIONS TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_Success() {
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	org2, err := model.NewOrganization("test-org-002", "Organization 2", "billing2@test.com")
	require.NoError(suite.T(), err)
	orgs := []*model.Organization{org1, org2}

	suite.mockRepo.On("ListOrganizations", mock.Anything, 10, 0).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.ListOrganizationsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Organizations, 2)
	assert.Equal(suite.T(), "organizations/test-org-001", response.Organizations[0].Name)
	assert.Equal(suite.T(), "organizations/test-org-002", response.Organizations[1].Name)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_WithPagination() {
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	orgs := []*model.Organization{org1}

	suite.mockRepo.On("ListOrganizations", mock.Anything, 5, 10).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations?pageSize=5&offset=10", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_InvalidPagination() {
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	orgs := []*model.Organization{org1}

	// Should fallback to defaults when invalid values are provided
	suite.mockRepo.On("ListOrganizations", mock.Anything, 10, 0).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations?pageSize=invalid&offset=-5", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_LargePagination() {
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	orgs := []*model.Organization{org1}

	// Should cap pageSize at 100
	suite.mockRepo.On("ListOrganizations", mock.Anything, 100, 0).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations?pageSize=500", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_ByAdminEmail() {
	adminEmail := "admin@test.com"
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	org1.AdminEmails = []string{adminEmail}
	orgs := []*model.Organization{org1}

	suite.mockRepo.On("ListOrganizationsByAdmin", mock.Anything, adminEmail).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations?admin_email="+adminEmail, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.ListOrganizationsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Organizations, 1)
	assert.Contains(suite.T(), response.Organizations[0].AdminEmails, adminEmail)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_RepositoryError() {
	suite.mockRepo.On("ListOrganizations", mock.Anything, 10, 0).Return(nil, errors.New("database error"))

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "list_organizations_failed", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_EmptyResult() {
	suite.mockRepo.On("ListOrganizations", mock.Anything, 10, 0).Return([]*model.Organization{}, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.ListOrganizationsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Organizations, 0)
	assert.NotNil(suite.T(), response.Organizations) // Should be empty array, not nil

	suite.mockRepo.AssertExpectations(suite.T())
}

// ============== UPDATE ORGANIZATION TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestUpdateOrganization_Success() {
	orgID := "test-org-123"
	updatedOrg := *suite.testOrg
	updatedOrg.DisplayName = "Updated Organization Name"
	updatedOrg.Description = "Updated description"

	req := firestorehttp.UpdateOrganizationRequest{
		DisplayName:     "Updated Organization Name",
		Description:     "Updated description",
		AdminEmails:     []string{"newadmin@test.com"},
		DefaultLocation: "us-east1",
	}

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(suite.testOrg, nil)
	suite.mockRepo.On("UpdateOrganization", mock.Anything, mock.MatchedBy(func(org *model.Organization) bool {
		return org.OrganizationID == orgID &&
			org.DisplayName == req.DisplayName &&
			org.Description == req.Description
	})).Return(nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.OrganizationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), req.DisplayName, response.DisplayName)
	assert.Equal(suite.T(), req.Description, response.Description)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestUpdateOrganization_PartialUpdate() {
	orgID := "test-org-123"
	req := firestorehttp.UpdateOrganizationRequest{
		DisplayName: "Only Update Display Name",
		// Other fields empty - should not override existing values
	}

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(suite.testOrg, nil)
	suite.mockRepo.On("UpdateOrganization", mock.Anything, mock.MatchedBy(func(org *model.Organization) bool {
		return org.OrganizationID == orgID &&
			org.DisplayName == req.DisplayName &&
			org.Description == suite.testOrg.Description // Should keep original description
	})).Return(nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestUpdateOrganization_NotFound() {
	orgID := "non-existent-org"
	req := firestorehttp.UpdateOrganizationRequest{
		DisplayName: "Updated Name",
	}

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(nil, model.ErrOrganizationNotFound)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "organization_not_found", errorResp["error"])

	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateOrganization")
}

func (suite *OrganizationHandlerTestSuite) TestUpdateOrganization_InvalidJSON() {
	orgID := "test-org-123"
	invalidJSON := `{"displayName": "incomplete`

	httpReq := httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader([]byte(invalidJSON)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "invalid_request_body", errorResp["error"])

	suite.mockRepo.AssertNotCalled(suite.T(), "GetOrganization")
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateOrganization")
}

func (suite *OrganizationHandlerTestSuite) TestUpdateOrganization_UpdateError() {
	orgID := "test-org-123"
	req := firestorehttp.UpdateOrganizationRequest{
		DisplayName: "Updated Name",
	}

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(suite.testOrg, nil)
	suite.mockRepo.On("UpdateOrganization", mock.Anything, mock.Anything).Return(errors.New("update failed"))

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "update_organization_failed", errorResp["error"])
}

// ============== DELETE ORGANIZATION TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestDeleteOrganization_Success() {
	orgID := "test-org-123"
	suite.mockRepo.On("DeleteOrganization", mock.Anything, orgID).Return(nil)

	httpReq := httptest.NewRequest(http.MethodDelete, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)

	// Response body should be empty for 204 No Content
	body := make([]byte, 1)
	n, _ := resp.Body.Read(body)
	assert.Equal(suite.T(), 0, n)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestDeleteOrganization_NotFound() {
	orgID := "non-existent-org"
	suite.mockRepo.On("DeleteOrganization", mock.Anything, orgID).Return(model.ErrOrganizationNotFound)

	httpReq := httptest.NewRequest(http.MethodDelete, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "organization_not_found", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestDeleteOrganization_RepositoryError() {
	orgID := "test-org-123"
	suite.mockRepo.On("DeleteOrganization", mock.Anything, orgID).Return(errors.New("database error"))

	httpReq := httptest.NewRequest(http.MethodDelete, "/organizations/"+orgID, nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "delete_organization_failed", errorResp["error"])
}

// ============== ORGANIZATION PROJECTS TESTS ==============

// ============== ORGANIZATION USAGE TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestGetOrganizationUsage_Success() {
	orgID := "test-org-123"

	// Create app with tenant middleware context
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		ctx := context.WithValue(c.UserContext(), contextkeys.OrganizationIDKey, orgID)
		c.SetUserContext(ctx)
		return c.Next()
	})
	suite.handler.RegisterRoutes(app)

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(suite.testOrg, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID+"/usage", nil)
	resp, err := app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response firestorehttp.OrganizationUsageResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), orgID, response.OrganizationID)
	assert.NotNil(suite.T(), response.Usage)
	assert.NotNil(suite.T(), response.Quotas)
	assert.Equal(suite.T(), suite.testOrg.Usage.ProjectCount, response.Usage.ProjectCount)

	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *OrganizationHandlerTestSuite) TestGetOrganizationUsage_MissingOrgID() {
	// Test endpoint without tenant middleware context - this should fail with missing org ID
	app := fiber.New()

	// Register only the usage endpoint directly without the middleware
	app.Get("/organizations/:organizationId/usage", suite.handler.GetOrganizationUsage)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/test-org-123/usage", nil)
	resp, err := app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "missing_organization_id", errorResp["error"])
}

func (suite *OrganizationHandlerTestSuite) TestGetOrganizationUsage_OrganizationNotFound() {
	orgID := "non-existent-org"

	// Create app with tenant middleware context
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		ctx := context.WithValue(c.UserContext(), contextkeys.OrganizationIDKey, orgID)
		c.SetUserContext(ctx)
		return c.Next()
	})
	suite.handler.RegisterRoutes(app)

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(nil, model.ErrOrganizationNotFound)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations/"+orgID+"/usage", nil)
	resp, err := app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Equal(suite.T(), "organization_not_found", errorResp["error"])
}

// ============== INTEGRATION TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestCompleteOrganizationLifecycle() {
	// Test complete CRUD operations
	orgID := "lifecycle-test-org"

	// 1. Create Organization
	createReq := firestorehttp.CreateOrganizationRequest{
		OrganizationID:  orgID,
		DisplayName:     "Lifecycle Test Org",
		BillingEmail:    "billing@lifecycle.com",
		AdminEmails:     []string{"admin@lifecycle.com"},
		DefaultLocation: "us-central1",
	}

	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.Anything).Return(nil).Once()

	body, _ := json.Marshal(createReq)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, _ := suite.app.Test(httpReq)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// 2. Get Organization
	createdOrg, _ := model.NewOrganization(orgID, createReq.DisplayName, createReq.BillingEmail)
	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(createdOrg, nil).Once()

	httpReq = httptest.NewRequest(http.MethodGet, "/organizations/"+orgID, nil)
	resp, _ = suite.app.Test(httpReq)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 3. Update Organization
	updateReq := firestorehttp.UpdateOrganizationRequest{
		DisplayName: "Updated Lifecycle Org",
		Description: "Updated description",
	}

	suite.mockRepo.On("GetOrganization", mock.Anything, orgID).Return(createdOrg, nil).Once()
	suite.mockRepo.On("UpdateOrganization", mock.Anything, mock.Anything).Return(nil).Once()

	body, _ = json.Marshal(updateReq)
	httpReq = httptest.NewRequest(http.MethodPut, "/organizations/"+orgID, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, _ = suite.app.Test(httpReq)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 4. Delete Organization
	suite.mockRepo.On("DeleteOrganization", mock.Anything, orgID).Return(nil).Once()

	httpReq = httptest.NewRequest(http.MethodDelete, "/organizations/"+orgID, nil)
	resp, _ = suite.app.Test(httpReq)
	assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

// ============== EDGE CASES AND BOUNDARY TESTS ==============

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_MaxLengthFields() {
	// Test with maximum length strings
	longString := make([]byte, 1000)
	for i := range longString {
		longString[i] = 'a'
	}

	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID:  "test-org-max",
		DisplayName:     string(longString),
		Description:     string(longString),
		BillingEmail:    "billing@test.com",
		DefaultLocation: string(longString),
	}

	// Should handle long strings gracefully
	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	// Should succeed or fail gracefully
	assert.True(suite.T(), resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest)
}

func (suite *OrganizationHandlerTestSuite) TestCreateOrganization_SpecialCharacters() {
	req := firestorehttp.CreateOrganizationRequest{
		OrganizationID: "test-org-special",
		DisplayName:    "Org with Special Chars: !@#$%^&*()",
		Description:    "Description with émojis 🚀 and unicode ñáéíóú",
		BillingEmail:   "billing@test.com",
	}

	suite.mockRepo.On("CreateOrganization", mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
}

func (suite *OrganizationHandlerTestSuite) TestListOrganizations_ZeroPagination() {
	// Test with pageSize=0 and offset=0
	org1, err := model.NewOrganization("test-org-001", "Organization 1", "billing1@test.com")
	require.NoError(suite.T(), err)
	orgs := []*model.Organization{org1}

	// Should use default pageSize when 0 is provided
	suite.mockRepo.On("ListOrganizations", mock.Anything, 10, 0).Return(orgs, nil)

	httpReq := httptest.NewRequest(http.MethodGet, "/organizations?pageSize=0&offset=0", nil)
	resp, err := suite.app.Test(httpReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}
