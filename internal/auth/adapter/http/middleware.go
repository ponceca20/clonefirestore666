package http

import (
	"strings"

	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware provides authentication middleware for Fiber
type AuthMiddleware struct {
	usecase    usecase.AuthUsecaseInterface
	cookieName string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(uc usecase.AuthUsecaseInterface, cookieName string) *AuthMiddleware {
	return &AuthMiddleware{
		usecase:    uc,
		cookieName: cookieName,
	}
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := m.extractToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "Authentication required",
				Message: "No authentication token provided",
				Code:    fiber.StatusUnauthorized,
			})
		}

		// Validate token
		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "Invalid token",
				Message: "The provided authentication token is invalid or expired",
				Code:    fiber.StatusUnauthorized,
			})
		}

		// Store user info in context for downstream handlers
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("token", token)

		return c.Next()
	}
}

// OptionalAuth middleware that optionally validates authentication
// Does not block if no token is provided, but validates if present
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := m.extractToken(c)
		if token != "" {
			// Validate token if present
			claims, err := m.usecase.ValidateToken(c.Context(), token)
			if err == nil {
				// Store user info in context if token is valid
				c.Locals("user_id", claims.UserID)
				c.Locals("user_email", claims.Email)
				c.Locals("token", token)
				c.Locals("authenticated", true)
			} else {
				c.Locals("authenticated", false)
			}
		} else {
			c.Locals("authenticated", false)
		}

		return c.Next()
	}
}

// extractToken extracts the token from Authorization header or cookie
func (m *AuthMiddleware) extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	return c.Cookies(m.cookieName)
}

// GetUserID helper function to get user ID from context
func GetUserID(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals("user_id").(string)
	return userID, ok
}

// GetUserEmail helper function to get user email from context
func GetUserEmail(c *fiber.Ctx) (string, bool) {
	email, ok := c.Locals("user_email").(string)
	return email, ok
}

// IsAuthenticated helper function to check if user is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	auth, ok := c.Locals("authenticated").(bool)
	return ok && auth
}
