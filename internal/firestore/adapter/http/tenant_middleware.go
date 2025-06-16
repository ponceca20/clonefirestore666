package http

import (
	"fmt"
	"strings"

	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/logger"
	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// TenantExtractor provides organization/tenant information extraction from requests
// Following Firestore's API path structure and Clean Architecture principles
type TenantExtractor struct {
	logger logger.Logger
}

// NewTenantExtractor creates a new TenantExtractor with dependency injection
func NewTenantExtractor(log logger.Logger) *TenantExtractor {
	return &TenantExtractor{
		logger: log,
	}
}

// ExtractOrganizationID extracts organization ID from multiple sources following priority order
// 1. URL path parameters (preferred for Firestore API compatibility)
// 2. Custom headers (backward compatibility)
// 3. Authorization header suffix (enterprise feature)
// 4. Query parameters (development/testing)
func (te *TenantExtractor) ExtractOrganizationID(c *fiber.Ctx) (string, error) {
	// 1. From URL path (preferred for Firestore API compatibility)
	//    /v1/organizations/{organizationId}/projects/{projectId}/...
	if orgID := c.Params("organizationId"); orgID != "" {
		if err := ValidateOrganizationIDFormat(orgID); err != nil {
			te.logger.Warn("Invalid organization ID format in URL path", "orgID", orgID, "error", err)
			return "", errors.NewValidationError("invalid organization ID format").WithCause(err)
		}
		return orgID, nil
	}

	// 2. From custom header (for backward compatibility)
	if orgID := c.Get("X-Organization-ID"); orgID != "" {
		if err := ValidateOrganizationIDFormat(orgID); err != nil {
			te.logger.Warn("Invalid organization ID format in header", "orgID", orgID, "error", err)
			return "", errors.NewValidationError("invalid organization ID format").WithCause(err)
		}
		return orgID, nil
	}

	// 3. From Authorization header suffix (enterprise feature)
	//    Authorization: Bearer token@org_id
	if auth := c.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if parts := strings.Split(token, "@"); len(parts) == 2 {
				orgID := parts[1]
				if err := ValidateOrganizationIDFormat(orgID); err != nil {
					te.logger.Warn("Invalid organization ID format in auth token", "orgID", orgID, "error", err)
					return "", errors.NewValidationError("invalid organization ID format").WithCause(err)
				}
				return orgID, nil
			}
		}
	}

	// 4. From query parameter (development/testing)
	if orgID := c.Query("organization_id"); orgID != "" {
		if err := ValidateOrganizationIDFormat(orgID); err != nil {
			te.logger.Warn("Invalid organization ID format in query param", "orgID", orgID, "error", err)
			return "", errors.NewValidationError("invalid organization ID format").WithCause(err)
		}
		return orgID, nil
	}

	// No organization ID found
	return "", errors.NewNotFoundError("organization ID")
}

// OrganizationMiddleware extracts organization/tenant information from requests
// Following Firestore's API path structure and hexagonal architecture principles
func OrganizationMiddleware(log logger.Logger) fiber.Handler {
	extractor := NewTenantExtractor(log)

	return func(c *fiber.Ctx) error {
		orgID, err := extractor.ExtractOrganizationID(c)
		if err != nil {
			log.Debug("Failed to extract organization ID", "path", c.Path(), "error", err)

			// Check if this is a Firestore-style path without organization
			if projectID := c.Params("projectID"); projectID != "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    "ORGANIZATION_ID_REQUIRED",
						"message": "Organization ID must be specified in URL path, header, or query parameter",
						"details": "Use /v1/organizations/{orgId}/projects/... or set X-Organization-ID header",
					},
				})
			}

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "ORGANIZATION_ID_MISSING",
					"message": "Organization ID is required for this endpoint",
					"details": "Provide organization ID via URL path, header, or query parameter",
				},
			})
		}

		// Set organization context for downstream handlers
		ctx := utils.WithOrganizationID(c.UserContext(), orgID)
		c.SetUserContext(ctx)

		log.Debug("Organization ID extracted successfully", "orgID", orgID, "path", c.Path())
		return c.Next()
	}
}

