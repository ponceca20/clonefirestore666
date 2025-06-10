package http

import (
	"firestore-clone/internal/firestore/domain/client" // Added import
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger" // Added import

	"github.com/gofiber/fiber/v2"
)

// HTTPHandler handles Firestore REST API endpoints with multi-tenant support
// Following Firestore's hierarchical architecture: Organization → Project → Database → Documents
type HTTPHandler struct {
	FirestoreUC         usecase.FirestoreUsecaseInterface
	SecurityUC          usecase.SecurityUsecase
	RealtimeUC          usecase.RealtimeUsecase
	AuthClient          client.AuthClient
	Log                 logger.Logger
	OrganizationHandler *OrganizationHandler // NEW: Organization management
}

// NewFirestoreHTTPHandler creates a new HTTPHandler with organization support
func NewFirestoreHTTPHandler(
	firestoreUC usecase.FirestoreUsecaseInterface,
	securityUC usecase.SecurityUsecase,
	realtimeUC usecase.RealtimeUsecase,
	authClient client.AuthClient,
	log logger.Logger,
	organizationHandler *OrganizationHandler,
) *HTTPHandler {
	return &HTTPHandler{
		FirestoreUC:         firestoreUC,
		SecurityUC:          securityUC,
		RealtimeUC:          realtimeUC,
		AuthClient:          authClient,
		Log:                 log,
		OrganizationHandler: organizationHandler,
	}
}

func (h *HTTPHandler) RegisterRoutes(router fiber.Router) {
	// Register organization management routes (admin API) SOLO si router es *fiber.App
	if h.OrganizationHandler != nil {
		if app, ok := router.(*fiber.App); ok {
			h.OrganizationHandler.RegisterRoutes(app)
		}
	}

	// Firestore API with organization hierarchy
	v1 := router.Group("/v1")

	// Organization-scoped Firestore API
	// /v1/organizations/{organizationId}/projects/{projectId}/databases/{databaseId}/documents/...
	orgAPI := v1.Group("/organizations/:organizationId", TenantMiddleware(), ValidateFirestoreHierarchy())
	projectAPI := orgAPI.Group("/projects/:projectID", ProjectMiddleware())
	dbAPI := projectAPI.Group("/databases/:databaseID")

	// Document endpoints
	dbAPI.Post("/documents/:collectionID", h.CreateDocument)
	dbAPI.Get("/documents/:collectionID/:documentID", h.GetDocument)
	dbAPI.Put("/documents/:collectionID/:documentID", h.UpdateDocument)
	dbAPI.Delete("/documents/:collectionID/:documentID", h.DeleteDocument)
	dbAPI.Post("/query/:collectionID", h.QueryDocuments)

	// TODO: Add more endpoints as they are implemented
	// dbAPI.Get("/collections", h.ListCollections)
	// dbAPI.Post("/collections", h.CreateCollection)
	// dbAPI.Post("/batchWrite", h.BatchWrite)
	// dbAPI.Post("/commit", h.CommitTransaction)
	// dbAPI.Get("/listen", h.ListenForChanges) // WebSocket endpoint
}

func (h *HTTPHandler) CreateDocument(c *fiber.Ctx) error {
	// Extract parameters following Firestore hierarchy
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")

	// Organization ID should be in context from middleware
	ctx := c.Context()

	var reqBody map[string]interface{}
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
			"code":    "INVALID_ARGUMENT",
		})
	}

	// Validate required parameters
	if projectID == "" || databaseID == "" || collectionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_required_parameters",
			"message": "Project ID, Database ID, and Collection ID are required",
			"code":    "INVALID_ARGUMENT",
		})
	}

	req := usecase.CreateDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		Data:         reqBody,
	}

	doc, err := h.FirestoreUC.CreateDocument(ctx, req)
	if err != nil {
		h.Log.WithFields(map[string]interface{}{
			"project_id":    projectID,
			"database_id":   databaseID,
			"collection_id": collectionID,
			"error":         err.Error(),
		}).Error("Failed to create document")

		// Return Firestore-compatible error response
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_document_failed",
			"message": err.Error(),
			"code":    "INTERNAL",
		})
	}

	h.Log.WithFields(map[string]interface{}{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   doc.DocumentID,
	}).Info("Document created successfully")

	return c.Status(fiber.StatusCreated).JSON(doc)
}

