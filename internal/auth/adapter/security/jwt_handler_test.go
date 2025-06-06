package security_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"firestore-clone/internal/auth/adapter/security"
	"firestore-clone/internal/auth/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type JWTTestSuite struct {
	suite.Suite
	config  *config.Config
	service *security.JWTokenService
}

func (suite *JWTTestSuite) SetupTest() {
	suite.config = &config.Config{
		JWTSecretKey:    "test-secret-key-32-characters-long-12345",
		JWTIssuer:       "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	service, err := security.NewJWTokenService(suite.config)
	require.NoError(suite.T(), err)
	suite.service = service
}

func (suite *JWTTestSuite) TestNewJWTokenService_Success() {
	// Act
	service, err := security.NewJWTokenService(suite.config)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), service)
}

func (suite *JWTTestSuite) TestNewJWTokenService_ValidationErrors() {
	testCases := []struct {
		name         string
		modifyConfig func(*config.Config)
		expectedErr  string
	}{
		{
			name: "empty secret key",
			modifyConfig: func(cfg *config.Config) {
				cfg.JWTSecretKey = ""
			},
			expectedErr: "jwt secret key cannot be empty",
		},
		{
			name: "empty issuer",
			modifyConfig: func(cfg *config.Config) {
				cfg.JWTIssuer = ""
			},
			expectedErr: "jwt issuer cannot be empty",
		},
		{
			name: "zero TTL",
			modifyConfig: func(cfg *config.Config) {
				cfg.AccessTokenTTL = 0
			},
			expectedErr: "jwt access token TTL must be positive",
		},
		{
			name: "negative TTL",
			modifyConfig: func(cfg *config.Config) {
				cfg.AccessTokenTTL = -1 * time.Minute
			},
			expectedErr: "jwt access token TTL must be positive",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cfg := *suite.config // Copy
			tc.modifyConfig(&cfg)

			service, err := security.NewJWTokenService(&cfg)

			assert.Error(suite.T(), err)
			assert.Nil(suite.T(), service)
			assert.Contains(suite.T(), err.Error(), tc.expectedErr)
		})
	}
}

func (suite *JWTTestSuite) TestGenerateToken_Success() {
	// Arrange
	ctx := context.Background()
	userID := "user-123"
	email := "test@example.com"

	// Act
	tokenString, err := suite.service.GenerateToken(ctx, userID, email)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), tokenString)

	// Verify token structure
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(suite.config.JWTSecretKey), nil
	})
	require.NoError(suite.T(), err)
	assert.True(suite.T(), token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), userID, claims["userID"])
	assert.Equal(suite.T(), email, claims["email"])
	assert.Equal(suite.T(), suite.config.JWTIssuer, claims["iss"])
}

func (suite *JWTTestSuite) TestValidateToken_Success() {
	// Arrange
	ctx := context.Background()
	userID := "user-123"
	email := "test@example.com"

	tokenString, err := suite.service.GenerateToken(ctx, userID, email)
	require.NoError(suite.T(), err)

	// Act
	claims, err := suite.service.ValidateToken(ctx, tokenString)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), claims)
	assert.Equal(suite.T(), userID, claims.UserID)
	assert.Equal(suite.T(), email, claims.Email)
	assert.Equal(suite.T(), suite.config.JWTIssuer, claims.Issuer)
}

func (suite *JWTTestSuite) TestValidateToken_InvalidSignature() {
	// Arrange
	ctx := context.Background()

	differentConfig := *suite.config
	differentConfig.JWTSecretKey = "different-secret-key-32-chars-long"
	differentService, err := security.NewJWTokenService(&differentConfig)
	require.NoError(suite.T(), err)

	tokenString, err := differentService.GenerateToken(ctx, "user-123", "test@example.com")
	require.NoError(suite.T(), err)

	// Act
	claims, err := suite.service.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), claims)
	assert.Equal(suite.T(), security.ErrTokenSignatureInvalid, err)
}

func (suite *JWTTestSuite) TestValidateToken_ExpiredToken() {
	// Arrange
	ctx := context.Background()

	shortConfig := *suite.config
	shortConfig.AccessTokenTTL = 1 * time.Millisecond
	shortService, err := security.NewJWTokenService(&shortConfig)
	require.NoError(suite.T(), err)

	tokenString, err := shortService.GenerateToken(ctx, "user-123", "test@example.com")
	require.NoError(suite.T(), err)

	time.Sleep(10 * time.Millisecond)

	// Act
	claims, err := shortService.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), claims)
	assert.Equal(suite.T(), security.ErrTokenExpired, err)
}

func (suite *JWTTestSuite) TestValidateToken_MalformedTokens() {
	ctx := context.Background()

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid.token.format"},
		{"malformed jwt", "header.payload"},
		{"random string", "not-a-jwt-token"},
		{"incomplete jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			claims, err := suite.service.ValidateToken(ctx, tc.token)

			assert.Error(suite.T(), err)
			assert.Nil(suite.T(), claims)
			assert.Equal(suite.T(), security.ErrTokenInvalid, err)
		})
	}
}

func (suite *JWTTestSuite) TestGenerateAndValidateToken_RoundTrip() {
	// Arrange
	ctx := context.Background()
	userID := "user-123"
	email := "test@example.com"

	// Act
	tokenString, err := suite.service.GenerateToken(ctx, userID, email)
	require.NoError(suite.T(), err)

	claims, err := suite.service.ValidateToken(ctx, tokenString)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), userID, claims.UserID)
	assert.Equal(suite.T(), email, claims.Email)
}

func (suite *JWTTestSuite) TestTokenLifecycle_MultipleCycles() {
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		userID := fmt.Sprintf("user-%d", i)
		email := fmt.Sprintf("user%d@example.com", i)

		token, err := suite.service.GenerateToken(ctx, userID, email)
		require.NoError(suite.T(), err)

		claims, err := suite.service.ValidateToken(ctx, token)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), userID, claims.UserID)
		assert.Equal(suite.T(), email, claims.Email)
	}
}

func TestJWTTestSuite(t *testing.T) {
	suite.Run(t, new(JWTTestSuite))
}

func BenchmarkGenerateToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecretKey:   "test-secret-key-32-characters-long-12345",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 15 * time.Minute,
	}
	service, _ := security.NewJWTokenService(cfg)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GenerateToken(ctx, "user-123", "test@example.com")
	}
}

func BenchmarkValidateToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecretKey:   "test-secret-key-32-characters-long-12345",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 15 * time.Minute,
	}
	service, _ := security.NewJWTokenService(cfg)
	ctx := context.Background()

	token, _ := service.GenerateToken(ctx, "user-123", "test@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ValidateToken(ctx, token)
	}
}
