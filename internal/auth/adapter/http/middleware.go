package http

import (
	"context"
	"regexp"
	"strings"
	"time"

	"firestore-clone/internal/auth/usecase"
	"firestore-clone/internal/shared/contextkeys"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"
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

// FirestorePathParser extracts PROJECT_ID and DATABASE_ID from Firestore paths
type FirestorePathParser struct {
	projectRegex  *regexp.Regexp
	databaseRegex *regexp.Regexp
}

// NewFirestorePathParser creates a new Firestore path parser
func NewFirestorePathParser() *FirestorePathParser {
	return &FirestorePathParser{
		projectRegex:  regexp.MustCompile(`/projects/([^/]+)`),
		databaseRegex: regexp.MustCompile(`/databases/([^/]+)`),
	}
}

// ExtractFirestoreContext extracts Firestore context from path
func (p *FirestorePathParser) ExtractFirestoreContext(path string) (projectID, databaseID string) {
	if matches := p.projectRegex.FindStringSubmatch(path); len(matches) > 1 {
		projectID = matches[1]
	}
	if matches := p.databaseRegex.FindStringSubmatch(path); len(matches) > 1 {
		databaseID = matches[1]
	}
	return projectID, databaseID
}

// CORS middleware with security headers
func (m *AuthMiddleware) CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001,https://your-domain.com",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
}

// SecurityHeaders adds security headers
func (m *AuthMiddleware) SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

// RateLimiter creates rate limiting middleware for auth endpoints
func (m *AuthMiddleware) RateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               10,              // 10 requests
		Expiration:        1 * time.Minute, // per minute
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Get("X-Forwarded-For", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
	})
}

// RequestID middleware
func (m *AuthMiddleware) RequestID() fiber.Handler {
	return requestid.New(requestid.Config{
		Header:     "X-Request-ID",
		ContextKey: string(contextkeys.RequestIDKey),
	})
}

// Protect returns middleware that requires authentication
func (m *AuthMiddleware) Protect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := m.extractToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Add user context using context.WithValue (no utils.WithUserID, etc.)
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.UserEmailKey, claims.Email)
		if claims.TenantID != "" {
			ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
		}
		if claims.OrganizationID != "" {
			ctx = context.WithValue(ctx, contextkeys.OrganizationIDKey, claims.OrganizationID)
		}
		if claims.ProjectID != "" {
			ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
		}
		if claims.DatabaseID != "" {
			ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)
		}

		c.SetUserContext(ctx)
		return c.Next()
	}
}

// RequireRole returns middleware that requires a specific role
func (m *AuthMiddleware) RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if context is already set by Protect() middleware
		ctx := c.UserContext()
		userIDVal := ctx.Value(contextkeys.UserIDKey)

		if userIDVal != nil {
			// Context already set by Protect(), just validate token for roles
			token, err := m.extractToken(c)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Authentication required",
				})
			}

			claims, err := m.usecase.ValidateToken(c.Context(), token)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token",
				})
			}

			if !claims.HasRole(role) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Insufficient permissions",
				})
			}

			return c.Next()
		}

		// Context not set, this middleware is being used standalone
		// Perform full authentication and authorization
		token, err := m.extractToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		if !claims.HasRole(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		// Set context for handlers
		ctx = c.UserContext()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.UserEmailKey, claims.Email)
		if claims.TenantID != "" {
			ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
		}
		if claims.OrganizationID != "" {
			ctx = context.WithValue(ctx, contextkeys.OrganizationIDKey, claims.OrganizationID)
		}
		if claims.ProjectID != "" {
			ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
		}
		if claims.DatabaseID != "" {
			ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)
		}

		c.SetUserContext(ctx)
		return c.Next()
	}
}

// RequirePermission returns middleware that requires a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := m.extractToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		if !claims.HasPermission(permission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		// Add user context using context.WithValue
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.UserEmailKey, claims.Email)
		if claims.TenantID != "" {
			ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
		}
		if claims.OrganizationID != "" {
			ctx = context.WithValue(ctx, contextkeys.OrganizationIDKey, claims.OrganizationID)
		}
		if claims.ProjectID != "" {
			ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
		}
		if claims.DatabaseID != "" {
			ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)
		}

		c.SetUserContext(ctx)
		return c.Next()
	}
}

// RequireAuth middleware that requires authentication with Firestore context injection
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := m.extractToken(c)
		if err != nil || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization token required",
			})
		}

		// Validate token
		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Extract Firestore context from path
		parser := NewFirestorePathParser()
		pathProjectID, pathDatabaseID := parser.ExtractFirestoreContext(c.Path())

		// Verify token project/database matches path context
		if pathProjectID != "" && claims.ProjectID != pathProjectID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Project access denied",
			})
		}
		if pathDatabaseID != "" && claims.DatabaseID != pathDatabaseID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Database access denied",
			})
		}
		// Inject context values
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
		ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)

		// Update Fiber context
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// OptionalAuth middleware that optionally validates authentication
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := m.extractToken(c)
		if err != nil || token == "" {
			return c.Next() // Continue without authentication
		}

		// Validate token if present
		claims, err := m.usecase.ValidateToken(c.Context(), token)
		if err != nil {
			// Invalid token, but continue without authentication
			return c.Next()
		}
		// Inject context values if token is valid
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, claims.ProjectID)
		ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, claims.DatabaseID)

		c.SetUserContext(ctx)
		return c.Next()
	}
}

// FirestoreProjectContext middleware injects Firestore project context from path
func (m *AuthMiddleware) FirestoreProjectContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		parser := NewFirestorePathParser()
		projectID, databaseID := parser.ExtractFirestoreContext(c.Path())
		if projectID != "" || databaseID != "" {
			ctx := c.UserContext()
			if projectID != "" {
				ctx = context.WithValue(ctx, contextkeys.ProjectIDKey, projectID)
			}
			if databaseID != "" {
				ctx = context.WithValue(ctx, contextkeys.DatabaseIDKey, databaseID)
			}
			c.SetUserContext(ctx)
		}

		return c.Next()
	}
}

// extractToken extracts the token from Authorization header or cookie
func (m *AuthMiddleware) extractToken(c *fiber.Ctx) (string, error) {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer "), nil
		}
	}

	// Try cookie
	token := c.Cookies(m.cookieName)
	if token != "" {
		return token, nil
	}

	// Try query parameter (for WebSocket connections)
	token = c.Query("token")
	if token != "" {
		return token, nil
	}

	return "", fiber.NewError(fiber.StatusUnauthorized, "No authentication token found")
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

// GetTenantID helper function to get tenant ID from context -- REMOVED
// func GetTenantID(c *fiber.Ctx) (string, bool) {
// 	tenantID, ok := c.Locals("tenantID").(string)
// 	return tenantID, ok
// }

// IsAuthenticated helper function to check if user is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	auth, ok := c.Locals("authenticated").(bool)
	return ok && auth
}
