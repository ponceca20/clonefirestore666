package security_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"firestore-clone/internal/auth/adapter/security"
	"firestore-clone/internal/auth/config"
)

type JWTTestSuite struct {
	suite.Suite
	config  *config.Config
	service *security.JWTokenService
}

func (suite *JWTTestSuite) SetupTest() {
	suite.config = &config.Config{
		JWTSecretKey:   "supersecretkeythatisatleast32characterslong!",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 2 * time.Hour,
	}
	service, err := security.NewJWTokenService(suite.config)
	assert.NoError(suite.T(), err)
	suite.service = service
}

func (suite *JWTTestSuite) TestGenerateAndValidateToken_Success() {
	ctx := context.Background()
	token, err := suite.service.GenerateToken(ctx, "user-1", "user@example.com", "tenant-1", "project-1", "db-1")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	claims, err := suite.service.ValidateToken(ctx, token)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "user-1", claims.UserID)
	assert.Equal(suite.T(), "user@example.com", claims.Email)
	assert.Equal(suite.T(), "tenant-1", claims.TenantID)
	assert.Equal(suite.T(), "project-1", claims.ProjectID)
	assert.Equal(suite.T(), "db-1", claims.DatabaseID)
	assert.Equal(suite.T(), suite.config.JWTIssuer, claims.Issuer)
	assert.WithinDuration(suite.T(), time.Now().Add(suite.config.AccessTokenTTL), claims.ExpiresAt.Time, 2*time.Second)
}

func (suite *JWTTestSuite) TestValidateToken_InvalidToken() {
	ctx := context.Background()
	_, err := suite.service.ValidateToken(ctx, "invalid.token.value")
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), security.ErrTokenInvalid, err)
}

func (suite *JWTTestSuite) TestValidateToken_ExpiredToken() {
	// Create a token with a very short expiry
	service, err := security.NewJWTokenService(&config.Config{
		JWTSecretKey:   "supersecretkeythatisatleast32characterslong!",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 1 * time.Second, // 1 segundo
	})
	assert.NoError(suite.T(), err)
	ctx := context.Background()
	token, err := service.GenerateToken(ctx, "user-1", "user@example.com", "tenant-1", "project-1", "db-1")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	time.Sleep(2 * time.Second) // Espera a que expire
	_, err = service.ValidateToken(ctx, token)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), security.ErrTokenExpired, err)
}

func (suite *JWTTestSuite) TestGenerateToken_EmptySecret() {
	badCfg := &config.Config{
		JWTSecretKey:   "",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 1 * time.Hour,
	}
	service, err := security.NewJWTokenService(badCfg)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), service)
}

func (suite *JWTTestSuite) TestGenerateToken_EmptyIssuer() {
	badCfg := &config.Config{
		JWTSecretKey:   "supersecretkeythatisatleast32characterslong!",
		JWTIssuer:      "",
		AccessTokenTTL: 1 * time.Hour,
	}
	service, err := security.NewJWTokenService(badCfg)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), service)
}

func (suite *JWTTestSuite) TestGenerateToken_NegativeTTL() {
	badCfg := &config.Config{
		JWTSecretKey:   "supersecretkeythatisatleast32characterslong!",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: -1 * time.Second,
	}
	service, err := security.NewJWTokenService(badCfg)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), service)
}

func (suite *JWTTestSuite) TestValidateToken_SignatureInvalid() {
	ctx := context.Background()
	// Create a token with a different secret
	otherService, _ := security.NewJWTokenService(&config.Config{
		JWTSecretKey:   "anothersecretkeythatisalsolongenough!",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 1 * time.Hour,
	})
	token, err := otherService.GenerateToken(ctx, "user-1", "user@example.com", "tenant-1", "project-1", "db-1")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	_, err = suite.service.ValidateToken(ctx, token)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), security.ErrTokenSignatureInvalid, err)
}

func TestJWTTestSuite(t *testing.T) {
	suite.Run(t, new(JWTTestSuite))
}