// ProjectExtractor handles project-specific context extraction
type ProjectExtractor struct {
	logger logger.Logger
}

// NewProjectExtractor creates a new ProjectExtractor
func NewProjectExtractor(log logger.Logger) *ProjectExtractor {
	return &ProjectExtractor{
		logger: log,
	}
}

// ExtractProjectContext extracts project and database information from URL parameters
func (pe *ProjectExtractor) ExtractProjectContext(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Extract project ID from URL
	if projectID := c.Params("projectID"); projectID != "" {
		if err := ValidateProjectIDFormat(projectID); err != nil {
			pe.logger.Warn("Invalid project ID format", "projectID", projectID, "error", err)
			return errors.NewValidationError("invalid project ID format").WithCause(err)
		}
		ctx = utils.WithProjectID(ctx, projectID)
		pe.logger.Debug("Project ID extracted", "projectID", projectID)
	}

	// Extract database ID from URL
	if databaseID := c.Params("databaseID"); databaseID != "" {
		if err := ValidateDatabaseIDFormat(databaseID); err != nil {
			pe.logger.Warn("Invalid database ID format", "databaseID", databaseID, "error", err)
			return errors.NewValidationError("invalid database ID format").WithCause(err)
		}
		ctx = utils.WithDatabaseID(ctx, databaseID)
		pe.logger.Debug("Database ID extracted", "databaseID", databaseID)
	}

	c.SetUserContext(ctx)
	return nil
}

// FirestoreProjectMiddleware extracts project information and validates organization access
func FirestoreProjectMiddleware(log logger.Logger) fiber.Handler {
	extractor := NewProjectExtractor(log)

	return func(c *fiber.Ctx) error {
		if err := extractor.ExtractProjectContext(c); err != nil {
			log.Warn("Failed to extract project context", "path", c.Path(), "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_PROJECT_CONTEXT",
					"message": err.Error(),
					"path":    c.Path(),
				},
			})
		}

		return c.Next()
	}
}

// HierarchyValidator validates Firestore hierarchy requirements
type HierarchyValidator struct {
	logger logger.Logger
}

// NewHierarchyValidator creates a new HierarchyValidator
func NewHierarchyValidator(log logger.Logger) *HierarchyValidator {
	return &HierarchyValidator{
		logger: log,
	}
}

// ValidateHierarchy validates that the request follows Firestore hierarchy rules
func (hv *HierarchyValidator) ValidateHierarchy(c *fiber.Ctx) error {
	ctx := c.UserContext()
	path := c.Path()

	// Validate required hierarchy components
	orgID, err := utils.GetOrganizationIDFromContext(ctx)
	if err != nil {
		hv.logger.Debug("Organization ID not found in context", "path", path)
		return errors.NewValidationError("organization ID is required").WithCode("MISSING_ORGANIZATION_ID")
	}

	// Validate organization ID format if present
	if err := ValidateOrganizationIDFormat(orgID); err != nil {
		hv.logger.Warn("Invalid organization ID format", "orgID", orgID, "error", err)
		return errors.NewValidationError("invalid organization ID format").
			WithCode("INVALID_ORGANIZATION_ID").
			WithCause(err)
	}

	// Validate project ID if this is a project-scoped path
	if strings.Contains(path, "/projects/") {
		projectID, err := utils.GetProjectIDFromContext(ctx)
		if err != nil {
			hv.logger.Debug("Project ID required but not found", "path", path)
			return errors.NewValidationError("project ID is required for this endpoint").WithCode("MISSING_PROJECT_ID")
		}

		if err := ValidateProjectIDFormat(projectID); err != nil {
			hv.logger.Warn("Invalid project ID format", "projectID", projectID, "error", err)
			return errors.NewValidationError("invalid project ID format").
				WithCode("INVALID_PROJECT_ID").
				WithCause(err)
		}
	}

	// Validate database ID if this is a database-scoped path
	if strings.Contains(path, "/databases/") {
		databaseID, err := utils.GetDatabaseIDFromContext(ctx)
		if err != nil {
			hv.logger.Debug("Database ID required but not found", "path", path)
			return errors.NewValidationError("database ID is required for this endpoint").WithCode("MISSING_DATABASE_ID")
		}

		if err := ValidateDatabaseIDFormat(databaseID); err != nil {
			hv.logger.Warn("Invalid database ID format", "databaseID", databaseID, "error", err)
			return errors.NewValidationError("invalid database ID format").
				WithCode("INVALID_DATABASE_ID").
				WithCause(err)
		}
	}

	return nil
}

