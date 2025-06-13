package http

import (
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/errors"

	"github.com/gofiber/fiber/v2"
)

// Database handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateDatabase(c *fiber.Ctx) error {
	h.Log.Debug("Creating database via HTTP", "projectID", c.Params("projectID"))

	var req usecase.CreateDatabaseRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Validate required fields
	if req.Database == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_database",
			"message": "Database is required",
		})
	}
	database, err := h.FirestoreUC.CreateDatabase(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)

		// Handle specific error types with appropriate HTTP status codes
		if errors.IsNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "project_not_found",
				"message": err.Error(),
			})
		}

		if errors.IsValidation(err) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_failed",
				"message": err.Error(),
			})
		}

		if appErr, ok := err.(*errors.AppError); ok && appErr.Type == errors.ErrorTypeConflict {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "database_already_exists",
				"message": err.Error(),
			})
		}

		// Default to internal server error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_database_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Database created successfully", "databaseID", database.DatabaseID)
	return c.Status(fiber.StatusCreated).JSON(database)
}

func (h *HTTPHandler) GetDatabase(c *fiber.Ctx) error {
	h.Log.Debug("Getting database via HTTP",
		"projectID", c.Params("projectID"),
		"databaseID", c.Params("databaseID"))

	req := usecase.GetDatabaseRequest{
		ProjectID:  c.Params("projectID"),
		DatabaseID: c.Params("databaseID"),
	}

	database, err := h.FirestoreUC.GetDatabase(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to get database", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "database_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(database)
}

func (h *HTTPHandler) UpdateDatabase(c *fiber.Ctx) error {
	h.Log.Debug("Updating database via HTTP",
		"projectID", c.Params("projectID"),
		"databaseID", c.Params("databaseID"))

	var req usecase.UpdateDatabaseRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Ensure databaseID from path is used
	if req.Database != nil {
		req.Database.DatabaseID = c.Params("databaseID")
	}

	database, err := h.FirestoreUC.UpdateDatabase(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to update database", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_database_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Database updated successfully", "databaseID", database.DatabaseID)
	return c.JSON(database)
}

func (h *HTTPHandler) DeleteDatabase(c *fiber.Ctx) error {
	h.Log.Debug("Deleting database via HTTP",
		"projectID", c.Params("projectID"),
		"databaseID", c.Params("databaseID"))

	req := usecase.DeleteDatabaseRequest{
		ProjectID:  c.Params("projectID"),
		DatabaseID: c.Params("databaseID"),
	}

	err := h.FirestoreUC.DeleteDatabase(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete database", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_database_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Database deleted successfully", "databaseID", req.DatabaseID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) ListDatabases(c *fiber.Ctx) error {
	h.Log.Debug("Listing databases via HTTP", "projectID", c.Params("projectID"))

	req := usecase.ListDatabasesRequest{
		ProjectID: c.Params("projectID"),
	}

	databases, err := h.FirestoreUC.ListDatabases(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list databases", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_databases_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Databases listed successfully", "count", len(databases))
	return c.JSON(fiber.Map{
		"databases": databases,
		"count":     len(databases),
	})
}
