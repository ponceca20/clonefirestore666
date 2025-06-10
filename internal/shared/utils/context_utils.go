package utils

import (
	"context"
	"errors"

	"firestore-clone/internal/shared/contextkeys"
)

// Common context errors
var (
	ErrTenantIDNotFound        = errors.New("tenantID not found in context")
	ErrTenantIDNotString       = errors.New("tenantID in context is not a string")
	ErrUserIDNotFound          = errors.New("userID not found in context")
	ErrUserIDNotString         = errors.New("userID in context is not a string")
	ErrProjectIDNotFound       = errors.New("projectID not found in context")
	ErrProjectIDNotString      = errors.New("projectID in context is not a string")
	ErrDatabaseIDNotFound      = errors.New("databaseID not found in context")
	ErrDatabaseIDNotString     = errors.New("databaseID in context is not a string")
	ErrRequestIDNotFound       = errors.New("requestID not found in context")
	ErrRequestIDNotString      = errors.New("requestID in context is not a string")
	ErrOrganizationIDNotFound  = errors.New("organizationID not found in context")
	ErrOrganizationIDNotString = errors.New("organizationID in context is not a string")
)

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

	return tenantID, nil
}

// GetOrganizationIDFromContext retrieves the organization ID from the context.
func GetOrganizationIDFromContext(ctx context.Context) (string, error) {
	orgIDVal := ctx.Value(contextkeys.OrganizationIDKey)
	if orgIDVal == nil {
		return "", ErrOrganizationIDNotFound
	}

	orgID, ok := orgIDVal.(string)
	if !ok {
		return "", ErrOrganizationIDNotString
	}

	return orgID, nil
}

// GetUserIDFromContext retrieves the user ID from the context.
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userIDVal := ctx.Value(contextkeys.UserIDKey)
	if userIDVal == nil {
		return "", ErrUserIDNotFound
	}

	userID, ok := userIDVal.(string)
	if !ok {
		return "", ErrUserIDNotString
	}

	return userID, nil
}

// GetProjectIDFromContext retrieves the project ID from the context.
func GetProjectIDFromContext(ctx context.Context) (string, error) {
	projectIDVal := ctx.Value(contextkeys.ProjectIDKey)
	if projectIDVal == nil {
		return "", ErrProjectIDNotFound
	}

	projectID, ok := projectIDVal.(string)
	if !ok {
		return "", ErrProjectIDNotString
	}

	return projectID, nil
}

// GetDatabaseIDFromContext retrieves the database ID from the context.
func GetDatabaseIDFromContext(ctx context.Context) (string, error) {
	databaseIDVal := ctx.Value(contextkeys.DatabaseIDKey)
	if databaseIDVal == nil {
		return "", ErrDatabaseIDNotFound
	}

	databaseID, ok := databaseIDVal.(string)
	if !ok {
		return "", ErrDatabaseIDNotString
	}

	return databaseID, nil
}

// GetRequestIDFromContext retrieves the request ID from the context.
func GetRequestIDFromContext(ctx context.Context) (string, error) {
	requestIDVal := ctx.Value(contextkeys.RequestIDKey)
	if requestIDVal == nil {
		return "", ErrRequestIDNotFound
	}

	requestID, ok := requestIDVal.(string)
	if !ok {
		return "", ErrRequestIDNotString
	}

	return requestID, nil
}

// GetUserEmailFromContext retrieves the user email from the context.
func GetUserEmailFromContext(ctx context.Context) (string, error) {
	emailVal := ctx.Value(contextkeys.UserEmailKey)
	if emailVal == nil {
		return "", errors.New("userEmail not found in context")
	}

	email, ok := emailVal.(string)
	if !ok {
		return "", errors.New("userEmail in context is not a string")
	}

	return email, nil
}

// Context builder functions

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextkeys.TenantIDKey, tenantID)
}

// WithOrganizationID adds organization ID to context
func WithOrganizationID(ctx context.Context, organizationID string) context.Context {
	return context.WithValue(ctx, contextkeys.OrganizationIDKey, organizationID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextkeys.UserIDKey, userID)
}

// WithProjectID adds project ID to context
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, contextkeys.ProjectIDKey, projectID)
}

// WithDatabaseID adds database ID to context
func WithDatabaseID(ctx context.Context, databaseID string) context.Context {
	return context.WithValue(ctx, contextkeys.DatabaseIDKey, databaseID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextkeys.RequestIDKey, requestID)
}

// WithUserEmail adds user email to context
func WithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, contextkeys.UserEmailKey, email)
}

// WithComponent adds component name to context
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, contextkeys.ComponentKey, component)
}

// WithOperation adds operation name to context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, contextkeys.OperationKey, operation)
}

// Optional getters that return default values instead of errors

// GetTenantIDOrDefault retrieves the tenant ID from context or returns a default value
func GetTenantIDOrDefault(ctx context.Context, defaultValue string) string {
	if tenantID, err := GetTenantIDFromContext(ctx); err == nil {
		return tenantID
	}
	return defaultValue
}

// GetOrganizationIDOrDefault retrieves the organization ID from context or returns a default value
func GetOrganizationIDOrDefault(ctx context.Context, defaultValue string) string {
	if organizationID, err := GetOrganizationIDFromContext(ctx); err == nil {
		return organizationID
	}
	return defaultValue
}

// GetUserIDOrDefault retrieves the user ID from context or returns a default value
func GetUserIDOrDefault(ctx context.Context, defaultValue string) string {
	if userID, err := GetUserIDFromContext(ctx); err == nil {
		return userID
	}
	return defaultValue
}

// GetProjectIDOrDefault retrieves the project ID from context or returns a default value
func GetProjectIDOrDefault(ctx context.Context, defaultValue string) string {
	if projectID, err := GetProjectIDFromContext(ctx); err == nil {
		return projectID
	}
	return defaultValue
}

// GetDatabaseIDOrDefault retrieves the database ID from context or returns a default value
func GetDatabaseIDOrDefault(ctx context.Context, defaultValue string) string {
	if databaseID, err := GetDatabaseIDFromContext(ctx); err == nil {
		return databaseID
	}
	return defaultValue
}

// HasTenantID checks if context has a tenant ID
func HasTenantID(ctx context.Context) bool {
	_, err := GetTenantIDFromContext(ctx)
	return err == nil
}

// HasOrganizationID checks if context has an organization ID
func HasOrganizationID(ctx context.Context) bool {
	_, err := GetOrganizationIDFromContext(ctx)
	return err == nil
}

// HasUserID checks if context has a user ID
func HasUserID(ctx context.Context) bool {
	_, err := GetUserIDFromContext(ctx)
	return err == nil
}

// HasProjectID checks if context has a project ID
func HasProjectID(ctx context.Context) bool {
	_, err := GetProjectIDFromContext(ctx)
	return err == nil
}

// HasDatabaseID checks if context has a database ID
func HasDatabaseID(ctx context.Context) bool {
	_, err := GetDatabaseIDFromContext(ctx)
	return err == nil
}
