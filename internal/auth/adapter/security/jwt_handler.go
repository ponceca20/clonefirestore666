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

// GenerateToken generates a new JWT token for the given user with Firestore context
func (s *JWTokenService) GenerateToken(ctx context.Context, userID, email, tenantID, projectID, databaseID string) (string, error) {
	now := time.Now()
	claims := &repository.Claims{
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		ProjectID:  projectID,
		DatabaseID: databaseID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  []string{projectID}, // Use projectID as audience
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
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

	// Additional validation for multitenant context
	if claims.UserID == "" || claims.Email == "" {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// GenerateRefreshToken generates a refresh token with extended TTL
func (s *JWTokenService) GenerateRefreshToken(ctx context.Context, userID, email, tenantID string) (string, error) {
	now := time.Now()
	claims := &repository.Claims{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(168 * time.Hour)), // 7 days
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (s *JWTokenService) ValidateRefreshToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
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

	// Additional validation for refresh tokens
	if claims.UserID == "" || claims.Email == "" {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
