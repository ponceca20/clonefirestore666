package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"firestore-clone/internal/firestore"
	"firestore-clone/internal/firestore/adapter/auth_client"
	"firestore-clone/internal/shared/logger"

	"github.com/caarlos0/env/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port string `env:"SERVER_PORT" envDefault:"3000"`
}

func main() {
	fmt.Println("Application Starting...")

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load server configuration
	serverCfg := &ServerConfig{}
	if err := env.Parse(serverCfg); err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// TODO: Initialize logger
	// TODO: Load main application configuration

	// Example: Initialize Dependency Injection Container
	// container, err := di.NewContainer()
	// if err != nil {
	// 	log.Fatalf("Failed to create DI container: %v", err)
	// }

	// Initialize Auth Module (example)
	// authModule, err := auth.NewAuthModule(/* auth dependencies from container */)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize Auth module: %v", err)
	// }
	// defer authModule.Stop() // Ensure graceful shutdown

	// TODO: Initialize AuthClient and Logger before passing them here
	authClient := auth_client.NewSimpleAuthClient()
	logger := logger.NewDefaultLogger()

	// Initialize Firestore Module
	firestoreModule, err := firestore.NewFirestoreModule(authClient, logger)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore module: %v", err)
	}
	defer firestoreModule.Stop() // Ensure graceful shutdown

	// Setup HTTP server (Fiber)
	app := fiber.New(fiber.Config{
		AppName: "Firestore Clone API v1.0",
	})

	// Add basic health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "OK",
			"message": "Firestore Clone API is running",
		})
	})

	// Register module routes
	// authModule.RegisterRoutes(app)
	firestoreModule.RegisterRoutes(app)
	firestoreModule.StartRealtimeServices() // If it has any background services for WebSockets

	serverAddr := fmt.Sprintf("%s:%s", serverCfg.Host, serverCfg.Port)
	fmt.Printf("Modules Initialized. Starting HTTP server on %s...\n", serverAddr)

	// Start server in a goroutine so we can handle graceful shutdown
	go func() {
		if err := app.Listen(serverAddr); err != nil {
			log.Printf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// Shutdown the server
	if err := app.Shutdown(); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Add any cleanup tasks for modules if necessary, though Stop() methods should handle it.
	fmt.Println("Application Stopped.")
}
