package http

import (
	"context"
	"strconv"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// OrganizationRepo defines the minimal interface needed for handler
// This matches the methods used by OrganizationHandler
// Place here for production and test use
// (You can move this to a separate file if preferred)
//
//go:generate mockgen -destination=organization_repo_mock.go -package=http . OrganizationRepo
type OrganizationRepo interface {
	CreateOrganization(ctx context.Context, org *model.Organization) error
	GetOrganization(ctx context.Context, id string) (*model.Organization, error)
	ListOrganizations(ctx context.Context, pageSize, offset int) ([]*model.Organization, error)
	ListOrganizationsByAdmin(ctx context.Context, adminEmail string) ([]*model.Organization, error)
	UpdateOrganization(ctx context.Context, org *model.Organization) error
	DeleteOrganization(ctx context.Context, id string) error
}

// OrganizationHandler handles organization management endpoints
// Following Firestore's hierarchical API structure
type OrganizationHandler struct {
	organizationRepo OrganizationRepo
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(organizationRepo OrganizationRepo) *OrganizationHandler {
	return &OrganizationHandler{
		organizationRepo: organizationRepo,
	}
}

// RegisterRoutes registers organization management routes
// Accepts any fiber.Router (app or group) and registers under the given group
func (h *OrganizationHandler) RegisterRoutes(router fiber.Router) {
	// Organizations endpoints
	orgs := router.Group("/organizations")
	orgs.Post("/", h.CreateOrganization)                  // POST /organizations
	orgs.Get("/", h.ListOrganizations)                    // GET /organizations
	orgs.Get("/:organizationId", h.GetOrganization)       // GET /organizations/{organizationId}
	orgs.Put("/:organizationId", h.UpdateOrganization)    // PUT /organizations/{organizationId}
	orgs.Delete("/:organizationId", h.DeleteOrganization) // DELETE /organizations/{organizationId}

	// Organization-scoped endpoints (Firestore hierarchy)
	orgScoped := orgs.Group("/:organizationId", TenantMiddleware())
	// Note: Project routes are handled by HTTPHandler.registerProjectRoutes to avoid conflicts
	orgScoped.Get("/usage", h.GetOrganizationUsage) // GET /organizations/{organizationId}/usage
}

// CreateOrganization creates a new organization
// POST /v1/organizations
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	var req CreateOrganizationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Validate required fields
	if req.OrganizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_organization_id",
			"message": "Organization ID is required",
		})
	}

	if req.DisplayName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_display_name",
			"message": "Display name is required",
		})
	}

	if req.BillingEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_billing_email",
			"message": "Billing email is required",
		})
	}

	// Create organization model
	org, err := model.NewOrganization(req.OrganizationID, req.DisplayName, req.BillingEmail)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_organization_data",
			"message": err.Error(),
		})
	}

	// Set optional fields
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.DefaultLocation != "" {
		org.DefaultLocation = req.DefaultLocation
	}
	if len(req.AdminEmails) > 0 {
		org.AdminEmails = req.AdminEmails
	}

	// Create organization
	err = h.organizationRepo.CreateOrganization(c.Context(), org)
	if err != nil {
		if err == model.ErrOrganizationExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "organization_already_exists",
				"message": "Organization with this ID already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_organization_failed",
			"message": "Failed to create organization",
		})
	}

	// Return created organization
	return c.Status(fiber.StatusCreated).JSON(OrganizationResponse{
		Name:            "organizations/" + org.OrganizationID,
		OrganizationID:  org.OrganizationID,
		DisplayName:     org.DisplayName,
		Description:     org.Description,
		BillingEmail:    org.BillingEmail,
		AdminEmails:     org.AdminEmails,
		DefaultLocation: org.DefaultLocation,
		State:           string(org.State),
		CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       org.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		ProjectCount:    org.ProjectCount,
	})
}

// GetOrganization retrieves an organization by ID
// GET /v1/organizations/{organizationId}
func (h *OrganizationHandler) GetOrganization(c *fiber.Ctx) error {
	organizationID := c.Params("organizationId")

	org, err := h.organizationRepo.GetOrganization(c.Context(), organizationID)
	if err != nil {
		if err == model.ErrOrganizationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "organization_not_found",
				"message": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_organization_failed",
			"message": "Failed to retrieve organization",
		})
	}

	// Return organization data
	return c.JSON(OrganizationResponse{
		Name:            "organizations/" + org.OrganizationID,
		OrganizationID:  org.OrganizationID,
		DisplayName:     org.DisplayName,
		Description:     org.Description,
		BillingEmail:    org.BillingEmail,
		AdminEmails:     org.AdminEmails,
		DefaultLocation: org.DefaultLocation,
		State:           string(org.State),
		CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       org.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		ProjectCount:    org.ProjectCount,
		Usage:           org.Usage,
		Quotas:          org.Quotas,
	})
}

