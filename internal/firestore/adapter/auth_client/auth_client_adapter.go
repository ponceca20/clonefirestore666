package auth_client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/shared/contextkeys"
)

// AuthClientAdapter provides integration with the auth module for Firestore project context
type AuthClientAdapter struct {
	authUsecase usecase.AuthUsecaseInterface
	tokenSvc    repository.TokenService
}

// NewAuthClientAdapter creates a new auth client adapter with full integration
func NewAuthClientAdapter(authUsecase usecase.AuthUsecaseInterface, tokenSvc repository.TokenService) client.AuthClient {
	return &AuthClientAdapter{
		authUsecase: authUsecase,
		tokenSvc:    tokenSvc,
	}
}

// ValidateToken validates the given authentication token and returns user ID with Firestore context
func (c *AuthClientAdapter) ValidateToken(ctx context.Context, tokenString string) (userID string, err error) {
	if tokenString == "" {
		return "", ErrInvalidToken
	}
	// Clean token if it has Bearer prefix
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}
	claims, err := c.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}
	if claims == nil {
		return "", ErrInvalidToken
	}
	if claims.ProjectID != "" {
		ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
	}
	if claims.DatabaseID != "" {
		ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)
	}
	if claims.TenantID != "" {
		ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
	}
	return claims.UserID, nil
}

// GetUserByID retrieves user details by ID with Firestore project context
func (c *AuthClientAdapter) GetUserByID(ctx context.Context, userID string, projectID string) (*model.User, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}
	user, err := c.authUsecase.GetUserByID(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if projectID != "" && user.ProjectID != projectID {
		return nil, ErrProjectAccessDenied
	}
	return user, nil
}

// ValidateProjectAccess validates if user has access to the specified Firestore project
func (c *AuthClientAdapter) ValidateProjectAccess(ctx context.Context, userID, projectID string) error {
	user, err := c.GetUserByID(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if user.ProjectID != projectID {
		return ErrProjectAccessDenied
	}
	return nil
}

// ExtractFirestoreContext extracts Firestore context from token and returns structured info
func (c *AuthClientAdapter) ExtractFirestoreContext(ctx context.Context, tokenString string) (*FirestoreContext, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}
	claims, err := c.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Firestore context: %w", err)
	}
	return &FirestoreContext{
		UserID:     claims.UserID,
		Email:      claims.Email,
		ProjectID:  claims.ProjectID,
		DatabaseID: claims.DatabaseID,
		TenantID:   claims.TenantID,
	}, nil
}

// SimpleAuthClient provides a simple implementation for development/testing
type SimpleAuthClient struct{}

// NewSimpleAuthClient creates a new simple auth client (for development/testing only)
func NewSimpleAuthClient() client.AuthClient {
	return &SimpleAuthClient{}
}

// ValidateToken validates the given authentication token (placeholder implementation)
func (c *SimpleAuthClient) ValidateToken(ctx context.Context, tokenString string) (userID string, err error) {
	if tokenString == "" {
		return "", ErrInvalidToken
	}
	return "test-user-" + tokenString[:min(8, len(tokenString))], nil
}

// GetUserByID retrieves user details by ID (placeholder implementation)
func (c *SimpleAuthClient) GetUserByID(ctx context.Context, userID string, projectID string) (*model.User, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}
	return &model.User{
		ID:         userID,
		Email:      "test@example.com",
		ProjectID:  projectID,
		DatabaseID: "(default)",
		FirstName:  "Test",
		LastName:   "User",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// FirestoreContext represents Firestore-specific context extracted from auth tokens
type FirestoreContext struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ProjectID  string `json:"project_id"`
	DatabaseID string `json:"database_id"`
	TenantID   string `json:"tenant_id"`
}

// Auth client errors
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrUserNotFound         = errors.New("user not found")
	ErrProjectAccessDenied  = errors.New("access denied to Firestore project")
	ErrDatabaseAccessDenied = errors.New("access denied to Firestore database")
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
