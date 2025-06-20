package domain

import (
	"time"
)

// FirestoreRuleset representa el contenido completo de un archivo firestore.rules
// optimizado para velocidad de procesamiento
type FirestoreRuleset struct {
	Service   string        `json:"service"`
	Matches   []*MatchBlock `json:"matches"`
	CreatedAt time.Time     `json:"created_at"`
	Version   string        `json:"version"`
}

// MatchBlock representa un bloque "match /path/{wildcard} { ... }"
// Optimizado con índices para búsqueda rápida
type MatchBlock struct {
	Path         string            `json:"path"`
	PathSegments []string          `json:"path_segments"` // Pre-procesado para matching rápido
	Variables    map[string]string `json:"variables"`     // Variables extraídas como {userId}
	Allow        []*AllowStatement `json:"allow"`
	Deny         []*DenyStatement  `json:"deny,omitempty"`
	Nested       []*MatchBlock     `json:"nested,omitempty"`
	ParentPath   string            `json:"parent_path,omitempty"`
	FullPath     string            `json:"full_path"` // Ruta completa pre-calculada
	Depth        int               `json:"depth"`     // Profundidad para prioridad
	Priority     string            `json:"priority"`  // Prioridad como string para comparación exacta
}

// AllowStatement representa una línea "allow operation: if condition;"
type AllowStatement struct {
	Operations []string `json:"operations"` // ["read", "write", "update", etc.]
	Condition  string   `json:"condition"`  // La condición "if" como string
	Line       int      `json:"line"`       // Línea del archivo para debugging
}

// DenyStatement representa una línea "deny operation: if condition;"
type DenyStatement struct {
	Operations []string `json:"operations"`
	Condition  string   `json:"condition"`
	Line       int      `json:"line"`
}

// ParseResult encapsula el resultado del parsing con metadatos de rendimiento
type ParseResult struct {
	Ruleset   *FirestoreRuleset `json:"ruleset"`
	ParseTime time.Duration     `json:"parse_time"`
	Errors    []ParseError      `json:"errors,omitempty"`
	Warnings  []ParseWarning    `json:"warnings,omitempty"`
	LineCount int               `json:"line_count"`
	RuleCount int               `json:"rule_count"`
}

// ParseError representa un error durante el parsing
type ParseError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ParseWarning representa una advertencia durante el parsing
type ParseWarning struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// TranslationResult encapsula el resultado de la traducción con métricas
type TranslationResult struct {
	Rules           interface{}   `json:"rules"` // []repository.SecurityRule
	TranslationTime time.Duration `json:"translation_time"`
	RulesGenerated  int           `json:"rules_generated"`
	OptimizedRules  int           `json:"optimized_rules"`
	Errors          []string      `json:"errors,omitempty"`
}

// CacheKey representa una clave optimizada para caché de reglas
type CacheKey struct {
	ProjectID  string `json:"project_id"`
	DatabaseID string `json:"database_id"`
	Version    string `json:"version"`
	Hash       string `json:"hash"` // Hash del contenido para invalidación
}

// Performance metrics para monitoreo
type PerformanceMetrics struct {
	ParseDuration      time.Duration `json:"parse_duration"`
	TranslateDuration  time.Duration `json:"translate_duration"`
	TotalDuration      time.Duration `json:"total_duration"`
	MemoryUsed         int64         `json:"memory_used"`
	RulesProcessed     int           `json:"rules_processed"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	OptimizationsSaved int           `json:"optimizations_saved"`
}

// TranslationMetrics contiene métricas de performance del traductor
// Debe ser idéntica a la usada en usecase/fast_translator.go
// Esto permite exponer métricas a través de la interfaz del dominio
type TranslationMetrics struct {
	TotalTranslations    int64         `json:"total_translations"`
	AverageTranslateTime time.Duration `json:"average_translate_time"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	OptimizationsSaved   int64         `json:"optimizations_saved"`
	ErrorRate            float64       `json:"error_rate"`
	LastTranslation      time.Time     `json:"last_translation"`
}
