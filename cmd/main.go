package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	authConfig "firestore-clone/internal/auth/config"
	"firestore-clone/internal/di"
	firestoreConfig "firestore-clone/internal/firestore/config"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/logger"

	"github.com/caarlos0/env/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port string `env:"SERVER_PORT" envDefault:"3030"`
}

// getStatusText returns HTTP status text for a given status code

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found: %v", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger()

	// Load configurations
	authCfg, err := authConfig.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load auth config: %v", err)
	}

	firestoreCfg, err := firestoreConfig.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load firestore config: %v", err)
	}

	// Initialize MongoDB client
	client, err := initMongoDB(authCfg.MongoDBURI)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Initialize tenant manager for multitenant support
	tenantConfig := &database.TenantConfig{
		DatabasePrefix:     authCfg.TenantDBPrefix,
		MaxConnections:     authCfg.MaxTenantConnections,
		ConnectionTimeout:  authCfg.TenantConnectionTTL,
		AutoCreateDatabase: true,
		MaxPoolSize:        10,
		MinPoolSize:        2,
	}
	tenantManager := database.NewTenantManager(client, tenantConfig, appLogger)

	// Initialize dependency injection container
	container, err := di.NewMultitenantContainer(client, tenantManager, authCfg, firestoreCfg, appLogger)
	if err != nil {
		log.Fatalf("Failed to initialize DI container: %v", err)
	}

	// Get modules from container
	authModule := container.GetAuthModule()
	firestoreModule := container.GetFirestoreModule()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Firestore Clone - Multitenant",
		ReadTimeout:  authCfg.ReadTimeout,
		WriteTimeout: authCfg.WriteTimeout,
		IdleTimeout:  authCfg.IdleTimeout,
		ErrorHandler: globalErrorHandler,
	})

	// Apply global middleware
	setupMiddleware(app, authCfg, firestoreCfg)

	// Health check endpoint
	app.Get("/health", healthCheck(client, tenantManager))

	// Monitoring endpoint
	app.Get("/metrics", monitor.New(monitor.Config{Title: "Firestore Clone Metrics"}))
	// Register module routes
	api := app.Group("/api/v1")

	// Auth routes (no tenant requirement for login/register)
	authModule.RegisterRoutes(api.Group("/auth"))

	// Organization routes (admin API, REST, CRUD)
	firestoreModule.OrganizationHandler.RegisterRoutes(api)

	// Firestore routes (tenant-aware) - Pass auth middleware for WebSocket authentication
	authMiddleware := authModule.GetMiddleware()
	firestoreModule.RegisterRoutes(api, authMiddleware)

	// Start background services
	go func() {
		firestoreModule.StartRealtimeServices()
		appLogger.Info("Realtime services started")
	}()

	// Load server config
	var serverCfg ServerConfig
	if err := env.Parse(&serverCfg); err != nil {
		log.Fatalf("Failed to load server config: %v", err)
	}

	// Start server
	port := serverCfg.Port
	if port == "" {
		port = "3030"
	}

	appLogger.Info("Starting Firestore Clone server", map[string]interface{}{
		"port":             port,
		"tenant_isolation": authCfg.TenantIsolationEnabled,
		"mongodb_uri":      maskMongoURI(authCfg.MongoDBURI),
		"cors_enabled":     authCfg.CORSEnabled,
		"rate_limit":       authCfg.RateLimitEnabled,
	})

	// Graceful shutdown setup
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	appLogger.Info("Shutting down server...", map[string]interface{}{})

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown modules
	firestoreModule.Stop()
	container.Close()

	// Shutdown server
	if err := app.ShutdownWithContext(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	appLogger.Info("Server exited", map[string]interface{}{})
}

// initMongoDB initializes MongoDB connection
func initMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

// setupMiddleware configures global middleware
func setupMiddleware(app *fiber.App, authCfg *authConfig.Config, firestoreCfg *firestoreConfig.FirestoreConfig) {
	// Security middleware
	app.Use(helmet.New())
	app.Use(recover.New())

	// CORS middleware (use FirestoreConfig for CORS if present, else fallback to authCfg)
	corsCfg := firestoreCfg.CORS
	// Solo permitir localhost:3000 para desarrollo local
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowMethods:     corsCfg.AllowMethods,
		AllowHeaders:     corsCfg.AllowHeaders,
		AllowCredentials: corsCfg.AllowCredentials,
	}))

	// Rate limiting middleware
	if authCfg.RateLimitEnabled {
		app.Use(limiter.New(limiter.Config{
			Max:        authCfg.RateLimitRPS,
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				// Use tenant ID + IP for rate limiting
				tenantID := c.Get("X-Organization-ID", "default")
				return tenantID + ":" + c.IP()
			},
		}))
	}
}

// healthCheck returns a health check handler
func healthCheck(client *mongo.Client, tenantManager *database.TenantManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		// Check MongoDB connection
		if err := client.Ping(ctx, nil); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":  "unhealthy",
				"mongodb": "disconnected",
				"error":   err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":             "healthy",
			"timestamp":          time.Now().Unix(),
			"mongodb":            "connected",
			"tenant_connections": tenantManager.GetConnectionCount(),
		})
	}
}

// globalErrorHandler handles application errors
func globalErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":     true,
		"message":   message,
		"timestamp": time.Now().Unix(),
	})
}

func maskMongoURI(uri string) string {
	// Simple masking for logging
	if len(uri) > 20 {
		return uri[:10] + "***" + uri[len(uri)-7:]
	}
	return "***"
}
