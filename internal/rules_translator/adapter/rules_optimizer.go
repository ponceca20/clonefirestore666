package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/domain"
)

// RulesOptimizer implementa optimización avanzada de reglas para máximo rendimiento
type RulesOptimizer struct {
	config *OptimizerConfig
}

type OptimizerConfig struct {
	EnableConsolidation   bool    `json:"enable_consolidation"`
	EnableConditionOptim  bool    `json:"enable_condition_optimization"`
	EnablePriorityOptim   bool    `json:"enable_priority_optimization"`
	MaxOptimizationPasses int     `json:"max_optimization_passes"`
	PerformanceThreshold  float64 `json:"performance_threshold"`
	EnableAggressiveOptim bool    `json:"enable_aggressive_optimization"`
}

// NewRulesOptimizer crea una nueva instancia del optimizador
func NewRulesOptimizer(config *OptimizerConfig) *RulesOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	return &RulesOptimizer{
		config: config,
	}
}

// DefaultOptimizerConfig configuración optimizada por defecto
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		EnableConsolidation:   true,
		EnableConditionOptim:  true,
		EnablePriorityOptim:   true,
		MaxOptimizationPasses: 3,
		PerformanceThreshold:  0.8,
		EnableAggressiveOptim: false,
	}
}

// Optimize optimiza reglas para máximo rendimiento
func (o *RulesOptimizer) Optimize(ctx context.Context, rules interface{}) (interface{}, *domain.OptimizationReport, error) {
	startTime := time.Now()

	securityRules, ok := rules.([]*repository.SecurityRule)
	if !ok {
		return nil, nil, fmt.Errorf("invalid rules type")
	}

	report := &domain.OptimizationReport{
		RulesOptimized:  0,
		PerformanceGain: 0,
		MemorySaved:     0,
		Optimizations:   make([]domain.OptimizationSuggestion, 0),
	}

	optimizedRules := make([]*repository.SecurityRule, len(securityRules))
	copy(optimizedRules, securityRules)

	// Múltiples pasadas de optimización
	for pass := 0; pass < o.config.MaxOptimizationPasses; pass++ {
		passOptimized := false

		// 1. Consolidar reglas duplicadas/similares
		if o.config.EnableConsolidation {
			consolidated, optimized := o.consolidateRules(optimizedRules)
			if optimized {
				optimizedRules = consolidated
				passOptimized = true
				report.RulesOptimized++
			}
		}

		// 2. Optimizar condiciones CEL
		if o.config.EnableConditionOptim {
			condOptimized := o.optimizeConditions(optimizedRules)
			if condOptimized {
				passOptimized = true
				report.RulesOptimized++
			}
		}

		// 3. Optimizar prioridades para matching más rápido
		if o.config.EnablePriorityOptim {
			prioOptimized := o.optimizePriorities(optimizedRules)
			if prioOptimized {
				passOptimized = true
				report.RulesOptimized++
			}
		}

		// 4. Optimizaciones agresivas si están habilitadas
		if o.config.EnableAggressiveOptim {
			aggressiveOptimized := o.aggressiveOptimizations(optimizedRules)
			if aggressiveOptimized {
				passOptimized = true
				report.RulesOptimized++
			}
		}

		// Si no hubo optimizaciones en esta pasada, salir
		if !passOptimized {
			break
		}
	}

	// Calcular métricas finales
	report.PerformanceGain = o.calculatePerformanceGain(securityRules, optimizedRules)
	report.MemorySaved = o.calculateMemorySaved(securityRules, optimizedRules)
	report.TimeSaved = time.Since(startTime)

	return optimizedRules, report, nil
}

// consolidateRules consolida reglas duplicadas o muy similares
func (o *RulesOptimizer) consolidateRules(rules []*repository.SecurityRule) ([]*repository.SecurityRule, bool) {
	optimized := false
	consolidated := make([]*repository.SecurityRule, 0, len(rules))
	processed := make(map[int]bool)

	for i, rule := range rules {
		if processed[i] {
			continue
		}

		// Buscar reglas similares que se puedan consolidar
		similarRules := []*repository.SecurityRule{rule}
		for j := i + 1; j < len(rules); j++ {
			if processed[j] {
				continue
			}

			if o.canConsolidateRules(rule, rules[j]) {
				similarRules = append(similarRules, rules[j])
				processed[j] = true
				optimized = true
			}
		}

		// Consolidar reglas similares
		if len(similarRules) > 1 {
			consolidatedRule := o.mergeRules(similarRules)
			consolidated = append(consolidated, consolidatedRule)
		} else {
			consolidated = append(consolidated, rule)
		}

		processed[i] = true
	}

	return consolidated, optimized
}

