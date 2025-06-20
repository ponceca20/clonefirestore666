package usecase_test

import (
	"context"
	repository "firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/firestore/usecase"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityRulesCRUDUsecase_ValidateRules(t *testing.T) {
	ctx := context.Background()
	// Mocks: orquestador y engine deben ser inicializados con implementaciones dummy o mocks
	orchestrator := &MockOrchestrator{}
	engine := &MockRulesEngine{}
	uc := usecase.NewSecurityRulesCRUDUsecase(orchestrator, engine)

	err := uc.ValidateRules(ctx, `rules_version = '2'; service cloud.firestore { match /databases/{database}/documents { match /users/{userId} { allow read: if true; } } }`)
	require.NoError(t, err)
}

// MockOrchestrator simula el orquestador
// MockRulesEngine simula el motor de reglas

type MockOrchestrator struct{}

func (m *MockOrchestrator) ImportAndDeployFirestoreRules(ctx context.Context, rulesContent, projectID, databaseID string) error {
	return nil
}

// Implementa el método parser() correctamente según la interfaz
func (m *MockOrchestrator) Parser() interface {
	ParseString(ctx context.Context, rulesText string) (interface{}, error)
} {
	return &mockParser{}
}

type mockParser struct{}

func (p *mockParser) ParseString(ctx context.Context, rulesText string) (interface{}, error) {
	return nil, nil
}

var _ = &MockOrchestrator{} // Ensure interface compliance

type MockRulesEngine struct{}

func (m *MockRulesEngine) GetRawRules(ctx context.Context, projectID, databaseID string) (string, error) {
	return "rules_version = '2';", nil
}
func (m *MockRulesEngine) DeleteRules(ctx context.Context, projectID, databaseID string) error {
	return nil
}
func (m *MockRulesEngine) ClearCache(projectID, databaseID string) {}
func (m *MockRulesEngine) EvaluateAccess(ctx context.Context, op repository.OperationType, secCtx *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	return &repository.RuleEvaluationResult{Allowed: true}, nil
}
func (m *MockRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return nil, nil
}
func (m *MockRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	return nil
}
func (m *MockRulesEngine) SetResourceAccessor(accessor repository.ResourceAccessor) {}
func (m *MockRulesEngine) ValidateRules(rules []*repository.SecurityRule) error     { return nil }
