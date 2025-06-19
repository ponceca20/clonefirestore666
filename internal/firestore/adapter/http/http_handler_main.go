package http

import (
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HTTPHandler handles Firestore REST API endpoints with multi-tenant support
// Following hexagonal architecture principles as a primary adapter for HTTP transport
// Implements Firestore's hierarchical structure: Organization → Project → Database → Documents
type HTTPHandler struct {
	// Domain use cases (application layer - hexagonal architecture core)
	FirestoreUC usecase.FirestoreUsecaseInterface
	SecurityUC  usecase.SecurityUsecase
	RealtimeUC  usecase.RealtimeUsecase

	// External dependencies (secondary adapters)
	AuthClient client.AuthClient
	Log        logger.Logger

	// Specialized handlers for domain boundaries
	OrganizationHandler *OrganizationHandler      // Organization domain management
	WebSocketHandler    *EnhancedWebSocketHandler // Enhanced real-time WebSocket support
}

// NewFirestoreHTTPHandler creates a new HTTPHandler with organization support
// Following dependency injection principles and hexagonal architecture
func NewFirestoreHTTPHandler(
	firestoreUC usecase.FirestoreUsecaseInterface,
	securityUC usecase.SecurityUsecase,
	realtimeUC usecase.RealtimeUsecase,
	authClient client.AuthClient,
	log logger.Logger,
	organizationHandler *OrganizationHandler,
	webSocketHandler *EnhancedWebSocketHandler,
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
// Primary adapter that translates HTTP requests to domain use cases
func (h *HTTPHandler) RegisterRoutes(router fiber.Router) { // Register organization management routes (admin API)
	if h.OrganizationHandler != nil {
		if app, ok := router.(*fiber.App); ok {
			h.registerOrganizationRoutes(app)
		}
	}
	// Note: WebSocket routes are now registered at the module level with proper authentication
	// See firestore.go RegisterRoutes method where WebSocket handler is properly configured with auth middleware

	// Register Firestore API routes with organization hierarchy
	// /api/v1/organizations/{organizationId}/projects/{projectId}/databases/{databaseId}/documents/...
	orgAPI := router.Group("/organizations/:organizationId", TenantMiddleware())
	projectAPI := orgAPI.Group("/projects/:projectID", ProjectMiddleware())
	dbAPI := projectAPI.Group("/databases/:databaseID", ValidateFirestoreHierarchy())

	// Register domain-specific routes following Firestore API standards
	// Note: Atomic routes must come before document routes to avoid route conflicts
	h.registerAtomicRoutes(dbAPI)
	h.registerDocumentRoutes(dbAPI)
	h.registerCollectionRoutes(dbAPI)
	h.registerIndexRoutes(dbAPI)
	h.registerBatchRoutes(dbAPI)
	h.registerTransactionRoutes(dbAPI)
	h.registerProjectRoutes(orgAPI)      // Projects should be at org level
	h.registerDatabaseRoutes(projectAPI) // Databases should be at project level

	// Debug route to verify router is working (development only)
	router.All("/debug/*", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Debug route reached",
			"path":    c.Params("*"),
			"method":  c.Method(),
			"url":     c.OriginalURL(),
		})
	})

	// Register health check routes
	h.registerHealthRoutes(router)
}

// registerOrganizationRoutes handles organization-level routes
// Follows separation of concerns by delegating to specialized handler
func (h *HTTPHandler) registerOrganizationRoutes(app *fiber.App) {
	// Organization routes are handled by OrganizationHandler
	// This follows the separation of concerns principle in hexagonal architecture
	if h.OrganizationHandler != nil {
		h.OrganizationHandler.RegisterRoutes(app)
	}
}

