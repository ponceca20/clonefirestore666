package auth_client

import (
	"context"
	"errors"
	"firestore-clone/internal/firestore/domain/client"
)

// SimpleAuthClient provides a simple implementation of the AuthClient interface
// This is a placeholder implementation for development purposes
type SimpleAuthClient struct{}

// NewSimpleAuthClient creates a new simple auth client
func NewSimpleAuthClient() client.AuthClient {
	return &SimpleAuthClient{}
}

// ValidateToken validates the given authentication token
// This is a placeholder implementation that always returns success for development
func (c *SimpleAuthClient) ValidateToken(ctx context.Context, tokenString string) (userID string, err error) {
	// TODO: Implement actual token validation logic
	// For now, return a placeholder user ID
	if tokenString == "" {
		return "", ErrInvalidToken
	}
	return "placeholder-user-id", nil
}

// Add the error to the client package for now
var ErrInvalidToken = errors.New("invalid token")

// TODO: Implement auth client adapter for firestore
