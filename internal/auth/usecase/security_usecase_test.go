package usecase_test

import (
	"context"
	"errors"
	"testing"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	firestoreusecase "firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockSecurityRulesEngine mocks the SecurityRulesEngine interface
// Only EvaluateAccess is needed for these tests

type MockSecurityRulesEngine struct {
	mock.Mock
}

func (m *MockSecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	args := m.Called(ctx, operation, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.RuleEvaluationResult), args.Error(1)
}
func (m *MockSecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return nil, nil
}
func (m *MockSecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	return nil
}
func (m *MockSecurityRulesEngine) ValidateRules(rules []*repository.SecurityRule) error {
	return nil
}
func (m *MockSecurityRulesEngine) ClearCache(projectID, databaseID string) {
	// Mock implementation - no-op
}
func (m *MockSecurityRulesEngine) SetResourceAccessor(accessor repository.ResourceAccessor) {
	// Mock implementation - no-op
}
func (m *MockSecurityRulesEngine) DeleteRules(ctx context.Context, projectID, databaseID string) error {
	// Mock implementation - no-op
	return nil
}

func (m *MockSecurityRulesEngine) GetRawRules(ctx context.Context, projectID, databaseID string) (string, error) {
	// Mock implementation - returns empty JSON
	return "{}", nil
}

// DummyLogger implements logger.Logger with no-ops for all methods
// Only the methods required by the interface are implemented for the test
type DummyLogger struct{}

func (d *DummyLogger) Debug(args ...interface{})                              {}
func (d *DummyLogger) Info(args ...interface{})                               {}
func (d *DummyLogger) Warn(args ...interface{})                               {}
func (d *DummyLogger) Error(args ...interface{})                              {}
func (d *DummyLogger) Fatal(args ...interface{})                              {}
func (d *DummyLogger) Debugf(format string, args ...interface{})              {}
func (d *DummyLogger) Infof(format string, args ...interface{})               {}
func (d *DummyLogger) Warnf(format string, args ...interface{})               {}
func (d *DummyLogger) Errorf(format string, args ...interface{})              {}
func (d *DummyLogger) Fatalf(format string, args ...interface{})              {}
func (d *DummyLogger) WithFields(fields map[string]interface{}) logger.Logger { return d }
func (d *DummyLogger) WithContext(ctx context.Context) logger.Logger          { return d }
func (d *DummyLogger) WithComponent(component string) logger.Logger           { return d }

func TestValidateRead_Allowed(t *testing.T) {
	mockEngine := new(MockSecurityRulesEngine)
	logger := &DummyLogger{}
	uc := firestoreusecase.NewSecurityUsecase(mockEngine, logger)
	user := &model.User{ID: primitive.NewObjectID()}
	path := "projects/proj1/databases/db1/documents/doc1"
	result := &repository.RuleEvaluationResult{Allowed: true, AllowedBy: "rule1"}
	mockEngine.On("EvaluateAccess", mock.Anything, repository.OperationRead, mock.AnythingOfType("*repository.SecurityContext")).Return(result, nil)
	err := uc.ValidateRead(context.Background(), user, path)
	assert.NoError(t, err)
	mockEngine.AssertExpectations(t)
}

func TestValidateRead_Denied(t *testing.T) {
	mockEngine := new(MockSecurityRulesEngine)
	logger := &DummyLogger{}
	uc := firestoreusecase.NewSecurityUsecase(mockEngine, logger)
	user := &model.User{ID: primitive.NewObjectID()}
	path := "projects/proj1/databases/db1/documents/doc1"
	result := &repository.RuleEvaluationResult{Allowed: false, DeniedBy: "rule2", Reason: "forbidden"}
	mockEngine.On("EvaluateAccess", mock.Anything, repository.OperationRead, mock.AnythingOfType("*repository.SecurityContext")).Return(result, nil)
	err := uc.ValidateRead(context.Background(), user, path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	mockEngine.AssertExpectations(t)
}

func TestValidateRead_EvaluateError(t *testing.T) {
	mockEngine := new(MockSecurityRulesEngine)
	logger := &DummyLogger{}
	uc := firestoreusecase.NewSecurityUsecase(mockEngine, logger)
	user := &model.User{ID: primitive.NewObjectID()}
	path := "projects/proj1/databases/db1/documents/doc1"
	mockEngine.On("EvaluateAccess", mock.Anything, repository.OperationRead, mock.AnythingOfType("*repository.SecurityContext")).Return(nil, errors.New("engine error"))
	err := uc.ValidateRead(context.Background(), user, path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security rules evaluation failed")
	mockEngine.AssertExpectations(t)
}
