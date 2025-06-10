package auth_client

import (
	"context"
	"errors"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"
	"firestore-clone/internal/firestore/domain/client"
)

// AuthClientAdapter adapts the auth module's usecase to the Firestore AuthClient interface
type AuthClientAdapter struct {
	authUsecase  usecase.AuthUsecaseInterface
	tokenService repository.TokenService
}

// NewAuthClientAdapter creates a new auth client adapter that integrates with the auth module
func NewAuthClientAdapter(authUsecase usecase.AuthUsecaseInterface, tokenService repository.TokenService) client.AuthClient {
	return &AuthClientAdapter{
		authUsecase:  authUsecase,
		tokenService: tokenService,
	}
}

// ValidateToken validates a JWT token and returns the userID
func (a *AuthClientAdapter) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("token is empty")
	}

	// Use the auth module's token validation
	claims, err := a.authUsecase.ValidateToken(ctx, tokenString)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

// GetUserByID retrieves user information by user ID using the auth module
func (a *AuthClientAdapter) GetUserByID(ctx context.Context, userID string, projectID string) (*model.User, error) {
	if userID == "" {
		return nil, errors.New("user ID is empty")
	}

	// Use the auth module's user retrieval
	user, err := a.authUsecase.GetUserByID(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// SimpleAuthClient implements AuthClient interface for basic authentication (for testing)
type SimpleAuthClient struct{}

// NewSimpleAuthClient creates a new SimpleAuthClient
func NewSimpleAuthClient() client.AuthClient {
	return &SimpleAuthClient{}
}

// ValidateToken validates a JWT token and returns userID
func (c *SimpleAuthClient) ValidateToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrInvalidToken
	}

	// Simple validation for testing - in production this would validate JWT
	// and extract user information from token claims
	if token == "invalid" {
		return "", ErrInvalidToken
	}

	// Return a mock user ID for valid tokens
	return "test-user-id", nil
}

// GetUserByID retrieves user information by user ID
func (c *SimpleAuthClient) GetUserByID(ctx context.Context, userID string, projectID string) (*model.User, error) {
	if userID == "" {
		return nil, errors.New("user ID is empty")
	}

	if userID == "not-found" {
		return nil, errors.New("user not found")
	}

	// Return a mock user
	return &model.User{
		UserID:         userID,
		Email:          "user@example.com",
		TenantID:       "default-tenant",
		OrganizationID: "default-org",
		Roles:          []string{"user"},
		IsActive:       true,
		IsVerified:     true,
		FirstName:      "Test",
		LastName:       "User",
	}, nil
}

// Auth client errors
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrUserNotFound         = errors.New("user not found")
	ErrProjectAccessDenied  = errors.New("access denied to Firestore project")
	ErrDatabaseAccessDenied = errors.New("access denied to Firestore database")
)
