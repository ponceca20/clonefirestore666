package http

import (
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// AtomicOperationsHandler provides Firestore-compatible atomic operations
// following hexagonal architecture principles for clean separation of concerns

// atomicOperationRequest represents common fields for atomic operations
type atomicOperationRequest struct {
	Field string `json:"field" validate:"required"`
}

// atomicIncrementRequest represents the request body for atomic increment operations
type atomicIncrementRequest struct {
	atomicOperationRequest
	IncrementBy interface{} `json:"incrementBy" validate:"required"`
}

// atomicArrayRequest represents the request body for array operations
type atomicArrayRequest struct {
	atomicOperationRequest
	Elements []interface{} `json:"elements" validate:"required"`
}

// validateFieldName validates that the field name is provided following Firestore standards
func validateFieldName(field string) *fiber.Map {
	if field == "" {
		return &fiber.Map{
			"error":   "missing_field",
			"message": "Field name is required for atomic operations",
		}
	}
	return nil
}

// validateElements validates that elements array is provided and not empty
func validateElements(elements []interface{}) *fiber.Map {
	if len(elements) == 0 {
		return &fiber.Map{
			"error":   "missing_elements",
			"message": "Elements array is required for atomic array operations",
		}
	}
	return nil
}

// extractPathParams extracts common path parameters for atomic operations
func extractPathParams(c *fiber.Ctx) (projectID, databaseID, collectionID, documentID string) {
	return c.Params("projectID"), c.Params("databaseID"),
		c.Params("collectionID"), c.Params("documentID")
}

// AtomicIncrement handles atomic increment operations following Firestore API standards
func (h *HTTPHandler) AtomicIncrement(c *fiber.Ctx) error {
	projectID, databaseID, collectionID, documentID := extractPathParams(c)

	h.Log.Debug("Performing atomic increment operation",
		"collection", collectionID,
		"document", documentID,
		"project", projectID,
		"database", databaseID)

	var reqBody atomicIncrementRequest
	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse atomic increment request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body for atomic increment operation",
		})
	}

	// Validate required fields following Firestore API standards
	if validationErr := validateFieldName(reqBody.Field); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}
	if reqBody.IncrementBy == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_increment_by",
			"message": "IncrementBy value is required for atomic increment operations",
		})
	}

	// Build use case request following hexagonal architecture
	req := usecase.AtomicIncrementRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Field:        reqBody.Field,
		IncrementBy:  reqBody.IncrementBy,
	}

	// Execute atomic operation through use case layer
	response, err := h.FirestoreUC.AtomicIncrement(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Atomic increment operation failed",
			"error", err,
			"field", req.Field,
			"document", documentID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_increment_failed",
			"message": "Failed to perform atomic increment: " + err.Error(),
		})
	}

	h.Log.Info("Atomic increment operation completed successfully",
		"field", req.Field,
		"newValue", response.NewValue,
		"document", documentID)

	return c.JSON(response)
}

// AtomicArrayUnion handles atomic array union operations following Firestore API standards
func (h *HTTPHandler) AtomicArrayUnion(c *fiber.Ctx) error {
	projectID, databaseID, collectionID, documentID := extractPathParams(c)

	h.Log.Debug("Performing atomic array union operation",
		"collection", collectionID,
		"document", documentID,
		"project", projectID,
		"database", databaseID)

	var reqBody atomicArrayRequest
	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse atomic array union request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body for atomic array union operation",
		})
	}

	// Validate required fields following Firestore API standards
	if validationErr := validateFieldName(reqBody.Field); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}

	if validationErr := validateElements(reqBody.Elements); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}

	// Build use case request following hexagonal architecture
	req := usecase.AtomicArrayUnionRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Field:        reqBody.Field,
		Elements:     reqBody.Elements,
	}

	// Execute atomic operation through use case layer
	err := h.FirestoreUC.AtomicArrayUnion(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Atomic array union operation failed",
			"error", err,
			"field", req.Field,
			"document", documentID,
			"elementsCount", len(req.Elements))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_array_union_failed",
			"message": "Failed to perform atomic array union: " + err.Error(),
		})
	}
	h.Log.Info("Atomic array union operation completed successfully",
		"field", req.Field,
		"document", documentID,
		"elementsAdded", len(req.Elements))

	return c.SendStatus(fiber.StatusOK)
}

// AtomicArrayRemove handles atomic array remove operations following Firestore API standards
func (h *HTTPHandler) AtomicArrayRemove(c *fiber.Ctx) error {
	projectID, databaseID, collectionID, documentID := extractPathParams(c)

	h.Log.Debug("Performing atomic array remove operation",
		"collection", collectionID,
		"document", documentID,
		"project", projectID,
		"database", databaseID)

	var reqBody atomicArrayRequest
	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse atomic array remove request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body for atomic array remove operation",
		})
	}

	// Validate required fields following Firestore API standards
	if validationErr := validateFieldName(reqBody.Field); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}

	if validationErr := validateElements(reqBody.Elements); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}

	// Build use case request following hexagonal architecture
	req := usecase.AtomicArrayRemoveRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Field:        reqBody.Field,
		Elements:     reqBody.Elements,
	}

	// Execute atomic operation through use case layer
	err := h.FirestoreUC.AtomicArrayRemove(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Atomic array remove operation failed",
			"error", err,
			"field", req.Field,
			"document", documentID,
			"elementsCount", len(req.Elements))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_array_remove_failed",
			"message": "Failed to perform atomic array remove: " + err.Error(),
		})
	}
	h.Log.Info("Atomic array remove operation completed successfully",
		"field", req.Field,
		"document", documentID,
		"elementsRemoved", len(req.Elements))

	return c.SendStatus(fiber.StatusOK)
}

// AtomicServerTimestamp handles atomic server timestamp operations following Firestore API standards
func (h *HTTPHandler) AtomicServerTimestamp(c *fiber.Ctx) error {
	projectID, databaseID, collectionID, documentID := extractPathParams(c)

	h.Log.Debug("Performing atomic server timestamp operation",
		"collection", collectionID,
		"document", documentID,
		"project", projectID,
		"database", databaseID)

	var reqBody atomicOperationRequest
	if err := c.BodyParser(&reqBody); err != nil {
		h.Log.Error("Failed to parse atomic server timestamp request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body for atomic server timestamp operation",
		})
	}

	// Validate required fields following Firestore API standards
	if validationErr := validateFieldName(reqBody.Field); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(*validationErr)
	}

	// Build use case request following hexagonal architecture
	req := usecase.AtomicServerTimestampRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Field:        reqBody.Field,
	}

	// Execute atomic operation through use case layer
	err := h.FirestoreUC.AtomicServerTimestamp(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Atomic server timestamp operation failed",
			"error", err,
			"field", req.Field,
			"document", documentID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "atomic_server_timestamp_failed",
			"message": "Failed to perform atomic server timestamp: " + err.Error(),
		})
	}

	h.Log.Info("Atomic server timestamp operation completed successfully",
		"field", req.Field,
		"document", documentID)

	return c.SendStatus(fiber.StatusOK)
}
