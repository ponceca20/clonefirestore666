package security

import (
	"context"
	"errors"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/repository"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid          = errors.New("token is invalid")
	ErrTokenExpired          = errors.New("token is expired")
	ErrTokenSignatureInvalid = errors.New("token signature is invalid")
)

// JWTokenService implements JWT token generation and validation
type JWTokenService struct {
	secretKey []byte
	issuer    string
	ttl       time.Duration
}

// NewJWTokenService creates a new JWT token service
func NewJWTokenService(cfg *config.Config) (*JWTokenService, error) {
	if cfg.JWTSecretKey == "" {
		return nil, errors.New("jwt secret key cannot be empty")
	}
	if cfg.JWTIssuer == "" {
		return nil, errors.New("jwt issuer cannot be empty")
	}
	if cfg.AccessTokenTTL <= 0 {
		return nil, errors.New("jwt access token TTL must be positive")
	}

	return &JWTokenService{
		secretKey: []byte(cfg.JWTSecretKey),
		issuer:    cfg.JWTIssuer,
		ttl:       cfg.AccessTokenTTL,
	}, nil
}

// GenerateToken generates a new JWT token for the given user
func (s *JWTokenService) GenerateToken(ctx context.Context, userID, email, tenantID string) (string, error) {
	now := time.Now()
	claims := &repository.Claims{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTokenService) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	if tokenString == "" {
		return nil, ErrTokenInvalid
	}

	token, err := jwt.ParseWithClaims(tokenString, &repository.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenSignatureInvalid
		}
		return s.secretKey, nil
	})

	if err != nil {
		// Check for specific JWT validation errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrTokenSignatureInvalid
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenInvalid
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenInvalid
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*repository.Claims)
	if !ok {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
