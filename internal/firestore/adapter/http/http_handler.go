package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2"
)

// FirestoreHTTPHandler handles HTTP requests for Firestore operations
type FirestoreHTTPHandler struct {
	firestoreUC usecase.FirestoreUsecase
	securityUC  usecase.SecurityUsecase
	authClient  client.AuthClient
	logger      logger.Logger
}

// NewFirestoreHTTPHandler creates a new HTTP handler for Firestore
func NewFirestoreHTTPHandler(
	firestoreUC usecase.FirestoreUsecase,
	securityUC usecase.SecurityUsecase,
	authClient client.AuthClient,
	log logger.Logger,
) *FirestoreHTTPHandler {
	return &FirestoreHTTPHandler{
		firestoreUC: firestoreUC,
		securityUC:  securityUC,
		authClient:  authClient,
		logger:      log,
	}
}

// RegisterRoutes registers all Firestore HTTP routes
func (h *FirestoreHTTPHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/v1")

	// Document operations
	api.Get("/documents/*", h.GetDocument)
	api.Post("/documents/*", h.CreateDocument)
	api.Patch("/documents/*", h.UpdateDocument)
	api.Delete("/documents/*", h.DeleteDocument)

	// Collection operations
	api.Get("/collections/:collectionId/documents", h.ListDocuments)

	h.logger.Info("Firestore HTTP routes registered")
}

// CreateDocumentRequest represents the request body for creating a document
type CreateDocumentRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// UpdateDocumentRequest represents the request body for updating a document
type UpdateDocumentRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// CreateDocument handles POST requests to create documents
func (h *FirestoreHTTPHandler) CreateDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	projectID := c.Params("projectId")
	collection := c.Params("collectionId")
	documentID := c.Params("documentId")

	if projectID == "" || collection == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID and collection are required",
		})
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		h.logger.Errorf("Failed to parse request body: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	path := fmt.Sprintf("projects/%s/databases/(default)/documents/%s", projectID, collection)
	if documentID != "" {
		path += "/" + documentID
	}

	// Validate authentication and get user ID
	var userID string
	token := h.extractToken(c)
	if token != "" {
		uid, err := h.authClient.ValidateToken(ctx, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication token",
			})
		}
		userID = uid
	}

	// Validate write permission
	if err := h.securityUC.ValidateWrite(ctx, userID, path, data); err != nil {
		if err == errors.ErrUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized access",
			})
		}
		if err == errors.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden access",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Create document
	document, err := h.firestoreUC.CreateDocument(ctx, path, data)
	if err != nil {
		if err == errors.ErrConflict {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Document already exists",
			})
		}
		h.logger.Error("Error creating document", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(http.StatusCreated).JSON(document)
}

// GetDocument handles GET requests to retrieve documents
func (h *FirestoreHTTPHandler) GetDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	projectID := c.Params("projectId")
	collection := c.Params("collectionId")
	documentID := c.Params("documentId")

	if projectID == "" || collection == "" || documentID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID, collection, and document ID are required",
		})
	}

	path := fmt.Sprintf("projects/%s/databases/(default)/documents/%s/%s", projectID, collection, documentID)

	// Validate authentication and get user ID
	var userID string
	token := h.extractToken(c)
	if token != "" {
		uid, err := h.authClient.ValidateToken(ctx, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication token",
			})
		}
		userID = uid
	}

	// Validate read permission
	if err := h.securityUC.ValidateRead(ctx, userID, path); err != nil {
		if err == errors.ErrUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized access",
			})
		}
		if err == errors.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden access",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Get document
	document, err := h.firestoreUC.GetDocument(ctx, path)
	if err != nil {
		if err == errors.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		h.logger.Error("Error getting document", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(document)
}

// UpdateDocument handles PATCH requests to update documents
func (h *FirestoreHTTPHandler) UpdateDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	projectID := c.Params("projectId")
	collection := c.Params("collectionId")
	documentID := c.Params("documentId")

	if projectID == "" || collection == "" || documentID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID, collection, and document ID are required",
		})
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		h.logger.Errorf("Failed to parse request body: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	path := fmt.Sprintf("projects/%s/databases/(default)/documents/%s/%s", projectID, collection, documentID)

	// Validate authentication and get user ID
	var userID string
	token := h.extractToken(c)
	if token != "" {
		uid, err := h.authClient.ValidateToken(ctx, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication token",
			})
		}
		userID = uid
	}

	// Validate write permission
	if err := h.securityUC.ValidateWrite(ctx, userID, path, data); err != nil {
		if err == errors.ErrUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized access",
			})
		}
		if err == errors.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden access",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update document
	document, err := h.firestoreUC.UpdateDocument(ctx, path, data)
	if err != nil {
		if err == errors.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		h.logger.Error("Error updating document", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(document)
}

// DeleteDocument handles DELETE requests to delete documents
func (h *FirestoreHTTPHandler) DeleteDocument(c *fiber.Ctx) error {
	ctx := c.Context()
	projectID := c.Params("projectId")
	collection := c.Params("collectionId")
	documentID := c.Params("documentId")

	if projectID == "" || collection == "" || documentID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID, collection, and document ID are required",
		})
	}

	path := fmt.Sprintf("projects/%s/databases/(default)/documents/%s/%s", projectID, collection, documentID)

	// Validate authentication and get user ID
	var userID string
	token := h.extractToken(c)
	if token != "" {
		uid, err := h.authClient.ValidateToken(ctx, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication token",
			})
		}
		userID = uid
	}

	// Validate delete permission
	if err := h.securityUC.ValidateDelete(ctx, userID, path); err != nil {
		if err == errors.ErrUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized access",
			})
		}
		if err == errors.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden access",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Delete document
	if err := h.firestoreUC.DeleteDocument(ctx, path); err != nil {
		if err == errors.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		h.logger.Error("Error deleting document", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListDocuments handles GET requests to list documents in a collection
func (h *FirestoreHTTPHandler) ListDocuments(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	collection := c.Params("collectionId")

	if projectID == "" || collection == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID and collection are required",
		})
	}

	// Parse query parameters
	limit := 50 // default limit
	if limitStr := c.Query("pageSize"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	pageToken := c.Query("pageToken")

	path := fmt.Sprintf("projects/%s/databases/(default)/documents/%s", projectID, collection)

	// Validate authentication and get user ID
	var userID string
	token := h.extractToken(c)
	if token != "" {
		uid, err := h.authClient.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication token",
			})
		}
		userID = uid
	}

	// For now, return placeholder response
	// TODO: Implement proper collection listing with query support
	_ = userID // Avoid unused variable warning
	docs, nextPageToken, err := h.firestoreUC.ListDocuments(c.Context(), path, limit, pageToken)
	if err != nil {
		h.logger.Errorf("Failed to list documents: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list documents",
		})
	}

	response := fiber.Map{
		"documents": docs,
	}

	if nextPageToken != "" {
		response["nextPageToken"] = nextPageToken
	}

	return c.JSON(response)
}

