package http

import (
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// Atomic operations handlers implementation following single responsibility principle
func (h *HTTPHandler) AtomicIncrement(c *fiber.Ctx) error {
	h.Log.Debug("Performing atomic increment via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqBody struct {
		Field       string      `json:"field" validate:"required"`
		IncrementBy interface{} `json:"incrementBy" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	if reqBody.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_field",
			"message": "Field name is required",
		})
	}
	if reqBody.IncrementBy == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_increment_by",
			"message": "IncrementBy is required",
		})
	}

	req := usecase.AtomicIncrementRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Field:        reqBody.Field,
		IncrementBy:  reqBody.IncrementBy,
	}

	response, err := h.FirestoreUC.AtomicIncrement(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to perform atomic increment", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_increment_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Atomic increment completed successfully",
		"field", req.Field,
		"newValue", response.NewValue)
	return c.JSON(response)
}

func (h *HTTPHandler) AtomicArrayUnion(c *fiber.Ctx) error {
	h.Log.Debug("Performing atomic array union via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqBody struct {
		Field    string        `json:"field" validate:"required"`
		Elements []interface{} `json:"elements" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	if reqBody.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_field",
			"message": "Field name is required",
		})
	}
	if reqBody.Elements == nil || len(reqBody.Elements) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_elements",
			"message": "Elements array is required",
		})
	}

	req := usecase.AtomicArrayUnionRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Field:        reqBody.Field,
		Elements:     reqBody.Elements,
	}

	err := h.FirestoreUC.AtomicArrayUnion(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to perform atomic array union", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_array_union_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Atomic array union completed successfully", "field", req.Field)
	return c.SendStatus(fiber.StatusOK)
}

func (h *HTTPHandler) AtomicArrayRemove(c *fiber.Ctx) error {
	h.Log.Debug("Performing atomic array remove via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqBody struct {
		Field    string        `json:"field" validate:"required"`
		Elements []interface{} `json:"elements" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	if reqBody.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_field",
			"message": "Field name is required",
		})
	}
	if reqBody.Elements == nil || len(reqBody.Elements) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_elements",
			"message": "Elements array is required",
		})
	}

	req := usecase.AtomicArrayRemoveRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Field:        reqBody.Field,
		Elements:     reqBody.Elements,
	}

	err := h.FirestoreUC.AtomicArrayRemove(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to perform atomic array remove", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_array_remove_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Atomic array remove completed successfully", "field", req.Field)
	return c.SendStatus(fiber.StatusOK)
}

func (h *HTTPHandler) AtomicServerTimestamp(c *fiber.Ctx) error {
	h.Log.Debug("Performing atomic server timestamp via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqBody struct {
		Field string `json:"field" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	if reqBody.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_field",
			"message": "Field name is required",
		})
	}

	req := usecase.AtomicServerTimestampRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Field:        reqBody.Field,
	}

	err := h.FirestoreUC.AtomicServerTimestamp(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to perform atomic server timestamp", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_server_timestamp_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Atomic server timestamp completed successfully", "field", req.Field)
	return c.SendStatus(fiber.StatusOK)
}
