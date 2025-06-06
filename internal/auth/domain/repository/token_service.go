package repository

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService defines the interface for token operations
type TokenService interface {
	GenerateToken(ctx context.Context, userID, email string) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"userID"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
