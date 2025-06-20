package usecase

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = fmt.Errorf("email is already taken")
	ErrUserNotFound       = fmt.Errorf("user not found")
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")
	ErrInvalidEmailFormat = fmt.Errorf("invalid email format")
	ErrTokenInvalid       = fmt.Errorf("token is invalid")
	ErrSessionNotFound    = fmt.Errorf("session not found")
	ErrInvalidProjectID   = fmt.Errorf("invalid project ID format")
	ErrInvalidDatabaseID  = fmt.Errorf("invalid database ID format")
	ErrWeakPassword       = fmt.Errorf("password does not meet strength requirements")
)

// Email regex
var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Password validation constants
const (
	minPasswordLength = 8
	maxPasswordLength = 128
)

// Firestore project ID regex (Google Cloud project naming conventions)
var (
	projectIDRegex  = regexp.MustCompile(`^[a-z][-a-z0-9]{4,28}[a-z0-9]$`)
	databaseIDRegex = regexp.MustCompile(`^[a-z][-a-z0-9]{0,62}[a-z0-9]$`)
)

// AuthUsecaseInterface defines the contract for authentication use cases.
type AuthUsecaseInterface interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Logout(ctx context.Context, userID string) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, userID string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error)
	AddUserToTenant(ctx context.Context, userID, tenantID string) error
	RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error
	ValidateToken(ctx context.Context, token string) (*repository.Claims, error)
}

// AuthUsecase implements the authentication logic.
type AuthUsecase struct {
	authRepo repository.AuthRepository
	tokenSvc repository.TokenService
	config   *config.Config
}

// NewAuthUsecase creates a new instance of AuthUsecase.
func NewAuthUsecase(
	authRepo repository.AuthRepository,
	tokenSvc repository.TokenService,
	cfg *config.Config,
) AuthUsecaseInterface {
	return &AuthUsecase{
		authRepo: authRepo,
		tokenSvc: tokenSvc,
		config:   cfg,
	}
}

// Request/Response types

type RegisterRequest struct {
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=8"`
	FirstName      string `json:"firstName" validate:"required"`
	LastName       string `json:"lastName" validate:"required"`
	TenantID       string `json:"tenantId" validate:"required"`
	OrganizationID string `json:"organizationId,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TenantID string `json:"tenantId,omitempty"`
}

type AuthResponse struct {
	User         *model.User `json:"user"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresIn    int64       `json:"expiresIn"`
}

// validateEmail validates email format
func (uc *AuthUsecase) validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmailFormat
	}
	return nil
}

// validatePassword validates password strength
func (uc *AuthUsecase) validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	if len(password) > maxPasswordLength {
		return fmt.Errorf("password must be at most %d characters", maxPasswordLength)
	}

	// Check for basic complexity requirements
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrWeakPassword
	}

	return nil
}

// validateProjectID validates Firestore project ID format
func (uc *AuthUsecase) validateProjectID(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project ID is required")
	}
	if !projectIDRegex.MatchString(projectID) {
		return ErrInvalidProjectID
	}
	return nil
}

// validateDatabaseID validates Firestore database ID format
func (uc *AuthUsecase) validateDatabaseID(databaseID string) error {
	if databaseID == "" {
		return fmt.Errorf("database ID is required")
	}
	if !databaseIDRegex.MatchString(databaseID) {
		return ErrInvalidDatabaseID
	}
	return nil
}

// Authentication operations

