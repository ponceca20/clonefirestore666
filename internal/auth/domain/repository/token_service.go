package repository

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService defines the interface for token operations
type TokenService interface {
	GenerateToken(ctx context.Context, userID, email, tenantID, projectID, databaseID string) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	GenerateRefreshToken(ctx context.Context, userID, email, tenantID string) (string, error)
	ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error)
}

// Claims represents JWT claims with multitenant support
type Claims struct {
	UserID         string   `json:"userID"`
	Email          string   `json:"email"`
	TenantID       string   `json:"tenantID,omitempty"`
	OrganizationID string   `json:"organizationID,omitempty"`
	ProjectID      string   `json:"projectID,omitempty"`
	DatabaseID     string   `json:"databaseID,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// HasRole checks if the user has a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission
func (c *Claims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetTenantContext returns the tenant context from claims
func (c *Claims) GetTenantContext() TenantContext {
	return TenantContext{
		TenantID:       c.TenantID,
		OrganizationID: c.OrganizationID,
		ProjectID:      c.ProjectID,
		DatabaseID:     c.DatabaseID,
	}
}

// TenantContext represents the tenant context from token claims
type TenantContext struct {
	TenantID       string
	OrganizationID string
	ProjectID      string
	DatabaseID     string
}
