package http_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authhttp "firestore-clone/internal/auth/adapter/http"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MiddlewareTestSuite struct {
	suite.Suite
	app        *fiber.App
	mockUC     *mockAuthUsecase
	middleware *authhttp.AuthMiddleware
}

func (suite *MiddlewareTestSuite) SetupTest() {
	suite.mockUC = &mockAuthUsecase{}
	suite.middleware = authhttp.NewAuthMiddleware(suite.mockUC, "auth_cookie")
	suite.app = fiber.New()
}

func (suite *MiddlewareTestSuite) TestRequireAuth_Success() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		userID, exists := authhttp.GetUserID(c)
		if !exists {
			return c.Status(500).JSON(fiber.Map{"error": "user_id not found"})
		}
		return c.JSON(fiber.Map{"user_id": userID, "authenticated": true})
	})

	token := "valid-token"
	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, token).Return(claims, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestRequireAuth_NoToken() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	suite.mockUC.AssertNotCalled(suite.T(), "ValidateToken")
}

func (suite *MiddlewareTestSuite) TestRequireAuth_InvalidToken() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	token := "invalid-token"
	suite.mockUC.On("ValidateToken", mock.Anything, token).
		Return((*repository.Claims)(nil), usecase.ErrTokenInvalid)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestRequireAuth_TokenFromCookie() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		userID, _ := authhttp.GetUserID(c)
		return c.JSON(fiber.Map{"user_id": userID})
	})

	token := "cookie-token"
	claims := &repository.Claims{
		UserID: "user-456",
		Email:  "cookie@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, token).Return(claims, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", fmt.Sprintf("auth_cookie=%s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestOptionalAuth_WithValidToken() {
	// Arrange
	suite.app.Use(suite.middleware.OptionalAuth())
	suite.app.Get("/optional", func(c *fiber.Ctx) error {
		isAuth := authhttp.IsAuthenticated(c)
		userID, hasUserID := authhttp.GetUserID(c)
		return c.JSON(fiber.Map{
			"authenticated": isAuth,
			"user_id":       userID,
			"has_user_id":   hasUserID,
		})
	})

	token := "valid-token"
	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, token).Return(claims, nil)

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestOptionalAuth_WithoutToken() {
	// Arrange
	suite.app.Use(suite.middleware.OptionalAuth())
	suite.app.Get("/optional", func(c *fiber.Ctx) error {
		isAuth := authhttp.IsAuthenticated(c)
		userID, hasUserID := authhttp.GetUserID(c)
		return c.JSON(fiber.Map{
			"authenticated": isAuth,
			"user_id":       userID,
			"has_user_id":   hasUserID,
		})
	})

	req := httptest.NewRequest("GET", "/optional", nil)

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertNotCalled(suite.T(), "ValidateToken")
}

func (suite *MiddlewareTestSuite) TestOptionalAuth_WithInvalidToken() {
	// Arrange
	suite.app.Use(suite.middleware.OptionalAuth())
	suite.app.Get("/optional", func(c *fiber.Ctx) error {
		isAuth := authhttp.IsAuthenticated(c)
		return c.JSON(fiber.Map{"authenticated": isAuth})
	})

	token := "invalid-token"
	suite.mockUC.On("ValidateToken", mock.Anything, token).
		Return((*repository.Claims)(nil), usecase.ErrTokenInvalid)

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestTokenExtraction_Priority() {
	// Arrange - Bearer token should take priority over cookie
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	bearerToken := "bearer-token"
	cookieToken := "cookie-token"

	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	// Should call with bearer token, not cookie token
	suite.mockUC.On("ValidateToken", mock.Anything, bearerToken).Return(claims, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Set("Cookie", fmt.Sprintf("auth_cookie=%s", cookieToken))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestHelperFunctions() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		userID, hasUserID := authhttp.GetUserID(c)
		email, hasEmail := authhttp.GetUserEmail(c)
		isAuth := authhttp.IsAuthenticated(c)

		return c.JSON(fiber.Map{
			"user_id":       userID,
			"has_user_id":   hasUserID,
			"email":         email,
			"has_email":     hasEmail,
			"authenticated": isAuth,
		})
	})

	token := "valid-token"
	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, token).Return(claims, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *MiddlewareTestSuite) TestMiddleware_ConcurrentRequests() {
	// Arrange
	suite.app.Use(suite.middleware.RequireAuth())
	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		userID, _ := authhttp.GetUserID(c)
		return c.JSON(fiber.Map{"user_id": userID})
	})

	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, mock.Anything).Return(claims, nil)

	// Act - Simulate concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			token := fmt.Sprintf("token-%d", id)
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp, err := suite.app.Test(req)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func (suite *MiddlewareTestSuite) TestErrorHandling_ContextValues() {
	// Arrange
	suite.app.Use(suite.middleware.OptionalAuth())
	suite.app.Get("/test", func(c *fiber.Ctx) error {
		// Test when user is not authenticated
		userID, hasUserID := authhttp.GetUserID(c)
		email, hasEmail := authhttp.GetUserEmail(c)
		isAuth := authhttp.IsAuthenticated(c)

		assert.Empty(suite.T(), userID)
		assert.False(suite.T(), hasUserID)
		assert.Empty(suite.T(), email)
		assert.False(suite.T(), hasEmail)
		assert.False(suite.T(), isAuth)

		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)

	// Act
	resp, err := suite.app.Test(req)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

// Benchmark tests
func BenchmarkRequireAuth_ValidToken(b *testing.B) {
	mockUC := &mockAuthUsecase{}
	middleware := authhttp.NewAuthMiddleware(mockUC, "auth_cookie")
	app := fiber.New()

	app.Use(middleware.RequireAuth())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	claims := &repository.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	mockUC.On("ValidateToken", mock.Anything, mock.Anything).Return(claims, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		app.Test(req)
	}
}

func BenchmarkOptionalAuth_NoToken(b *testing.B) {
	mockUC := &mockAuthUsecase{}
	middleware := authhttp.NewAuthMiddleware(mockUC, "auth_cookie")
	app := fiber.New()

	app.Use(middleware.OptionalAuth())
	app.Get("/optional", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/optional", nil)
		app.Test(req)
	}
}