func (h *HTTPHandler) GetDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	// Validate required parameters
	if projectID == "" || databaseID == "" || collectionID == "" || documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_required_parameters",
			"message": "Project ID, Database ID, Collection ID, and Document ID are required",
			"code":    "INVALID_ARGUMENT",
		})
	}

	req := usecase.GetDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}

	doc, err := h.FirestoreUC.GetDocument(c.Context(), req)
	if err != nil {
		h.Log.WithFields(map[string]interface{}{
			"project_id":    projectID,
			"database_id":   databaseID,
			"collection_id": collectionID,
			"document_id":   documentID,
			"error":         err.Error(),
		}).Error("Failed to get document")

		// Check if document not found
		if err.Error() == "document not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "document_not_found",
				"message": "The requested document was not found",
				"code":    "NOT_FOUND",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_document_failed",
			"message": err.Error(),
			"code":    "INTERNAL",
		})
	}

	return c.JSON(doc)
}

func (h *HTTPHandler) UpdateDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	// Validate required parameters
	if projectID == "" || databaseID == "" || collectionID == "" || documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_required_parameters",
			"message": "Project ID, Database ID, Collection ID, and Document ID are required",
			"code":    "INVALID_ARGUMENT",
		})
	}

	var reqBody map[string]interface{}
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
			"code":    "INVALID_ARGUMENT",
		})
	}

	req := usecase.UpdateDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Data:         reqBody,
	}

	doc, err := h.FirestoreUC.UpdateDocument(c.Context(), req)
	if err != nil {
		h.Log.WithFields(map[string]interface{}{
			"project_id":    projectID,
			"database_id":   databaseID,
			"collection_id": collectionID,
			"document_id":   documentID,
			"error":         err.Error(),
		}).Error("Failed to update document")

		// Check if document not found
		if err.Error() == "document not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "document_not_found",
				"message": "The requested document was not found",
				"code":    "NOT_FOUND",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_document_failed",
			"message": err.Error(),
			"code":    "INTERNAL",
		})
	}

	h.Log.WithFields(map[string]interface{}{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}).Info("Document updated successfully")
	return c.JSON(doc)
}

func (h *HTTPHandler) DeleteDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	// Validate required parameters
	if projectID == "" || databaseID == "" || collectionID == "" || documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_required_parameters",
			"message": "Project ID, Database ID, Collection ID, and Document ID are required",
			"code":    "INVALID_ARGUMENT",
		})
	}

	req := usecase.DeleteDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}

	err := h.FirestoreUC.DeleteDocument(c.Context(), req)
	if err != nil {
		h.Log.WithFields(map[string]interface{}{
			"project_id":    projectID,
			"database_id":   databaseID,
			"collection_id": collectionID,
			"document_id":   documentID,
			"error":         err.Error(),
		}).Error("Failed to delete document")

		// Check if document not found
		if err.Error() == "document not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "document_not_found",
				"message": "The requested document was not found",
				"code":    "NOT_FOUND",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_document_failed",
			"message": err.Error(),
			"code":    "INTERNAL",
		})
	}

	h.Log.WithFields(map[string]interface{}{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}).Info("Document deleted successfully")

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) QueryDocuments(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")

	// Validate required parameters
	if projectID == "" || databaseID == "" || collectionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_required_parameters",
			"message": "Project ID, Database ID, and Collection ID are required",
			"code":    "INVALID_ARGUMENT",
		})
	}

	var query model.Query
	if err := c.BodyParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_query_body",
			"message": "Failed to parse query body",
			"code":    "INVALID_ARGUMENT",
		})
	}

	req := usecase.QueryRequest{
		ProjectID:       projectID,
		DatabaseID:      databaseID,
		StructuredQuery: &query,
		Parent:          "projects/" + projectID + "/databases/" + databaseID + "/documents/" + collectionID,
	}

	results, err := h.FirestoreUC.RunQuery(c.Context(), req)
	if err != nil {
		h.Log.WithFields(map[string]interface{}{
			"project_id":    projectID,
			"database_id":   databaseID,
			"collection_id": collectionID,
			"error":         err.Error(),
		}).Error("Failed to run query")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "query_failed",
			"message": err.Error(),
			"code":    "INTERNAL",
		})
	}

	return c.JSON(results)
}
