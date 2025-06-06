package repository

import (
	"context"
	// "firestore-clone/internal/auth/domain/model" // Assuming User model exists in auth module
)

// SecurityRulesEngine defines the interface for evaluating Firestore security rules.
type SecurityRulesEngine interface {
	// CheckAccess checks if a given operation is allowed based on security rules.
	// userID might be an empty string for unauthenticated requests.
	// resourcePath is the path to the document or collection being accessed.
	// operationType could be "read", "write", "delete", "list".
	// requestData is relevant for write operations.
	CheckAccess(ctx context.Context, userID string, resourcePath string, operationType string, requestData map[string]interface{}) (bool, error)

	// LoadRules loads and compiles the security rules.
	// This would typically be called when the rules are updated.
	LoadRules(ctx context.Context, rules string) error
}

// TODO: Define common operation types
const (
	OperationRead   = "read"
	OperationWrite  = "write" // Covers create and update
	OperationDelete = "delete"
	OperationList   = "list"
)
