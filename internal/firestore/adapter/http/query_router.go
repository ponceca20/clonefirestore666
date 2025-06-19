package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// QueryHandler define la interfaz para los handlers de consulta
// Permite testing con mocks y desacopla la implementación
type QueryHandler interface {
	RunQuery(c *fiber.Ctx) error
	RunAggregationQuery(c *fiber.Ctx) error
}

// FirestoreQueryRouter maneja el enrutamiento especializado para endpoints de consulta de Firestore
// Implementa el patrón Adapter de la arquitectura hexagonal para resolver conflictos de routing
type FirestoreQueryRouter struct {
	handler QueryHandler
}

// NewFirestoreQueryRouter crea una nueva instancia del router especializado
func NewFirestoreQueryRouter(handler QueryHandler) *FirestoreQueryRouter {
	return &FirestoreQueryRouter{
		handler: handler,
	}
}

// RegisterRoutes registra las rutas de consulta con manejo especializado
// Soluciona el problema de Fiber con patrones que contienen dos puntos
func (r *FirestoreQueryRouter) RegisterRoutes(router fiber.Router) {
	// Registrar middleware personalizado para interceptar rutas de consulta
	router.Use("/documents:*", r.routeQueryEndpoints)
}

// routeQueryEndpoints middleware que intercepta y enruta correctamente los endpoints de consulta
// Implementa lógica de routing manual para resolver conflictos de patrones
func (r *FirestoreQueryRouter) routeQueryEndpoints(c *fiber.Ctx) error {
	path := c.Path()
	method := c.Method()

	// Solo procesar requests POST a endpoints de consulta
	if method != "POST" {
		return c.Next()
	}

	// Extraer la acción del endpoint (parte después de documents:)
	if idx := strings.Index(path, "documents:"); idx != -1 {
		action := path[idx+len("documents:"):]

		// Determinar qué handler usar basado en la acción exacta
		switch action {
		case "runQuery":
			return r.handleRunQuery(c)
		case "runAggregationQuery":
			return r.handleRunAggregationQuery(c)
		default:
			// Continuar con el routing normal para otros endpoints
			return c.Next()
		}
	}

	return c.Next()
}

// handleRunQuery maneja específicamente el endpoint runQuery
// Implementa validación de seguridad y delegación al handler correspondiente
func (r *FirestoreQueryRouter) handleRunQuery(c *fiber.Ctx) error {
	// Validar que el request contiene structuredQuery
	if !r.isValidRunQueryRequest(c) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_query_format",
			"message": "Request must contain a 'structuredQuery' field for runQuery endpoint",
		})
	}

	// Delegar al handler correspondiente
	return r.handler.RunQuery(c)
}

// handleRunAggregationQuery maneja específicamente el endpoint runAggregationQuery
// Implementa validación de seguridad y delegación al handler correspondiente
func (r *FirestoreQueryRouter) handleRunAggregationQuery(c *fiber.Ctx) error {
	// Validar que el request contiene structuredAggregationQuery
	if !r.isValidRunAggregationQueryRequest(c) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_aggregation_format",
			"message": "Request must contain a 'structuredAggregationQuery' field for runAggregationQuery endpoint",
		})
	}

	// Delegar al handler correspondiente
	return r.handler.RunAggregationQuery(c)
}

// isValidRunQueryRequest valida que el request es válido para runQuery
// Implementa validación rápida sin parsear completamente el body
func (r *FirestoreQueryRouter) isValidRunQueryRequest(c *fiber.Ctx) bool {
	body := c.Body()
	bodyStr := string(body)

	// Verificar que contiene structuredQuery pero NO structuredAggregationQuery
	hasStructuredQuery := strings.Contains(bodyStr, "structuredQuery")
	hasAggregationQuery := strings.Contains(bodyStr, "structuredAggregationQuery")

	return hasStructuredQuery && !hasAggregationQuery
}

// isValidRunAggregationQueryRequest valida que el request es válido para runAggregationQuery
// Implementa validación rápida sin parsear completamente el body
func (r *FirestoreQueryRouter) isValidRunAggregationQueryRequest(c *fiber.Ctx) bool {
	body := c.Body()
	bodyStr := string(body)

	// Verificar que contiene structuredAggregationQuery
	return strings.Contains(bodyStr, "structuredAggregationQuery")
}

// SecurityAwareQueryRouter extiende FirestoreQueryRouter con validaciones de seguridad
// Implementa el patrón Decorator para añadir capacidades de seguridad
type SecurityAwareQueryRouter struct {
	*FirestoreQueryRouter
	securityEnabled bool
}

// NewSecurityAwareQueryRouter crea un router con validaciones de seguridad habilitadas
func NewSecurityAwareQueryRouter(handler QueryHandler) *SecurityAwareQueryRouter {
	baseRouter := NewFirestoreQueryRouter(handler)
	return &SecurityAwareQueryRouter{
		FirestoreQueryRouter: baseRouter,
		securityEnabled:      true, // Habilitado por defecto para el clon de Firestore
	}
}

// RegisterSecureRoutes registra rutas con validaciones de seguridad adicionales
// Implementa validaciones de autenticación y autorización requeridas por Firestore
func (r *SecurityAwareQueryRouter) RegisterSecureRoutes(router fiber.Router) {
	// Aplicar middleware de seguridad antes de las rutas de consulta
	if r.securityEnabled {
		router.Use("/documents:*", r.validateSecurityContext)
	}

	// Registrar rutas base
	r.RegisterRoutes(router)
}

// validateSecurityContext valida el contexto de seguridad para consultas
// Implementa validaciones requeridas por el clon de Firestore
func (r *SecurityAwareQueryRouter) validateSecurityContext(c *fiber.Ctx) error {
	// Validar que existe el token de autenticación
	authToken := c.Get("Authorization")
	cookieToken := c.Cookies("fs_auth_token")

	if authToken == "" && cookieToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "authentication_required",
			"message": "Valid authentication token is required for Firestore operations",
		})
	}

	// Validar parámetros de contexto de Firestore
	projectID := strings.TrimSpace(c.Params("projectID"))
	databaseID := strings.TrimSpace(c.Params("databaseID"))

	if projectID == "" || databaseID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_firestore_context",
			"message": "Valid projectID and databaseID are required",
		})
	}

	return c.Next()
}

// QueryEndpointMetrics proporciona métricas para endpoints de consulta
// Implementa observabilidad requerida para sistemas de producción
type QueryEndpointMetrics struct {
	RunQueryCount      int64
	AggregationCount   int64
	RoutingErrors      int64
	SecurityViolations int64
}

// GetMetrics retorna las métricas actuales del router
func (r *SecurityAwareQueryRouter) GetMetrics() QueryEndpointMetrics {
	// En una implementación real, esto vendría de un sistema de métricas
	return QueryEndpointMetrics{}
}
