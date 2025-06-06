package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"firestore-clone/internal/auth"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/testutil"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuthIntegrationTestSuite struct {
	suite.Suite
	app      *fiber.App
	client   *mongo.Client
	database *mongo.Database
	module   *auth.AuthModule
	testData *testutil.TestData
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Setup test configuration
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "test_auth_integration_db",
		JWTSecretKey:    "test-secret-key-32-characters-long-12345",
		JWTIssuer:       "test-integration-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "test_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	// Setup MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	require.NoError(suite.T(), err)

	suite.client = client
	suite.database = client.Database(cfg.DatabaseName)

	// Initialize auth module
	module, err := auth.NewAuthModule(suite.database, cfg)
	require.NoError(suite.T(), err)
	suite.module = module

	// Setup Fiber app
	suite.app = fiber.New()
	suite.module.RegisterRoutes(suite.app)

	// Initialize test data
	suite.testData = testutil.NewTestData()
}

func (suite *AuthIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	ctx := context.Background()
	err := suite.database.Drop(ctx)
	require.NoError(suite.T(), err)
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.client.Disconnect(ctx)
	}
}

func (suite *AuthIntegrationTestSuite) TestCompleteAuthFlow() {
	email := "integration@example.com"
	password := "password123"

	// Step 1: Register user
	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := suite.app.Test(registerReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)

	// Extract token from register response
	var registerResponse map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerResponse)
	require.NoError(suite.T(), err)
	registerToken := registerResponse["token"].(string)
	assert.NotEmpty(suite.T(), registerToken)

	// Step 2: Get current user with token
	meReq := httptest.NewRequest("GET", "/auth/me", nil)
	meReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", registerToken))

	meResp, err := suite.app.Test(meReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, meResp.StatusCode)

	var meResponse map[string]interface{}
	err = json.NewDecoder(meResp.Body).Decode(&meResponse)
	require.NoError(suite.T(), err)
	userData := meResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), email, userData["email"])

	// Step 3: Logout
	logoutReq := httptest.NewRequest("POST", "/auth/logout", nil)
	logoutResp, err := suite.app.Test(logoutReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, logoutResp.StatusCode)

	// Step 4: Login with same credentials
	loginPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := suite.app.Test(loginReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, loginResp.StatusCode)

	var loginResponse map[string]interface{}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	require.NoError(suite.T(), err)
	loginToken := loginResponse["token"].(string)
	assert.NotEmpty(suite.T(), loginToken)

	// Step 5: Verify new token works
	meReq2 := httptest.NewRequest("GET", "/auth/me", nil)
	meReq2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginToken))

	meResp2, err := suite.app.Test(meReq2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, meResp2.StatusCode)
}

func (suite *AuthIntegrationTestSuite) TestRegisterWithDuplicateEmail() {
	email := "duplicate@example.com"
	password := "password123"

	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(registerPayload)

	// First registration - should succeed
	req1 := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := suite.app.Test(req1)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp1.StatusCode)

	// Second registration - should fail
	req2 := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := suite.app.Test(req2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusConflict, resp2.StatusCode)

	var errorResponse map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&errorResponse)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse["error"], "Email already taken")
}

func (suite *AuthIntegrationTestSuite) TestLoginWithInvalidCredentials() {
	// Register user first
	email := "test@example.com"
	password := "correctpassword"

	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := suite.app.Test(registerReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)

	// Try to login with wrong password
	loginPayload := map[string]string{
		"email":    email,
		"password": "wrongpassword",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := suite.app.Test(loginReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, loginResp.StatusCode)

	var errorResponse map[string]interface{}
	err = json.NewDecoder(loginResp.Body).Decode(&errorResponse)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse["error"], "Invalid credentials")
}

func (suite *AuthIntegrationTestSuite) TestTokenValidationFlow() {
	// Register and get token
	email := "token@example.com"
	password := "password123"

	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := suite.app.Test(registerReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)

	var registerResponse map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerResponse)
	require.NoError(suite.T(), err)
	token := registerResponse["token"].(string)

	// Test valid token
	validReq := httptest.NewRequest("GET", "/auth/validate", nil)
	validReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	validResp, err := suite.app.Test(validReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, validResp.StatusCode)

	// Test invalid token
	invalidReq := httptest.NewRequest("GET", "/auth/validate", nil)
	invalidReq.Header.Set("Authorization", "Bearer invalid-token")

	invalidResp, err := suite.app.Test(invalidReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, invalidResp.StatusCode)

	// Test no token
	noTokenReq := httptest.NewRequest("GET", "/auth/validate", nil)

	noTokenResp, err := suite.app.Test(noTokenReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, noTokenResp.StatusCode)
}

