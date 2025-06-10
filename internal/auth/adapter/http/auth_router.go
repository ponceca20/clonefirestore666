package http

import (
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/usecase"
	"firestore-clone/internal/shared/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthHTTPHandler handles HTTP requests for authentication
type AuthHTTPHandler struct {
	usecase        usecase.AuthUsecaseInterface
	cookieName     string
	cookiePath     string
	cookieDomain   string
	cookieMaxAge   int
	cookieSecure   bool
	cookieHTTPOnly bool
	cookieSameSite string
}

// NewAuthHTTPHandler creates a new authentication HTTP handler
func NewAuthHTTPHandler(
	uc usecase.AuthUsecaseInterface,
	cookieName, cookiePath, cookieDomain string,
	cookieMaxAge int,
	cookieSecure, cookieHTTPOnly bool,
	cookieSameSite string,
) *AuthHTTPHandler {
	return &AuthHTTPHandler{
		usecase:        uc,
		cookieName:     cookieName,
		cookiePath:     cookiePath,
		cookieDomain:   cookieDomain,
		cookieMaxAge:   cookieMaxAge,
		cookieSecure:   cookieSecure,
		cookieHTTPOnly: cookieHTTPOnly,
		cookieSameSite: cookieSameSite,
	}
}

// SetupAuthRoutesWithMiddleware sets up authentication routes with middleware
func (h *AuthHTTPHandler) SetupAuthRoutesWithMiddleware(router fiber.Router, middleware *AuthMiddleware) {
	// Public routes (no authentication required)
	router.Post("/register", h.Register)
	router.Post("/login", h.Login)
	router.Post("/refresh", h.RefreshToken)

	// Protected routes (authentication required)
	protected := router.Group("/", middleware.Protect())
	protected.Post("/logout", h.Logout)
	protected.Get("/me", h.GetCurrentUser)
	protected.Put("/me", h.UpdateCurrentUser)
	protected.Post("/change-password", h.ChangePassword)

	// Admin routes (for tenant management)
	admin := router.Group("/admin", middleware.RequireRole("admin"))
	admin.Get("/users", h.ListUsers)
	admin.Get("/users/:userId", h.GetUser)
	admin.Delete("/users/:userId", h.DeleteUser)
}

// Register handles user registration
func (h *AuthHTTPHandler) Register(c *fiber.Ctx) error {
	var req usecase.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Add tenant context if available
	if tenantID := c.Get("X-Tenant-ID"); tenantID != "" {
		req.TenantID = tenantID
	}
	if orgID := c.Get("X-Organization-ID"); orgID != "" {
		req.OrganizationID = orgID
	}
	response, err := h.usecase.Register(c.Context(), req)
	if err != nil {
		// Handle specific error types
		if err == model.ErrUserExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set cookie
	h.setCookie(c, response.AccessToken)

	return c.Status(fiber.StatusCreated).JSON(response)
}

// Login handles user login
func (h *AuthHTTPHandler) Login(c *fiber.Ctx) error {
	var req usecase.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Add tenant context if available
	if tenantID := c.Get("X-Tenant-ID"); tenantID != "" {
		req.TenantID = tenantID
	}
	response, err := h.usecase.Login(c.Context(), req)
	if err != nil {
		// Handle specific error types
		switch err {
		case model.ErrUserNotFound, model.ErrInvalidPassword:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		case model.ErrAccountLocked:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Account is locked",
			})
		case model.ErrAccountInactive:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Account is inactive",
			})
		case model.ErrTenantMismatch:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied for this tenant",
			})
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	// Set cookie
	h.setCookie(c, response.AccessToken)

	return c.JSON(response)
}

// Logout handles user logout
func (h *AuthHTTPHandler) Logout(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c.Context())
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	err = h.usecase.Logout(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Clear cookie
	h.clearCookie(c)

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
func (h *AuthHTTPHandler) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	response, err := h.usecase.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set new cookie
	h.setCookie(c, response.AccessToken)

	return c.JSON(response)
}

// GetCurrentUser returns current user information
func (h *AuthHTTPHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c.Context())
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	user, err := h.usecase.GetUserByID(c.Context(), userID, "")
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// UpdateCurrentUser updates current user information
func (h *AuthHTTPHandler) UpdateCurrentUser(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c.Context())
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get current user
	user, err := h.usecase.GetUserByID(c.Context(), userID, "")
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Parse update request
	var updateReq struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Phone     string `json:"phone"`
	}
	if err := c.BodyParser(&updateReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update user fields
	if updateReq.FirstName != "" {
		user.FirstName = updateReq.FirstName
	}
	if updateReq.LastName != "" {
		user.LastName = updateReq.LastName
	}
	if updateReq.Phone != "" {
		user.Phone = updateReq.Phone
	}

	err = h.usecase.UpdateUser(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// ChangePassword handles password change
func (h *AuthHTTPHandler) ChangePassword(c *fiber.Ctx) error {
	userID, err := utils.GetUserIDFromContext(c.Context())
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.usecase.ChangePassword(c.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// ListUsers lists users (admin only)
func (h *AuthHTTPHandler) ListUsers(c *fiber.Ctx) error {
	tenantID, err := utils.GetTenantIDFromContext(c.Context())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tenant ID required",
		})
	}

	users, err := h.usecase.GetUsersByTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": len(users),
	})
}

// GetUser gets a specific user (admin only)
func (h *AuthHTTPHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID required",
		})
	}

	user, err := h.usecase.GetUserByID(c.Context(), userID, "")
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// DeleteUser deletes a user (admin only)
func (h *AuthHTTPHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID required",
		})
	}

	err := h.usecase.DeleteUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// Helper methods

func (h *AuthHTTPHandler) setCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     h.cookiePath,
		Domain:   h.cookieDomain,
		MaxAge:   h.cookieMaxAge,
		Secure:   h.cookieSecure,
		HTTPOnly: h.cookieHTTPOnly,
		SameSite: h.cookieSameSite,
		Expires:  time.Now().Add(time.Duration(h.cookieMaxAge) * time.Second),
	})
}

func (h *AuthHTTPHandler) clearCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     h.cookiePath,
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HTTPOnly: h.cookieHTTPOnly,
		SameSite: h.cookieSameSite,
		Expires:  time.Now().Add(-1 * time.Hour),
	})
}
