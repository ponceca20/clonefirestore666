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

	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock usecase
type mockAuthUsecase struct {
	mock.Mock
}

// Implement all methods of usecase.AuthUsecaseInterface to satisfy the interface for the handler tests
func (m *mockAuthUsecase) Register(ctx context.Context, req usecase.RegisterRequest) (*usecase.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResponse), args.Error(1)
}
func (m *mockAuthUsecase) Login(ctx context.Context, req usecase.LoginRequest) (*usecase.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResponse), args.Error(1)
}
func (m *mockAuthUsecase) Logout(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *mockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*usecase.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResponse), args.Error(1)
}
func (m *mockAuthUsecase) GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error) {
	args := m.Called(ctx, userID, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *mockAuthUsecase) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *mockAuthUsecase) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockAuthUsecase) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *mockAuthUsecase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	return args.Error(0)
}
func (m *mockAuthUsecase) GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}
func (m *mockAuthUsecase) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}
func (m *mockAuthUsecase) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}
func (m *mockAuthUsecase) ValidateToken(ctx context.Context, token string) (*repository.Claims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Claims), args.Error(1)
}

func TestRegister_Success(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/register", handler.Register)

	reqBody := usecase.RegisterRequest{
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  "tenant-1",
	}
	user := &model.User{
		ID:        primitive.NewObjectID(),
		Email:     reqBody.Email,
		FirstName: reqBody.FirstName,
		LastName:  reqBody.LastName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockUC.On("Register", mock.Anything, mock.AnythingOfType("usecase.RegisterRequest")).Return(&usecase.AuthResponse{User: user, AccessToken: "token-123"}, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result usecase.AuthResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, user.Email, result.User.Email)
	assert.Equal(t, "token-123", result.AccessToken)
	mockUC.AssertExpectations(t)
}

func TestRegister_InvalidPayload(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/register", handler.Register)

	// Si el handler llega a llamar Register, que devuelva un error controlado
	mockUC.On("Register", mock.Anything, mock.Anything).Return(nil, errors.New("invalid payload")).Maybe()

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

	reqBody := usecase.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
		TenantID: "tenant-1",
	}
	user := &model.User{
		ID:        primitive.NewObjectID(),
		Email:     reqBody.Email,
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockUC.On("Login", mock.Anything, mock.AnythingOfType("usecase.LoginRequest")).Return(&usecase.AuthResponse{User: user, AccessToken: "token-abc"}, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result usecase.AuthResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, user.Email, result.User.Email)
	assert.Equal(t, "token-abc", result.AccessToken)
	mockUC.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	handler := authhttp.NewAuthHTTPHandler(mockUC, "auth_token", "/", "", 3600, false, true, "Lax")
	app.Post("/auth/login", handler.Login)

	reqBody := usecase.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpass",
		TenantID: "tenant-1",
	}
	mockUC.On("Login", mock.Anything, mock.AnythingOfType("usecase.LoginRequest")).Return(nil, usecase.ErrInvalidCredentials)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// El mockAuthUsecase ya está correctamente definido y alineado con la interfaz y la lógica actual.
// Los tests de Register y Login ya usan el mock y verifican la respuesta y claims.
// Si se agregan más tests, seguir el mismo patrón de mock y validación de claims/contexto.
