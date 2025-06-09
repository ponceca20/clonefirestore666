package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authhttp "firestore-clone/internal/auth/adapter/http"
	repo "firestore-clone/internal/auth/domain/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MiddlewareTestSuite struct {
	suite.Suite
	app        *fiber.App
	mockUC     *mockAuthUsecase
	middleware *authhttp.AuthMiddleware
}

func (suite *MiddlewareTestSuite) SetupTest() {
	suite.app = fiber.New()
	suite.mockUC = new(mockAuthUsecase)
	suite.middleware = authhttp.NewAuthMiddleware(suite.mockUC, "auth_token")
}

func TestRequireAuth_Unauthorized(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	middleware := authhttp.NewAuthMiddleware(mockUC, "auth_token")
	app.Get("/protected", middleware.RequireAuth(), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRequireAuth_Authorized(t *testing.T) {
	app := fiber.New()
	mockUC := new(mockAuthUsecase)
	middleware := authhttp.NewAuthMiddleware(mockUC, "auth_token")
	claims := &repo.Claims{UserID: "user-1", Email: "test@example.com"}
	mockUC.On("ValidateToken", mock.Anything, "valid-token").Return(claims, nil)
	app.Get("/protected", middleware.RequireAuth(), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}
