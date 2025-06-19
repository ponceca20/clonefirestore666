package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// QueryEndpointLogger proporciona logging estructurado para endpoints de consulta
// Implementa observabilidad requerida para sistemas de producción siguiendo arquitectura hexagonal
type QueryEndpointLogger struct {
	logger *logrus.Logger
}

// NewQueryEndpointLogger crea una nueva instancia del logger especializado
func NewQueryEndpointLogger() *QueryEndpointLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	return &QueryEndpointLogger{
		logger: logger,
	}
}

// LogQueryRequest registra información detallada de requests de consulta
func (l *QueryEndpointLogger) LogQueryRequest(c *fiber.Ctx, endpoint string, startTime time.Time) {
	duration := time.Since(startTime)

	l.logger.WithFields(logrus.Fields{
		"component":    "firestore_query_router",
		"endpoint":     endpoint,
		"method":       c.Method(),
		"path":         c.Path(),
		"project_id":   c.Params("projectID"),
		"database_id":  c.Params("databaseID"),
		"org_id":       c.Params("orgID"),
		"duration_ms":  duration.Milliseconds(),
		"content_type": c.Get("Content-Type"),
		"user_agent":   c.Get("User-Agent"),
		"ip_address":   c.IP(),
		"status_code":  c.Response().StatusCode(),
		"body_size":    len(c.Body()),
	}).Info("Query endpoint request processed")
}

// LogRoutingDecision registra decisiones de enrutamiento para debugging
func (l *QueryEndpointLogger) LogRoutingDecision(c *fiber.Ctx, endpoint string, decision string, reason string) {
	l.logger.WithFields(logrus.Fields{
		"component":   "firestore_query_router",
		"endpoint":    endpoint,
		"path":        c.Path(),
		"decision":    decision,
		"reason":      reason,
		"project_id":  c.Params("projectID"),
		"database_id": c.Params("databaseID"),
	}).Debug("Query routing decision made")
}

// LogSecurityEvent registra eventos de seguridad
func (l *QueryEndpointLogger) LogSecurityEvent(c *fiber.Ctx, eventType string, details string) {
	l.logger.WithFields(logrus.Fields{
		"component":   "firestore_query_security",
		"event_type":  eventType,
		"details":     details,
		"path":        c.Path(),
		"ip_address":  c.IP(),
		"user_agent":  c.Get("User-Agent"),
		"project_id":  c.Params("projectID"),
		"database_id": c.Params("databaseID"),
		"org_id":      c.Params("orgID"),
	}).Warn("Security event detected")
}

// LogError registra errores de manera estructurada
func (l *QueryEndpointLogger) LogError(c *fiber.Ctx, err error, context string) {
	l.logger.WithFields(logrus.Fields{
		"component":   "firestore_query_router",
		"error":       err.Error(),
		"context":     context,
		"path":        c.Path(),
		"method":      c.Method(),
		"project_id":  c.Params("projectID"),
		"database_id": c.Params("databaseID"),
	}).Error("Query router error occurred")
}

// EnhancedFirestoreQueryRouter extiende el router base con logging y métricas
// Implementa el patrón Decorator para añadir capacidades de observabilidad
type EnhancedFirestoreQueryRouter struct {
	*SecurityAwareQueryRouter
	logger  *QueryEndpointLogger
	metrics *QueryRouterMetrics
}

// NewEnhancedFirestoreQueryRouter crea un router con capacidades completas de producción
func NewEnhancedFirestoreQueryRouter(handler *HTTPHandler) *EnhancedFirestoreQueryRouter {
	baseRouter := NewSecurityAwareQueryRouter(handler)
	logger := NewQueryEndpointLogger()
	metrics := NewQueryRouterMetrics()

	return &EnhancedFirestoreQueryRouter{
		SecurityAwareQueryRouter: baseRouter,
		logger:                   logger,
		metrics:                  metrics,
	}
}

// RegisterProductionRoutes registra rutas con todas las capacidades de producción
func (r *EnhancedFirestoreQueryRouter) RegisterProductionRoutes(router fiber.Router) {
	// Middleware de métricas y logging
	router.Use("/documents:*", r.metricsAndLoggingMiddleware)

	// Registrar rutas base con seguridad
	r.RegisterSecureRoutes(router)
}

