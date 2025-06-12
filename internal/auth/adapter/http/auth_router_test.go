package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authhttp "firestore-clone/internal/auth/adapter/http"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Note: mockAuthUsecase is defined in mock_auth_usecase_test.go

// --- Test Suite Setup ---
type AuthRouterTestSuite struct {
	suite.Suite
	app            *fiber.App
	mockUC         *mockAuthUsecase
	authHandler    *authhttp.AuthHTTPHandler
	authMiddleware *authhttp.AuthMiddleware
}

func (suite *AuthRouterTestSuite) SetupTest() {
	suite.app = fiber.New()
	suite.mockUC = new(mockAuthUsecase)

	suite.authHandler = authhttp.NewAuthHTTPHandler(
		suite.mockUC,
		"test_auth_cookie", // cookieName
		"/",                // cookiePath
		"",                 // cookieDomain (empty for localhost tests usually)
		3600,               // cookieMaxAge
		false,              // cookieSecure
		true,               // cookieHTTPOnly
		"Lax",              // cookieSameSite
	)

	suite.authMiddleware = authhttp.NewAuthMiddleware(suite.mockUC, "test_auth_cookie")
	suite.authHandler.SetupAuthRoutesWithMiddleware(suite.app, suite.authMiddleware)
}

func TestAuthRouterTestSuite(t *testing.T) {
	suite.Run(t, new(AuthRouterTestSuite))
}

// --- Test Cases ---

// Public Routes
func (suite *AuthRouterTestSuite) TestRegister_Success() {
	registerReq := usecase.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  "tenant-123",
	}
	userObjID := primitive.NewObjectID()
	expectedUser := &model.User{
		ID:     userObjID,
		UserID: "user1",
		Email:  registerReq.Email,
	}
	authResp := &usecase.AuthResponse{
		User:         expectedUser,
		AccessToken:  "newAccessToken",
		RefreshToken: "newRefreshToken",
	}
	suite.mockUC.On("Register", mock.Anything, registerReq).Return(authResp, nil).Once()

	requestBody, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusCreated, resp.StatusCode)

	var response usecase.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal(authResp.AccessToken, response.AccessToken)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestLogin_Success() {
	loginReq := usecase.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
		TenantID: "tenant-123",
	}
	userObjID := primitive.NewObjectID()
	expectedUser := &model.User{
		ID:     userObjID,
		UserID: "user1",
		Email:  loginReq.Email,
	}
	authResp := &usecase.AuthResponse{
		User:         expectedUser,
		AccessToken:  "accessToken",
		RefreshToken: "refreshToken",
	}
	suite.mockUC.On("Login", mock.Anything, loginReq).Return(authResp, nil).Once()

	requestBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response usecase.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal(authResp.AccessToken, response.AccessToken)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestRefreshToken_Success() {
	refreshReqBody := map[string]string{"refreshToken": "oldRefreshToken"}
	expectedResponse := &usecase.AuthResponse{
		AccessToken:  "newAccessToken",
		RefreshToken: "newRefreshTokenAgain",
	}
	suite.mockUC.On("RefreshToken", mock.Anything, "oldRefreshToken").Return(expectedResponse, nil).Once()

	requestBody, _ := json.Marshal(refreshReqBody)
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody usecase.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	suite.NoError(err)
	suite.Equal(expectedResponse.AccessToken, respBody.AccessToken)
	suite.Equal(expectedResponse.RefreshToken, respBody.RefreshToken)

	foundCookie := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "test_auth_cookie" {
			foundCookie = true
			suite.Equal(expectedResponse.AccessToken, cookie.Value)
			break
		}
	}
	suite.True(foundCookie, "Auth cookie not updated after token refresh")
	suite.mockUC.AssertExpectations(suite.T())
}

// Protected Routes (require authentication)

