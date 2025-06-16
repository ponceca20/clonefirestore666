package http

import (
	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// Middleware functions for HTTP handlers following hexagonal architecture principles

// ProjectMiddleware validates project context within organization
func ProjectMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Log path, method, and extracted parameters
		println("[ProjectMiddleware] Path:", c.Path(), "Method:", c.Method())
		projectID := c.Params("projectID")
		println("[ProjectMiddleware] Extracted projectID:", projectID)
		if projectID == "" {
			println("[ProjectMiddleware] Missing projectID!")
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

		println("[ProjectMiddleware] Set projectID in context and locals.")

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
		// Log path, method, and extracted parameters
		println("[ValidateFirestoreHierarchy] Path:", c.Path(), "Method:", c.Method())
		databaseID := c.Params("databaseID")
		println("[ValidateFirestoreHierarchy] Extracted databaseID:", databaseID)
		if databaseID == "" {
			println("[ValidateFirestoreHierarchy] Missing databaseID!")
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

		println("[ValidateFirestoreHierarchy] Set databaseID in context and locals.")

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