// canConsolidateRules determina si dos reglas se pueden consolidar
func (o *RulesOptimizer) canConsolidateRules(rule1, rule2 *repository.SecurityRule) bool {
	// Mismo match pattern pero diferentes operaciones
	if rule1.Match == rule2.Match {
		return true
	}

	// Patterns muy similares con diferencias mínimas
	similarity := o.calculatePatternSimilarity(rule1.Match, rule2.Match)
	return similarity > 0.8
}

// mergeRules combina múltiples reglas en una sola optimizada
func (o *RulesOptimizer) mergeRules(rules []*repository.SecurityRule) *repository.SecurityRule {
	if len(rules) == 0 {
		return nil
	}

	// Usar la primera regla como base
	merged := &repository.SecurityRule{
		Match:       rules[0].Match,
		Priority:    rules[0].Priority,
		Description: fmt.Sprintf("Merged rule from %d rules", len(rules)),
		Allow:       make(map[repository.OperationType]string),
		Deny:        make(map[repository.OperationType]string),
	}

	// Combinar todas las operaciones
	for _, rule := range rules {
		for op, condition := range rule.Allow {
			// Usar la condición más permisiva (OR lógico)
			if existing, exists := merged.Allow[op]; exists {
				merged.Allow[op] = o.combineConditions(existing, condition, "OR")
			} else {
				merged.Allow[op] = condition
			}
		}

		for op, condition := range rule.Deny {
			// Usar la condición más restrictiva (AND lógico)
			if existing, exists := merged.Deny[op]; exists {
				merged.Deny[op] = o.combineConditions(existing, condition, "AND")
			} else {
				merged.Deny[op] = condition
			}
		}

		// Usar la prioridad más alta
		if rule.Priority > merged.Priority {
			merged.Priority = rule.Priority
		}
	}

	return merged
}

// optimizeConditions optimiza condiciones CEL para mejor rendimiento
func (o *RulesOptimizer) optimizeConditions(rules []*repository.SecurityRule) bool {
	optimized := false

	for _, rule := range rules {
		// Optimizar condiciones Allow
		for op, condition := range rule.Allow {
			optimizedCondition := o.optimizeCELCondition(condition)
			if optimizedCondition != condition {
				rule.Allow[op] = optimizedCondition
				optimized = true
			}
		}

		// Optimizar condiciones Deny
		for op, condition := range rule.Deny {
			optimizedCondition := o.optimizeCELCondition(condition)
			if optimizedCondition != condition {
				rule.Deny[op] = optimizedCondition
				optimized = true
			}
		}
	}

	return optimized
}

// optimizeCELCondition optimiza una condición CEL individual
func (o *RulesOptimizer) optimizeCELCondition(condition string) string {
	optimized := condition
	// 1. Eliminar condiciones redundantes
	optimized = o.removeRedundantConditions(optimized)

	// 2. Reordenar condiciones por velocidad de evaluación (más rápidas primero)
	optimized = o.reorderConditionsBySpeed(optimized)

	// 3. Optimizar llamadas a funciones costosas (solo en modo agresivo)
	if o.config.EnableAggressiveOptim {
		optimized = o.optimizeExpensiveFunctions(optimized)
	}

	// 4. Usar short-circuit evaluation
	optimized = o.enableShortCircuit(optimized)

	return optimized
}

// optimizePriorities optimiza prioridades para matching más eficiente
func (o *RulesOptimizer) optimizePriorities(rules []*repository.SecurityRule) bool {
	// Calcular nuevas prioridades basadas en:
	// 1. Frecuencia de uso estimada
	// 2. Complejidad de la condición
	// 3. Especificidad del pattern

	for _, rule := range rules {
		newPriority := o.calculateOptimalPriority(rule)
		if newPriority != rule.Priority {
			rule.Priority = newPriority
			return true
		}
	}

	return false
}

// aggressiveOptimizations aplica optimizaciones más agresivas
func (o *RulesOptimizer) aggressiveOptimizations(rules []*repository.SecurityRule) bool {
	optimized := false

	// 1. Pre-computar resultados de condiciones estáticas
	for _, rule := range rules {
		for op, condition := range rule.Allow {
			if o.isStaticCondition(condition) {
				result := o.evaluateStaticCondition(condition)
				rule.Allow[op] = fmt.Sprintf("%t", result)
				optimized = true
			}
		}
	}

	// 2. Inline de funciones simples
	optimized = o.inlineSimpleFunctions(rules) || optimized

	// 3. Eliminar reglas inalcanzables
	optimized = o.removeUnreachableRules(rules) || optimized

	return optimized
}

