package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email is already taken")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidProjectID   = errors.New("invalid project ID format")
	ErrInvalidDatabaseID  = errors.New("invalid database ID format")
	ErrWeakPassword       = errors.New("password does not meet strength requirements")
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
	emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// AuthUsecaseInterface defines the contract for authentication use cases.
type AuthUsecaseInterface interface {
	Register(ctx context.Context, req RegisterRequest) (*model.User, string, error)
	Login(ctx context.Context, req LoginRequest) (*model.User, string, error)
	Logout(ctx context.Context, tokenString string) error
	ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error)
	RefreshToken(ctx context.Context, tokenString string) (string, error)
	GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error)
	GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error)
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
	TenantID   string `json:"tenantId,omitempty"`
	FirstName  string `json:"firstName" validate:"required"`
	LastName   string `json:"lastName" validate:"required"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
}

// AuthUsecase implements the authentication logic.
type AuthUsecase struct {
	repo     repository.AuthRepository
	tokenSvc repository.TokenService
	config   *config.Config
}

// NewAuthUsecase creates a new instance of AuthUsecase.
func NewAuthUsecase(
	repo repository.AuthRepository,
	tokenSvc repository.TokenService,
	cfg *config.Config,
) *AuthUsecase {
	return &AuthUsecase{
		repo:     repo,
		tokenSvc: tokenSvc,
		config:   cfg,
	}
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

// Register creates a new user with Firestore project association
func (uc *AuthUsecase) Register(ctx context.Context, req RegisterRequest) (*model.User, string, error) {
	// Validate email
	if err := uc.validateEmail(req.Email); err != nil {
		return nil, "", err
	}

	// Validate password
	if err := uc.validatePassword(req.Password); err != nil {
		return nil, "", err
	}

	// Validate project ID
	if err := uc.validateProjectID(req.ProjectID); err != nil {
		return nil, "", err
	}

	// Validate database ID
	if err := uc.validateDatabaseID(req.DatabaseID); err != nil {
		return nil, "", err
	}

	// Validate required fields
	if strings.TrimSpace(req.FirstName) == "" {
		return nil, "", fmt.Errorf("firstName is required")
	}
	if strings.TrimSpace(req.LastName) == "" {
		return nil, "", fmt.Errorf("lastName is required")
	}

	// Check if user already exists in this project
	existingUser, err := uc.repo.GetUserByEmail(ctx, req.Email, req.ProjectID)
	if err != nil && err != ErrUserNotFound {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with Firestore project context
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		TenantID:     req.TenantID,
		ProjectID:    req.ProjectID,
		DatabaseID:   req.DatabaseID,
		PasswordHash: string(hashedPassword),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		AvatarURL:    strings.TrimSpace(req.AvatarURL),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Validate user fields
	if err := user.ValidateFields(); err != nil {
		return nil, "", err
	}

	err = uc.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token with Firestore context
	token, err := uc.tokenSvc.GenerateToken(ctx, user.ID, user.Email, user.TenantID, user.ProjectID, user.DatabaseID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Clear password hash before returning
	user.PasswordHash = ""
	return user, token, nil
}

// Login authenticates a user with Firestore project context
func (uc *AuthUsecase) Login(ctx context.Context, req LoginRequest) (*model.User, string, error) {
	// Validate email
	if err := uc.validateEmail(req.Email); err != nil {
		return nil, "", err
	}

	// Validate project ID
	if err := uc.validateProjectID(req.ProjectID); err != nil {
		return nil, "", err
	}

	// Validate database ID
	if err := uc.validateDatabaseID(req.DatabaseID); err != nil {
		return nil, "", err
	}

	// Get user by email within project context
	user, err := uc.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)), req.ProjectID)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Verify database ID matches
	if user.DatabaseID != req.DatabaseID {
		return nil, "", ErrInvalidCredentials
	}

	// Generate token with Firestore context
	token, err := uc.tokenSvc.GenerateToken(ctx, user.ID, user.Email, user.TenantID, user.ProjectID, user.DatabaseID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Clear password hash before returning
	user.PasswordHash = ""
	return user, token, nil
}

// Logout invalidates a session
func (uc *AuthUsecase) Logout(ctx context.Context, tokenString string) error {
	// Validate token to ensure it's legitimate
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return ErrTokenInvalid
	}

	// Delete user sessions
	err = uc.repo.DeleteUserSessions(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// ValidateToken validates a JWT string
func (uc *AuthUsecase) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// RefreshToken generates a new token for valid existing token
func (uc *AuthUsecase) RefreshToken(ctx context.Context, tokenString string) (string, error) {
	// Validate current token
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return "", ErrTokenInvalid
	}

	// Get user to ensure they still exist
	user, err := uc.repo.GetUserByID(ctx, claims.UserID, claims.ProjectID)
	if err != nil {
		return "", ErrUserNotFound
	}

	// Generate new token
	newToken, err := uc.tokenSvc.GenerateToken(ctx, user.ID, user.Email, user.TenantID, user.ProjectID, user.DatabaseID)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	return newToken, nil
}

// GetUserFromToken validates a token and fetches the associated user
func (uc *AuthUsecase) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
	// Validate token
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Get user by ID with project context
	user, err := uc.repo.GetUserByID(ctx, claims.UserID, claims.ProjectID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	// Clear password hash
	user.PasswordHash = ""
	return user, nil
}

// GetUserByID retrieves a user by ID with project context
func (uc *AuthUsecase) GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	// Get user by ID with project context
	user, err := uc.repo.GetUserByID(ctx, userID, projectID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Clear password hash for security
	user.PasswordHash = ""
	return user, nil
}

// Ensure AuthUsecase implements AuthUsecaseInterface
var _ AuthUsecaseInterface = (*AuthUsecase)(nil)
