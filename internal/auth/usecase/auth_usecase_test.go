package usecase_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

// Mock repository
type mockAuthRepository struct {
	mock.Mock
}

func (m *mockAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockAuthRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockAuthRepository) CreateSession(ctx context.Context, session *model.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockAuthRepository) GetSessionByID(ctx context.Context, id string) (*model.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *mockAuthRepository) DeleteSession(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Mock token service
type mockTokenService struct {
	mock.Mock
}

func (m *mockTokenService) GenerateToken(ctx context.Context, userID, email, tenantID string) (string, error) { // Added tenantID
	args := m.Called(ctx, userID, email, tenantID) // Added tenantID
	return args.String(0), args.Error(1)
}

func (m *mockTokenService) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Claims), args.Error(1)
}

type AuthUsecaseTestSuite struct {
	suite.Suite
	mockRepo  *mockAuthRepository
	mockToken *mockTokenService
	usecase   *usecase.AuthUsecase
	config    *config.Config
}

func (suite *AuthUsecaseTestSuite) SetupTest() {
	suite.mockRepo = &mockAuthRepository{}
	suite.mockToken = &mockTokenService{}
	suite.config = &config.Config{
		JWTSecretKey:   "test-secret-key",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 15 * time.Minute,
	}

	suite.usecase = usecase.NewAuthUsecase(suite.mockRepo, suite.mockToken, suite.config)
}

func (suite *AuthUsecaseTestSuite) TestRegister_Success() {
	// Arrange
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	tenantID := "tenant-123" // Added tenantID
	firstName := "TestFirst"
	lastName := "TestLast"
	avatarURL := "http://example.com/avatar.png"
	token := "jwt-token-123"

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(nil, usecase.ErrUserNotFound)
	// Expect CreateUser to be called with a user object that includes the tenantID, firstName, lastName, avatarURL
	suite.mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(user *model.User) bool {
		return user.Email == email && user.TenantID == tenantID && user.FirstName == firstName && user.LastName == lastName && user.AvatarURL == avatarURL
	})).Return(nil)
	suite.mockToken.On("GenerateToken", ctx, mock.AnythingOfType("string"), email, tenantID).Return(token, nil) // Added tenantID

	// Act
	user, resultToken, err := suite.usecase.Register(ctx, email, password, tenantID, firstName, lastName, avatarURL) // Added tenantID and new fields

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), email, user.Email)
	assert.Equal(suite.T(), tenantID, user.TenantID) // Assert TenantID on returned user
	assert.Equal(suite.T(), firstName, user.FirstName)
	assert.Equal(suite.T(), lastName, user.LastName)
	assert.Equal(suite.T(), avatarURL, user.AvatarURL)
	assert.Equal(suite.T(), token, resultToken)
	assert.NotEmpty(suite.T(), user.PasswordHash)
	assert.NotEqual(suite.T(), password, user.PasswordHash) // Password should be hashed

	// Verify password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	assert.NoError(suite.T(), err)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestRegister_EmailAlreadyTaken() {
	// Arrange
	ctx := context.Background()
	email := "existing@example.com"
	password := "password123"
	tenantID := "tenant-456" // Added tenantID
	firstNamePlaceholder := "first"
	lastNamePlaceholder := "last"
	avatarURLPlaceholder := "url"

	existingUser := &model.User{
		ID:    "existing-user-id",
		Email: email,
		// TenantID might or might not be set on existingUser for this test's purpose,
		// as the check is for email existence before TenantID is deeply involved.
	}

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(existingUser, nil)

	// Act
	user, token, err := suite.usecase.Register(ctx, email, password, tenantID, firstNamePlaceholder, lastNamePlaceholder, avatarURLPlaceholder) // Added tenantID and new fields

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrEmailTaken, err)
	assert.Nil(suite.T(), user)
	assert.Empty(suite.T(), token)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestRegister_InvalidEmailFormat() {
	// Arrange
	ctx := context.Background()
	invalidEmails := []string{
		"invalid-email",
		"@example.com",
		"test@",
		"test.example.com",
		"",
	}

	for _, email := range invalidEmails {
		suite.Run("invalid_email_"+email, func() {
			// Act
			user, token, err := suite.usecase.Register(ctx, email, "password123", "tenant-789", "first", "last", "url") // Added tenantID and new fields

			// Assert
			assert.Error(suite.T(), err)
			assert.Equal(suite.T(), usecase.ErrInvalidEmailFormat, err)
			assert.Nil(suite.T(), user)
			assert.Empty(suite.T(), token)
		})
	}

	suite.mockRepo.AssertNotCalled(suite.T(), "GetUserByEmail")
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestLogin_Success() {
	// Arrange
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	tenantID := "tenant-123" // Added tenantID for consistency
	token := "jwt-token-456"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.User{
		ID:           "user-123",
		Email:        email,
		TenantID:     tenantID, // User from DB should have TenantID
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(user, nil)
	suite.mockToken.On("GenerateToken", ctx, user.ID, email, tenantID).Return(token, nil) // Added tenantID

	// Act
	resultUser, resultToken, err := suite.usecase.Login(ctx, email, password)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, resultUser.ID)
	assert.Equal(suite.T(), user.Email, resultUser.Email)
	assert.Equal(suite.T(), token, resultToken)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestLogin_InvalidCredentials() {
	// Arrange
	ctx := context.Background()
	email := "test@example.com"
	wrongPassword := "wrongpassword"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	user := &model.User{
		ID:           "user-123",
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(user, nil)

	// Act
	resultUser, token, err := suite.usecase.Login(ctx, email, wrongPassword)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrInvalidCredentials, err)
	assert.Nil(suite.T(), resultUser)
	assert.Empty(suite.T(), token)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestLogin_UserNotFound() {
	// Arrange
	ctx := context.Background()
	email := "nonexistent@example.com"
	password := "password123"

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(nil, usecase.ErrUserNotFound)

	// Act
	user, token, err := suite.usecase.Login(ctx, email, password)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrInvalidCredentials, err)
	assert.Nil(suite.T(), user)
	assert.Empty(suite.T(), token)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestValidateToken_Success() {
	// Arrange
	ctx := context.Background()
	tokenString := "valid-token"
	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "test-issuer",
		},
	}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)

	// Act
	resultClaims, err := suite.usecase.ValidateToken(ctx, tokenString)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), claims.UserID, resultClaims.UserID)
	assert.Equal(suite.T(), claims.Email, resultClaims.Email)

	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestValidateToken_InvalidToken() {
	// Arrange
	ctx := context.Background()
	tokenString := "invalid-token"

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(nil, usecase.ErrTokenInvalid)

	// Act
	claims, err := suite.usecase.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrTokenInvalid, err)
	assert.Nil(suite.T(), claims)

	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestGetUserFromToken_Success() {
	// Arrange
	ctx := context.Background()
	tokenString := "valid-token"
	userID := "user-123"
	email := "test@example.com"

	claims := &repository.Claims{
		UserID: userID,
		Email:  email,
	}

	user := &model.User{
		ID:        userID,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)
	suite.mockRepo.On("GetUserByID", ctx, userID).Return(user, nil)

	// Act
	resultUser, err := suite.usecase.GetUserFromToken(ctx, tokenString)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, resultUser.ID)
	assert.Equal(suite.T(), user.Email, resultUser.Email)

	suite.mockToken.AssertExpectations(suite.T())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestGetUserFromToken_InvalidToken() {
	// Arrange
	ctx := context.Background()
	tokenString := "invalid-token"

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(nil, usecase.ErrTokenInvalid)

	// Act
	user, err := suite.usecase.GetUserFromToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrTokenInvalid, err)
	assert.Nil(suite.T(), user)

	suite.mockToken.AssertExpectations(suite.T())
	suite.mockRepo.AssertNotCalled(suite.T(), "GetUserByID")
}

