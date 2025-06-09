package repository

import (
	"context"

	"firestore-clone/internal/auth/domain/model"
)

// AuthRepository defines the interface for authentication data operations
type AuthRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email, projectID string) (*model.User, error)
	GetUserByID(ctx context.Context, id, projectID string) (*model.User, error)
	GetUsersByProject(ctx context.Context, projectID string) ([]*model.User, error)

	// Session operations
	CreateSession(ctx context.Context, session *model.Session) error
	GetSessionByID(ctx context.Context, id string) (*model.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error
}