func (suite *AuthIntegrationTestSuite) TestCookieAuthentication() {
	// Register user
	email := "cookie@example.com"
	password := "password123"

	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := suite.app.Test(registerReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)

	// Extract cookie from response
	cookies := registerResp.Cookies()
	require.Len(suite.T(), cookies, 1)
	authCookie := cookies[0]

	// Use cookie for authentication
	meReq := httptest.NewRequest("GET", "/auth/me", nil)
	meReq.Header.Set("Cookie", fmt.Sprintf("%s=%s", authCookie.Name, authCookie.Value))

	meResp, err := suite.app.Test(meReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, meResp.StatusCode)

	var meResponse map[string]interface{}
	err = json.NewDecoder(meResp.Body).Decode(&meResponse)
	require.NoError(suite.T(), err)
	userData := meResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), email, userData["email"])
}

func (suite *AuthIntegrationTestSuite) TestConcurrentRegistrations() {
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			email := fmt.Sprintf("concurrent%d@example.com", id)
			registerPayload := map[string]string{
				"email":    email,
				"password": "password123",
			}
			body, _ := json.Marshal(registerPayload)

			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.app.Test(req)
			if err != nil {
				results <- 0
				return
			}
			results <- resp.StatusCode
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		statusCode := <-results
		if statusCode == http.StatusCreated {
			successCount++
		}
	}

	assert.Equal(suite.T(), numGoroutines, successCount, "All concurrent registrations should succeed")
}

func (suite *AuthIntegrationTestSuite) TestDataPersistence() {
	email := "persistence@example.com"
	password := "password123"

	// Register user
	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := suite.app.Test(registerReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)

	// Get user ID from response
	var registerResponse map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerResponse)
	require.NoError(suite.T(), err)
	userResponse := registerResponse["user"].(map[string]interface{})
	userID := userResponse["id"].(string)

	// Verify user exists in database by attempting login
	loginPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := suite.app.Test(loginReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, loginResp.StatusCode)

	var loginResponse map[string]interface{}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	require.NoError(suite.T(), err)
	loginUserResponse := loginResponse["user"].(map[string]interface{})
	loginUserID := loginUserResponse["id"].(string)

	// Verify same user ID
	assert.Equal(suite.T(), userID, loginUserID)
}

func (suite *AuthIntegrationTestSuite) TestEmailValidation() {
	invalidEmails := testutil.InvalidEmails
	password := "password123"

	for _, email := range invalidEmails {
		suite.Run("invalid_email_"+email, func() {
			registerPayload := map[string]string{
				"email":    email,
				"password": password,
			}
			body, _ := json.Marshal(registerPayload)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.app.Test(req)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func (suite *AuthIntegrationTestSuite) TestPasswordValidation() {
	email := "password@example.com"
	invalidPasswords := testutil.InvalidPasswords

	for _, password := range invalidPasswords {
		suite.Run("invalid_password_"+password, func() {
			registerPayload := map[string]string{
				"email":    email,
				"password": password,
			}
			body, _ := json.Marshal(registerPayload)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.app.Test(req)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	// Skip if MongoDB is not available or in short mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Try to connect to MongoDB to check availability
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("Skipping integration tests: MongoDB not available")
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		t.Skip("Skipping integration tests: MongoDB not reachable")
	}

	client.Disconnect(ctx)

	suite.Run(t, new(AuthIntegrationTestSuite))
}

// Helper function for quick integration test
func TestQuickIntegrationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Minimal integration test that doesn't require full suite setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "quick_test_db",
		JWTSecretKey:    "quick-test-secret-key-32-chars-long",
		JWTIssuer:       "quick-test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "quick_test_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		t.Skip("MongoDB not available")
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)
	defer database.Drop(ctx)

	module, err := auth.NewAuthModule(database, cfg)
	require.NoError(t, err)

	app := fiber.New()
	module.RegisterRoutes(app)

	// Quick register test
	registerPayload := map[string]string{
		"email":    "quick@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