func (suite *AuthRouterTestSuite) TestLogout_Success() {
	userID := "user123"
	accessToken := "validAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, accessToken).
		Return(&repository.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID}}, nil).Once()
	suite.mockUC.On("Logout", mock.Anything, userID).Return(nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: accessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestLogout_Unauthorized_NoToken() {
	// No ValidateToken mock needed as middleware should block before calling it
	// No Logout mock needed as handler won't be reached

	req := httptest.NewRequest(http.MethodPost, "/logout", nil) // No cookie

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode) // Middleware should deny
	suite.mockUC.AssertNotCalled(suite.T(), "Logout", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestGetCurrentUser_Success() {
	userID := "user123"
	accessToken := "validAccessToken"
	userObjID := primitive.NewObjectID()
	expectedUser := &model.User{
		ID:     userObjID,
		UserID: userID,
		Email:  "current@example.com",
	}

	suite.mockUC.On("ValidateToken", mock.Anything, accessToken).
		Return(&repository.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID}}, nil).Once()
	suite.mockUC.On("GetUserByID", mock.Anything, userID, "").Return(expectedUser, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: accessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	suite.NoError(err)
	suite.Equal(expectedUser.Email, user.Email)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestUpdateCurrentUser_Success() {
	userID := "user123"
	accessToken := "validAccessToken"

	userObjID := primitive.NewObjectID()
	originalUser := &model.User{
		ID:        userObjID,
		UserID:    userID,
		FirstName: "OldFirst",
		LastName:  "OldLast",
		Phone:     "123",
	}
	updateReq := map[string]string{
		"firstName": "NewFirst",
		"lastName":  "NewLast",
		// Phone not updated to test partial update
	}

	suite.mockUC.On("ValidateToken", mock.Anything, accessToken).
		Return(&repository.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID}}, nil).Once()
	suite.mockUC.On("GetUserByID", mock.Anything, userID, "").Return(originalUser, nil).Once()

	// Important: The argument to UpdateUser should be the user object *after* modifications.
	// We use mock.MatchedBy to check if the user passed to UpdateUser has the correct fields.
	suite.mockUC.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.UserID == userID && u.FirstName == updateReq["firstName"] && u.LastName == updateReq["lastName"] && u.Phone == originalUser.Phone
	})).Return(nil).Once()

	requestBody, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/me", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: accessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respUser model.User
	err = json.NewDecoder(resp.Body).Decode(&respUser)
	suite.NoError(err)
	suite.Equal(updateReq["firstName"], respUser.FirstName)
	suite.Equal(updateReq["lastName"], respUser.LastName)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestChangePassword_Success() {
	userID := "user123"
	accessToken := "validAccessToken"
	passwords := map[string]string{
		"oldPassword": "oldPassword123", // Corrected from old_password
		"newPassword": "newPassword456", // Corrected from new_password
	}

	suite.mockUC.On("ValidateToken", mock.Anything, accessToken).
		Return(&repository.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID}}, nil).Once()
	suite.mockUC.On("ChangePassword", mock.Anything, userID, passwords["oldPassword"], passwords["newPassword"]).Return(nil).Once()

	requestBody, _ := json.Marshal(passwords)
	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: accessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

// Admin Routes (require admin authentication)

func (suite *AuthRouterTestSuite) TestListUsers_Admin_Success() {
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"
	userObjID1 := primitive.NewObjectID()
	userObjID2 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
		{ID: userObjID2, UserID: "user2", Email: "user2@example.com"},
	}
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{UserID: adminUserID, TenantID: tenantID, Roles: []string{"admin"}, RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID}}, nil).Twice()
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})
	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Len(response.Users, 2)
	suite.Equal(2, response.Total)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestListUsers_Forbidden_NonAdmin() {
	nonAdminUserID := "nonAdminUser"
	tenantID := "tenant123"
	nonAdminAccessToken := "nonAdminAccessToken"
	suite.mockUC.On("ValidateToken", mock.Anything, nonAdminAccessToken).
		Return(&repository.Claims{UserID: nonAdminUserID, TenantID: tenantID, Roles: []string{"user"}, RegisteredClaims: jwt.RegisteredClaims{Subject: nonAdminUserID}}, nil).Twice()
	// GetUsersByTenant should not be called

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode) // Should be denied by admin middleware
	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestGetUser_Admin_Success() {
	adminUserID := "adminUser"
	targetUserID := "targetUser123"
	adminAccessToken := "adminAccessToken"
	targetUserObjID := primitive.NewObjectID()
	expectedUser := &model.User{
		ID:     targetUserObjID,
		UserID: targetUserID,
		Email:  "target@example.com",
	}
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{UserID: adminUserID, Roles: []string{"admin"}, RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID}}, nil).Twice()
	suite.mockUC.On("GetUserByID", mock.Anything, targetUserID, "").Return(expectedUser, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	suite.NoError(err)
	suite.Equal(expectedUser.Email, user.Email)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Admin_Success() {
	adminUserID := "adminUser"
	targetUserID := "targetUser123"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{UserID: adminUserID, Roles: []string{"admin"}, RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID}}, nil).Twice()
	suite.mockUC.On("DeleteUser", mock.Anything, targetUserID).Return(nil).Once()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.mockUC.AssertExpectations(suite.T())
}

