package http

import (
	"firestore-clone/internal/firestore/usecase"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Document handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateDocument(c *fiber.Ctx) error {
	h.Log.Debug("Creating document via HTTP", "collection", c.Params("collectionID"))

	var req usecase.CreateDocumentRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")
	req.CollectionID = c.Params("collectionID")
	req.DocumentID = c.Query("documentId") // Optional from query params
	// Parse request body
	if err := c.BodyParser(&req.Data); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Validate required fields - check if data is nil or empty map
	if req.Data == nil || len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_data",
			"message": "Document data is required",
		})
	}

	// Call usecase
	document, err := h.FirestoreUC.CreateDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document created successfully", "documentID", document.DocumentID)
	return c.Status(fiber.StatusCreated).JSON(document)
}

func (h *HTTPHandler) GetDocument(c *fiber.Ctx) error {
	h.Log.Debug("Getting document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	req := usecase.GetDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
	}

	document, err := h.FirestoreUC.GetDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to get document", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "document_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(document)
}

func (h *HTTPHandler) UpdateDocument(c *fiber.Ctx) error {
	h.Log.Debug("Updating document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqData map[string]any
	if err := c.BodyParser(&reqData); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	req := usecase.UpdateDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Data:         reqData,
		Mask:         parseUpdateMaskQuery(c), // Parse update mask from query as []string
	}

	document, err := h.FirestoreUC.UpdateDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to update document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document updated successfully", "documentID", document.DocumentID)
	return c.JSON(document)
}

func (h *HTTPHandler) DeleteDocument(c *fiber.Ctx) error {
	h.Log.Debug("Deleting document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	req := usecase.DeleteDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
	}

	err := h.FirestoreUC.DeleteDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document deleted successfully", "documentID", req.DocumentID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) QueryDocuments(c *fiber.Ctx) error {
	h.Log.Debug("Querying documents via HTTP", "collection", c.Params("collectionID"))

	var req usecase.QueryRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")

	// Parse request body for structured query
	if err := c.BodyParser(&req); err != nil {
		h.Log.Error("Failed to parse query request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_query_request",
			"message": "Failed to parse query request",
		})
	}

	documents, err := h.FirestoreUC.QueryDocuments(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to execute query", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "query_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Query executed successfully", "resultCount", len(documents))
	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

// ListDocuments lists all documents in a collection with pagination
func (h *HTTPHandler) ListDocuments(c *fiber.Ctx) error {
	h.Log.Debug("Listing documents via HTTP", "collection", c.Params("collectionID"))

	req := usecase.ListDocumentsRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	// Parse optional query parameters
	if pageSize := c.QueryInt("pageSize"); pageSize > 0 {
		req.PageSize = int32(pageSize)
	}
	req.PageToken = c.Query("pageToken")
	req.OrderBy = c.Query("orderBy")
	req.ShowMissing = c.QueryBool("showMissing")

	documents, err := h.FirestoreUC.ListDocuments(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list documents", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_documents_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Documents listed successfully", "count", len(documents))
	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

// Helper to parse updateMask query param as []string (comma-separated)
func parseUpdateMaskQuery(c *fiber.Ctx) []string {
	maskParam := c.Query("updateMask")
	if maskParam == "" {
		return nil
	}
	// Firestore API expects comma-separated field paths
	fields := strings.Split(maskParam, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}
