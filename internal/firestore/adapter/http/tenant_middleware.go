package http

import (
	"strings"

	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// TenantMiddleware extracts organization/tenant information from requests
// Following Firestore's API path structure
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract organization ID from different sources:

		// 1. From URL path (preferred for Firestore API compatibility)
		//    /v1/organizations/{organizationId}/projects/{projectId}/...
		if orgID := c.Params("organizationId"); orgID != "" {
			ctx := utils.WithOrganizationID(c.Context(), orgID)
			c.SetUserContext(ctx)
			return c.Next()
		}

		// 2. From custom header (for backward compatibility)
		if orgID := c.Get("X-Organization-ID"); orgID != "" {
			ctx := utils.WithOrganizationID(c.Context(), orgID)
			c.SetUserContext(ctx)
			return c.Next()
		}

		// 3. From Authorization header suffix (enterprise feature)
		//    Authorization: Bearer token@org_id
		if auth := c.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				if parts := strings.Split(token, "@"); len(parts) == 2 {
					orgID := parts[1]
					ctx := utils.WithOrganizationID(c.Context(), orgID)
					c.SetUserContext(ctx)
					return c.Next()
				}
			}
		}

		// 4. From query parameter (development/testing)
		if orgID := c.Query("organization_id"); orgID != "" {
			ctx := utils.WithOrganizationID(c.Context(), orgID)
			c.SetUserContext(ctx)
			return c.Next()
		}

		// 5. Try to extract from Firestore path structure
		//    /firestore/projects/{projectId}/databases/{databaseId}/...
		//    In this case, we need to look up the organization for the project
		if projectID := c.Params("projectID"); projectID != "" {
			// For now, we'll require explicit organization ID
			// In a full implementation, you'd look up the organization for this project
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "organization_id_required",
				"message": "Organization ID must be specified in URL path, header, or query parameter",
				"code":    "ORGANIZATION_ID_REQUIRED",
			})
		}

		// If no organization ID found, return error for protected routes
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "organization_id_missing",
			"message": "Organization ID is required for this endpoint",
			"code":    "ORGANIZATION_ID_MISSING",
		})
	}
}

// ProjectMiddleware extracts project information and validates organization access
func ProjectMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract project ID from URL
		if projectID := c.Params("projectID"); projectID != "" {
			ctx := utils.WithProjectID(c.Context(), projectID)
			c.SetUserContext(ctx)
		}

		// Extract database ID from URL
		if databaseID := c.Params("databaseID"); databaseID != "" {
			ctx := utils.WithDatabaseID(c.Context(), databaseID)
			c.SetUserContext(ctx)
		}

		return c.Next()
	}
}

// FirestorePathMiddleware handles Firestore-compatible path extraction
// Supports organization-aware paths
func FirestorePathMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Organization format: /v1/organizations/{orgId}/projects/{projectId}/databases/{dbId}/documents/...
		if strings.Contains(path, "/organizations/") {
			return TenantMiddleware()(c)
		}
		// Default to requiring organization ID from headers
		if orgID := c.Get("X-Organization-ID"); orgID != "" {
			ctx := utils.WithOrganizationID(c.Context(), orgID)
			c.SetUserContext(ctx)
			return ProjectMiddleware()(c)
		} // Require organization ID for all requests
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "organization_id_required",
			"message": "This endpoint requires organization ID. Use /v1/organizations/{orgId}/projects/... or set X-Organization-ID header",
			"code":    "ORGANIZATION_ID_REQUIRED",
		})
	}
}

// ValidateFirestoreHierarchy validates that the request follows Firestore hierarchy rules
func ValidateFirestoreHierarchy() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Validate required hierarchy components
		orgID, err := utils.GetOrganizationIDFromContext(ctx)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "missing_organization_id",
				"message": "Organization ID is required",
				"code":    "MISSING_ORGANIZATION_ID",
			})
		}

		projectID, err := utils.GetProjectIDFromContext(ctx)
		if err != nil && strings.Contains(c.Path(), "/projects/") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "missing_project_id",
				"message": "Project ID is required for this endpoint",
				"code":    "MISSING_PROJECT_ID",
			})
		}

		// Validate ID formats
		if orgID != "" {
			if err := validateOrganizationIDFormat(orgID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "invalid_organization_id",
					"message": err.Error(),
					"code":    "INVALID_ORGANIZATION_ID",
				})
			}
		}

		if projectID != "" {
			if err := validateProjectIDFormat(projectID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "invalid_project_id",
					"message": err.Error(),
					"code":    "INVALID_PROJECT_ID",
				})
			}
		}

		return c.Next()
	}
}

// Helper functions for validation

func validateOrganizationIDFormat(orgID string) error {
	if len(orgID) < 3 || len(orgID) > 30 {
		return fiber.NewError(fiber.StatusBadRequest, "organization ID must be 3-30 characters")
	}

	// Must start with letter
	if !((orgID[0] >= 'a' && orgID[0] <= 'z') || (orgID[0] >= 'A' && orgID[0] <= 'Z')) {
		return fiber.NewError(fiber.StatusBadRequest, "organization ID must start with a letter")
	}

	// Can contain letters, numbers, hyphens
	for _, char := range orgID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fiber.NewError(fiber.StatusBadRequest, "organization ID can only contain letters, numbers, and hyphens")
		}
	}

	return nil
}

func validateProjectIDFormat(projectID string) error {
	if len(projectID) < 6 || len(projectID) > 30 {
		return fiber.NewError(fiber.StatusBadRequest, "project ID must be 6-30 characters")
	}

	// Must start with letter
	if !((projectID[0] >= 'a' && projectID[0] <= 'z') || (projectID[0] >= 'A' && projectID[0] <= 'Z')) {
		return fiber.NewError(fiber.StatusBadRequest, "project ID must start with a letter")
	}

	// Can contain letters, numbers, hyphens
	for _, char := range projectID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fiber.NewError(fiber.StatusBadRequest, "project ID can only contain lowercase letters, numbers, and hyphens")
		}
	}

	return nil
}
