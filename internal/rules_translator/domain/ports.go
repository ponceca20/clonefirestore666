package domain

import (
	"context"
	"io"
	"time"
)

// RulesParser define el puerto para parsing de archivos firestore.rules
// Optimizado para máximo rendimiento
type RulesParser interface {
	// Parse convierte un archivo .rules en AST optimizado
	Parse(ctx context.Context, content io.Reader) (*ParseResult, error)

	// ParseString versión optimizada para strings
	ParseString(ctx context.Context, content string) (*ParseResult, error)

	// Validate verifica la sintaxis sin hacer parsing completo (más rápido)
	Validate(ctx context.Context, content io.Reader) ([]ParseError, error)

	// GetMetrics retorna métricas de rendimiento del parser
	GetMetrics() *ParserMetrics
}

// RulesTranslator define el puerto para traducción de AST a SecurityRules
type RulesTranslator interface {
	// Translate convierte un FirestoreRuleset a reglas del sistema
	Translate(ctx context.Context, ruleset *FirestoreRuleset) (*TranslationResult, error)

	// TranslateWithCache usa caché para optimizar traducciones repetidas
	TranslateWithCache(ctx context.Context, ruleset *FirestoreRuleset, cacheKey *CacheKey) (*TranslationResult, error)

	// ValidateTranslation verifica que la traducción sea correcta
	ValidateTranslation(ctx context.Context, original *FirestoreRuleset, translated interface{}) error

	// GetOptimizationSuggestions analiza las reglas para sugerir optimizaciones
	GetOptimizationSuggestions(ctx context.Context, ruleset *FirestoreRuleset) ([]OptimizationSuggestion, error)

	// GetMetrics retorna métricas de traducción
	GetMetrics() *TranslationMetrics
}

// RulesCache define el puerto para caché de reglas optimizado
type RulesCache interface {
	// Get obtiene reglas desde caché con TTL
	Get(ctx context.Context, key *CacheKey) (*TranslationResult, error)

	// Set guarda reglas en caché con TTL configurable
	Set(ctx context.Context, key *CacheKey, result *TranslationResult, ttl int) error

	// Invalidate invalida reglas específicas
	Invalidate(ctx context.Context, key *CacheKey) error

	// InvalidateAll limpia toda la caché
	InvalidateAll(ctx context.Context) error

	// GetStats retorna estadísticas de caché
	GetStats() *CacheStats

	// Preload precarga reglas frecuentemente usadas
	Preload(ctx context.Context, keys []*CacheKey) error
}

// RulesDeployer define el puerto para despliegue de reglas
type RulesDeployer interface {
	// Deploy despliega reglas al motor de seguridad
	Deploy(ctx context.Context, projectID, databaseID string, rules interface{}) error

	// DeployWithValidation despliega con validación previa
	DeployWithValidation(ctx context.Context, projectID, databaseID string, rules interface{}) (*DeployResult, error)

	// Rollback revierte al conjunto de reglas anterior
	Rollback(ctx context.Context, projectID, databaseID string) error

	// GetCurrentVersion obtiene la versión actual de reglas
	GetCurrentVersion(ctx context.Context, projectID, databaseID string) (string, error)

	// GetDeployHistory obtiene historial de despliegues
	GetDeployHistory(ctx context.Context, projectID, databaseID string, limit int) ([]*DeployHistory, error)
}

// RulesOptimizer define el puerto para optimización de reglas
type RulesOptimizer interface {
	// Optimize optimiza reglas para máximo rendimiento
	Optimize(ctx context.Context, rules interface{}) (interface{}, *OptimizationReport, error)

	// AnalyzePerformance analiza el impacto en rendimiento
	AnalyzePerformance(ctx context.Context, rules interface{}) (*PerformanceAnalysis, error)

	// SuggestImprovements sugiere mejoras en las reglas
	SuggestImprovements(ctx context.Context, rules interface{}) ([]OptimizationSuggestion, error)
}

// Structs de soporte para métricas y análisis

type ParserMetrics struct {
	TotalParsed      int64         `json:"total_parsed"`
	AverageParseTime time.Duration `json:"average_parse_time"`
	ErrorRate        float64       `json:"error_rate"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	LastParseTime    time.Time     `json:"last_parse_time"`
	MemoryUsage      int64         `json:"memory_usage"`
}

type CacheStats struct {
	HitRate        float64       `json:"hit_rate"`
	MissRate       float64       `json:"miss_rate"`
	TotalRequests  int64         `json:"total_requests"`
	CacheSize      int64         `json:"cache_size"`
	MemoryUsage    int64         `json:"memory_usage"`
	AverageLatency time.Duration `json:"average_latency"`
	LastAccess     time.Time     `json:"last_access"`
	EvictionCount  int64         `json:"eviction_count"`
}

type DeployResult struct {
	Success       bool          `json:"success"`
	Version       string        `json:"version"`
	DeployTime    time.Duration `json:"deploy_time"`
	RulesDeployed int           `json:"rules_deployed"`
	Errors        []string      `json:"errors,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
}

type DeployHistory struct {
	Version    string    `json:"version"`
	DeployedAt time.Time `json:"deployed_at"`
	DeployedBy string    `json:"deployed_by"`
	RulesCount int       `json:"rules_count"`
	Status     string    `json:"status"`
	RollbackOf string    `json:"rollback_of,omitempty"`
}

type OptimizationSuggestion struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"`
	Rule        string `json:"rule"`
	Line        int    `json:"line"`
}

type OptimizationReport struct {
	RulesOptimized  int                      `json:"rules_optimized"`
	PerformanceGain float64                  `json:"performance_gain"`
	MemorySaved     int64                    `json:"memory_saved"`
	Optimizations   []OptimizationSuggestion `json:"optimizations"`
	TimeSaved       time.Duration            `json:"time_saved"`
}

type PerformanceAnalysis struct {
	ExpectedLatency   time.Duration `json:"expected_latency"`
	MemoryRequirement int64         `json:"memory_requirement"`
	DatabaseCallCount int           `json:"database_call_count"`
	ComplexityScore   float64       `json:"complexity_score"`
	Bottlenecks       []string      `json:"bottlenecks"`
	Recommendations   []string      `json:"recommendations"`
}
