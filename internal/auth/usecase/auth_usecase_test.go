package usecase_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

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

func (m *mockAuthRepository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	args := m.Called(ctx, userID)
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

// Implementa los métodos faltantes en el mock para cumplir la interfaz
func (m *mockAuthRepository) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}
func (m *mockAuthRepository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}
func (m *mockAuthRepository) GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}
func (m *mockAuthRepository) CheckUserTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Bool(0), args.Error(1)
}
func (m *mockAuthRepository) CleanupExpiredSessions(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *mockAuthRepository) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *mockAuthRepository) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *mockAuthRepository) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}
func (m *mockAuthRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Session), args.Error(1)
}
func (m *mockAuthRepository) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *mockAuthRepository) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}
func (m *mockAuthRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Error(0)
}
func (m *mockAuthRepository) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockAuthRepository) VerifyPassword(ctx context.Context, userID, hashedPassword string) (bool, error) {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Bool(0), args.Error(1)
}

// Mock token service
type mockTokenService struct {
	mock.Mock
}

func (m *mockTokenService) GenerateToken(ctx context.Context, userID, email, tenantID, projectID, databaseID string, roles []string) (string, error) {
	args := m.Called(ctx, userID, email, tenantID, projectID, databaseID, roles)
	return args.String(0), args.Error(1)
}

func (m *mockTokenService) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Claims), args.Error(1)
}

// Métodos faltantes del mockTokenService
func (m *mockTokenService) GenerateRefreshToken(ctx context.Context, userID, email, tenantID string) (string, error) {
	args := m.Called(ctx, userID, email, tenantID)
	return args.String(0), args.Error(1)
}
func (m *mockTokenService) ValidateRefreshToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
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

	suite.usecase = usecase.NewAuthUsecase(suite.mockRepo, suite.mockToken, suite.config).(*usecase.AuthUsecase)
}

