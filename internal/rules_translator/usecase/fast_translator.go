package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/domain"
)

// FastTranslator implementa RulesTranslator optimizado para velocidad máxima
type FastTranslator struct {
	cache        domain.RulesCache
	optimizer    domain.RulesOptimizer
	metrics      *TranslationMetrics
	metricsMutex sync.RWMutex

	// Pre-mapeo de operaciones para velocidad
	operationMap map[string]repository.OperationType

	// Pool de objetos para evitar allocations
	rulePool sync.Pool

	// Configuración de optimización
	config *TranslatorConfig
}

type TranslatorConfig struct {
	EnableCache         bool          `json:"enable_cache"`
	EnableOptimization  bool          `json:"enable_optimization"`
	MaxCacheSize        int           `json:"max_cache_size"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	EnableMetrics       bool          `json:"enable_metrics"`
	OptimizeConditions  bool          `json:"optimize_conditions"`
	EnableValidation    bool          `json:"enable_validation"`
	ParallelTranslation bool          `json:"parallel_translation"`
}

type TranslationMetrics struct {
	TotalTranslations    int64         `json:"total_translations"`
	AverageTranslateTime time.Duration `json:"average_translate_time"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	OptimizationsSaved   int64         `json:"optimizations_saved"`
	ErrorRate            float64       `json:"error_rate"`
	LastTranslation      time.Time     `json:"last_translation"`
}

// NewFastTranslator crea una nueva instancia optimizada del traductor
func NewFastTranslator(cache domain.RulesCache, optimizer domain.RulesOptimizer, config *TranslatorConfig) *FastTranslator {
	if config == nil {
		config = DefaultTranslatorConfig()
	}

	translator := &FastTranslator{
		cache:     cache,
		optimizer: optimizer,
		config:    config, metrics: &TranslationMetrics{}, // Pre-mapeo para operaciones comunes (velocidad máxima)
		operationMap: map[string]repository.OperationType{
			"read":   repository.OperationRead,
			"list":   repository.OperationList,
			"get":    repository.OperationRead, // get maps to read in Firestore
			"write":  repository.OperationWrite,
			"create": repository.OperationCreate,
			"update": repository.OperationUpdate,
			"delete": repository.OperationDelete,
		},
	}

	// Pool de SecurityRule objects para reducir GC pressure
	translator.rulePool = sync.Pool{
		New: func() interface{} {
			return &repository.SecurityRule{
				Allow: make(map[repository.OperationType]string),
				Deny:  make(map[repository.OperationType]string),
			}
		},
	}

	return translator
}

// DefaultTranslatorConfig configuración optimizada por defecto
func DefaultTranslatorConfig() *TranslatorConfig {
	return &TranslatorConfig{
		EnableCache:         true,
		EnableOptimization:  true,
		MaxCacheSize:        1000,
		CacheTTL:            time.Hour,
		EnableMetrics:       true,
		OptimizeConditions:  true,
		EnableValidation:    true,
		ParallelTranslation: true,
	}
}

// Translate convierte un FirestoreRuleset a SecurityRules optimizado
func (t *FastTranslator) Translate(ctx context.Context, ruleset *domain.FirestoreRuleset) (*domain.TranslationResult, error) {
	startTime := time.Now()

	result := &domain.TranslationResult{
		Errors: make([]string, 0),
	}

	// Traducir todos los matches de forma eficiente
	var securityRules []*repository.SecurityRule

	if t.config.ParallelTranslation && len(ruleset.Matches) > 2 {
		securityRules = t.translateParallel(ctx, ruleset.Matches)
	} else {
		securityRules = t.translateSequential(ctx, ruleset.Matches)
	}

	// Optimizar reglas si está habilitado
	if t.config.EnableOptimization && t.optimizer != nil {
		optimizedRules, report, err := t.optimizer.Optimize(ctx, securityRules)
		if err == nil && report != nil {
			securityRules = optimizedRules.([]*repository.SecurityRule)
			result.OptimizedRules = len(securityRules)
		}
	}

	result.Rules = securityRules
	result.RulesGenerated = len(securityRules)
	result.TranslationTime = time.Since(startTime)

	// Actualizar métricas
	t.updateMetrics(result.TranslationTime, len(result.Errors) == 0)

	return result, nil
}

