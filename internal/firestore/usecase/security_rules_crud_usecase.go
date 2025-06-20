package usecase

import (
	"context"
	"firestore-clone/internal/firestore/domain/repository"
)

// SecurityRulesCRUDUsecase define el CRUD de reglas de seguridad al estilo Firestore
// Orquesta parser, traductor y motor de reglas

type SecurityRulesCRUDUsecase interface {
	GetRules(ctx context.Context, projectID, databaseID string) (string, error)
	PutRules(ctx context.Context, projectID, databaseID, rulesText, user string) error
	PatchRules(ctx context.Context, projectID, databaseID, partialText, user string) error
	DeleteRules(ctx context.Context, projectID, databaseID string) error
	ValidateRules(ctx context.Context, rulesText string) error
}

// SecurityRulesOrchestrator define el contrato para el orquestador de reglas
// Debe tener métodos exportados para que los mocks funcionen correctamente en los tests
type SecurityRulesOrchestrator interface {
	ImportAndDeployFirestoreRules(ctx context.Context, rulesContent, projectID, databaseID string) error
	Parser() interface {
		ParseString(ctx context.Context, rulesText string) (interface{}, error)
	}
}

type securityRulesCRUDUsecase struct {
	orchestrator SecurityRulesOrchestrator
	engine       repository.SecurityRulesEngine
}

func NewSecurityRulesCRUDUsecase(
	orchestrator SecurityRulesOrchestrator,
	engine repository.SecurityRulesEngine,
) SecurityRulesCRUDUsecase {
	return &securityRulesCRUDUsecase{orchestrator, engine}
}

func (uc *securityRulesCRUDUsecase) GetRules(ctx context.Context, projectID, databaseID string) (string, error) {
	// Cargar reglas actuales desde el motor de reglas (debe devolver el texto original .rules)
	return uc.engine.GetRawRules(ctx, projectID, databaseID)
}

func (uc *securityRulesCRUDUsecase) PutRules(ctx context.Context, projectID, databaseID, rulesText, user string) error {
	return uc.orchestrator.ImportAndDeployFirestoreRules(ctx, rulesText, projectID, databaseID)
}

func (uc *securityRulesCRUDUsecase) PatchRules(ctx context.Context, projectID, databaseID, partialText, user string) error {
	return nil // PATCH opcional, implementar si se requiere
}

func (uc *securityRulesCRUDUsecase) DeleteRules(ctx context.Context, projectID, databaseID string) error {
	return uc.engine.DeleteRules(ctx, projectID, databaseID)
}

func (uc *securityRulesCRUDUsecase) ValidateRules(ctx context.Context, rulesText string) error {
	_, err := uc.orchestrator.Parser().ParseString(ctx, rulesText)
	return err
}
