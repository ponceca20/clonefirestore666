package usecase

import (
	"context"
	"errors"
	"regexp"
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
)

// AuthUsecaseInterface defines the contract for authentication use cases.
type AuthUsecaseInterface interface {
	Register(ctx context.Context, email, password string) (*model.User, string, error)
	Login(ctx context.Context, email, password string) (*model.User, string, error)
	Logout(ctx context.Context, tokenString string) error
	ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error)
	GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error)
}

// emailRegex is a simple regex for email validation.
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

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

// Register creates a new user, hashes their password, and returns the user and a JWT.
func (uc *AuthUsecase) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	// Validate email format
	if !emailRegex.MatchString(email) {
		return nil, "", ErrInvalidEmailFormat
	}

	// Check if user already exists
	existingUser, err := uc.repo.GetUserByEmail(ctx, email)
	if err != nil && err != ErrUserNotFound {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// Create user
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = uc.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, "", err
	}

	// Generate token
	token, err := uc.tokenSvc.GenerateToken(ctx, user.ID, email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user and returns the user and a JWT.
func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	// Get user by email
	user, err := uc.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Generate token
	token, err := uc.tokenSvc.GenerateToken(ctx, user.ID, email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Logout invalidates a session. For stateless JWTs, this is complex.
// Current implementation just validates the token as a placeholder.
// A real implementation might add the token to a denylist.
func (uc *AuthUsecase) Logout(ctx context.Context, tokenString string) error {
	// Validate token to ensure it's legitimate before "logout"
	_, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return err
	}

	// For JWT, we can't really invalidate the token without a blacklist
	// This is a placeholder implementation
	return nil
}

// ValidateToken validates a JWT string.
func (uc *AuthUsecase) ValidateToken(ctx context.Context, tokenString string) (*repository.Claims, error) {
	return uc.tokenSvc.ValidateToken(ctx, tokenString)
}

// GetUserFromToken validates a token and fetches the associated user.
func (uc *AuthUsecase) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
	// Validate token
	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Get user by ID from claims
	user, err := uc.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Ensure AuthUsecase implements AuthUsecaseInterface
var _ AuthUsecaseInterface = (*AuthUsecase)(nil)