// translateParallel traduce matches en paralelo para máxima velocidad
func (t *FastTranslator) translateParallel(ctx context.Context, matches []*domain.MatchBlock) []*repository.SecurityRule {
	resultChan := make(chan []*repository.SecurityRule, len(matches))
	var wg sync.WaitGroup

	// Procesar cada match en su propia goroutine
	for _, match := range matches {
		wg.Add(1)
		go func(m *domain.MatchBlock) {
			defer wg.Done()
			rules := t.translateMatchBlock(ctx, m, "")
			resultChan <- rules
		}(match)
	}

	// Esperar a que terminen todas las goroutines
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Recolectar resultados
	var allRules []*repository.SecurityRule
	for rules := range resultChan {
		allRules = append(allRules, rules...)
	}

	return allRules
}

// translateSequential traduce matches secuencialmente
func (t *FastTranslator) translateSequential(ctx context.Context, matches []*domain.MatchBlock) []*repository.SecurityRule {
	var allRules []*repository.SecurityRule

	for _, match := range matches {
		rules := t.translateMatchBlock(ctx, match, "")
		allRules = append(allRules, rules...)
	}

	return allRules
}

// translateMatchBlock convierte un MatchBlock a SecurityRules (recursivo optimizado)
func (t *FastTranslator) translateMatchBlock(ctx context.Context, block *domain.MatchBlock, parentPath string) []*repository.SecurityRule {
	var rules []*repository.SecurityRule

	// Construir ruta completa de forma eficiente
	fullPath := t.buildFullPath(parentPath, block.Path)

	// Crear regla principal si tiene declaraciones allow/deny
	if len(block.Allow) > 0 || len(block.Deny) > 0 {
		rule := t.getSecurityRule() // Pool object
		rule.Match = fullPath
		rule.Priority = t.calculatePriority(fullPath, block.Depth)
		rule.Description = fmt.Sprintf("Auto-generated from match %s", block.Path)

		// Procesar allow statements optimizado
		t.processAllowStatements(rule, block.Allow)

		// Procesar deny statements optimizado
		t.processDenyStatements(rule, block.Deny)

		rules = append(rules, rule)
	}

	// Procesar bloques anidados recursivamente
	for _, nested := range block.Nested {
		nestedRules := t.translateMatchBlock(ctx, nested, fullPath)
		rules = append(rules, nestedRules...)
	}

	return rules
}

// processAllowStatements procesa declaraciones allow de forma optimizada
func (t *FastTranslator) processAllowStatements(rule *repository.SecurityRule, allowStmts []*domain.AllowStatement) {
	for _, stmt := range allowStmts {
		condition := t.optimizeCondition(stmt.Condition)

		for _, opStr := range stmt.Operations {
			// Expandir operaciones compuestas para compatibilidad total con Firestore
			operations := t.expandOperation(opStr)

			for _, op := range operations {
				if mappedOp, exists := t.operationMap[op]; exists {
					rule.Allow[mappedOp] = condition
				}
			}
		}
	}
}

// processDenyStatements procesa declaraciones deny de forma optimizada
func (t *FastTranslator) processDenyStatements(rule *repository.SecurityRule, denyStmts []*domain.DenyStatement) {
	for _, stmt := range denyStmts {
		condition := t.optimizeCondition(stmt.Condition)

		for _, opStr := range stmt.Operations {
			operations := t.expandOperation(opStr)

			for _, op := range operations {
				if mappedOp, exists := t.operationMap[op]; exists {
					rule.Deny[mappedOp] = condition
				}
			}
		}
	}
}

// expandOperation expande operaciones compuestas (ej: "read" -> ["read", "list"], "write" -> ["create", "update", "delete"])
func (t *FastTranslator) expandOperation(operation string) []string {
	switch strings.TrimSpace(operation) {
	case "read":
		return []string{"read", "list"} // Firestore read maps to read and list
	case "write":
		return []string{"create", "update", "delete"} // Firestore write maps to create, update, delete
	default:
		return []string{operation} // Keep other operations as-is
	}
}