// ListOrganizations lists organizations with pagination
// GET /v1/organizations
func (h *OrganizationHandler) ListOrganizations(c *fiber.Ctx) error { // Parse pagination parameters
	pageSize := 10 // default
	if ps := c.Query("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			if parsed > 100 {
				pageSize = 100 // Cap at 100
			} else {
				pageSize = parsed
			}
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Check if filtering by admin email
	adminEmail := c.Query("admin_email")

	var organizations []*model.Organization
	var err error

	if adminEmail != "" {
		organizations, err = h.organizationRepo.ListOrganizationsByAdmin(c.Context(), adminEmail)
	} else {
		organizations, err = h.organizationRepo.ListOrganizations(c.Context(), pageSize, offset)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_organizations_failed",
			"message": "Failed to list organizations",
		})
	}

	// Convert to response format
	var orgResponses []OrganizationResponse
	for _, org := range organizations {
		// Add nil check for safety
		if org == nil {
			continue
		}
		orgResponses = append(orgResponses, OrganizationResponse{
			Name:            "organizations/" + org.OrganizationID,
			OrganizationID:  org.OrganizationID,
			DisplayName:     org.DisplayName,
			Description:     org.Description,
			BillingEmail:    org.BillingEmail,
			AdminEmails:     org.AdminEmails,
			DefaultLocation: org.DefaultLocation,
			State:           string(org.State),
			CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       org.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			ProjectCount:    org.ProjectCount,
		})
	}

	// Ensure orgResponses is never nil
	if orgResponses == nil {
		orgResponses = []OrganizationResponse{}
	}

	return c.JSON(ListOrganizationsResponse{
		Organizations: orgResponses,
		NextPageToken: "", // TODO: Implement proper pagination tokens
	})
}

// UpdateOrganization updates an existing organization
// PUT /v1/organizations/{organizationId}
func (h *OrganizationHandler) UpdateOrganization(c *fiber.Ctx) error {
	organizationID := c.Params("organizationId")

	var req UpdateOrganizationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Get existing organization
	org, err := h.organizationRepo.GetOrganization(c.Context(), organizationID)
	if err != nil {
		if err == model.ErrOrganizationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "organization_not_found",
				"message": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_organization_failed",
			"message": "Failed to retrieve organization",
		})
	}

	// Update fields
	if req.DisplayName != "" {
		org.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.DefaultLocation != "" {
		org.DefaultLocation = req.DefaultLocation
	}
	if len(req.AdminEmails) > 0 {
		org.AdminEmails = req.AdminEmails
	}

	// Update organization
	err = h.organizationRepo.UpdateOrganization(c.Context(), org)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_organization_failed",
			"message": "Failed to update organization",
		})
	}

	// Return updated organization
	return c.JSON(OrganizationResponse{
		Name:            "organizations/" + org.OrganizationID,
		OrganizationID:  org.OrganizationID,
		DisplayName:     org.DisplayName,
		Description:     org.Description,
		BillingEmail:    org.BillingEmail,
		AdminEmails:     org.AdminEmails,
		DefaultLocation: org.DefaultLocation,
		State:           string(org.State),
		CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       org.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		ProjectCount:    org.ProjectCount,
	})
}

// DeleteOrganization deletes an organization
// DELETE /v1/organizations/{organizationId}
func (h *OrganizationHandler) DeleteOrganization(c *fiber.Ctx) error {
	organizationID := c.Params("organizationId")

	err := h.organizationRepo.DeleteOrganization(c.Context(), organizationID)
	if err != nil {
		if err == model.ErrOrganizationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "organization_not_found",
				"message": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_organization_failed",
			"message": "Failed to delete organization",
		})
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetOrganizationUsage gets usage statistics for an organization
// GET /v1/organizations/{organizationId}/usage
func (h *OrganizationHandler) GetOrganizationUsage(c *fiber.Ctx) error {
	organizationID, err := utils.GetOrganizationIDFromContext(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_organization_id",
			"message": "Organization ID is required",
		})
	}

	org, err := h.organizationRepo.GetOrganization(c.Context(), organizationID)
	if err != nil {
		if err == model.ErrOrganizationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "organization_not_found",
				"message": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_organization_failed",
			"message": "Failed to retrieve organization",
		})
	}

	return c.JSON(OrganizationUsageResponse{
		OrganizationID: organizationID,
		Usage:          org.Usage,
		Quotas:         org.Quotas,
	})
}

// Request/Response types

type CreateOrganizationRequest struct {
	OrganizationID  string   `json:"organizationId"`
	DisplayName     string   `json:"displayName"`
	Description     string   `json:"description,omitempty"`
	BillingEmail    string   `json:"billingEmail"`
	AdminEmails     []string `json:"adminEmails,omitempty"`
	DefaultLocation string   `json:"defaultLocation,omitempty"`
}

type UpdateOrganizationRequest struct {
	DisplayName     string   `json:"displayName,omitempty"`
	Description     string   `json:"description,omitempty"`
	AdminEmails     []string `json:"adminEmails,omitempty"`
	DefaultLocation string   `json:"defaultLocation,omitempty"`
}

type OrganizationResponse struct {
	Name            string                    `json:"name"`
	OrganizationID  string                    `json:"organizationId"`
	DisplayName     string                    `json:"displayName"`
	Description     string                    `json:"description,omitempty"`
	BillingEmail    string                    `json:"billingEmail"`
	AdminEmails     []string                  `json:"adminEmails,omitempty"`
	DefaultLocation string                    `json:"defaultLocation"`
	State           string                    `json:"state"`
	CreatedAt       string                    `json:"createdAt"`
	UpdatedAt       string                    `json:"updatedAt"`
	ProjectCount    int                       `json:"projectCount"`
	Usage           *model.OrganizationUsage  `json:"usage,omitempty"`
	Quotas          *model.OrganizationQuotas `json:"quotas,omitempty"`
}

type ListOrganizationsResponse struct {
	Organizations []OrganizationResponse `json:"organizations"`
	NextPageToken string                 `json:"nextPageToken,omitempty"`
}

type OrganizationUsageResponse struct {
	OrganizationID string                    `json:"organizationId"`
	Usage          *model.OrganizationUsage  `json:"usage"`
	Quotas         *model.OrganizationQuotas `json:"quotas"`
}
