package usecase

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"

	"go.uber.org/zap"
)

// SecurityRulesUseCase handles business logic for security rules evaluation and management
type SecurityRulesUseCase struct {
	securityEngine repository.SecurityRulesEngine
	log            logger.Logger
}

// NewSecurityRulesUseCase creates a new SecurityRulesUseCase
func NewSecurityRulesUseCase(securityEngine repository.SecurityRulesEngine, log logger.Logger) *SecurityRulesUseCase {
	return &SecurityRulesUseCase{
		securityEngine: securityEngine,
		log:            log,
	}
}

// AccessRequest represents a request to access a resource
type AccessRequest struct {
	User       *model.User              `json:"user,omitempty"`
	ProjectID  string                   `json:"projectId"`
	DatabaseID string                   `json:"databaseId"`
	Path       string                   `json:"path"`
	Operation  repository.OperationType `json:"operation"`
	Resource   map[string]interface{}   `json:"resource,omitempty"`
	Request    map[string]interface{}   `json:"request,omitempty"`
}

// AccessResponse represents the response of an access evaluation
type AccessResponse struct {
	Allowed          bool                             `json:"allowed"`
	Reason           string                           `json:"reason,omitempty"`
	RuleMatch        string                           `json:"ruleMatch,omitempty"`
	EvaluationTimeMs int64                            `json:"evaluationTimeMs"`
	Timestamp        time.Time                        `json:"timestamp"`
	Details          *repository.RuleEvaluationResult `json:"details,omitempty"`
}

// RulesManagementRequest represents a request to manage security rules
type RulesManagementRequest struct {
	ProjectID  string                     `json:"projectId"`
	DatabaseID string                     `json:"databaseId"`
	Rules      []*repository.SecurityRule `json:"rules"`
}

// EvaluateAccess evaluates whether a user can perform an operation on a resource
func (uc *SecurityRulesUseCase) EvaluateAccess(ctx context.Context, request *AccessRequest) (*AccessResponse, error) {
	startTime := time.Now()

	// Validate input
	if err := uc.validateAccessRequest(request); err != nil {
		uc.log.Error("Invalid access request", zap.Error(err))
		return nil, fmt.Errorf("invalid access request: %w", err)
	}

	// Create security context
	securityContext := &repository.SecurityContext{
		User:       request.User,
		ProjectID:  request.ProjectID,
		DatabaseID: request.DatabaseID,
		Path:       request.Path,
		Resource:   request.Resource,
		Request:    request.Request,
		Timestamp:  startTime.Unix(),
	}

	// Evaluate access using the security engine
	result, err := uc.securityEngine.EvaluateAccess(ctx, request.Operation, securityContext)
	if err != nil {
		uc.log.Error("Failed to evaluate access",
			zap.String("projectID", request.ProjectID),
			zap.String("databaseID", request.DatabaseID),
			zap.String("path", request.Path),
			zap.String("operation", string(request.Operation)),
			zap.Error(err))
		return nil, fmt.Errorf("access evaluation failed: %w", err)
	}
	// Log access decision for audit purposes
	userID := "anonymous"
	if request.User != nil {
		userID = request.User.ID.Hex()
	}

	if result.Allowed {
		uc.log.Info("Access granted",
			zap.String("userID", userID),
			zap.String("projectID", request.ProjectID),
			zap.String("databaseID", request.DatabaseID),
			zap.String("path", request.Path),
			zap.String("operation", string(request.Operation)),
			zap.String("allowedBy", result.AllowedBy),
			zap.Int64("evaluationTimeMs", result.EvaluationTimeMs))
	} else {
		uc.log.Warn("Access denied",
			zap.String("userID", userID),
			zap.String("projectID", request.ProjectID),
			zap.String("databaseID", request.DatabaseID),
			zap.String("path", request.Path),
			zap.String("operation", string(request.Operation)),
			zap.String("deniedBy", result.DeniedBy),
			zap.String("reason", result.Reason),
			zap.Int64("evaluationTimeMs", result.EvaluationTimeMs))
	}

	return &AccessResponse{
		Allowed:          result.Allowed,
		Reason:           result.Reason,
		RuleMatch:        result.RuleMatch,
		EvaluationTimeMs: result.EvaluationTimeMs,
		Timestamp:        startTime,
		Details:          result,
	}, nil
}

// DeployRules deploys new security rules for a project/database
func (uc *SecurityRulesUseCase) DeployRules(ctx context.Context, request *RulesManagementRequest) error {
	// Validate input
	if err := uc.validateRulesManagementRequest(request); err != nil {
		uc.log.Error("Invalid rules management request", zap.Error(err))
		return fmt.Errorf("invalid rules request: %w", err)
	}

	// Validate rules syntax and logic (handle nil rules)
	rules := request.Rules
	if rules == nil {
		rules = []*repository.SecurityRule{} // Convert nil to empty slice
	}
	if err := uc.securityEngine.ValidateRules(rules); err != nil {
		uc.log.Error("Rules validation failed",
			zap.String("projectID", request.ProjectID),
			zap.String("databaseID", request.DatabaseID),
			zap.Error(err))
		return fmt.Errorf("rules validation failed: %w", err)
	}
	// Save rules
	if err := uc.securityEngine.SaveRules(ctx, request.ProjectID, request.DatabaseID, rules); err != nil {
		uc.log.Error("Failed to save security rules",
			zap.String("projectID", request.ProjectID),
			zap.String("databaseID", request.DatabaseID),
			zap.Error(err))
		return fmt.Errorf("failed to save rules: %w", err)
	}

	uc.log.Info("Security rules deployed successfully",
		zap.String("projectID", request.ProjectID),
		zap.String("databaseID", request.DatabaseID),
		zap.Int("rulesCount", len(rules)))

	return nil
}

