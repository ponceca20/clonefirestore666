package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authhttp "firestore-clone/internal/auth/adapter/http"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock usecase
type mockAuthUsecase struct {
	mock.Mock
}

// Implement all methods of usecase.AuthUsecaseInterface to satisfy the interface for the handler tests
func (m *mockAuthUsecase) Register(ctx context.Context, req usecase.RegisterRequest) (*model.User, string, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}
func (m *mockAuthUsecase) Login(ctx context.Context, req usecase.LoginRequest) (*model.User, string, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
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
func (m *mockAuthUsecase) RefreshToken(ctx context.Context, tokenString string) (string, error) {
	args := m.Called(ctx, tokenString)
	return args.String(0), args.Error(1)
}
func (m *mockAuthUsecase) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *mockAuthUsecase) GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error) {
	args := m.Called(ctx, userID, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func TestRegister_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/register", handler.Register)

	reqBody := authhttp.RegisterRequest{
		Email:      "test@example.com",
		Password:   "Password123!",
		ProjectID:  "testproj",
		DatabaseID: "testdb",
		FirstName:  "Test",
		LastName:   "User",
	}
	user := &model.User{
		ID:        "user-1",
		Email:     reqBody.Email,
		FirstName: reqBody.FirstName,
		LastName:  reqBody.LastName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockUC.On("Register", mock.Anything, mock.AnythingOfType("usecase.RegisterRequest")).Return(user, "token-123", nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result authhttp.AuthResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, user.Email, result.User.Email)
	assert.Equal(t, "token-123", result.Token)
	mockUC.AssertExpectations(t)
}

func TestRegister_InvalidPayload(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/register", handler.Register)

	// Missing required fields
	body := []byte(`{"email":"bademail"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLogin_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/login", handler.Login)

	reqBody := authhttp.LoginRequest{
		Email:      "test@example.com",
		Password:   "Password123!",
		ProjectID:  "testproj",
		DatabaseID: "testdb",
	}
	user := &model.User{
		ID:        "user-1",
		Email:     reqBody.Email,
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockUC.On("Login", mock.Anything, mock.AnythingOfType("usecase.LoginRequest")).Return(user, "token-abc", nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result authhttp.AuthResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, user.Email, result.User.Email)
	assert.Equal(t, "token-abc", result.Token)
	mockUC.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/login", handler.Login)

	reqBody := authhttp.LoginRequest{
		Email:      "test@example.com",
		Password:   "wrongpass",
		ProjectID:  "testproj",
		DatabaseID: "testdb",
	}
	mockUC.On("Login", mock.Anything, mock.AnythingOfType("usecase.LoginRequest")).Return(nil, "", usecase.ErrInvalidCredentials)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
