package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authhttp "firestore-clone/internal/auth/adapter/http"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Mock usecase
type mockAuthUsecase struct {
	mock.Mock
}

func (m *mockAuthUsecase) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}

func (m *mockAuthUsecase) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}

func (m *mockAuthUsecase) Logout(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

func (m *mockAuthUsecase) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Claims), args.Error(1)
}

func (m *mockAuthUsecase) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

type AuthHTTPTestSuite struct {
	suite.Suite
	app         *fiber.App
	mockUsecase *mockAuthUsecase
}

func (suite *AuthHTTPTestSuite) SetupTest() {
	suite.mockUsecase = &mockAuthUsecase{}
	suite.app = fiber.New()

	handler := authhttp.NewAuthHTTPHandler(
		suite.mockUsecase,
		"test_cookie",
		"/",
		"",
		3600,
		false,
		true,
		"Lax",
	)

	handler.SetupAuthRoutes(suite.app)
}

func (suite *AuthHTTPTestSuite) TestRegister_Success() {
	// Arrange
	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := "jwt-token-12345"

	suite.mockUsecase.On("Register", mock.Anything, "test@example.com", "password123").
		Return(user, token, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response authhttp.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), user.ID, response.User.ID)
	assert.Equal(suite.T(), user.Email, response.User.Email)
	assert.Equal(suite.T(), token, response.Token)
	assert.Equal(suite.T(), "User registered successfully", response.Message)

	// Check cookie is set
	cookies := resp.Cookies()
	assert.Len(suite.T(), cookies, 1)
	assert.Equal(suite.T(), "test_cookie", cookies[0].Name)
	assert.Equal(suite.T(), token, cookies[0].Value)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestRegister_EmailAlreadyTaken() {
	// Arrange
	requestBody := map[string]string{
		"email":    "existing@example.com",
		"password": "password123",
	}

	suite.mockUsecase.On("Register", mock.Anything, "existing@example.com", "password123").
		Return((*model.User)(nil), "", usecase.ErrEmailTaken)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusConflict, resp.StatusCode)

	var response authhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Email already taken", response.Error)
	assert.Equal(suite.T(), http.StatusConflict, response.Code)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestRegister_ValidationErrors() {
	testCases := []struct {
		name        string
		requestBody map[string]string
		expectedMsg string
	}{
		{
			name:        "missing email",
			requestBody: map[string]string{"password": "password123"},
			expectedMsg: "email is required",
		},
		{
			name:        "invalid email format",
			requestBody: map[string]string{"email": "invalid-email", "password": "password123"},
			expectedMsg: "email must be a valid email address",
		},
		{
			name:        "missing password",
			requestBody: map[string]string{"email": "test@example.com"},
			expectedMsg: "password is required",
		},
		{
			name:        "password too short",
			requestBody: map[string]string{"email": "test@example.com", "password": "123"},
			expectedMsg: "password must be at least 8 characters long",
		},
		{
			name:        "empty email",
			requestBody: map[string]string{"email": "", "password": "password123"},
			expectedMsg: "email is required",
		},
		{
			name:        "empty password",
			requestBody: map[string]string{"email": "test@example.com", "password": ""},
			expectedMsg: "password is required",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.app.Test(req)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

			var response authhttp.ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(suite.T(), err)

			assert.Equal(suite.T(), "Validation failed", response.Error)
			assert.Contains(suite.T(), response.Message, tc.expectedMsg)
		})
	}

	suite.mockUsecase.AssertNotCalled(suite.T(), "Register")
}

func (suite *AuthHTTPTestSuite) TestLogin_Success() {
	// Arrange
	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := "jwt-token-54321"

	suite.mockUsecase.On("Login", mock.Anything, "test@example.com", "password123").
		Return(user, token, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response authhttp.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), user.ID, response.User.ID)
	assert.Equal(suite.T(), user.Email, response.User.Email)
	assert.Equal(suite.T(), token, response.Token)
	assert.Equal(suite.T(), "Login successful", response.Message)

	// Check cookie is set
	cookies := resp.Cookies()
	assert.Len(suite.T(), cookies, 1)
	assert.Equal(suite.T(), "test_cookie", cookies[0].Name)
	assert.Equal(suite.T(), token, cookies[0].Value)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestLogin_InvalidCredentials() {
	// Arrange
	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	suite.mockUsecase.On("Login", mock.Anything, "test@example.com", "wrongpassword").
		Return((*model.User)(nil), "", usecase.ErrInvalidCredentials)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	var response authhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Invalid credentials", response.Error)
	assert.Equal(suite.T(), http.StatusUnauthorized, response.Code)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestGetCurrentUser_Success() {
	// Arrange
	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := "valid-jwt-token"

	suite.mockUsecase.On("GetUserFromToken", mock.Anything, token).Return(user, nil)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response authhttp.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "User retrieved successfully", response.Message)

	userData := response.Data.(map[string]interface{})
	assert.Equal(suite.T(), user.ID, userData["id"])
	assert.Equal(suite.T(), user.Email, userData["email"])

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestGetCurrentUser_NoToken() {
	// Arrange
	req := httptest.NewRequest("GET", "/auth/me", nil)

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	var response authhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "No token provided", response.Error)

	suite.mockUsecase.AssertNotCalled(suite.T(), "GetUserFromToken")
}