// LoadRules loads the current security rules for a project/database
func (uc *SecurityRulesUseCase) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	if projectID == "" || databaseID == "" {
		return nil, fmt.Errorf("projectID and databaseID are required")
	}

	rules, err := uc.securityEngine.LoadRules(ctx, projectID, databaseID)
	if err != nil {
		uc.log.Error("Failed to load security rules",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	uc.log.Debug("Loaded security rules",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID),
		zap.Int("rulesCount", len(rules)))

	return rules, nil
}

// ClearRulesCache clears the rules cache for improved performance on rule updates
func (uc *SecurityRulesUseCase) ClearRulesCache(projectID, databaseID string) {
	uc.securityEngine.ClearCache(projectID, databaseID)

	uc.log.Debug("Cleared security rules cache",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID))
}

// TestRules tests a set of rules against various scenarios (useful for rule development)
func (uc *SecurityRulesUseCase) TestRules(ctx context.Context, request *RulesManagementRequest, testCases []*AccessRequest) ([]*AccessResponse, error) {
	// Validate rules first
	if err := uc.securityEngine.ValidateRules(request.Rules); err != nil {
		return nil, fmt.Errorf("rules validation failed: %w", err)
	}

	// Save rules temporarily for testing
	originalRules, err := uc.securityEngine.LoadRules(ctx, request.ProjectID, request.DatabaseID)
	if err != nil {
		uc.log.Warn("Could not load original rules for testing", zap.Error(err))
	}

	// Deploy test rules
	if err := uc.securityEngine.SaveRules(ctx, request.ProjectID, request.DatabaseID, request.Rules); err != nil {
		return nil, fmt.Errorf("failed to deploy test rules: %w", err)
	}

	// Clear cache to ensure test rules are used
	uc.securityEngine.ClearCache(request.ProjectID, request.DatabaseID)

	// Execute test cases
	results := make([]*AccessResponse, len(testCases))
	for i, testCase := range testCases {
		result, err := uc.EvaluateAccess(ctx, testCase)
		if err != nil {
			uc.log.Error("Test case evaluation failed",
				zap.Int("testCaseIndex", i),
				zap.Error(err))
			// Continue with other test cases
			results[i] = &AccessResponse{
				Allowed:   false,
				Reason:    fmt.Sprintf("Test execution error: %v", err),
				Timestamp: time.Now(),
			}
		} else {
			results[i] = result
		}
	}

	// Restore original rules if they existed
	if originalRules != nil {
		if err := uc.securityEngine.SaveRules(ctx, request.ProjectID, request.DatabaseID, originalRules); err != nil {
			uc.log.Error("Failed to restore original rules after testing", zap.Error(err))
		}
	}

	uc.log.Info("Rules testing completed",
		zap.String("projectID", request.ProjectID),
		zap.String("databaseID", request.DatabaseID),
		zap.Int("testCasesCount", len(testCases)))

	return results, nil
}

// validateAccessRequest validates an access request
func (uc *SecurityRulesUseCase) validateAccessRequest(request *AccessRequest) error {
	if request == nil {
		return fmt.Errorf("request is required")
	}
	if request.ProjectID == "" {
		return fmt.Errorf("projectID is required")
	}
	if request.DatabaseID == "" {
		return fmt.Errorf("databaseID is required")
	}
	if request.Path == "" {
		return fmt.Errorf("path is required")
	}
	if request.Operation == "" {
		return fmt.Errorf("operation is required")
	}

	// Validate operation type
	validOps := map[repository.OperationType]bool{
		repository.OperationRead:   true,
		repository.OperationWrite:  true,
		repository.OperationDelete: true,
		repository.OperationCreate: true,
		repository.OperationUpdate: true,
		repository.OperationList:   true,
	}
	if !validOps[request.Operation] {
		return fmt.Errorf("invalid operation type: %s", request.Operation)
	}

	return nil
}

// validateRulesManagementRequest validates a rules management request
func (uc *SecurityRulesUseCase) validateRulesManagementRequest(request *RulesManagementRequest) error {
	if request == nil {
		return fmt.Errorf("request is required")
	}
	if request.ProjectID == "" {
		return fmt.Errorf("projectID is required")
	}
	if request.DatabaseID == "" {
		return fmt.Errorf("databaseID is required")
	}
	// Note: Rules can be empty (clearing all rules)
	return nil
}