func (suite *AuthUsecaseTestSuite) TestRegister_Success() {
	ctx := context.Background()
	email := "test@example.com"
	password := "Password123!"
	tenantID := "tenant-123"
	firstName := "TestFirst"
	lastName := "TestLast"
	token := "jwt-token-123"
	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(nil, model.ErrUserNotFound)
	suite.mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(user *model.User) bool {
		return user.Email == email && user.TenantID == tenantID && user.FirstName == firstName && user.LastName == lastName
	})).Return(nil)
	suite.mockToken.On("GenerateToken", ctx, mock.AnythingOfType("string"), email, tenantID, "", "", []string{"user"}).Return(token, nil)
	suite.mockToken.On("GenerateRefreshToken", ctx, mock.AnythingOfType("string"), email, tenantID).Return("refresh-token", nil)
	suite.mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*model.Session")).Return(nil)

	registerReq := usecase.RegisterRequest{
		Email:          email,
		Password:       password,
		TenantID:       tenantID,
		FirstName:      firstName,
		LastName:       lastName,
		OrganizationID: "",
	}
	resp, err := suite.usecase.Register(ctx, registerReq)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), email, resp.User.Email)
	assert.Equal(suite.T(), tenantID, resp.User.TenantID)
	assert.Equal(suite.T(), firstName, resp.User.FirstName)
	assert.Equal(suite.T(), lastName, resp.User.LastName)
	assert.Equal(suite.T(), token, resp.AccessToken)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestRegister_EmailAlreadyTaken() {
	ctx := context.Background()
	email := "existing@example.com"
	password := "Password123!"
	tenantID := "tenant-456"
	firstNamePlaceholder := "first"
	lastNamePlaceholder := "last"

	existingUser := &model.User{
		Email: email,
	}
	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(existingUser, nil)

	registerReq := usecase.RegisterRequest{
		Email:          email,
		Password:       password,
		TenantID:       tenantID,
		FirstName:      firstNamePlaceholder,
		LastName:       lastNamePlaceholder,
		OrganizationID: "",
	}
	resp, err := suite.usecase.Register(ctx, registerReq)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), model.ErrUserExists, err)
	assert.Nil(suite.T(), resp)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestLogin_Success() {
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	tenantID := "tenant-123"
	token := "jwt-token-456"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.User{
		UserID:     "user-123",
		Email:      email,
		TenantID:   tenantID,
		Password:   string(hashedPassword),
		IsActive:   true,
		IsVerified: true,
		Roles:      []string{"user"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(user, nil)
	suite.mockRepo.On("UpdateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)
	suite.mockToken.On("GenerateToken", ctx, user.UserID, email, tenantID, "", "", []string{"user"}).Return(token, nil)
	suite.mockToken.On("GenerateRefreshToken", ctx, user.UserID, email, tenantID).Return("refresh-token", nil)
	suite.mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*model.Session")).Return(nil)

	loginReq := usecase.LoginRequest{
		Email:    email,
		Password: password,
		TenantID: tenantID,
	}
	resp, err := suite.usecase.Login(ctx, loginReq)

	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.UserID, resp.User.UserID)
	assert.Equal(suite.T(), user.Email, resp.User.Email)
	assert.Equal(suite.T(), token, resp.AccessToken)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertExpectations(suite.T())
}

func (suite *AuthUsecaseTestSuite) TestLogin_InvalidCredentials() {
	ctx := context.Background()
	email := "test@example.com"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	user := &model.User{
		UserID:     "user-123",
		Email:      email,
		TenantID:   "tenant-123",
		Password:   string(hashedPassword),
		IsActive:   true,
		IsVerified: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(user, nil)
	suite.mockRepo.On("UpdateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)

	loginReq := usecase.LoginRequest{
		Email:    email,
		Password: wrongPassword,
		TenantID: "tenant-123", // debe coincidir para que falle por password
	}
	resp, err := suite.usecase.Login(ctx, loginReq)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), model.ErrInvalidPassword, err)
	assert.Nil(suite.T(), resp)

	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestLogin_UserNotFound() {
	ctx := context.Background()
	email := "nouser@example.com" // email válido y formato aceptado
	password := "password123"
	suite.mockRepo.On("GetUserByEmail", ctx, email).Return(nil, model.ErrUserNotFound)

	loginReq := usecase.LoginRequest{
		Email:    email, // formato válido
		Password: password,
		TenantID: "tenant-notfound",
	}
	resp, err := suite.usecase.Login(ctx, loginReq)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), model.ErrUserNotFound, err)
	assert.Nil(suite.T(), resp)

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
	expectedErrors := []string{
		"invalid email format",
		"invalid email format",
		"invalid email format",
		"invalid email format",
		"email is required", // <- aquí el error esperado
	}

	for i, email := range invalidEmails {
		registerReq := usecase.RegisterRequest{
			Email:          email,
			Password:       "Password123!",
			TenantID:       "tenant-789",
			FirstName:      "First",
			LastName:       "Last",
			OrganizationID: "",
		}
		// No se debe esperar llamada a GetUserByEmail para emails inválidos
		suite.mockRepo.ExpectedCalls = nil // Limpiar expectativas previas
		suite.mockRepo.Calls = nil         // Limpiar llamadas previas
		resp, err := suite.usecase.Register(ctx, registerReq)
		assert.Error(suite.T(), err, "invalid_email_%s", email)
		assert.Contains(suite.T(), err.Error(), expectedErrors[i])
		assert.Nil(suite.T(), resp)
		// Verifica que NO se llamó a GetUserByEmail
		for _, call := range suite.mockRepo.Calls {
			assert.NotEqual(suite.T(), "GetUserByEmail", call.Method, "No debe llamarse GetUserByEmail para email inválido")
		}
	}

	suite.mockToken.AssertNotCalled(suite.T(), "GenerateToken")
}

func (suite *AuthUsecaseTestSuite) TestLogout_Success() {
	// Arrange
	ctx := context.Background()
	tokenString := "valid-token"
	claims := &repository.Claims{UserID: "user-123"}

	suite.mockToken.On("ValidateToken", ctx, tokenString).Return(claims, nil)
	suite.mockRepo.On("DeleteSessionsByUserID", ctx, claims.UserID).Return(nil)

	// Act
	err := suite.usecase.LogoutByToken(ctx, tokenString)

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
	// No se debe esperar llamada a DeleteSessionsByUserID si el token es inválido

	// Act
	err := suite.usecase.LogoutByToken(ctx, tokenString)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrTokenInvalid, err)

	suite.mockToken.AssertExpectations(suite.T())
	// Verifica que NO se llamó a DeleteSessionsByUserID
	for _, call := range suite.mockRepo.Calls {
		assert.NotEqual(suite.T(), "DeleteSessionsByUserID", call.Method, "No debe llamarse DeleteSessionsByUserID para token inválido")
	}
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
		return u.TenantID == "bench-tenant" && u.FirstName == "BenchFirst" && u.LastName == "BenchLast"
	})).Return(nil)
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant", "bench-project", "bench-database", []string{"user"}).Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registerReq := usecase.RegisterRequest{
			Email:          "test@example.com",
			Password:       "password123",
			TenantID:       "bench-tenant",
			FirstName:      "BenchFirst",
			LastName:       "BenchLast",
			OrganizationID: "",
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
		UserID:    "user-123",
		Email:     "test@example.com",
		TenantID:  "bench-tenant", // User from DB should have TenantID
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.On("GetUserByEmail", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
	mockToken.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, "bench-tenant", "bench-project", "bench-database", []string{"user"}).Return("token", nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loginReq := usecase.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
			TenantID: "bench-tenant",
		}
		uc.Login(ctx, loginReq)
	}
}
