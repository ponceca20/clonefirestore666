package usecase

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/repository"
	rtdomain "firestore-clone/internal/rules_translator/domain"
)

// SecurityRulesTranslatorOrchestrator integra el módulo rules_translator con el motor de seguridad principal
// para garantizar compatibilidad total con Firestore.
type SecurityRulesTranslatorOrchestrator struct {
	parser     rtdomain.RulesParser
	translator rtdomain.RulesTranslator
	optimizer  rtdomain.RulesOptimizer // opcional
	engine     repository.SecurityRulesEngine
}

// NewSecurityRulesTranslatorOrchestrator crea una nueva instancia del orquestador
func NewSecurityRulesTranslatorOrchestrator(
	parser rtdomain.RulesParser,
	translator rtdomain.RulesTranslator,
	optimizer rtdomain.RulesOptimizer,
	engine repository.SecurityRulesEngine,
) *SecurityRulesTranslatorOrchestrator {
	return &SecurityRulesTranslatorOrchestrator{
		parser:     parser,
		translator: translator,
		optimizer:  optimizer,
		engine:     engine,
	}
}

// ImportAndDeployFirestoreRules importa, valida, traduce y despliega reglas Firestore
func (o *SecurityRulesTranslatorOrchestrator) ImportAndDeployFirestoreRules(ctx context.Context, rulesContent, projectID, databaseID string) error {
	// 1. Parsear reglas Firestore
	parseResult, err := o.parser.ParseString(ctx, rulesContent)
	if err != nil {
		return fmt.Errorf("error al parsear reglas Firestore: %w", err)
	}

	// 2. Traducir AST a reglas internas
	translationResult, err := o.translator.Translate(ctx, parseResult.Ruleset)
	if err != nil {
		return fmt.Errorf("error al traducir reglas Firestore: %w", err)
	}

	// 3. (Opcional) Optimizar reglas
	finalRules := translationResult.Rules // Corrección: usar el campo correcto
	if o.optimizer != nil {
		optimized, _, err := o.optimizer.Optimize(ctx, finalRules) // Corrección: capturar los 3 valores
		if err == nil && optimized != nil {
			finalRules = optimized
		}
	}

	// 4. Validar reglas con el motor de seguridad
	rulesSlice, ok := finalRules.([]*repository.SecurityRule)
	if !ok {
		return fmt.Errorf("el resultado de la traducción no es []*repository.SecurityRule, sino %T", finalRules)
	}
	if err := o.engine.ValidateRules(rulesSlice); err != nil {
		return fmt.Errorf("reglas traducidas no válidas: %w", err)
	}

	// 5. Desplegar reglas en el motor de seguridad
	if err := o.engine.SaveRules(ctx, projectID, databaseID, rulesSlice); err != nil {
		return fmt.Errorf("error al guardar reglas en el motor de seguridad: %w", err)
	}

	// 6. Limpiar caché para aplicar reglas nuevas
	o.engine.ClearCache(projectID, databaseID)

	return nil
}
