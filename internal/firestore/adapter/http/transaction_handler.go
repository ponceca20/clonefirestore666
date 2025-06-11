package http

import (
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// Transaction and batch handlers implementation following single responsibility principle
func (h *HTTPHandler) BatchWrite(c *fiber.Ctx) error {
	h.Log.Debug("Performing batch write via HTTP")

	var req usecase.BatchWriteRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Validate required fields
	if len(req.Writes) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_writes",
			"message": "Write operations are required",
		})
	}

	response, err := h.FirestoreUC.RunBatchWrite(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to execute batch write", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "batch_write_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Batch write completed successfully", "operationsCount", len(req.Writes))
	return c.JSON(response)
}

func (h *HTTPHandler) BeginTransaction(c *fiber.Ctx) error {
	h.Log.Debug("Beginning transaction via HTTP", "projectID", c.Params("projectID"))

	projectID := c.Params("projectID")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_project_id",
			"message": "Project ID is required",
		})
	}

	transactionID, err := h.FirestoreUC.BeginTransaction(c.UserContext(), projectID)
	if err != nil {
		h.Log.Error("Failed to begin transaction", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "begin_transaction_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Transaction started successfully", "transactionID", transactionID)
	return c.JSON(fiber.Map{
		"transactionId": transactionID,
	})
}

func (h *HTTPHandler) CommitTransaction(c *fiber.Ctx) error {
	h.Log.Debug("Committing transaction via HTTP", "projectID", c.Params("projectID"))

	var reqBody struct {
		TransactionID string `json:"transactionId" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	projectID := c.Params("projectID")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_project_id",
			"message": "Project ID is required",
		})
	}

	if reqBody.TransactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_transaction_id",
			"message": "Transaction ID is required",
		})
	}

	err := h.FirestoreUC.CommitTransaction(c.UserContext(), projectID, reqBody.TransactionID)
	if err != nil {
		h.Log.Error("Failed to commit transaction", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "commit_transaction_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Transaction committed successfully", "transactionID", reqBody.TransactionID)
	return c.SendStatus(fiber.StatusOK)
}
