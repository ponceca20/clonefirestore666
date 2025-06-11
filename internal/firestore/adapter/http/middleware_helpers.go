package http

import (
	"regexp"
	"strings"

	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// Middleware functions for HTTP handlers following hexagonal architecture principles

// Organization ID validation regex - must start with letter, be 3+ chars, contain only letters/numbers/hyphens
var orgIDRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{2,}$`)

// validateOrganizationID validates organization ID format
func validateOrganizationID(orgID string) bool {
	return orgIDRegex.MatchString(orgID)
}

// TenantMiddleware validates tenant context and organization access
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "organization_id_missing",
				"message": "Organization ID is required",
			})
		}

		// Validate organization ID format
		if !validateOrganizationID(organizationID) {
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

		// TODO: Add tenant validation logic here
		// - Validate organization exists
		// - Check user access to organization
		// - Set tenant database context

		return c.Next()
	}
}

// ProjectMiddleware validates project context within organization
func ProjectMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract project ID from path
		projectID := c.Params("projectID")
		if projectID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "missing_project_id",
				"message": "Project ID is required",
			})
		}

		// Set project context for downstream handlers
		c.Locals("projectID", projectID)

		// Add project ID to the Go context
		ctx := c.UserContext()
		ctx = utils.WithProjectID(ctx, projectID)
		c.SetUserContext(ctx)

		// TODO: Add project validation logic here
		// - Validate project exists within organization
		// - Check user access to project
		// - Set project-specific context

		return c.Next()
	}
}

// ValidateFirestoreHierarchy validates the complete Firestore hierarchy
func ValidateFirestoreHierarchy() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract database ID from path
		databaseID := c.Params("databaseID")
		if databaseID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "missing_database_id",
				"message": "Database ID is required",
			})
		}

		// Set database context for downstream handlers
		c.Locals("databaseID", databaseID)

		// Add database ID to the Go context
		ctx := c.UserContext()
		ctx = utils.WithDatabaseID(ctx, databaseID)
		c.SetUserContext(ctx)

		// TODO: Add hierarchy validation logic here
		// - Validate database exists within project
		// - Check user access to database
		// - Validate complete organization -> project -> database hierarchy
		// - Set database-specific context

		return c.Next()
	}
}

// ErrorHandlerMiddleware provides consistent error handling across all handlers
func ErrorHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add error handling context
		c.Locals("requestStartTime", c.Context().Time())

		// Continue to next handler
		err := c.Next()

		// Handle any errors that occurred
		if err != nil {
			// Log error with context
			// Return consistent error response
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_server_error",
				"message": "An internal error occurred",
			})
		}

		return nil
	}
}

// LoggingMiddleware provides request/response logging
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Log request details
		// Add request ID for tracing
		// Continue to next handler
		return c.Next()
	}
}

// AuthenticationMiddleware validates user authentication
func AuthenticationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Add authentication logic here
		// - Extract and validate auth token
		// - Set user context
		// - Check user permissions

		return c.Next()
	}
}