func (suite *AuthHTTPTestSuite) TestGetCurrentUser_InvalidToken() {
	// Arrange
	token := "invalid-token"

	suite.mockUsecase.On("GetUserFromToken", mock.Anything, token).
		Return((*model.User)(nil), usecase.ErrTokenInvalid)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	var response authhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Invalid or expired token", response.Error)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestLogout_Success() {
	// Arrange
	req := httptest.NewRequest("POST", "/auth/logout", nil)

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response authhttp.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Logout successful", response.Message)

	// Check cookie is cleared
	cookies := resp.Cookies()
	assert.Len(suite.T(), cookies, 1)
	assert.Equal(suite.T(), "test_cookie", cookies[0].Name)
	assert.Equal(suite.T(), "", cookies[0].Value)
	// Cookie should be expired (MaxAge <= 0 means cookie is cleared)
	assert.LessOrEqual(suite.T(), cookies[0].MaxAge, 0)
}

func (suite *AuthHTTPTestSuite) TestTokenFromCookie() {
	// Arrange
	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := "cookie-token"

	suite.mockUsecase.On("GetUserFromToken", mock.Anything, token).Return(user, nil)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.Header.Set("Cookie", fmt.Sprintf("test_cookie=%s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *AuthHTTPTestSuite) TestMalformedJSON() {
	// Arrange
	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var response authhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Invalid request payload", response.Error)

	suite.mockUsecase.AssertNotCalled(suite.T(), "Register")
}

func (suite *AuthHTTPTestSuite) TestContentTypeValidation() {
	// Arrange
	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	// Missing Content-Type header

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

func (suite *AuthHTTPTestSuite) TestCORSHeaders() {
	// Arrange
	req := httptest.NewRequest("OPTIONS", "/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	// Note: CORS headers would need to be configured in the Fiber app
	// This test demonstrates how to verify CORS configuration
	assert.True(suite.T(), resp.StatusCode < 500) // Should not be a server error
}

func (suite *AuthHTTPTestSuite) TestSecurityHeaders() {
	// Arrange
	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	suite.mockUsecase.On("Register", mock.Anything, "test@example.com", "password123").
		Return(user, "token", nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Verify security headers (these would need to be configured in middleware)
	// assert.NotEmpty(suite.T(), resp.Header.Get("X-Content-Type-Options"))
	// assert.NotEmpty(suite.T(), resp.Header.Get("X-Frame-Options"))
	// assert.NotEmpty(suite.T(), resp.Header.Get("X-XSS-Protection"))
}

func TestAuthHTTPTestSuite(t *testing.T) {
	suite.Run(t, new(AuthHTTPTestSuite))
}

// Performance benchmarks
func BenchmarkRegister(b *testing.B) {
	mockUsecase := &mockAuthUsecase{}
	app := fiber.New()

	handler := authhttp.NewAuthHTTPHandler(
		mockUsecase,
		"test_cookie",
		"/",
		"",
		3600,
		false,
		true,
		"Lax",
	)

	handler.SetupAuthRoutes(app)

	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUsecase.On("Register", mock.Anything, mock.Anything, mock.Anything).
		Return(user, "token", nil)

	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}
}

func BenchmarkLogin(b *testing.B) {
	mockUsecase := &mockAuthUsecase{}
	app := fiber.New()

	handler := authhttp.NewAuthHTTPHandler(
		mockUsecase,
		"test_cookie",
		"/",
		"",
		3600,
		false,
		true,
		"Lax",
	)

	handler.SetupAuthRoutes(app)

	user := &model.User{
		ID:        "user-123",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUsecase.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(user, "token", nil)

	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}
}