// RunQuery handles POST requests to run structured queries
func (h *FirestoreHTTPHandler) RunQuery(c *fiber.Ctx) error {
	projectID := c.Params("projectId")

	if projectID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID is required",
		})
	}

	var queryReq map[string]interface{}
	if err := c.BodyParser(&queryReq); err != nil {
		h.logger.Errorf("Failed to parse query request: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query request",
		})
	}

	// Convert query request to JSON for processing
	queryBytes, err := json.Marshal(queryReq)
	if err != nil {
		h.logger.Errorf("Failed to marshal query: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query format",
		})
	}

	docs, err := h.firestoreUC.RunQuery(c.Context(), projectID, string(queryBytes))
	if err != nil {
		h.logger.Errorf("Failed to run query: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to execute query",
		})
	}

	return c.JSON(fiber.Map{
		"documents": docs,
	})
}

// BeginTransaction handles POST requests to begin transactions
func (h *FirestoreHTTPHandler) BeginTransaction(c *fiber.Ctx) error {
	projectID := c.Params("projectId")

	if projectID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID is required",
		})
	}

	transactionID, err := h.firestoreUC.BeginTransaction(c.Context(), projectID)
	if err != nil {
		h.logger.Errorf("Failed to begin transaction: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to begin transaction",
		})
	}

	return c.JSON(fiber.Map{
		"transaction": transactionID,
	})
}

// CommitTransaction handles POST requests to commit transactions
func (h *FirestoreHTTPHandler) CommitTransaction(c *fiber.Ctx) error {
	projectID := c.Params("projectId")

	if projectID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Project ID is required",
		})
	}

	var commitReq map[string]interface{}
	if err := c.BodyParser(&commitReq); err != nil {
		h.logger.Errorf("Failed to parse commit request: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid commit request",
		})
	}

	transactionID, ok := commitReq["transaction"].(string)
	if !ok {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Transaction ID is required",
		})
	}

	err := h.firestoreUC.CommitTransaction(c.Context(), projectID, transactionID)
	if err != nil {
		h.logger.Errorf("Failed to commit transaction: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	return c.JSON(fiber.Map{
		"commitTime": "2024-01-01T00:00:00Z", // Placeholder timestamp
	})
}

// extractToken extracts the authentication token from the request
func (h *FirestoreHTTPHandler) extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	auth := c.Get("Authorization")
	if auth != "" && strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Try query parameter
	return c.Query("access_token")
}

// SetupRoutes configures the HTTP routes for Firestore operations
func (h *FirestoreHTTPHandler) SetupRoutes(router fiber.Router) {
	v1 := router.Group("/v1")

	// Document operations
	v1.Post("/projects/:projectId/databases/(default)/documents/:collectionId", h.CreateDocument)
	v1.Post("/projects/:projectId/databases/(default)/documents/:collectionId/:documentId", h.CreateDocument)
	v1.Get("/projects/:projectId/databases/(default)/documents/:collectionId/:documentId", h.GetDocument)
	v1.Patch("/projects/:projectId/databases/(default)/documents/:collectionId/:documentId", h.UpdateDocument)
	v1.Delete("/projects/:projectId/databases/(default)/documents/:collectionId/:documentId", h.DeleteDocument)
	v1.Get("/projects/:projectId/databases/(default)/documents/:collectionId", h.ListDocuments)

	// Query operations
	v1.Post("/projects/:projectId/databases/(default)/documents:runQuery", h.RunQuery)

	// Transaction operations
	v1.Post("/projects/:projectId/databases/(default)/documents:beginTransaction", h.BeginTransaction)
	v1.Post("/projects/:projectId/databases/(default)/documents:commit", h.CommitTransaction)
}
