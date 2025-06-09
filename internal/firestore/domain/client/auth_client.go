package client

import (
	"context"
	authModel "firestore-clone/internal/auth/domain/model"
)

// AuthClient defines the interface for communicating with the Auth module.
// This is used by the Firestore module to validate tokens and get user information.
type AuthClient interface {
	// ValidateToken validates the given authentication token.
	// It should return the userID associated with the token if valid,
	// and an error if the token is invalid or expired.
	ValidateToken(ctx context.Context, tokenString string) (userID string, err error)
	// GetUserByID retrieves user details from the Auth module.
	// This is needed for security rule evaluation that depends on user information.
	GetUserByID(ctx context.Context, userID string, projectID string) (*authModel.User, error)
}

// For now, let's define a placeholder User struct if we don't want to import it yet,
// or assume ValidateToken just returns userID and claims.
// Example of a placeholder User struct that might be returned by GetUserByID.
// type User struct {
//	ID string
//	Email string
//	IsActive bool
//	// Add other relevant fields like roles, custom claims, etc.
// }