// metricsAndLoggingMiddleware maneja métricas y logging para todos los requests
func (r *EnhancedFirestoreQueryRouter) metricsAndLoggingMiddleware(c *fiber.Ctx) error {
	startTime := time.Now()

	// Determinar endpoint
	endpoint := r.extractEndpointFromPath(c.Path())

	// Incrementar contador de requests
	r.metrics.IncrementRequestCount(endpoint)

	// Procesar request
	err := c.Next()

	// Log del request completado
	r.logger.LogQueryRequest(c, endpoint, startTime)

	// Actualizar métricas de duración
	duration := time.Since(startTime)
	r.metrics.RecordRequestDuration(endpoint, duration)

	// Si hubo error, registrarlo
	if err != nil {
		r.logger.LogError(c, err, "request_processing")
		r.metrics.IncrementErrorCount(endpoint)
	}

	return err
}

// extractEndpointFromPath extrae el tipo de endpoint del path
func (r *EnhancedFirestoreQueryRouter) extractEndpointFromPath(path string) string {
	if contains := func(s, substr string) bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}; contains(path, "documents:runQuery") {
		return "runQuery"
	} else if contains(path, "documents:runAggregationQuery") {
		return "runAggregationQuery"
	}
	return "unknown"
}

// QueryRouterMetrics maneja métricas específicas del router de consultas
type QueryRouterMetrics struct {
	requestCounts      map[string]int64
	errorCounts        map[string]int64
	durations          map[string][]time.Duration
	securityViolations int64
}

// NewQueryRouterMetrics crea una nueva instancia de métricas
func NewQueryRouterMetrics() *QueryRouterMetrics {
	return &QueryRouterMetrics{
		requestCounts: make(map[string]int64),
		errorCounts:   make(map[string]int64),
		durations:     make(map[string][]time.Duration),
	}
}

// IncrementRequestCount incrementa el contador de requests para un endpoint
func (m *QueryRouterMetrics) IncrementRequestCount(endpoint string) {
	m.requestCounts[endpoint]++
}

// IncrementErrorCount incrementa el contador de errores para un endpoint
func (m *QueryRouterMetrics) IncrementErrorCount(endpoint string) {
	m.errorCounts[endpoint]++
}

// RecordRequestDuration registra la duración de un request
func (m *QueryRouterMetrics) RecordRequestDuration(endpoint string, duration time.Duration) {
	if m.durations[endpoint] == nil {
		m.durations[endpoint] = make([]time.Duration, 0)
	}
	m.durations[endpoint] = append(m.durations[endpoint], duration)

	// Mantener solo las últimas 1000 mediciones para evitar memory leaks
	if len(m.durations[endpoint]) > 1000 {
		m.durations[endpoint] = m.durations[endpoint][1:]
	}
}

// GetMetricsSummary retorna un resumen de las métricas actuales
func (m *QueryRouterMetrics) GetMetricsSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	// Contadores de requests
	summary["request_counts"] = m.requestCounts
	summary["error_counts"] = m.errorCounts
	summary["security_violations"] = m.securityViolations

	// Estadísticas de duración
	durationStats := make(map[string]interface{})
	for endpoint, durations := range m.durations {
		if len(durations) > 0 {
			total := time.Duration(0)
			min := durations[0]
			max := durations[0]

			for _, d := range durations {
				total += d
				if d < min {
					min = d
				}
				if d > max {
					max = d
				}
			}

			avg := total / time.Duration(len(durations))

			durationStats[endpoint] = map[string]interface{}{
				"count":   len(durations),
				"average": avg.Milliseconds(),
				"min":     min.Milliseconds(),
				"max":     max.Milliseconds(),
			}
		}
	}
	summary["duration_stats"] = durationStats

	return summary
}

// HealthStatus representa el estado de salud del router
type HealthStatus struct {
	Status          string                 `json:"status"`
	Timestamp       time.Time              `json:"timestamp"`
	Version         string                 `json:"version"`
	Metrics         map[string]interface{} `json:"metrics"`
	ActiveEndpoints []string               `json:"active_endpoints"`
	SecurityEnabled bool                   `json:"security_enabled"`
}

// GetHealthStatus retorna el estado de salud del router
func (r *EnhancedFirestoreQueryRouter) GetHealthStatus() HealthStatus {
	return HealthStatus{
		Status:          "healthy",
		Timestamp:       time.Now(),
		Version:         "1.0.0",
		Metrics:         r.metrics.GetMetricsSummary(),
		ActiveEndpoints: []string{"runQuery", "runAggregationQuery"},
		SecurityEnabled: r.securityEnabled,
	}
}
