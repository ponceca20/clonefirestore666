package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// mockOrgRepo implements the minimal OrganizationRepo interface for handler tests
// Only the methods used in handler are mocked

type mockOrgRepo struct {
	CreateOrganizationFn       func(ctx context.Context, org *model.Organization) error
	GetOrganizationFn          func(ctx context.Context, id string) (*model.Organization, error)
	ListOrganizationsFn        func(ctx context.Context, pageSize, offset int) ([]*model.Organization, error)
	UpdateOrganizationFn       func(ctx context.Context, org *model.Organization) error
	DeleteOrganizationFn       func(ctx context.Context, id string) error
	ListOrganizationsByAdminFn func(ctx context.Context, adminEmail string) ([]*model.Organization, error)
}

func (m *mockOrgRepo) CreateOrganization(ctx context.Context, org *model.Organization) error {
	return m.CreateOrganizationFn(ctx, org)
}
func (m *mockOrgRepo) GetOrganization(ctx context.Context, id string) (*model.Organization, error) {
	return m.GetOrganizationFn(ctx, id)
}
func (m *mockOrgRepo) ListOrganizations(ctx context.Context, pageSize, offset int) ([]*model.Organization, error) {
	return m.ListOrganizationsFn(ctx, pageSize, offset)
}
func (m *mockOrgRepo) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	return m.UpdateOrganizationFn(ctx, org)
}
func (m *mockOrgRepo) DeleteOrganization(ctx context.Context, id string) error {
	return m.DeleteOrganizationFn(ctx, id)
}
func (m *mockOrgRepo) ListOrganizationsByAdmin(ctx context.Context, adminEmail string) ([]*model.Organization, error) {
	return m.ListOrganizationsByAdminFn(ctx, adminEmail)
}

func newHandlerWithMockRepo(mock *mockOrgRepo) *OrganizationHandler {
	return NewOrganizationHandler(mock)
}

func TestCreateOrganization_Success(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		CreateOrganizationFn: func(ctx context.Context, org *model.Organization) error { return nil },
		// Stubs for unused methods to avoid nil panic
		GetOrganizationFn:          func(ctx context.Context, id string) (*model.Organization, error) { return nil, nil },
		ListOrganizationsFn:        func(ctx context.Context, pageSize, offset int) ([]*model.Organization, error) { return nil, nil },
		UpdateOrganizationFn:       func(ctx context.Context, org *model.Organization) error { return nil },
		DeleteOrganizationFn:       func(ctx context.Context, id string) error { return nil },
		ListOrganizationsByAdminFn: func(ctx context.Context, adminEmail string) ([]*model.Organization, error) { return nil, nil },
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Post("/v1/organizations", h.CreateOrganization)

	body := map[string]interface{}{
		"organizationId": "test-org-123", // Use valid org ID format
		"displayName":    "Test Org",
		"billingEmail":   "admin@test.com",
	}
	b, _ := json.Marshal(body)
	// Hexagonal/clean: siempre especificar el content-type correcto
	req := httptest.NewRequest("POST", "/v1/organizations", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCreateOrganization_BadRequest(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{}
	h := newHandlerWithMockRepo(mockRepo)
	app.Post("/v1/organizations", h.CreateOrganization)
	resp, _ := app.Test(httptest.NewRequest("POST", "/v1/organizations", bytes.NewReader([]byte("invalid-json"))))
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetOrganization_Success(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		GetOrganizationFn: func(ctx context.Context, id string) (*model.Organization, error) {
			return &model.Organization{
				OrganizationID:  "org1",
				DisplayName:     "Test Org",
				BillingEmail:    "admin@test.com",
				AdminEmails:     []string{"admin@test.com"},
				DefaultLocation: "us-central1",
				State:           model.OrganizationStateActive,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				ProjectCount:    0,
			}, nil
		},
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Get("/v1/organizations/:organizationId", h.GetOrganization)
	resp, _ := app.Test(httptest.NewRequest("GET", "/v1/organizations/org1", nil))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetOrganization_NotFound(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		GetOrganizationFn: func(ctx context.Context, id string) (*model.Organization, error) {
			return nil, model.ErrOrganizationNotFound
		},
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Get("/v1/organizations/:organizationId", h.GetOrganization)
	resp, _ := app.Test(httptest.NewRequest("GET", "/v1/organizations/org404", nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateOrganization_Success(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		GetOrganizationFn: func(ctx context.Context, id string) (*model.Organization, error) {
			return &model.Organization{
				OrganizationID:  "org1",
				DisplayName:     "Test Org",
				BillingEmail:    "admin@test.com",
				AdminEmails:     []string{"admin@test.com"},
				DefaultLocation: "us-central1",
				State:           model.OrganizationStateActive,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				ProjectCount:    0,
			}, nil
		},
		UpdateOrganizationFn: func(ctx context.Context, org *model.Organization) error { return nil },
		// Stubs for unused methods to evitar nil panic
		CreateOrganizationFn:       func(ctx context.Context, org *model.Organization) error { return nil },
		ListOrganizationsFn:        func(ctx context.Context, pageSize, offset int) ([]*model.Organization, error) { return nil, nil },
		DeleteOrganizationFn:       func(ctx context.Context, id string) error { return nil },
		ListOrganizationsByAdminFn: func(ctx context.Context, adminEmail string) ([]*model.Organization, error) { return nil, nil },
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Put("/v1/organizations/:organizationId", h.UpdateOrganization)
	body := map[string]interface{}{"displayName": "Updated Org"}
	b, _ := json.Marshal(body)
	// Hexagonal/clean: siempre especificar el content-type correcto
	req := httptest.NewRequest("PUT", "/v1/organizations/org1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestDeleteOrganization_Success(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		DeleteOrganizationFn: func(ctx context.Context, id string) error { return nil },
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Delete("/v1/organizations/:organizationId", h.DeleteOrganization)
	resp, _ := app.Test(httptest.NewRequest("DELETE", "/v1/organizations/org1", nil))
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteOrganization_NotFound(t *testing.T) {
	app := fiber.New()
	mockRepo := &mockOrgRepo{
		DeleteOrganizationFn: func(ctx context.Context, id string) error { return model.ErrOrganizationNotFound },
	}
	h := newHandlerWithMockRepo(mockRepo)
	app.Delete("/v1/organizations/:organizationId", h.DeleteOrganization)
	resp, _ := app.Test(httptest.NewRequest("DELETE", "/v1/organizations/org404", nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
