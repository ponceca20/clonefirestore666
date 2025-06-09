package usecase_test

import (
	"context"
	"fmt"
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

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email, projectID string) (*model.User, error) {
	args := m.Called(ctx, email, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockAuthRepository) GetUserByID(ctx context.Context, id, projectID string) (*model.User, error) {
	args := m.Called(ctx, id, projectID)
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

func (m *mockAuthRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockAuthRepository) GetUsersByProject(ctx context.Context, projectID string) ([]*model.User, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

// Mock token service
type mockTokenService struct {
	mock.Mock
}

func (m *mockTokenService) GenerateToken(ctx context.Context, userID, email, tenantID, projectID, databaseID string) (string, error) {
	args := m.Called(ctx, userID, email, tenantID, projectID, databaseID)
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
	password := "Password123!" // password fuerte
	tenantID := "tenant-123"
	firstName := "TestFirst"
	lastName := "TestLast"
	avatarURL := "http://example.com/avatar.png"
	token := "jwt-token-123"
	suite.mockRepo.On("GetUserByEmail", ctx, email, "project-123").Return(nil, usecase.ErrUserNotFound)
	suite.mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(user *model.User) bool {
		return user.Email == email && user.TenantID == tenantID && user.FirstName == firstName && user.LastName == lastName && user.AvatarURL == avatarURL
	})).Return(nil)
	suite.mockToken.On("GenerateToken", ctx, mock.AnythingOfType("string"), email, tenantID, "project-123", "database-123").Return(token, nil)

	registerReq := usecase.RegisterRequest{
		Email:      email,
		Password:   password,
		ProjectID:  "project-123",
		DatabaseID: "database-123",
		TenantID:   tenantID,
		FirstName:  firstName,
		LastName:   lastName,
		AvatarURL:  avatarURL,
	}
	user, resultToken, err := suite.usecase.Register(ctx, registerReq)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), email, user.Email)
	assert.Equal(suite.T(), tenantID, user.TenantID)
	assert.Equal(suite.T(), firstName, user.FirstName)
	assert.Equal(suite.T(), lastName, user.LastName)
	assert.Equal(suite.T(), avatarURL, user.AvatarURL)
	assert.Equal(suite.T(), token, resultToken)
	// No verificar el hash de la contraseña porque la lógica lo limpia antes de retornar
	// assert.NotEmpty(suite.T(), user.PasswordHash)
	// assert.NotEqual(suite.T(), password, user.PasswordHash)
	// err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	// assert.NoError(suite.T(), err)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestRegister_EmailAlreadyTaken() {
	ctx := context.Background()
	email := "existing@example.com"
	password := "Password123!" // password fuerte
	tenantID := "tenant-456"
	firstNamePlaceholder := "first"
	lastNamePlaceholder := "last"
	avatarURLPlaceholder := "url"

	existingUser := &model.User{
		ID:    "existing-user-id",
		Email: email,
	}
	suite.mockRepo.On("GetUserByEmail", ctx, email, "project-456").Return(existingUser, nil)

	registerReq := usecase.RegisterRequest{
		Email:      email,
		Password:   password,
		ProjectID:  "project-456",
		DatabaseID: "database-456",
		TenantID:   tenantID,
		FirstName:  firstNamePlaceholder,
		LastName:   lastNamePlaceholder,
		AvatarURL:  avatarURLPlaceholder,
	}
	user, token, err := suite.usecase.Register(ctx, registerReq)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrEmailTaken, err)
	assert.Nil(suite.T(), user)
	assert.Empty(suite.T(), token)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestRegister_InvalidEmailFormat() {
	ctx := context.Background()
	invalidEmails := []string{
		"invalid-email",
		"@example.com",
		"test@",
		"test.example.com",
		"", // <- este caso debe esperar "email is required"
	}
	expectedErrors := []error{
		usecase.ErrInvalidEmailFormat,
		usecase.ErrInvalidEmailFormat,
		usecase.ErrInvalidEmailFormat,
		usecase.ErrInvalidEmailFormat,
		fmt.Errorf("email is required"), // <- aquí el error esperado
	}

	for i, email := range invalidEmails {
		registerReq := usecase.RegisterRequest{
			Email:      email,
			Password:   "Password123!",
			ProjectID:  "project-789",
			DatabaseID: "database-789",
			TenantID:   "tenant-789",
			FirstName:  "First",
			LastName:   "Last",
		}
		user, token, err := suite.usecase.Register(ctx, registerReq)
		assert.Error(suite.T(), err, "invalid_email_%s", email)
		if email == "" {
			assert.EqualError(suite.T(), err, "email is required")
		} else {
			assert.Equal(suite.T(), expectedErrors[i], err)
		}
		assert.Nil(suite.T(), user)
		assert.Empty(suite.T(), token)
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
		ProjectID:    "project-123",
		DatabaseID:   "database-123",
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	suite.mockRepo.On("GetUserByEmail", ctx, email, "project-123").Return(user, nil)
	suite.mockToken.On("GenerateToken", ctx, user.ID, email, tenantID, "project-123", "database-123").Return(token, nil)

	// Act
	loginReq := usecase.LoginRequest{
		Email:      email,
		Password:   password,
		ProjectID:  "project-123",
		DatabaseID: "database-123",
	}
	resultUser, resultToken, err := suite.usecase.Login(ctx, loginReq)

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
	suite.mockRepo.On("GetUserByEmail", ctx, email, "project-invalid").Return(user, nil)

	// Act
	loginReq := usecase.LoginRequest{
		Email:      email,
		Password:   wrongPassword,
		ProjectID:  "project-invalid",
		DatabaseID: "database-invalid",
	}
	resultUser, token, err := suite.usecase.Login(ctx, loginReq)

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
	suite.mockRepo.On("GetUserByEmail", ctx, email, "project-notfound").Return(nil, usecase.ErrUserNotFound)

	// Act
	loginReq := usecase.LoginRequest{
		Email:      email,
		Password:   password,
		ProjectID:  "project-notfound",
		DatabaseID: "database-notfound",
	}
	user, token, err := suite.usecase.Login(ctx, loginReq)

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
	projectID := "project-123"

	claims := &repository.Claims{
		UserID:    userID,
		Email:     email,
		ProjectID: projectID,
	}

	user := &model.User{
		ID:        userID,
		Email:     email,
		ProjectID: projectID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)
	suite.mockRepo.On("GetUserByID", ctx, userID, projectID).Return(user, nil)

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
	projectID := "project-123"

	claims := &repository.Claims{
		UserID:    userID,
		Email:     "test@example.com",
		ProjectID: projectID,
	}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)
	suite.mockRepo.On("GetUserByID", ctx, userID, projectID).Return(nil, usecase.ErrUserNotFound)

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
	suite.mockRepo.On("DeleteUserSessions", ctx, claims.UserID).Return(nil)

	// Act
	err := suite.usecase.Logout(ctx, tokenString)

	// Assert
	assert.NoError(suite.T(), err)

	suite.mockToken.AssertExpectations(suite.T())
	suite.mockRepo.AssertExpectations(suite.T())
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
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant", "bench-project", "bench-database").Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registerReq := usecase.RegisterRequest{
			Email:      "test@example.com",
			Password:   "password123",
			ProjectID:  "bench-project",
			DatabaseID: "bench-database",
			TenantID:   "bench-tenant",
			FirstName:  "BenchFirst",
			LastName:   "BenchLast",
			AvatarURL:  "bench.url",
		}
		uc.Register(ctx, registerReq)
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
		ProjectID:    "bench-project",
		DatabaseID:   "bench-database",
		PasswordHash: string(hashedPassword),
	}

	mockRepo.On("GetUserByEmail", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant", "bench-project", "bench-database").Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loginReq := usecase.LoginRequest{
			Email:      "test@example.com",
			Password:   "password123",
			ProjectID:  "bench-project",
			DatabaseID: "bench-database",
		}
		uc.Login(ctx, loginReq)
	}
}
