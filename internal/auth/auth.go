package auth

import (
	"fmt"

	authhttp "firestore-clone/internal/auth/adapter/http"
	"firestore-clone/internal/auth/adapter/persistence/mongodb"
	"firestore-clone/internal/auth/adapter/security"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// AuthModule represents the complete authentication module
type AuthModule struct {
	repository repository.AuthRepository
	tokenSvc   repository.TokenService
	usecase    usecase.AuthUsecaseInterface
	handler    *authhttp.AuthHTTPHandler
	config     *config.Config
}

// NewAuthModule creates a new authentication module instance
func NewAuthModule(db *mongo.Database, cfg *config.Config) (*AuthModule, error) {
	// Initialize repository
	authRepo, err := mongodb.NewMongoAuthRepository(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth repository: %w", err)
	}

	// Initialize token service
	tokenSvc, err := security.NewJWTokenService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create token service: %w", err)
	}

	// Initialize usecase
	authUsecase := usecase.NewAuthUsecase(authRepo, tokenSvc, cfg)

	// Initialize HTTP handler
	handler := authhttp.NewAuthHTTPHandler(
		authUsecase,
		cfg.CookieName,
		cfg.CookiePath,
		cfg.CookieDomain,
		int(cfg.AccessTokenTTL.Seconds()),
		cfg.CookieSecure,
		cfg.CookieHTTPOnly,
		cfg.CookieSameSite,
	)

	return &AuthModule{
		repository: authRepo,
		tokenSvc:   tokenSvc,
		usecase:    authUsecase,
		handler:    handler,
		config:     cfg,
	}, nil
}

// RegisterRoutes registers authentication routes with the provided router
func (am *AuthModule) RegisterRoutes(router fiber.Router) {
	middleware := am.GetMiddleware()
	am.handler.SetupAuthRoutesWithMiddleware(router, middleware)
}

// GetUsecase returns the auth usecase for external access
func (am *AuthModule) GetUsecase() usecase.AuthUsecaseInterface {
	return am.usecase
}

// GetMiddleware returns the auth middleware
func (am *AuthModule) GetMiddleware() *authhttp.AuthMiddleware {
	return authhttp.NewAuthMiddleware(am.usecase, am.config.CookieName)
}

// Stop performs cleanup when the module is shut down
func (am *AuthModule) Stop() error {
	// Cleanup resources if needed
	// For now, no specific cleanup is required
	return nil
}

// InitAuthModule initializes the authentication module and registers routes
// Deprecated: Use NewAuthModule instead
func InitAuthModule(app *fiber.App, db *mongo.Database, cfg *config.Config) error {
	module, err := NewAuthModule(db, cfg)
	if err != nil {
		return err
	}

	module.RegisterRoutes(app)
	return nil
}