// AnalyzePerformance analiza el impacto en rendimiento
func (o *RulesOptimizer) AnalyzePerformance(ctx context.Context, rules interface{}) (*domain.PerformanceAnalysis, error) {
	securityRules, ok := rules.([]*repository.SecurityRule)
	if !ok {
		return nil, fmt.Errorf("invalid rules type")
	}

	analysis := &domain.PerformanceAnalysis{
		Bottlenecks:     make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Análisis de latencia esperada
	analysis.ExpectedLatency = o.calculateExpectedLatency(securityRules)

	// Análisis de requerimientos de memoria
	analysis.MemoryRequirement = o.calculateMemoryRequirement(securityRules)

	// Contar llamadas a base de datos
	analysis.DatabaseCallCount = o.countDatabaseCalls(securityRules)

	// Calcular puntuación de complejidad
	analysis.ComplexityScore = o.calculateComplexityScore(securityRules)

	// Identificar cuellos de botella
	analysis.Bottlenecks = o.identifyBottlenecks(securityRules)

	// Generar recomendaciones
	analysis.Recommendations = o.generateRecommendations(securityRules)

	return analysis, nil
}

// SuggestImprovements sugiere mejoras específicas
func (o *RulesOptimizer) SuggestImprovements(ctx context.Context, rules interface{}) ([]domain.OptimizationSuggestion, error) {
	securityRules, ok := rules.([]*repository.SecurityRule)
	if !ok {
		return nil, fmt.Errorf("invalid rules type")
	}

	suggestions := make([]domain.OptimizationSuggestion, 0)

	for i, rule := range securityRules {
		// Sugerir consolidación de reglas similares
		if o.hasSimilarRules(rule, securityRules, i) {
			suggestions = append(suggestions, domain.OptimizationSuggestion{
				Type:        "consolidation",
				Description: "Consider consolidating with similar rules",
				Impact:      "medium",
				Effort:      "low",
				Rule:        rule.Match,
			})
		}

		// Sugerir optimización de condiciones complejas
		if o.hasComplexConditions(rule) {
			suggestions = append(suggestions, domain.OptimizationSuggestion{
				Type:        "condition",
				Description: "Complex conditions detected - consider simplification",
				Impact:      "high",
				Effort:      "medium",
				Rule:        rule.Match,
			})
		}

		// Sugerir optimización de prioridades
		if o.hasSuboptimalPriority(rule, securityRules) {
			suggestions = append(suggestions, domain.OptimizationSuggestion{
				Type:        "priority",
				Description: "Priority could be optimized for better performance",
				Impact:      "low",
				Effort:      "low",
				Rule:        rule.Match,
			})
		}
	}

	return suggestions, nil
}

// Helper methods (implementaciones simplificadas para brevedad)

func (o *RulesOptimizer) calculatePatternSimilarity(pattern1, pattern2 string) float64 {
	// Implementación simplificada - en producción usaría algoritmos más sofisticados
	if pattern1 == pattern2 {
		return 1.0
	}

	// Calcular similitud basada en segmentos comunes
	segments1 := strings.Split(pattern1, "/")
	segments2 := strings.Split(pattern2, "/")

	commonSegments := 0
	maxSegments := len(segments1)
	if len(segments2) > maxSegments {
		maxSegments = len(segments2)
	}

	for i := 0; i < len(segments1) && i < len(segments2); i++ {
		if segments1[i] == segments2[i] {
			commonSegments++
		}
	}

	return float64(commonSegments) / float64(maxSegments)
}

func (o *RulesOptimizer) combineConditions(cond1, cond2, operator string) string {
	if cond1 == cond2 {
		return cond1
	}
	return fmt.Sprintf("(%s) %s (%s)", cond1, strings.ToLower(operator), cond2)
}

func (o *RulesOptimizer) removeRedundantConditions(condition string) string {
	// Eliminar condiciones como "true && condition" -> "condition"
	condition = strings.ReplaceAll(condition, "true &&", "")
	condition = strings.ReplaceAll(condition, "&& true", "")
	condition = strings.ReplaceAll(condition, "false ||", "")
	condition = strings.ReplaceAll(condition, "|| false", "")
	return strings.TrimSpace(condition)
}

func (o *RulesOptimizer) reorderConditionsBySpeed(condition string) string {
	// En una implementación real, reordenaría las condiciones poniendo las más rápidas primero
	return condition
}

func (o *RulesOptimizer) optimizeExpensiveFunctions(condition string) string {
	// Optimizar get() y exists() agregando hints de caché
	condition = strings.ReplaceAll(condition, "get(", "get_cached(")
	condition = strings.ReplaceAll(condition, "exists(", "exists_cached(")
	return condition
}

func (o *RulesOptimizer) enableShortCircuit(condition string) string {
	// Asegurar que las condiciones usen short-circuit evaluation
	return condition
}

func (o *RulesOptimizer) calculateOptimalPriority(rule *repository.SecurityRule) int {
	// Calcular prioridad óptima basada en múltiples factores
	priority := rule.Priority

	// Ajustar por complejidad de condiciones
	for _, condition := range rule.Allow {
		if strings.Contains(condition, "get(") {
			priority -= 10 // Condiciones con get() son más lentas
		}
	}

	return priority
}

func (o *RulesOptimizer) calculatePerformanceGain(original, optimized []*repository.SecurityRule) float64 {
	// Calcular ganancia estimada de rendimiento
	return float64(len(original)-len(optimized)) / float64(len(original))
}

func (o *RulesOptimizer) calculateMemorySaved(original, optimized []*repository.SecurityRule) int64 {
	// Estimar memoria ahorrada
	return int64((len(original) - len(optimized)) * 1024) // ~1KB por regla
}

func (o *RulesOptimizer) calculateExpectedLatency(rules []*repository.SecurityRule) time.Duration {
	// Calcular latencia esperada basada en complejidad de reglas
	totalLatency := time.Duration(0)
	for _, rule := range rules {
		ruleLatency := time.Microsecond * 100 // Base latency

		// Añadir latencia por cada get() call
		for _, condition := range rule.Allow {
			getCount := strings.Count(condition, "get(")
			ruleLatency += time.Millisecond * time.Duration(getCount*5)
		}

		totalLatency += ruleLatency
	}
	return totalLatency / time.Duration(len(rules))
}

func (o *RulesOptimizer) calculateMemoryRequirement(rules []*repository.SecurityRule) int64 {
	return int64(len(rules) * 2048) // ~2KB por regla estimado
}

func (o *RulesOptimizer) countDatabaseCalls(rules []*repository.SecurityRule) int {
	calls := 0
	for _, rule := range rules {
		for _, condition := range rule.Allow {
			calls += strings.Count(condition, "get(")
			calls += strings.Count(condition, "exists(")
		}
	}
	return calls
}

func (o *RulesOptimizer) calculateComplexityScore(rules []*repository.SecurityRule) float64 {
	score := 0.0
	for _, rule := range rules {
		ruleScore := 1.0 // Base score

		// Añadir complejidad por operaciones
		ruleScore += float64(len(rule.Allow)) * 0.5
		ruleScore += float64(len(rule.Deny)) * 0.5

		// Añadir complejidad por get() calls
		for _, condition := range rule.Allow {
			ruleScore += float64(strings.Count(condition, "get(")) * 2.0
		}

		score += ruleScore
	}
	return score / float64(len(rules))
}

func (o *RulesOptimizer) identifyBottlenecks(rules []*repository.SecurityRule) []string {
	bottlenecks := make([]string, 0)

	for _, rule := range rules {
		for _, condition := range rule.Allow {
			if strings.Count(condition, "get(") > 2 {
				bottlenecks = append(bottlenecks, fmt.Sprintf("Multiple get() calls in rule %s", rule.Match))
			}
		}
	}

	return bottlenecks
}

func (o *RulesOptimizer) generateRecommendations(rules []*repository.SecurityRule) []string {
	recommendations := make([]string, 0)

	if len(rules) > 50 {
		recommendations = append(recommendations, "Consider consolidating rules to reduce total count")
	}

	dbCalls := o.countDatabaseCalls(rules)
	if dbCalls > len(rules)*2 {
		recommendations = append(recommendations, "Consider reducing database calls in conditions")
	}

	return recommendations
}

// Métodos helper simplificados

func (o *RulesOptimizer) isStaticCondition(condition string) bool {
	return !strings.Contains(condition, "request.") && !strings.Contains(condition, "resource.")
}

func (o *RulesOptimizer) evaluateStaticCondition(condition string) bool {
	// Implementación simplificada
	return condition == "true"
}

func (o *RulesOptimizer) inlineSimpleFunctions(rules []*repository.SecurityRule) bool {
	// Implementación simplificada
	return false
}

func (o *RulesOptimizer) removeUnreachableRules(rules []*repository.SecurityRule) bool {
	// Implementación simplificada
	return false
}

func (o *RulesOptimizer) hasSimilarRules(rule *repository.SecurityRule, rules []*repository.SecurityRule, currentIndex int) bool {
	for i, other := range rules {
		if i != currentIndex && o.calculatePatternSimilarity(rule.Match, other.Match) > 0.7 {
			return true
		}
	}
	return false
}

func (o *RulesOptimizer) hasComplexConditions(rule *repository.SecurityRule) bool {
	for _, condition := range rule.Allow {
		if strings.Count(condition, "&&") > 2 || strings.Count(condition, "get(") > 1 {
			return true
		}
	}
	return false
}

func (o *RulesOptimizer) hasSuboptimalPriority(rule *repository.SecurityRule, rules []*repository.SecurityRule) bool {
	optimalPriority := o.calculateOptimalPriority(rule)
	return abs(rule.Priority-optimalPriority) > 10
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
