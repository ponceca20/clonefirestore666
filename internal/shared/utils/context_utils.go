package utils

import (
	"context"
	"errors"

	"firestore-clone/internal/shared/contextkeys"
)

var ErrTenantIDNotFound = errors.New("tenantID not found in context")
var ErrTenantIDNotString = errors.New("tenantID in context is not a string")

// GetTenantIDFromContext retrieves the tenant ID from the context.
// It returns the tenant ID and an error if the tenant ID is not found or is not a string.
func GetTenantIDFromContext(ctx context.Context) (string, error) {
	tenantIDVal := ctx.Value(contextkeys.TenantIDKey)
	if tenantIDVal == nil {
		return "", ErrTenantIDNotFound
	}

	tenantID, ok := tenantIDVal.(string)
	if !ok {
		return "", ErrTenantIDNotString
	}

	if tenantID == "" { // Consider if an empty tenantID is valid or should also be an error.
		// For now, allowing empty string if it was explicitly set as such.
		// Depending on business logic, might return an error like:
		// return "", errors.New("tenantID in context is an empty string")
	}

	return tenantID, nil
}