// optimizeCondition optimiza condiciones CEL para máximo rendimiento
func (t *FastTranslator) optimizeCondition(condition string) string {
	if !t.config.OptimizeConditions {
		return condition
	}

	// Optimizaciones comunes de sintaxis Firestore -> CEL
	optimized := condition

	// Mapeo de funciones comunes
	optimized = strings.ReplaceAll(optimized, "request.auth.uid", "request.auth.uid")
	optimized = strings.ReplaceAll(optimized, "resource.data.", "resource.data.")
	optimized = strings.ReplaceAll(optimized, "request.resource.data.", "request.resource.data.")

	// Optimizar llamadas get() y exists() para mejor rendimiento
	optimized = t.optimizeGetCalls(optimized)
	optimized = t.optimizeExistsCalls(optimized)

	return optimized
}

// optimizeGetCalls optimiza llamadas a get() para reducir latencia
func (t *FastTranslator) optimizeGetCalls(condition string) string {
	// TODO: Implementar optimizaciones específicas de get()
	// Por ejemplo, combinar múltiples get() del mismo documento
	return condition
}

// optimizeExistsCalls optimiza llamadas a exists() para reducir latencia
func (t *FastTranslator) optimizeExistsCalls(condition string) string {
	// TODO: Implementar optimizaciones específicas de exists()
	// Por ejemplo, usar caché para exists() frecuentes
	return condition
}

// buildFullPath construye la ruta completa de forma eficiente
func (t *FastTranslator) buildFullPath(parentPath, currentPath string) string {
	if parentPath == "" {
		return currentPath
	}

	// Normalizar rutas para consistencia
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	return strings.TrimSuffix(parentPath, "/") + currentPath
}

// calculatePriority calcula prioridad basada en especificidad (como Firestore)
func (t *FastTranslator) calculatePriority(path string, depth int) int {
	// Base priority - more specific paths have higher priority
	priority := 1000 // Start with base priority to avoid negatives

	// Add priority for greater depth (more specific)
	priority += depth * 100

	// Count path segments for specificity
	segments := strings.Split(strings.Trim(path, "/"), "/")

	// Add points for specific (non-variable) segments
	specificSegments := 0
	variableSegments := 0
	wildcardSegments := 0

	for _, segment := range segments {
		if strings.Contains(segment, "**") {
			wildcardSegments++
		} else if strings.Contains(segment, "{") {
			variableSegments++
		} else {
			specificSegments++
		}
	}

	// Higher priority for more specific segments
	priority += specificSegments * 50

	// Lower priority for variable segments
	priority -= variableSegments * 10

	// Lowest priority for wildcard segments
	priority -= wildcardSegments * 100

	// Ensure priority is never negative
	if priority < 0 {
		priority = 1
	}

	return priority
}

// TranslateWithCache implementa traducción con caché optimizado
func (t *FastTranslator) TranslateWithCache(ctx context.Context, ruleset *domain.FirestoreRuleset, cacheKey *domain.CacheKey) (*domain.TranslationResult, error) {
	if !t.config.EnableCache || t.cache == nil {
		return t.Translate(ctx, ruleset)
	}

	// Intentar obtener de caché primero
	if cached, err := t.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		t.updateCacheHitMetrics(true)
		return cached, nil
	}

	t.updateCacheHitMetrics(false)

	// No está en caché, traducir y guardar
	result, err := t.Translate(ctx, ruleset)
	if err != nil {
		return nil, err
	}

	// Guardar en caché de forma asíncrona para no bloquear
	go func() {
		if cacheErr := t.cache.Set(ctx, cacheKey, result, int(t.config.CacheTTL.Seconds())); cacheErr != nil {
			// Log error but don't fail the operation
		}
	}()

	return result, nil
}

// ValidateTranslation valida que la traducción sea correcta
func (t *FastTranslator) ValidateTranslation(ctx context.Context, original *domain.FirestoreRuleset, translated interface{}) error {
	if !t.config.EnableValidation {
		return nil
	}

	// TODO: Implementar validación comprehensiva
	// - Verificar que todas las reglas fueron traducidas
	// - Verificar que las condiciones son válidas CEL
	// - Verificar que las operaciones están mapeadas correctamente

	return nil
}