// FirestoreHierarchyMiddleware validates that the request follows Firestore hierarchy rules
func FirestoreHierarchyMiddleware(log logger.Logger) fiber.Handler {
	validator := NewHierarchyValidator(log)

	return func(c *fiber.Ctx) error {
		if err := validator.ValidateHierarchy(c); err != nil {
			log.Debug("Hierarchy validation failed", "path", c.Path(), "error", err)

			var appErr *errors.AppError
			// Use errors2 for type assertion since errors.As is not available in the custom errors package
			if e, ok := err.(*errors.AppError); ok {
				appErr = e
				return c.Status(appErr.HTTPCode).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    appErr.Code,
						"message": appErr.Message,
						"type":    appErr.Type,
					},
				})
			}

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "HIERARCHY_VALIDATION_ERROR",
					"message": err.Error(),
				},
			})
		}

		return c.Next()
	}
}

// FirestorePathHandler handles Firestore-compatible path extraction with organization awareness
type FirestorePathHandler struct {
	logger logger.Logger
}

// NewFirestorePathHandler creates a new FirestorePathHandler
func NewFirestorePathHandler(log logger.Logger) *FirestorePathHandler {
	return &FirestorePathHandler{
		logger: log,
	}
}

// HandleFirestorePath handles Firestore-compatible path extraction
// Supports organization-aware paths and falls back to header-based organization ID
func (fph *FirestorePathHandler) HandleFirestorePath(c *fiber.Ctx) fiber.Handler {
	path := c.Path()

	// Organization format: /v1/organizations/{orgId}/projects/{projectId}/databases/{dbId}/documents/...
	if strings.Contains(path, "/organizations/") {
		return OrganizationMiddleware(fph.logger)
	}

	// Default to requiring organization ID from headers
	return func(c *fiber.Ctx) error {
		if orgID := c.Get("X-Organization-ID"); orgID != "" {
			if err := ValidateOrganizationIDFormat(orgID); err != nil {
				fph.logger.Warn("Invalid organization ID in header", "orgID", orgID, "error", err)
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    "INVALID_ORGANIZATION_ID",
						"message": "Invalid organization ID format in header",
					},
				})
			}

			ctx := utils.WithOrganizationID(c.UserContext(), orgID)
			c.SetUserContext(ctx)
			return FirestoreProjectMiddleware(fph.logger)(c)
		}

		// Require organization ID for all requests
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "ORGANIZATION_ID_REQUIRED",
				"message": "This endpoint requires organization ID. Use /v1/organizations/{orgId}/projects/... or set X-Organization-ID header",
			},
		})
	}
}

// PathAwareMiddleware provides intelligent path-based middleware selection
func PathAwareMiddleware(log logger.Logger) fiber.Handler {
	handler := NewFirestorePathHandler(log)

	return func(c *fiber.Ctx) error {
		return handler.HandleFirestorePath(c)(c)
	}
}

// Validation helper functions for Firestore ID formats

// ValidateOrganizationIDFormat validates organization ID format
// Organization IDs must be 3-30 characters, start with letter, contain only letters, numbers, and hyphens
func ValidateOrganizationIDFormat(orgID string) error {
	if len(orgID) < 3 || len(orgID) > 30 {
		return fmt.Errorf("organization ID must be 3-30 characters, got %d", len(orgID))
	}

	// Must start with letter
	if !((orgID[0] >= 'a' && orgID[0] <= 'z') || (orgID[0] >= 'A' && orgID[0] <= 'Z')) {
		return fmt.Errorf("organization ID must start with a letter")
	}

	// Can contain letters, numbers, hyphens
	for i, char := range orgID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("organization ID can only contain letters, numbers, and hyphens (invalid character at position %d)", i)
		}
	}

	return nil
}

