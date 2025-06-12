package http_test

import (
	"context"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/stretchr/testify/mock"
)

// mockAuthUsecase is a shared mock type for the AuthUsecaseInterface
type mockAuthUsecase struct {
	mock.Mock
}

// --- Mock Implementations for usecase.AuthUsecaseInterface ---

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

func (m *mockAuthUsecase) RefreshToken(ctx context.Context, refreshTokenString string) (*usecase.AuthResponse, error) {
	args := m.Called(ctx, refreshTokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AuthResponse), args.Error(1)
}

func (m *mockAuthUsecase) GetUserByID(ctx context.Context, userID string, projectID string) (*model.User, error) {
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

func (m *mockAuthUsecase) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockAuthUsecase) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Claims), args.Error(1)
}

// Add any missing methods that might be required by the AuthUsecaseInterface
func (m *mockAuthUsecase) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}

func (m *mockAuthUsecase) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}

// Ensure mockAuthUsecase implements all methods of AuthUsecaseInterface
var _ usecase.AuthUsecaseInterface = (*mockAuthUsecase)(nil)
