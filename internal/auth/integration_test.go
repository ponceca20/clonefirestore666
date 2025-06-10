package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"firestore-clone/internal/auth"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/testutil"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
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

// Helper to create a JSON request body
func jsonBody(data []byte) *bytes.Buffer {
	return bytes.NewBuffer(data)
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Setup MongoDB test instance (replace with test container or mock for CI)
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:Ponceca120@127.0.0.1:27017/?authSource=admin"
	}
	// Use the real mongo.Connect for integration, or skip if not available
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	assert.NoError(suite.T(), err)

	// Test the connection
	err = client.Ping(context.Background(), nil)
	assert.NoError(suite.T(), err)

	db := client.Database("auth_integration_test")
	suite.client = client
	suite.database = db
	cfg := &config.Config{
		JWTSecretKey:   "integration-secret-key-that-is-at-least-32-chars-long",
		JWTIssuer:      "integration-test",
		AccessTokenTTL: time.Hour,
		CookieName:     "auth_token",
		CookiePath:     "/",
		CookieDomain:   "",
		CookieSecure:   false,
		CookieHTTPOnly: true, CookieSameSite: "Lax",
	}
	module, err := auth.NewAuthModule(db, cfg)
	assert.NoError(suite.T(), err)
	suite.module = module
	suite.app = fiber.New()
	// Register auth routes under /auth prefix
	authGroup := suite.app.Group("/auth")
	suite.module.RegisterRoutes(authGroup)
	suite.testData = testutil.NewTestData()
	// Clean test DB before each suite (manual cleanup if needed)
	db.Collection("users").Drop(context.Background())
	db.Collection("sessions").Drop(context.Background())
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	suite.database.Collection("users").Drop(context.Background())
	suite.database.Collection("sessions").Drop(context.Background())
	_ = suite.client.Disconnect(context.Background())
}

func (suite *AuthIntegrationTestSuite) TestRegisterAndLogin_Success() {
	registerReq := map[string]interface{}{
		"email":      "integration@example.com",
		"password":   "Password123!",
		"projectId":  "test-project-123",
		"databaseId": "test-database",
		"tenantId":   "test-tenant",
		"firstName":  "Integration",
		"lastName":   "Test",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	assert.NoError(suite.T(), err)

	// Debug: Print response body if not 201
	if resp.StatusCode != http.StatusCreated {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		suite.T().Logf("Register response body: %s", string(bodyBytes[:n]))
		resp.Body.Close()
	}

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	// Login
	loginReq := map[string]interface{}{
		"email":      "integration@example.com",
		"password":   "Password123!",
		"projectId":  "test-project-123",
		"databaseId": "test-database",
		"tenantId":   "test-tenant",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = suite.app.Test(req)
	assert.NoError(suite.T(), err)

	// Debug: Print response body if not 200
	if resp.StatusCode != http.StatusOK {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		suite.T().Logf("Login response body: %s", string(bodyBytes[:n]))
		resp.Body.Close()
	}

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *AuthIntegrationTestSuite) TestRegister_DuplicateEmail() {
	user := suite.testData.Users.UserWithEmail("dup@example.com")
	// Set valid tenant ID (ProjectID/DatabaseID are not part of User struct)
	user.TenantID = "test-tenant-123"
	db := suite.database
	db.Collection("users").InsertOne(context.Background(), user)
	registerReq := map[string]interface{}{
		"email":      user.Email,
		"password":   "Password123!",
		"projectId":  "test-project-123",
		"databaseId": "test-database",
		"tenantId":   "test-tenant-123",
		"firstName":  "Dup",
		"lastName":   "User",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusConflict, resp.StatusCode)
}

func (suite *AuthIntegrationTestSuite) TestLogin_InvalidCredentials() {
	loginReq := map[string]interface{}{
		"email":      "notfound@example.com",
		"password":   "WrongPass!",
		"projectId":  "test-project-123",
		"databaseId": "test-database",
		"tenantId":   "test-tenant",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
