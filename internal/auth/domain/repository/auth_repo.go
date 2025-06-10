package repository

import (
	"context"

	"firestore-clone/internal/auth/domain/model"
)

// AuthRepository defines the interface for authentication data operations
type AuthRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByID(ctx context.Context, userID string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*model.User, error)

	// Password operations
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
	VerifyPassword(ctx context.Context, userID, hashedPassword string) (bool, error)

	// Session operations
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	GetSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	CleanupExpiredSessions(ctx context.Context) error

	// Tenant/Organization operations
	GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error)
	CheckUserTenantAccess(ctx context.Context, userID, tenantID string) (bool, error)
	AddUserToTenant(ctx context.Context, userID, tenantID string) error
	RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error

	// Health check
	HealthCheck(ctx context.Context) error
}
