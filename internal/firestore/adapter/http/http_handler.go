package http

import (
	"firestore-clone/internal/firestore/domain/client" // Added import
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger" // Added import

	"github.com/gofiber/fiber/v2"
)

// HTTPHandler handles Firestore REST API endpoints. (Comment updated for clarity)
// It should be initialized with the Firestore usecase and other dependencies.
type HTTPHandler struct {
	FirestoreUC usecase.FirestoreUsecaseInterface // Changed to Interface
	SecurityUC  usecase.SecurityUsecase
	RealtimeUC  usecase.RealtimeUsecase
	AuthClient  client.AuthClient
	Log         logger.Logger // Field name 'Log' to avoid potential conflicts
}

// NewFirestoreHTTPHandler creates a new HTTPHandler. (Name and signature updated)
func NewFirestoreHTTPHandler(
	firestoreUC usecase.FirestoreUsecaseInterface,
	securityUC usecase.SecurityUsecase,
	realtimeUC usecase.RealtimeUsecase,
	authClient client.AuthClient,
	log logger.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		FirestoreUC: firestoreUC,
		SecurityUC:  securityUC,
		RealtimeUC:  realtimeUC,
		AuthClient:  authClient,
		Log:         log,
	}
}

func (h *HTTPHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/firestore")
	api.Post("/projects/:projectID/databases/:databaseID/documents/:collectionID", h.CreateDocument)
	api.Get("/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.GetDocument)
	api.Put("/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.UpdateDocument)
	api.Delete("/projects/:projectID/databases/:databaseID/documents/:collectionID/:documentID", h.DeleteDocument)
	api.Post("/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)
}

func (h *HTTPHandler) CreateDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")

	var reqBody map[string]interface{}
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	req := usecase.CreateDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		Data:         reqBody,
	}
	doc, err := h.FirestoreUC.CreateDocument(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(doc)
}

func (h *HTTPHandler) GetDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	req := usecase.GetDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}
	doc, err := h.FirestoreUC.GetDocument(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(doc)
}

func (h *HTTPHandler) UpdateDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	var reqBody map[string]interface{}
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(doc)
}

func (h *HTTPHandler) DeleteDocument(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")
	documentID := c.Params("documentID")

	req := usecase.DeleteDocumentRequest{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}
	err := h.FirestoreUC.DeleteDocument(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) QueryDocuments(c *fiber.Ctx) error {
	projectID := c.Params("projectID")
	databaseID := c.Params("databaseID")
	collectionID := c.Params("collectionID")

	var query model.Query
	if err := c.BodyParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query body"})
	}

	req := usecase.QueryRequest{
		ProjectID:       projectID,
		DatabaseID:      databaseID,
		StructuredQuery: &query,
		Parent:          "projects/" + projectID + "/databases/" + databaseID + "/documents/" + collectionID,
	}

	results, err := h.FirestoreUC.RunQuery(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(results)
}