// --- ADDITIONAL ADMIN ROUTE TESTS ---

// Tests for edge cases and error conditions in admin routes

func (suite *AuthRouterTestSuite) TestListUsers_Admin_NoTenantInContext() {
	adminUserID := "adminUser"
	adminAccessToken := "adminAccessToken"

	// Mock token validation but don't set TenantID in claims
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Tenant ID required", response["error"])

	// Should not call GetUsersByTenant since tenant ID is missing
	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestListUsers_Admin_DatabaseError() {
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock database error
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).
		Return(nil, fmt.Errorf("database connection failed")).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("database connection failed", response["error"])
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestListUsers_Admin_EmptyTenant() {
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock empty tenant (no users)
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).
		Return([]*model.User{}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal(0, response.Total)
	suite.Len(response.Users, 0)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestGetUser_Admin_UserNotFound() {
	adminUserID := "adminUser"
	targetUserID := "nonexistentUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock user not found error
	suite.mockUC.On("GetUserByID", mock.Anything, targetUserID, "").
		Return(nil, model.ErrUserNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Contains(response["error"], "user not found")
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestGetUser_Admin_EmptyUserID() {
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock GetUsersByTenant since the route /admin/users/ will hit the ListUsers handler
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).
		Return([]*model.User{}, nil).Once()

	// Use empty string as userID parameter - this should reach the handler
	req := httptest.NewRequest(http.MethodGet, "/admin/users/", nil) // Empty userID
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	// When userID is empty, the route /admin/users/ doesn't match /admin/users/:userId
	// So it hits the ListUsers endpoint instead, which returns a successful empty list
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	// Since it hits ListUsers instead of GetUser, it returns an empty list
	suite.Equal(0, response.Total)
	suite.Len(response.Users, 0)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestGetUser_Forbidden_NonAdmin() {
	nonAdminUserID := "nonAdminUser"
	targetUserID := "targetUser123"
	nonAdminAccessToken := "nonAdminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, nonAdminAccessToken).
		Return(&repository.Claims{
			UserID:           nonAdminUserID,
			Roles:            []string{"user"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: nonAdminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])

	// Should not call GetUserByID since user lacks admin role
	suite.mockUC.AssertNotCalled(suite.T(), "GetUserByID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Admin_UserNotFound() {
	adminUserID := "adminUser"
	targetUserID := "nonexistentUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock user not found error
	suite.mockUC.On("DeleteUser", mock.Anything, targetUserID).
		Return(model.ErrUserNotFound).Once()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Contains(response["error"], "user not found")
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Admin_DatabaseError() {
	adminUserID := "adminUser"
	targetUserID := "targetUser123"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	// Mock database error
	suite.mockUC.On("DeleteUser", mock.Anything, targetUserID).
		Return(fmt.Errorf("database constraint violation")).Once()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("database constraint violation", response["error"])
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Forbidden_NonAdmin() {
	nonAdminUserID := "nonAdminUser"
	targetUserID := "targetUser123"
	nonAdminAccessToken := "nonAdminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, nonAdminAccessToken).
		Return(&repository.Claims{
			UserID:           nonAdminUserID,
			Roles:            []string{"user"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: nonAdminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])

	// Should not call DeleteUser since user lacks admin role
	suite.mockUC.AssertNotCalled(suite.T(), "DeleteUser", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Admin_EmptyUserID() {
	adminUserID := "adminUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/", nil) // Empty userID
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	// Fiber returns 405 Method Not Allowed when DELETE is used on /admin/users/
	// because it expects a userID parameter
	suite.Equal(http.StatusMethodNotAllowed, resp.StatusCode)

	// Should not call DeleteUser since route doesn't match properly
	suite.mockUC.AssertNotCalled(suite.T(), "DeleteUser", mock.Anything, mock.Anything)
}

// Test for admin routes without authentication token

func (suite *AuthRouterTestSuite) TestListUsers_Unauthorized_NoToken() {
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil) // No cookie

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Authentication required", response["error"])

	// Should not call any usecase methods
	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestGetUser_Unauthorized_NoToken() {
	targetUserID := "targetUser123"

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil) // No cookie

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Authentication required", response["error"])

	// Should not call any usecase methods
	suite.mockUC.AssertNotCalled(suite.T(), "GetUserByID", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestDeleteUser_Unauthorized_NoToken() {
	targetUserID := "targetUser123"

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil) // No cookie

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Authentication required", response["error"])

	// Should not call any usecase methods
	suite.mockUC.AssertNotCalled(suite.T(), "DeleteUser", mock.Anything, mock.Anything)
}

// Test with invalid tokens

func (suite *AuthRouterTestSuite) TestListUsers_Unauthorized_InvalidToken() {
	invalidToken := "invalidToken"

	suite.mockUC.On("ValidateToken", mock.Anything, invalidToken).
		Return(nil, fmt.Errorf("invalid token")).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: invalidToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Invalid token", response["error"])

	// Should not call GetUsersByTenant
	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
	suite.mockUC.AssertExpectations(suite.T())
}

// Test admin routes with multiple roles including admin

func (suite *AuthRouterTestSuite) TestListUsers_Admin_MultipleRoles() {
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"
	userObjID1 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
	}

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"user", "admin", "moderator"}, // Multiple roles including admin
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Len(response.Users, 1)
	suite.Equal(1, response.Total)
	suite.mockUC.AssertExpectations(suite.T())
}

// Additional comprehensive tests for admin role evaluation

func (suite *AuthRouterTestSuite) TestAdminRoutes_RoleValidation_ExactMatch() {
	// Test that the role check is case-sensitive and exact
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"

	// User with "Admin" (capital A) should not be allowed
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"Admin", "user"}, // Capital 'A' - should not match "admin"
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])
	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
}

func (suite *AuthRouterTestSuite) TestAdminRoutes_EmptyRoles() {
	// Test user with empty roles array
	adminUserID := "adminUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{}, // Empty roles
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])
}

func (suite *AuthRouterTestSuite) TestAdminRoutes_NilRoles() {
	// Test user with nil roles array
	adminUserID := "adminUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            nil, // Nil roles
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])
}

func (suite *AuthRouterTestSuite) TestAdminRoutes_SimilarButNotExactRoles() {
	// Test user with similar roles that should not match
	adminUserID := "adminUser"
	adminAccessToken := "adminAccessToken"

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			Roles:            []string{"administrator", "admin-user", "super-admin", "user"}, // Similar but not exact
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal("Insufficient permissions", response["error"])
}

func (suite *AuthRouterTestSuite) TestAdminRoutes_AdminAsOnlyRole() {
	// Test user with only "admin" role (no other roles)
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"
	userObjID1 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
	}

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"}, // Only admin role
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Len(response.Users, 1)
	suite.Equal(1, response.Total)
	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestAllAdminEndpoints_RoleValidation() {
	// Test all admin endpoints with proper admin role
	adminUserID := "adminUser"
	tenantID := "tenant123"
	targetUserID := "targetUser"
	adminAccessToken := "adminAccessToken"

	userObjID1 := primitive.NewObjectID()
	userObjID2 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
		{ID: userObjID2, UserID: targetUserID, Email: "target@example.com"},
	}

	// Test GET /admin/users
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Times(6) // 2 calls for each of 3 endpoints
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})
	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Test GET /admin/users/:userId
	suite.mockUC.On("GetUserByID", mock.Anything, targetUserID, "").Return(expectedUsers[1], nil).Once()

	req = httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})
	resp, err = suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Test DELETE /admin/users/:userId
	suite.mockUC.On("DeleteUser", mock.Anything, targetUserID).Return(nil).Once()

	req = httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})
	resp, err = suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	suite.mockUC.AssertExpectations(suite.T())
}

