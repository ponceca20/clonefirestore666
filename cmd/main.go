package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/di"
	"firestore-clone/internal/shared/logger"

	"github.com/caarlos0/env/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port string `env:"SERVER_PORT" envDefault:"3000"`
}

func main() {
	fmt.Println("🚀 Firestore Clone - Starting Application...")

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}
	// Load server configuration
	serverCfg := &ServerConfig{}
	if err := env.Parse(serverCfg); err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger()
	appLogger.Info("Application configuration loaded successfully")

	// Initialize Dependency Injection Container
	container := di.NewContainer()
	defer func() {
		if err := container.Close(); err != nil {
			appLogger.Error("Failed to close container: %v", err)
		}
	}()

	// Initialize MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
		appLogger.Warn("MONGODB_URI not set, using default: %s", mongoURI)
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			appLogger.Error("Failed to disconnect MongoDB: %v", err)
		}
	}()

	// Verify MongoDB connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	appLogger.Info("MongoDB connection established successfully")

	// Load auth configuration
	authConfig, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load auth configuration: %v", err)
	}

	// Get MongoDB database
	mongoDB := mongoClient.Database(authConfig.DatabaseName)

	// Initialize Auth Module through container
	if err := container.InitializeAuth(mongoDB, authConfig); err != nil {
		log.Fatalf("Failed to initialize Auth module: %v", err)
	}
	appLogger.Info("Auth module initialized successfully")

	// Initialize Firestore Module through container
	if err := container.InitializeFirestore(); err != nil {
		log.Fatalf("Failed to initialize Firestore module: %v", err)
	}
	appLogger.Info("Firestore module initialized successfully")

	// Setup HTTP server (Fiber) with middleware
	app := fiber.New(fiber.Config{
		AppName:      "Firestore Clone API v1.0",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			appLogger.Error("HTTP Error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		},
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Add health check endpoint with container health status
	app.Get("/health", func(c *fiber.Ctx) error {
		healthCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		if err := container.HealthCheck(healthCtx); err != nil {
			appLogger.Error("Health check failed: %v", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":  "UNHEALTHY",
				"error":   err.Error(),
				"message": "One or more services are unhealthy",
			})
		}

		return c.JSON(fiber.Map{
			"status":    "HEALTHY",
			"message":   "Firestore Clone API is running",
			"timestamp": time.Now().UTC(),
			"modules": fiber.Map{
				"auth":      "initialized",
				"firestore": "initialized",
			},
		})
	})

	// Register module routes
	authModule := container.GetAuthModule()
	firestoreModule := container.GetFirestoreModule()

	if authModule != nil {
		authModule.RegisterRoutes(app)
		appLogger.Info("Auth routes registered")
	}

	if firestoreModule != nil {
		firestoreModule.RegisterRoutes(app)
		firestoreModule.StartRealtimeServices() // Start WebSocket services
		appLogger.Info("Firestore routes and realtime services registered")
	}

	serverAddr := fmt.Sprintf("%s:%s", serverCfg.Host, serverCfg.Port)
	appLogger.Info("🌟 All modules initialized. Starting HTTP server on %s", serverAddr)

	// Start server in a goroutine for graceful shutdown
	serverShutdown := make(chan error, 1)
	go func() {
		serverShutdown <- app.Listen(serverAddr)
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverShutdown:
		if err != nil {
			appLogger.Error("Server failed to start: %v", err)
			log.Fatalf("Server startup failed: %v", err)
		}
	case sig := <-quit:
		appLogger.Info("Received shutdown signal: %v", sig)
		fmt.Println("🛑 Shutting down server gracefully...")

		// Shutdown the server with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			appLogger.Error("Server forced to shutdown: %v", err)
		}

		appLogger.Info("HTTP server stopped")
	}

	fmt.Println("✅ Application stopped gracefully.")
}
