package repository

import (
	"context"
	"firestore-clone/internal/auth/domain/model"
)

// OperationType defines the type of operation being performed
type OperationType string

const (
	OperationRead   OperationType = "read"
	OperationWrite  OperationType = "write"
	OperationDelete OperationType = "delete"
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationList   OperationType = "list"
)

// SecurityContext contains context information for rule evaluation
type SecurityContext struct {
	User       *model.User            `json:"user,omitempty"`
	ProjectID  string                 `json:"projectId"`
	DatabaseID string                 `json:"databaseId"`
	Resource   map[string]interface{} `json:"resource,omitempty"`
	Request    map[string]interface{} `json:"request,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
	Path       string                 `json:"path"`
	// Variables extracted from path matching (e.g., {userId} -> userId: "123")
	Variables map[string]string `json:"variables,omitempty"`
}

// SecurityRule represents a single security rule
type SecurityRule struct {
	// Match pattern for paths this rule applies to
	Match string `json:"match"`

	// Allow conditions for different operations
	Allow map[OperationType]string `json:"allow,omitempty"`

	// Deny conditions for different operations
	Deny map[OperationType]string `json:"deny,omitempty"`

	// Priority of this rule (higher priority rules are evaluated first)
	Priority int `json:"priority"`

	// Optional metadata
	Description string `json:"description,omitempty"`
}

// RuleEvaluationResult represents the result of rule evaluation
type RuleEvaluationResult struct {
	Allowed   bool   `json:"allowed"`
	DeniedBy  string `json:"deniedBy,omitempty"`
	AllowedBy string `json:"allowedBy,omitempty"`
	Reason    string `json:"reason,omitempty"`
	RuleMatch string `json:"ruleMatch,omitempty"`
	// Performance metrics
	EvaluationTimeMs int64 `json:"evaluationTimeMs,omitempty"`
}

// ResourceAccessor provides access to Firestore resources for rule evaluation
// This follows the hexagonal architecture pattern - a port that can be implemented by adapters
type ResourceAccessor interface {
	// GetDocument retrieves a document by path for use in get() function
	GetDocument(ctx context.Context, projectID, databaseID, path string) (map[string]interface{}, error)

	// ExistsDocument checks if a document exists for use in exists() function
	ExistsDocument(ctx context.Context, projectID, databaseID, path string) (bool, error)
}

// SecurityRulesEngine defines the interface for evaluating Firestore security rules.
type SecurityRulesEngine interface {
	// EvaluateAccess checks if a user can perform an operation on a resource
	EvaluateAccess(ctx context.Context, operation OperationType, securityContext *SecurityContext) (*RuleEvaluationResult, error)

	// LoadRules loads security rules from storage
	LoadRules(ctx context.Context, projectID, databaseID string) ([]*SecurityRule, error)

	// SaveRules saves security rules to storage
	SaveRules(ctx context.Context, projectID, databaseID string, rules []*SecurityRule) error

	// ValidateRules validates the syntax and logic of security rules
	ValidateRules(rules []*SecurityRule) error

	// ClearCache clears the rules cache for a specific project/database
	ClearCache(projectID, databaseID string)

	// SetResourceAccessor sets the resource accessor for CEL functions
	SetResourceAccessor(accessor ResourceAccessor)

	// GetRawRules retrieves the raw rules as a string
	GetRawRules(ctx context.Context, projectID, databaseID string) (string, error)

	// DeleteRules deletes the security rules
	DeleteRules(ctx context.Context, projectID, databaseID string) error
}