func (suite *AuthRouterTestSuite) TestAllAdminEndpoints_NonAdminRejection() {
	// Test that all admin endpoints reject non-admin users
	nonAdminUserID := "nonAdminUser"
	targetUserID := "targetUser"
	nonAdminAccessToken := "nonAdminAccessToken"

	// Claims for non-admin user
	nonAdminClaims := &repository.Claims{
		UserID:           nonAdminUserID,
		Roles:            []string{"user", "moderator"}, // No admin role
		RegisteredClaims: jwt.RegisteredClaims{Subject: nonAdminUserID},
	}

	// Test GET /admin/users
	suite.mockUC.On("ValidateToken", mock.Anything, nonAdminAccessToken).
		Return(nonAdminClaims, nil).Times(6) // 2 calls for each of 3 endpoints

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})
	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	// Test GET /admin/users/:userId
	req = httptest.NewRequest(http.MethodGet, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})
	resp, err = suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	// Test DELETE /admin/users/:userId
	req = httptest.NewRequest(http.MethodDelete, "/admin/users/"+targetUserID, nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: nonAdminAccessToken})
	resp, err = suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusForbidden, resp.StatusCode)

	// Verify no business logic methods were called	suite.mockUC.AssertNotCalled(suite.T(), "GetUsersByTenant", mock.Anything, mock.Anything)
	suite.mockUC.AssertNotCalled(suite.T(), "GetUserByID", mock.Anything, mock.Anything, mock.Anything)
	suite.mockUC.AssertNotCalled(suite.T(), "DeleteUser", mock.Anything, mock.Anything)
	suite.mockUC.AssertExpectations(suite.T())
}

