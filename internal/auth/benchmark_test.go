package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"firestore-clone/internal/auth"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/auth/testutil"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func BenchmarkCompleteAuthFlow(b *testing.B) {
	// Skip if MongoDB is not available
	if testing.Short() {
		b.Skip("Skipping MongoDB benchmarks in short mode")
	}

	// Setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "benchmark_auth_db",
		JWTSecretKey:    "benchmark-secret-key-32-chars-long",
		JWTIssuer:       "benchmark-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "benchmark_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		b.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)

	module, err := auth.NewAuthModule(database, cfg)
	if err != nil {
		b.Fatalf("Failed to create auth module: %v", err)
	}

	app := fiber.New()
	module.RegisterRoutes(app)

	// Cleanup after benchmark
	defer database.Drop(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		email := "benchmark" + string(rune(i)) + "@example.com"
		password := "password123"

		// Register user
		registerPayload := map[string]string{
			"email":    email,
			"password": password,
		}
		registerBody, _ := json.Marshal(registerPayload)
		registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
		registerReq.Header.Set("Content-Type", "application/json")

		_, err := app.Test(registerReq)
		if err != nil {
			b.Errorf("Register failed: %v", err)
		}

		// Login user
		loginPayload := map[string]string{
			"email":    email,
			"password": password,
		}
		loginBody, _ := json.Marshal(loginPayload)
		loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")

		_, err = app.Test(loginReq)
		if err != nil {
			b.Errorf("Login failed: %v", err)
		}
	}
}

func BenchmarkRegisterOnly(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping MongoDB benchmarks in short mode")
	}

	// Setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "benchmark_register_db",
		JWTSecretKey:    "benchmark-secret-key-32-chars-long",
		JWTIssuer:       "benchmark-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "benchmark_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		b.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)

	module, err := auth.NewAuthModule(database, cfg)
	if err != nil {
		b.Fatalf("Failed to create auth module: %v", err)
	}

	app := fiber.New()
	module.RegisterRoutes(app)

	// Cleanup after benchmark
	defer database.Drop(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		email := "benchmark" + string(rune(i)) + "@example.com"
		password := "password123"

		registerPayload := map[string]string{
			"email":    email,
			"password": password,
		}
		registerBody, _ := json.Marshal(registerPayload)
		registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
		registerReq.Header.Set("Content-Type", "application/json")

		_, err := app.Test(registerReq)
		if err != nil {
			b.Errorf("Register failed: %v", err)
		}
	}
}

func BenchmarkLoginWithExistingUsers(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping MongoDB benchmarks in short mode")
	}

	// Setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "benchmark_login_db",
		JWTSecretKey:    "benchmark-secret-key-32-chars-long",
		JWTIssuer:       "benchmark-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "benchmark_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		b.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)
	defer database.Drop(ctx)

	module, err := auth.NewAuthModule(database, cfg)
	if err != nil {
		b.Fatalf("Failed to create auth module: %v", err)
	}

	app := fiber.New()
	module.RegisterRoutes(app)

	// Pre-create users for login benchmarks
	testUsers := 100
	for i := 0; i < testUsers; i++ {
		email := "benchuser" + string(rune(i)) + "@example.com"
		password := "password123"

		registerPayload := map[string]string{
			"email":    email,
			"password": password,
		}
		registerBody, _ := json.Marshal(registerPayload)
		registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
		registerReq.Header.Set("Content-Type", "application/json")

		app.Test(registerReq)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		userIndex := i % testUsers
		email := "benchuser" + string(rune(userIndex)) + "@example.com"
		password := "password123"

		loginPayload := map[string]string{
			"email":    email,
			"password": password,
		}
		loginBody, _ := json.Marshal(loginPayload)
		loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")

		_, err := app.Test(loginReq)
		if err != nil {
			b.Errorf("Login failed: %v", err)
		}
	}
}

func BenchmarkTokenValidation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping MongoDB benchmarks in short mode")
	}

	// Setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "benchmark_validation_db",
		JWTSecretKey:    "benchmark-secret-key-32-chars-long",
		JWTIssuer:       "benchmark-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "benchmark_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		b.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)
	defer database.Drop(ctx)

	module, err := auth.NewAuthModule(database, cfg)
	if err != nil {
		b.Fatalf("Failed to create auth module: %v", err)
	}

	app := fiber.New()
	module.RegisterRoutes(app)

	// Create a user and get token
	email := "validationuser@example.com"
	password := "password123"

	registerPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := app.Test(registerReq)
	if err != nil {
		b.Fatalf("Failed to register user: %v", err)
	}

	var registerResponse map[string]interface{}
	json.NewDecoder(registerResp.Body).Decode(&registerResponse)
	token := registerResponse["token"].(string)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		meReq := httptest.NewRequest("GET", "/auth/me", nil)
		meReq.Header.Set("Authorization", "Bearer "+token)

		_, err := app.Test(meReq)
		if err != nil {
			b.Errorf("Token validation failed: %v", err)
		}
	}
}

func BenchmarkConcurrentAuthentication(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping MongoDB benchmarks in short mode")
	}

	// Setup
	cfg := &config.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		DatabaseName:    "benchmark_concurrent_db",
		JWTSecretKey:    "benchmark-secret-key-32-chars-long",
		JWTIssuer:       "benchmark-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieName:      "benchmark_auth_cookie",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false,
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		b.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database(cfg.DatabaseName)
	defer database.Drop(ctx)

	module, err := auth.NewAuthModule(database, cfg)
	if err != nil {
		b.Fatalf("Failed to create auth module: %v", err)
	}

	app := fiber.New()
	module.RegisterRoutes(app)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		userCounter := 0
		for pb.Next() {
			email := "concurrent" + string(rune(userCounter)) + "@example.com"
			password := "password123"

			registerPayload := map[string]string{
				"email":    email,
				"password": password,
			}
			registerBody, _ := json.Marshal(registerPayload)
			registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerBody))
			registerReq.Header.Set("Content-Type", "application/json")

			app.Test(registerReq)
			userCounter++
		}
	})
}

func BenchmarkPasswordHashing(b *testing.B) {
	// Test data
	testData := testutil.NewTestData()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This simulates the password hashing that happens during registration
		testData.Users.UserWithPassword("bench@example.com", "password123")
	}
}
