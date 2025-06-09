package http

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Firestore project ID validation regex
var (
	projectIDRegex  = regexp.MustCompile(`^[a-z][-a-z0-9]{4,28}[a-z0-9]$`)
	databaseIDRegex = regexp.MustCompile(`^[a-z][-a-z0-9]{0,62}[a-z0-9]$`)
)

// --- DTOs ---

// RegisterRequest defines the structure for user registration with Firestore context.
type RegisterRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
	TenantID   string `json:"tenantId,omitempty"`
	FirstName  string `json:"firstName" validate:"required"`
	LastName   string `json:"lastName" validate:"required"`
	AvatarURL  string `json:"avatarUrl,omitempty" validate:"omitempty,url"`
}

// LoginRequest defines the structure for user login with Firestore context.
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
}

// UserResponse is the DTO for user information.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthResponse is the response for successful authentication (register/login).
type AuthResponse struct {
	User    UserResponse `json:"user"`
	Token   string       `json:"token,omitempty"` // Omitted if only sent in cookie
	Message string       `json:"message"`
}

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a standardized success response.
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// --- HTTP Handler ---

// CookieConfig holds configuration for HTTP cookies.
type CookieConfig struct {
	Name          string
	Path          string
	Domain        string
	MaxAgeSeconds int
	Secure        bool
	HTTPOnly      bool
	SameSite      string
}

// AuthHTTPHandler handles HTTP requests for authentication using Fiber.
type AuthHTTPHandler struct {
	usecase   usecase.AuthUsecaseInterface
	validate  *validator.Validate
	cookieCfg CookieConfig
}

// NewAuthHTTPHandler creates a new instance of AuthHTTPHandler.
func NewAuthHTTPHandler(
	uc usecase.AuthUsecaseInterface,
	cookieName string,
	cookiePath string,
	cookieDomain string,
	cookieMaxAge int, // in seconds
	cookieSecure bool,
	cookieHttpOnly bool,
	cookieSameSite string, // "lax", "strict", "none"
) *AuthHTTPHandler {
	return &AuthHTTPHandler{
		usecase:  uc,
		validate: validator.New(),
		cookieCfg: CookieConfig{
			Name:          cookieName,
			Path:          cookiePath,
			Domain:        cookieDomain,
			MaxAgeSeconds: cookieMaxAge,
			Secure:        cookieSecure,
			HTTPOnly:      cookieHttpOnly,
			SameSite:      cookieSameSite,
		},
	}
}

// SetupAuthRoutes registers the authentication routes with the provided Fiber router.
func (h *AuthHTTPHandler) SetupAuthRoutes(router fiber.Router) {
	authGroup := router.Group("/auth")
	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/logout", h.Logout)
	authGroup.Get("/validate", h.Validate)
	authGroup.Get("/me", h.GetCurrentUser) // Alternative endpoint name for validation
}

// SetupAuthRoutesWithMiddleware registers routes with optional middleware for protected endpoints
func (h *AuthHTTPHandler) SetupAuthRoutesWithMiddleware(router fiber.Router, middleware *AuthMiddleware) {
	authGroup := router.Group("/auth")
	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/logout", h.Logout)

	// Protected endpoints
	if middleware != nil {
		authGroup.Get("/validate", middleware.RequireAuth(), h.Validate)
		authGroup.Get("/me", middleware.RequireAuth(), h.GetCurrentUser)
	} else {
		authGroup.Get("/validate", h.Validate)
		authGroup.Get("/me", h.GetCurrentUser)
	}
}

// Register handles user registration with Firestore project validation.
func (h *AuthHTTPHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload", err)
	}

	if err := h.validate.Struct(&req); err != nil {
		return h.sendValidationErrorResponse(c, err)
	}

	// Validate Firestore project ID format
	if !projectIDRegex.MatchString(req.ProjectID) {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid project ID format. Must follow Google Cloud project naming conventions", nil)
	}

	// Validate Firestore database ID format
	if !databaseIDRegex.MatchString(req.DatabaseID) {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid database ID format", nil)
	}

	// Convert to usecase request format
	ucReq := usecase.RegisterRequest{
		Email:      req.Email,
		Password:   req.Password,
		ProjectID:  req.ProjectID,
		DatabaseID: req.DatabaseID,
		TenantID:   req.TenantID,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		AvatarURL:  req.AvatarURL,
	}

	user, token, err := h.usecase.Register(c.Context(), ucReq)
	if err != nil {
		return h.mapAuthErrorToFiberError(c, err)
	}

	h.setCookie(c, token)

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		User:    h.modelUserToUserResponse(user),
		Token:   token,
		Message: "User registered successfully",
	})
}

// Login handles user login.
func (h *AuthHTTPHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload", err)
	}
	if err := h.validate.Struct(&req); err != nil {
		return h.sendValidationErrorResponse(c, err)
	}

	// Validate Firestore project ID format
	if !projectIDRegex.MatchString(req.ProjectID) {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid project ID format. Must follow Google Cloud project naming conventions", nil)
	}

	// Validate Firestore database ID format
	if !databaseIDRegex.MatchString(req.DatabaseID) {
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid database ID format", nil)
	}

	// Convert to usecase request format
	ucReq := usecase.LoginRequest{
		Email:      req.Email,
		Password:   req.Password,
		ProjectID:  req.ProjectID,
		DatabaseID: req.DatabaseID,
	}
	user, token, err := h.usecase.Login(c.Context(), ucReq)
	if err != nil {
		return h.mapAuthErrorToFiberError(c, err)
	}

	h.setCookie(c, token)

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		User:    h.modelUserToUserResponse(user),
		Token:   token,
		Message: "Login successful",
	})
}