// GetOptimizationSuggestions analiza reglas para sugerir optimizaciones
func (t *FastTranslator) GetOptimizationSuggestions(ctx context.Context, ruleset *domain.FirestoreRuleset) ([]domain.OptimizationSuggestion, error) {
	var suggestions []domain.OptimizationSuggestion

	for _, match := range ruleset.Matches {
		suggestions = append(suggestions, t.analyzeMatchBlock(match)...)
	}

	return suggestions, nil
}

// analyzeMatchBlock analiza un bloque de match para optimizaciones
func (t *FastTranslator) analyzeMatchBlock(block *domain.MatchBlock) []domain.OptimizationSuggestion {
	var suggestions []domain.OptimizationSuggestion

	// Detectar reglas redundantes
	if len(block.Allow) > 3 {
		suggestions = append(suggestions, domain.OptimizationSuggestion{
			Type:        "redundancy",
			Description: "Consider consolidating multiple allow statements",
			Impact:      "medium",
			Effort:      "low",
			Rule:        block.Path,
		})
	}

	// Detectar condiciones complejas que podrían ser optimizadas
	for _, allow := range block.Allow {
		if strings.Count(allow.Condition, "get(") > 2 {
			suggestions = append(suggestions, domain.OptimizationSuggestion{
				Type:        "performance",
				Description: "Multiple get() calls detected - consider caching or restructuring",
				Impact:      "high",
				Effort:      "medium",
				Rule:        block.Path,
				Line:        allow.Line,
			})
		}
	}

	return suggestions
}

// Helper methods

func (t *FastTranslator) getSecurityRule() *repository.SecurityRule {
	rule := t.rulePool.Get().(*repository.SecurityRule)

	// Reset para reutilización
	rule.Match = ""
	rule.Priority = 0
	rule.Description = ""

	// Limpiar maps pero mantener capacidad
	for k := range rule.Allow {
		delete(rule.Allow, k)
	}
	for k := range rule.Deny {
		delete(rule.Deny, k)
	}

	return rule
}

func (t *FastTranslator) returnSecurityRule(rule *repository.SecurityRule) {
	t.rulePool.Put(rule)
}

func (t *FastTranslator) updateMetrics(translationTime time.Duration, success bool) {
	if !t.config.EnableMetrics {
		return
	}

	t.metricsMutex.Lock()
	defer t.metricsMutex.Unlock()

	t.metrics.TotalTranslations++
	t.metrics.LastTranslation = time.Now()

	// Moving average para tiempo de traducción
	if t.metrics.AverageTranslateTime == 0 {
		t.metrics.AverageTranslateTime = translationTime
	} else {
		t.metrics.AverageTranslateTime = time.Duration(
			float64(t.metrics.AverageTranslateTime)*0.9 + float64(translationTime)*0.1,
		)
	}

	if !success {
		errorRate := float64(1) / float64(t.metrics.TotalTranslations)
		if t.metrics.ErrorRate == 0 {
			t.metrics.ErrorRate = errorRate
		} else {
			t.metrics.ErrorRate = t.metrics.ErrorRate*0.9 + errorRate*0.1
		}
	}
}

func (t *FastTranslator) updateCacheHitMetrics(hit bool) {
	if !t.config.EnableMetrics {
		return
	}

	t.metricsMutex.Lock()
	defer t.metricsMutex.Unlock()

	if hit {
		t.metrics.CacheHitRate = t.metrics.CacheHitRate*0.9 + 0.1
	} else {
		t.metrics.CacheHitRate = t.metrics.CacheHitRate * 0.9
	}
}

// GetMetrics retorna métricas de traducción en el tipo del dominio
func (t *FastTranslator) GetMetrics() *domain.TranslationMetrics {
	t.metricsMutex.RLock()
	defer t.metricsMutex.RUnlock()

	metrics := *t.metrics
	return (*domain.TranslationMetrics)(&metrics)
}
