package http

import (
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

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
	OrganizationHandler *OrganizationHandler // Organization management
	WebSocketHandler    *WebSocketHandler    // Real-time WebSocket support
}

// NewFirestoreHTTPHandler creates a new HTTPHandler with organization support
func NewFirestoreHTTPHandler(
	firestoreUC usecase.FirestoreUsecaseInterface,
	securityUC usecase.SecurityUsecase,
	realtimeUC usecase.RealtimeUsecase,
	authClient client.AuthClient,
	log logger.Logger,
	organizationHandler *OrganizationHandler,
	webSocketHandler *WebSocketHandler,
) *HTTPHandler {
	return &HTTPHandler{
		FirestoreUC:         firestoreUC,
		SecurityUC:          securityUC,
		RealtimeUC:          realtimeUC,
		AuthClient:          authClient,
		Log:                 log,
		OrganizationHandler: organizationHandler,
		WebSocketHandler:    webSocketHandler,
	}
}

// RegisterRoutes registers all Firestore API routes following hexagonal architecture
func (h *HTTPHandler) RegisterRoutes(router fiber.Router) {
	// Register organization management routes (admin API)
	if h.OrganizationHandler != nil {
		if app, ok := router.(*fiber.App); ok {
			h.registerOrganizationRoutes(app)
		}
	}

	// Register WebSocket routes for real-time updates
	if h.WebSocketHandler != nil {
		h.WebSocketHandler.RegisterRoutes(router)
	}

	// Register Firestore API routes with organization hierarchy
	// /api/v1/organizations/{organizationId}/projects/{projectId}/databases/{databaseId}/documents/...
	orgAPI := router.Group("/organizations/:organizationId", TenantMiddleware())
	projectAPI := orgAPI.Group("/projects/:projectID", ProjectMiddleware())
	dbAPI := projectAPI.Group("/databases/:databaseID", ValidateFirestoreHierarchy())
	// Register domain-specific routes
	// Note: Atomic routes must come before document routes to avoid route conflicts
	h.registerAtomicRoutes(dbAPI)
	h.registerDocumentRoutes(dbAPI)
	h.registerCollectionRoutes(dbAPI)
	h.registerIndexRoutes(dbAPI)
	h.registerBatchRoutes(dbAPI)
	h.registerTransactionRoutes(dbAPI)
	h.registerProjectRoutes(orgAPI)      // Projects should be at org level
	h.registerDatabaseRoutes(projectAPI) // Databases should be at project level

	// Debug route to verify router is working
	router.All("/debug/*", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Debug route reached",
			"path":    c.Params("*"),
			"method":  c.Method(),
			"url":     c.OriginalURL(),
		})
	})
}

// registerOrganizationRoutes handles organization-level routes
func (h *HTTPHandler) registerOrganizationRoutes(app *fiber.App) {
	// Organization routes are handled by OrganizationHandler
	// This follows the separation of concerns principle
	if h.OrganizationHandler != nil {
		h.OrganizationHandler.RegisterRoutes(app)
	}
}

// registerDocumentRoutes registers document-related endpoints
// registerDocumentRoutes registers document-related endpoints
func (h *HTTPHandler) registerDocumentRoutes(router fiber.Router) {
	// Query endpoints (most specific routes first)
	router.Post("/documents:runQuery", h.RunQuery)        // Firestore-compliant query endpoint
	router.Post("/query/:collectionID", h.QueryDocuments) // Legacy endpoint for backward compatibility

	// Specific subcollection routes (most specific first)
	router.Post("/documents/:col1/:doc1/:col2/:doc2/:col3", h.CreateDocumentInSubcollection)     // 5 segments - deep subcollection
	router.Get("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.GetDocumentFromSubcollection) // 6 segments
	router.Put("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.UpdateDocumentInSubcollection)
	router.Delete("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.DeleteDocumentFromSubcollection)

	router.Post("/documents/:col1/:doc1/:col2", h.CreateDocumentInSubcollection)     // 3 segments - single subcollection
	router.Get("/documents/:col1/:doc1/:col2/:doc2", h.GetDocumentFromSubcollection) // 4 segments
	router.Put("/documents/:col1/:doc1/:col2/:doc2", h.UpdateDocumentInSubcollection)
	router.Delete("/documents/:col1/:doc1/:col2/:doc2", h.DeleteDocumentFromSubcollection)

	// Standard document endpoints
	router.Post("/documents/:collectionID", h.CreateDocument)
	router.Get("/documents/:collectionID/:documentID", h.GetDocument)
	router.Put("/documents/:collectionID/:documentID", h.UpdateDocument)
	router.Delete("/documents/:collectionID/:documentID", h.DeleteDocument)
	router.Get("/documents/:collectionID", h.ListDocuments) // List all documents in a collection
}

// registerCollectionRoutes registers collection-related endpoints
func (h *HTTPHandler) registerCollectionRoutes(router fiber.Router) {
	router.Get("/collections", h.ListCollections)
	router.Post("/collections", h.CreateCollection)
	router.Get("/collections/:collectionID", h.GetCollection)
	router.Put("/collections/:collectionID", h.UpdateCollection)
	router.Delete("/collections/:collectionID", h.DeleteCollection)
	router.Get("/documents/:collectionID/:documentID/subcollections", h.ListSubcollections)
}

// registerIndexRoutes registers index-related endpoints
func (h *HTTPHandler) registerIndexRoutes(router fiber.Router) {
	router.Post("/collections/:collectionID/indexes", h.CreateIndex)
	router.Get("/collections/:collectionID/indexes", h.ListIndexes)
	router.Delete("/collections/:collectionID/indexes/:indexID", h.DeleteIndex)
}

// registerBatchRoutes registers batch operation endpoints
func (h *HTTPHandler) registerBatchRoutes(router fiber.Router) {
	router.Post("/batchWrite", h.BatchWrite)
}

// registerTransactionRoutes registers transaction-related endpoints
func (h *HTTPHandler) registerTransactionRoutes(router fiber.Router) {
	router.Post("/beginTransaction", h.BeginTransaction)
	router.Post("/commit", h.CommitTransaction)
}

// registerAtomicRoutes registers atomic operation endpoints
func (h *HTTPHandler) registerAtomicRoutes(router fiber.Router) {
	router.Post("/documents/:collectionID/:documentID/increment", h.AtomicIncrement)
	router.Post("/documents/:collectionID/:documentID/arrayUnion", h.AtomicArrayUnion)
	router.Post("/documents/:collectionID/:documentID/arrayRemove", h.AtomicArrayRemove)
	router.Post("/documents/:collectionID/:documentID/serverTimestamp", h.AtomicServerTimestamp)
}

// registerProjectRoutes registers project-related endpoints
func (h *HTTPHandler) registerProjectRoutes(router fiber.Router) {
	router.Post("/projects", h.CreateProject)
	router.Get("/projects/:projectID", h.GetProject)
	router.Put("/projects/:projectID", h.UpdateProject)
	router.Delete("/projects/:projectID", h.DeleteProject)
	router.Get("/projects", h.ListProjects)
}

// registerDatabaseRoutes registers database-related endpoints
func (h *HTTPHandler) registerDatabaseRoutes(router fiber.Router) {
	router.Post("/databases", h.CreateDatabase)
	router.Get("/databases/:databaseID", h.GetDatabase)
	router.Put("/databases/:databaseID", h.UpdateDatabase)
	router.Delete("/databases/:databaseID", h.DeleteDatabase)
	router.Get("/databases", h.ListDatabases)
}