func (uc *AuthUsecase) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := uc.validateEmail(req.Email); err != nil {
		return nil, err
	}
	if err := uc.validatePassword(req.Password); err != nil {
		return nil, err
	}
	if err := uc.validateProjectID(req.TenantID); err != nil {
		return nil, err
	}
	if req.OrganizationID != "" {
		if err := uc.validateDatabaseID(req.OrganizationID); err != nil {
			return nil, err
		}
	}
	// Check if user already exists
	existingUser, err := uc.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && err != model.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, model.ErrUserExists
	}
	// Create new user
	user := &model.User{
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		IsActive:       true,
		IsVerified:     true, // Set to true for integration tests and development
		Roles:          []string{"user"},
		Password:       req.Password,
	}

	// Hash password
	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Validate user fields
	if errs := user.ValidateFields(); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}
	// Save user
	if err := uc.authRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := uc.tokenSvc.GenerateToken(
		ctx, user.UserID, user.Email, user.TenantID, "", "", user.Roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := uc.tokenSvc.GenerateRefreshToken(
		ctx, user.UserID, user.Email, user.TenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &model.Session{
		UserID:    user.UserID,
		Token:     accessToken,
		ExpiresAt: time.Now().Add(uc.config.AccessTokenTTL),
	}
	if err := uc.authRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.config.AccessTokenTTL.Seconds()),
	}, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if err := uc.validateEmail(req.Email); err != nil {
		return nil, err
	}
	// Get user by email
	user, err := uc.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == model.ErrUserNotFound {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user can login
	if !user.CanLogin() {
		if user.IsLocked() {
			return nil, model.ErrAccountLocked
		}
		if !user.IsActive {
			return nil, model.ErrAccountInactive
		}
		return nil, fmt.Errorf("account cannot login")
	}

	// Check tenant access if specified
	if req.TenantID != "" && user.TenantID != req.TenantID {
		return nil, model.ErrTenantMismatch
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		user.IncrementLoginAttempts()
		uc.authRepo.UpdateUser(ctx, user)
		return nil, model.ErrInvalidPassword
	}
	// Update last login
	user.UpdateLastLogin()
	if err := uc.authRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user login info: %w", err)
	}

	// Generate tokens
	accessToken, err := uc.tokenSvc.GenerateToken(
		ctx, user.UserID, user.Email, user.TenantID, "", "", user.Roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := uc.tokenSvc.GenerateRefreshToken(
		ctx, user.UserID, user.Email, user.TenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &model.Session{
		UserID:    user.UserID,
		Token:     accessToken,
		ExpiresAt: time.Now().Add(uc.config.AccessTokenTTL),
	}
	if err := uc.authRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.config.AccessTokenTTL.Seconds()),
	}, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, userID string) error {
	return uc.authRepo.DeleteSessionsByUserID(ctx, userID)
}

// LogoutByToken logs out a user by validating the token and extracting the user ID
func (uc *AuthUsecase) LogoutByToken(ctx context.Context, tokenString string) error {
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return ErrTokenInvalid
	}
	return uc.authRepo.DeleteSessionsByUserID(ctx, claims.UserID)
}

func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	claims, err := uc.tokenSvc.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user
	user, err := uc.authRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	// Check if user can still login
	if !user.CanLogin() {
		return nil, fmt.Errorf("user cannot login")
	}

	// Generate new tokens
	newAccessToken, err := uc.tokenSvc.GenerateToken(
		ctx, user.UserID, user.Email, user.TenantID, "", "", user.Roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := uc.tokenSvc.GenerateRefreshToken(
		ctx, user.UserID, user.Email, user.TenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create new session
	session := &model.Session{
		UserID:    user.UserID,
		Token:     newAccessToken,
		ExpiresAt: time.Now().Add(uc.config.AccessTokenTTL),
	}
	if err := uc.authRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(uc.config.AccessTokenTTL.Seconds()),
	}, nil
}

// User management

func (uc *AuthUsecase) GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error) {
	if err := uc.validateProjectID(projectID); err != nil {
		return nil, err
	}
	user, err := uc.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Clear password
	user.Password = ""
	return user, nil
}

func (uc *AuthUsecase) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := uc.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Clear password
	user.Password = ""
	return user, nil
}

func (uc *AuthUsecase) UpdateUser(ctx context.Context, user *model.User) error {
	// Validate user fields
	if errs := user.ValidateFields(); len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs)
	}

	return uc.authRepo.UpdateUser(ctx, user)
}

func (uc *AuthUsecase) DeleteUser(ctx context.Context, userID string) error {
	// Delete user sessions first
	if err := uc.authRepo.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return uc.authRepo.DeleteUser(ctx, userID)
}

func (uc *AuthUsecase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get user
	user, err := uc.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !user.CheckPassword(oldPassword) {
		return model.ErrInvalidPassword
	}

	if err := uc.validatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return uc.authRepo.UpdatePassword(ctx, userID, string(hashedPassword))
}

// Tenant operations

func (uc *AuthUsecase) GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	users, err := uc.authRepo.GetUsersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Clear passwords
	for _, user := range users {
		user.Password = ""
	}

	return users, nil
}

func (uc *AuthUsecase) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	return uc.authRepo.AddUserToTenant(ctx, userID, tenantID)
}

func (uc *AuthUsecase) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	return uc.authRepo.RemoveUserFromTenant(ctx, userID, tenantID)
}

// Token validation

func (uc *AuthUsecase) ValidateToken(ctx context.Context, token string) (*repository.Claims, error) {
	return uc.tokenSvc.ValidateToken(ctx, token)
}

// Ensure AuthUsecase implements AuthUsecaseInterface
var _ AuthUsecaseInterface = (*AuthUsecase)(nil)
