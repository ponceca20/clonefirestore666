package client

import (
	"context"
	// Assuming the User model will be defined in the auth module.
	// Adjust the import path if necessary once the auth module's User model is finalized.
	// For now, we might need a placeholder or a more generic return type if the auth module is not yet implemented.
	// authModel "firestore-clone/internal/auth/domain/model"
)

// AuthClient defines the interface for communicating with the Auth module.
// This is used by the Firestore module to validate tokens and get user information.
type AuthClient interface {
	// ValidateToken validates the given authentication token.
	// It should return the userID associated with the token if valid,
	// and an error if the token is invalid or expired.
	ValidateToken(ctx context.Context, tokenString string) (userID string, err error)

	// GetUserByID retrieves user details from the Auth module.
	// This might be needed for security rule evaluation that depends on user roles or custom claims.
	// If the User model from the auth module is not available yet, this can be commented out
	// or return a generic map[string]interface{}.
	// GetUserByID(ctx context.Context, userID string) (*authModel.User, error) // Placeholder
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