// ValidateProjectIDFormat validates project ID format
// Project IDs must be 6-30 characters, start with letter, contain only lowercase letters, numbers, and hyphens
func ValidateProjectIDFormat(projectID string) error {
	if len(projectID) < 6 || len(projectID) > 30 {
		return fmt.Errorf("project ID must be 6-30 characters, got %d", len(projectID))
	}

	// Must start with letter
	if !((projectID[0] >= 'a' && projectID[0] <= 'z') || (projectID[0] >= 'A' && projectID[0] <= 'Z')) {
		return fmt.Errorf("project ID must start with a letter")
	}

	// Can contain letters, numbers, hyphens (Firestore prefers lowercase)
	for i, char := range projectID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("project ID can only contain letters, numbers, and hyphens (invalid character at position %d)", i)
		}
	}

	return nil
}

// ValidateDatabaseIDFormat validates database ID format
// Database IDs can be "(default)" or follow similar rules to project IDs
func ValidateDatabaseIDFormat(databaseID string) error {
	// Allow the special "(default)" database ID
	if databaseID == "(default)" {
		return nil
	}

	if len(databaseID) < 3 || len(databaseID) > 30 {
		return fmt.Errorf("database ID must be 3-30 characters or '(default)', got %d", len(databaseID))
	}

	// Must start with letter
	if !((databaseID[0] >= 'a' && databaseID[0] <= 'z') || (databaseID[0] >= 'A' && databaseID[0] <= 'Z')) {
		return fmt.Errorf("database ID must start with a letter")
	}

	// Can contain letters, numbers, hyphens
	for i, char := range databaseID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("database ID can only contain letters, numbers, and hyphens (invalid character at position %d)", i)
		}
	}

	return nil
}

// TenantMiddleware validates tenant context and organization access
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Println("[TenantMiddleware] INICIO: path=", c.Path(), "method=", c.Method(), "params=", c.Params("organizationId"), c.Params("projectID"), c.Params("databaseID"))
		// Extract organization ID from path, header, query, or Authorization
		organizationID := c.Params("organizationId")
		if organizationID == "" {
			organizationID = c.Get("X-Organization-ID")
		}
		if organizationID == "" {
			if auth := c.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				if parts := strings.Split(token, "@"); len(parts) == 2 {
					organizationID = parts[1]
				}
			}
		}
		if organizationID == "" {
			organizationID = c.Query("organization_id")
		}
		if organizationID == "" {
			fmt.Println("[TenantMiddleware] organization_id_missing")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "organization_id_missing",
				"message": "Organization ID is required",
			})
		}

		// Validate organization ID format
		if err := ValidateOrganizationIDFormat(organizationID); err != nil {
			fmt.Println("[TenantMiddleware] invalid_organization_id", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_organization_id",
				"message": "Organization ID must start with a letter and be at least 3 characters long",
			})
		}

		// Set organization context for downstream handlers
		c.Locals("organizationId", organizationID)

		// Add organization ID to the Go context
		ctx := c.UserContext()
		ctx = utils.WithOrganizationID(ctx, organizationID)
		c.SetUserContext(ctx)

		// Extract project ID from path parameter if present
		if projectID := c.Params("projectId"); projectID != "" {
			fmt.Println("[TenantMiddleware] projectId=", projectID)
			ctx = utils.WithProjectID(ctx, projectID)
			c.SetUserContext(ctx)
		}

		// Extract database ID from path parameter if present
		if databaseID := c.Params("databaseId"); databaseID != "" {
			fmt.Println("[TenantMiddleware] databaseId=", databaseID)
			ctx = utils.WithDatabaseID(ctx, databaseID)
			c.SetUserContext(ctx)
		}

		fmt.Println("[TenantMiddleware] FIN: organizationId=", organizationID, "projectId=", c.Params("projectId"), "databaseId=", c.Params("databaseId"))
		// TODO: Add tenant validation logic here
		// - Validate organization exists
		// - Check user access to organization
		// - Set tenant database context

		return c.Next()
	}
}