func (suite *AuthUsecaseTestSuite) TestGetUserFromToken_UserNotFound() {
	// Arrange
	ctx := context.Background()
	tokenString := "valid-token"
	userID := "nonexistent-user"

	claims := &repository.Claims{
		UserID: userID,
		Email:  "test@example.com",
	}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)
	suite.mockRepo.On("GetUserByID", ctx, userID).Return(nil, usecase.ErrUserNotFound)

	// Act
	user, err := suite.usecase.GetUserFromToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrUserNotFound, err)
	assert.Nil(suite.T(), user)

	suite.mockToken.AssertExpectations(suite.T())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestLogout_Success() {
	// Arrange
	ctx := context.Background()
	tokenString := "valid-token"
	claims := &repository.Claims{UserID: "user-123"}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)

	// Act
	err := suite.usecase.Logout(ctx, tokenString)

	// Assert
	assert.NoError(suite.T(), err)

	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestLogout_InvalidToken() {
	// Arrange
	ctx := context.Background()
	tokenString := "invalid-token"

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(nil, usecase.ErrTokenInvalid)

	// Act
	err := suite.usecase.Logout(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrTokenInvalid, err)

	suite.mockToken.AssertExpectations(suite.T())
}

func TestAuthUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(AuthUsecaseTestSuite))
}

// Benchmark tests
func BenchmarkRegister(b *testing.B) {
	mockRepo := &mockAuthRepository{}
	mockToken := &mockTokenService{}
	cfg := &config.Config{
		JWTSecretKey:   "test-secret-key",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 15 * time.Minute,
	}
	uc := usecase.NewAuthUsecase(mockRepo, mockToken, cfg)

	mockRepo.On("GetUserByEmail", mock.Anything, mock.Anything).Return(nil, usecase.ErrUserNotFound)
	mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.TenantID == "bench-tenant" && u.FirstName == "BenchFirst" && u.LastName == "BenchLast" && u.AvatarURL == "bench.url"
	})).Return(nil)
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant").Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.Register(ctx, "test@example.com", "password123", "bench-tenant", "BenchFirst", "BenchLast", "bench.url")
	}
}

func BenchmarkLogin(b *testing.B) {
	mockRepo := &mockAuthRepository{}
	mockToken := &mockTokenService{}
	cfg := &config.Config{
		JWTSecretKey:   "test-secret-key",
		JWTIssuer:      "test-issuer",
		AccessTokenTTL: 15 * time.Minute,
	}
	uc := usecase.NewAuthUsecase(mockRepo, mockToken, cfg)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		ID:           "user-123",
		Email:        "test@example.com",
		TenantID:     "bench-tenant", // User from DB should have TenantID
		PasswordHash: string(hashedPassword),
	}

	mockRepo.On("GetUserByEmail", mock.Anything, mock.Anything).Return(user, nil)
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant").Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.Login(ctx, "test@example.com", "password123")
	}
}