// Test middleware chain execution order
func (suite *AuthRouterTestSuite) TestAdminMiddleware_ChainExecution() {
	// Test that both Protect() and RequireRole("admin") middlewares are executed in correct order
	adminUserID := "adminUser"
	tenantID := "tenant123"
	adminAccessToken := "adminAccessToken"

	// First call from Protect() middleware, second call from RequireRole() middleware
	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(&repository.Claims{
			UserID:           adminUserID,
			TenantID:         tenantID,
			Roles:            []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
		}, nil).Twice()

	userObjID1 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
	}
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Verify that ValidateToken was called exactly twice (once for Protect, once for RequireRole)
	suite.mockUC.AssertExpectations(suite.T())
}

// Test context propagation through middleware chain
func (suite *AuthRouterTestSuite) TestAdminMiddleware_ContextPropagation() {
	// Verify that user context is properly set and accessible in handlers
	adminUserID := "adminUser"
	tenantID := "tenant123"
	projectID := "project123"
	databaseID := "database123"
	adminAccessToken := "adminAccessToken"

	claims := &repository.Claims{
		UserID:           adminUserID,
		TenantID:         tenantID,
		ProjectID:        projectID,
		DatabaseID:       databaseID,
		Roles:            []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{Subject: adminUserID},
	}

	suite.mockUC.On("ValidateToken", mock.Anything, adminAccessToken).
		Return(claims, nil).Twice()

	userObjID1 := primitive.NewObjectID()
	expectedUsers := []*model.User{
		{ID: userObjID1, UserID: "user1", Email: "user1@example.com"},
	}
	suite.mockUC.On("GetUsersByTenant", mock.Anything, tenantID).Return(expectedUsers, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_auth_cookie", Value: adminAccessToken})

	resp, err := suite.app.Test(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var response struct {
		Users []*model.User `json:"users"`
		Total int           `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.NoError(err)
	suite.Equal(1, response.Total)

	// Verify GetUsersByTenant was called with the correct tenantID from context
	suite.mockUC.AssertExpectations(suite.T())
}