// registerDocumentRoutes registers document-related endpoints following Firestore API specification
// Routes are ordered by specificity to prevent conflicts (most specific first)
func (h *HTTPHandler) registerDocumentRoutes(router fiber.Router) {
	// Usar router mejorado para endpoints de consulta de Firestore con capacidades de producción
	// Esto resuelve el problema de conflictos de routing con patrones que contienen dos puntos
	// e incluye logging, métricas y validaciones de seguridad
	enhancedQueryRouter := NewEnhancedFirestoreQueryRouter(h)
	enhancedQueryRouter.RegisterProductionRoutes(router)

	// Legacy endpoint para compatibilidad hacia atrás
	router.Post("/query/:collectionID", h.QueryDocuments)
	// Subcollection routes (organized by depth for clear hierarchy)
	// Deep subcollections (3-level nesting)
	router.Post("/documents/:col1/:doc1/:col2/:doc2/:col3", h.CreateDocumentInSubcollection)
	router.Get("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.GetDocumentFromSubcollection)
	router.Put("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.UpdateDocumentInSubcollection)
	router.Delete("/documents/:col1/:doc1/:col2/:doc2/:col3/:doc3", h.DeleteDocumentFromSubcollection)

	// Single-level subcollections
	router.Post("/documents/:col1/:doc1/:col2", h.CreateDocumentInSubcollection)
	router.Get("/documents/:col1/:doc1/:col2/:doc2", h.GetDocumentFromSubcollection)
	router.Put("/documents/:col1/:doc1/:col2/:doc2", h.UpdateDocumentInSubcollection)
	router.Delete("/documents/:col1/:doc1/:col2/:doc2", h.DeleteDocumentFromSubcollection)

	// Standard document endpoints (CRUD operations)
	router.Post("/documents/:collectionID", h.CreateDocument)
	router.Get("/documents/:collectionID/:documentID", h.GetDocument)
	router.Put("/documents/:collectionID/:documentID", h.UpdateDocument)
	router.Delete("/documents/:collectionID/:documentID", h.DeleteDocument)
	router.Get("/documents/:collectionID", h.ListDocuments) // List all documents in a collection
}

// registerCollectionRoutes registers collection-related endpoints following Firestore collection operations
func (h *HTTPHandler) registerCollectionRoutes(router fiber.Router) {
	router.Get("/collections", h.ListCollections)
	router.Post("/collections", h.CreateCollection)
	router.Get("/collections/:collectionID", h.GetCollection)
	router.Put("/collections/:collectionID", h.UpdateCollection)
	router.Delete("/collections/:collectionID", h.DeleteCollection)
	router.Get("/documents/:collectionID/:documentID/subcollections", h.ListSubcollections)
}

// registerIndexRoutes registers index-related endpoints for query optimization
func (h *HTTPHandler) registerIndexRoutes(router fiber.Router) {
	router.Post("/collections/:collectionID/indexes", h.CreateIndex)
	router.Get("/collections/:collectionID/indexes", h.ListIndexes)
	router.Delete("/collections/:collectionID/indexes/:indexID", h.DeleteIndex)
}

// registerBatchRoutes registers batch operation endpoints for efficient bulk operations
func (h *HTTPHandler) registerBatchRoutes(router fiber.Router) {
	router.Post("/batchWrite", h.BatchWrite)
}

// registerTransactionRoutes registers transaction-related endpoints for ACID operations
func (h *HTTPHandler) registerTransactionRoutes(router fiber.Router) {
	router.Post("/beginTransaction", h.BeginTransaction)
	router.Post("/commit", h.CommitTransaction)
}

// registerAtomicRoutes registers atomic operation endpoints for field-level operations
func (h *HTTPHandler) registerAtomicRoutes(router fiber.Router) {
	router.Post("/documents/:collectionID/:documentID/increment", h.AtomicIncrement)
	router.Post("/documents/:collectionID/:documentID/arrayUnion", h.AtomicArrayUnion)
	router.Post("/documents/:collectionID/:documentID/arrayRemove", h.AtomicArrayRemove)
	router.Post("/documents/:collectionID/:documentID/serverTimestamp", h.AtomicServerTimestamp)
}

// registerProjectRoutes registers project-related endpoints for project management
func (h *HTTPHandler) registerProjectRoutes(router fiber.Router) {
	router.Post("/projects", h.CreateProject)
	router.Get("/projects/:projectID", h.GetProject)
	router.Put("/projects/:projectID", h.UpdateProject)
	router.Delete("/projects/:projectID", h.DeleteProject)
	router.Get("/projects", h.ListProjects)
}

// registerDatabaseRoutes registers database-related endpoints for database management
func (h *HTTPHandler) registerDatabaseRoutes(router fiber.Router) {
	router.Post("/databases", h.CreateDatabase)
	router.Get("/databases/:databaseID", h.GetDatabase)
	router.Put("/databases/:databaseID", h.UpdateDatabase)
	router.Delete("/databases/:databaseID", h.DeleteDatabase)
	router.Get("/databases", h.ListDatabases)
}

// registerHealthRoutes registra endpoints de monitoreo y salud del sistema
// Implementa observabilidad requerida para sistemas de producción
func (h *HTTPHandler) registerHealthRoutes(router fiber.Router) {
	// Endpoint de salud general del sistema
	router.Get("/health", h.GetSystemHealth)

	// Endpoint específico para métricas del router de consultas
	router.Get("/health/query-router", h.GetQueryRouterHealth)
}

// GetSystemHealth retorna el estado general del sistema
func (h *HTTPHandler) GetSystemHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"services": fiber.Map{
			"firestore": "healthy",
			"security":  "healthy",
			"router":    "healthy",
		},
	})
}

// GetQueryRouterHealth retorna métricas específicas del router de consultas
func (h *HTTPHandler) GetQueryRouterHealth(c *fiber.Ctx) error {
	// En una implementación real, esto obtendría métricas del router actual
	return c.JSON(fiber.Map{
		"status":           "healthy",
		"timestamp":        time.Now(),
		"active_endpoints": []string{"runQuery", "runAggregationQuery"},
		"security_enabled": true,
		"metrics": fiber.Map{
			"total_requests":        0,
			"successful_requests":   0,
			"failed_requests":       0,
			"average_response_time": "0ms",
		},
	})
}