// Logout handles user logout.
func (h *AuthHTTPHandler) Logout(c *fiber.Ctx) error {
	// Clear the authentication cookie
	h.clearCookie(c)

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Logout successful",
	})
}

// Validate handles token validation and returns user info.
// This endpoint would typically be protected by an auth middleware.
func (h *AuthHTTPHandler) Validate(c *fiber.Ctx) error {
	return h.GetCurrentUser(c)
}

// GetCurrentUser returns the current authenticated user.
func (h *AuthHTTPHandler) GetCurrentUser(c *fiber.Ctx) error {
	tokenString := h.extractToken(c)
	if tokenString == "" {
		return h.sendErrorResponse(c, fiber.StatusUnauthorized, "No token provided", nil)
	}

	user, err := h.usecase.GetUserFromToken(c.Context(), tokenString)
	if err != nil {
		return h.sendErrorResponse(c, fiber.StatusUnauthorized, "Invalid or expired token", err)
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "User retrieved successfully",
		Data:    h.modelUserToUserResponse(user),
	})
}

// --- Helper methods ---

// extractToken extracts the token from Authorization header or cookie.
func (h *AuthHTTPHandler) extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	return c.Cookies(h.cookieCfg.Name)
}

// setCookie sets the authentication cookie.
func (h *AuthHTTPHandler) setCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieCfg.Name,
		Value:    token,
		Path:     h.cookieCfg.Path,
		Domain:   h.cookieCfg.Domain,
		MaxAge:   h.cookieCfg.MaxAgeSeconds,
		Secure:   h.cookieCfg.Secure,
		HTTPOnly: h.cookieCfg.HTTPOnly,
		SameSite: h.cookieCfg.SameSite,
	})
}

// clearCookie clears the authentication cookie.
func (h *AuthHTTPHandler) clearCookie(c *fiber.Ctx) {
	// Set the cookie with both MaxAge -1 and an expired date to ensure clearing
	expiredTime := time.Now().Add(-24 * time.Hour)
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieCfg.Name,
		Value:    "",
		Path:     h.cookieCfg.Path,
		Domain:   h.cookieCfg.Domain,
		MaxAge:   -1,
		Expires:  expiredTime,
		Secure:   h.cookieCfg.Secure,
		HTTPOnly: h.cookieCfg.HTTPOnly,
		SameSite: h.cookieCfg.SameSite,
	})
}

// modelUserToUserResponse converts model.User to UserResponse.
func (h *AuthHTTPHandler) modelUserToUserResponse(user *model.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// sendErrorResponse sends a standardized error response.
func (h *AuthHTTPHandler) sendErrorResponse(c *fiber.Ctx, status int, message string, err error) error {
	response := ErrorResponse{
		Error:   message,
		Message: message,
		Code:    status,
	}

	if err != nil && fiber.IsChild() { // Only include error details in development
		response.Error = err.Error()
	}

	return c.Status(status).JSON(response)
}

// sendValidationErrorResponse sends a validation error response.
func (h *AuthHTTPHandler) sendValidationErrorResponse(c *fiber.Ctx, err error) error {
	var validationErrors []string

	if validatorErr, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validatorErr {
			validationErrors = append(validationErrors, h.getValidationErrorMessage(fieldErr))
		}
	}

	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Error:   "Validation failed",
		Message: strings.Join(validationErrors, "; "),
		Code:    fiber.StatusBadRequest,
	})
}

// getValidationErrorMessage returns a human-readable validation error message.
func (h *AuthHTTPHandler) getValidationErrorMessage(err validator.FieldError) string {
	fieldName := strings.ToLower(err.Field())
	switch err.Tag() {
	case "required":
		return fieldName + " is required"
	case "email":
		return fieldName + " must be a valid email address"
	case "min":
		return fieldName + " must be at least " + err.Param() + " characters long"
	default:
		return fieldName + " is invalid"
	}
}

// mapAuthErrorToFiberError traduce errores de negocio a códigos HTTP correctos
func (h *AuthHTTPHandler) mapAuthErrorToFiberError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecase.ErrEmailTaken):
		return h.sendErrorResponse(c, fiber.StatusConflict, "Email is already registered", err)
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return h.sendErrorResponse(c, fiber.StatusUnauthorized, "Invalid credentials", err)
	case errors.Is(err, usecase.ErrUserNotFound):
		return h.sendErrorResponse(c, fiber.StatusUnauthorized, "Invalid credentials", err)
	case errors.Is(err, usecase.ErrInvalidEmailFormat):
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid email format", err)
	case errors.Is(err, usecase.ErrWeakPassword):
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Password does not meet strength requirements", err)
	case errors.Is(err, usecase.ErrInvalidProjectID):
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid project ID format", err)
	case errors.Is(err, usecase.ErrInvalidDatabaseID):
		return h.sendErrorResponse(c, fiber.StatusBadRequest, "Invalid database ID format", err)
	case strings.Contains(err.Error(), "required"):
		return h.sendErrorResponse(c, fiber.StatusBadRequest, err.Error(), err)
	case strings.Contains(err.Error(), "failed to get user"):
		return h.sendErrorResponse(c, fiber.StatusUnauthorized, "Invalid credentials", err)
	case strings.Contains(err.Error(), "failed to check existing user"):
		return h.sendErrorResponse(c, fiber.StatusInternalServerError, "Internal server error", err)
	case strings.Contains(err.Error(), "failed to create user"):
		return h.sendErrorResponse(c, fiber.StatusInternalServerError, "Internal server error", err)
	case strings.Contains(err.Error(), "failed to generate token"):
		return h.sendErrorResponse(c, fiber.StatusInternalServerError, "Internal server error", err)
	default:
		return h.sendErrorResponse(c, fiber.StatusInternalServerError, "Internal server error", err)
	}
}
