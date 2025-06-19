package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"firestore-clone/internal/auth"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/firestore"
	authClientAdapter "firestore-clone/internal/firestore/adapter/auth_client"
	firestoreConfig "firestore-clone/internal/firestore/config"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// Container represents a dependency injection container with proper lifecycle management
// Following hexagonal architecture principles and Firestore-compatible service management
type Container struct {
	mu        sync.RWMutex
	services  map[reflect.Type]interface{}
	factories map[reflect.Type]func() (interface{}, error)

	// Core module instances - primary adapters in hexagonal architecture
	AuthModule      *auth.AuthModule
	FirestoreModule *firestore.FirestoreModule

	// Infrastructure dependencies - secondary adapters
	MongoDB     *mongo.Database
	RedisClient *redis.Client

	// Configuration - application settings
	AuthConfig      *config.Config
	FirestoreConfig *firestoreConfig.FirestoreConfig

	// Cross-cutting concerns
	Logger logger.Logger
}

// NewContainer creates a new DI container with Firestore-compatible service initialization
// Initializes core infrastructure for dependency management following clean architecture
func NewContainer() *Container {
	return &Container{
		services:  make(map[reflect.Type]interface{}),
		factories: make(map[reflect.Type]func() (interface{}, error)),
	}
}

// InitializeAuth initializes the authentication module with proper Firestore project support
// Establishes the auth domain boundary following hexagonal architecture principles
func (c *Container) InitializeAuth(mongoDB *mongo.Database, authConfig *config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if mongoDB == nil {
		return fmt.Errorf("mongoDB is required for auth module initialization")
	}
	if authConfig == nil {
		return fmt.Errorf("authConfig is required for auth module initialization")
	}

	// Store infrastructure dependencies
	c.MongoDB = mongoDB
	c.AuthConfig = authConfig

	// Initialize auth module with Firestore project support
	authModule, err := auth.NewAuthModule(mongoDB, authConfig)
	if err != nil {
		return fmt.Errorf("failed to create auth module: %w", err)
	}

	c.AuthModule = authModule
	return nil
}

// InitializeFirestore initializes the Firestore module with Redis support
// Establishes the firestore domain boundary following hexagonal architecture principles
func (c *Container) InitializeFirestore(cfg *firestoreConfig.FirestoreConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cfg == nil {
		return fmt.Errorf("firestoreConfig is required for Firestore module initialization")
	}
	if c.AuthModule == nil {
		return fmt.Errorf("auth module must be initialized before Firestore module")
	}
	if c.MongoDB == nil {
		return fmt.Errorf("MongoDB must be initialized before Firestore module")
	} // Store Firestore configuration
	c.FirestoreConfig = cfg
	// Initialize Redis client using configuration
	redisClient := firestoreConfig.NewRedisClient(&cfg.Redis)
	c.RedisClient = redisClient

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		c.Logger.Warn("Redis connection test failed, continuing without Redis", "error", err)
		// Continue without Redis for development/testing scenarios
	} else {
		c.Logger.Info("Redis connected successfully")
	}

	// Create integrated auth client using the auth module (adapter pattern)
	authUsecase := c.AuthModule.GetUsecase()
	tokenSvc := c.AuthModule.GetTokenService()
	authClient := authClientAdapter.NewAuthClientAdapter(authUsecase, tokenSvc)

	// Initialize Firestore module with auth integration and Redis
	// Following dependency inversion principle: high-level module depends on abstractions
	mongoClient := c.MongoDB.Client()
	masterDB := c.MongoDB // Use the same database as master for simplicity
	firestoreModule, err := firestore.NewFirestoreModule(authClient, c.Logger, mongoClient, masterDB, redisClient)
	if err != nil {
		return fmt.Errorf("failed to create Firestore module: %w", err)
	}

	c.FirestoreModule = firestoreModule
	return nil
}

// Register registers a service instance following dependency injection principles
func (c *Container) Register(service interface{}) error {
	if service == nil {
		return fmt.Errorf("cannot register nil service")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	serviceType := reflect.TypeOf(service)
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	c.services[serviceType] = service
	return nil
}

// RegisterFactory registers a factory function for lazy service instantiation
func (c *Container) RegisterFactory(serviceType reflect.Type, factory func() (interface{}, error)) error {
	if serviceType == nil {
		return fmt.Errorf("serviceType cannot be nil")
	}
	if factory == nil {
		return fmt.Errorf("factory function cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories[serviceType] = factory
	return nil
}

// Resolve resolves a service by type with thread-safe access
func (c *Container) Resolve(serviceType reflect.Type) (interface{}, error) {
	if serviceType == nil {
		return nil, fmt.Errorf("serviceType cannot be nil")
	}

	c.mu.RLock()

	// Check if service instance exists
	if service, exists := c.services[serviceType]; exists {
		c.mu.RUnlock()
		return service, nil
	}

	// Check if factory exists
	if factory, exists := c.factories[serviceType]; exists {
		c.mu.RUnlock()

		// Create new instance using factory
		service, err := factory()
		if err != nil {
			return nil, fmt.Errorf("failed to create service of type %v: %w", serviceType, err)
		}

		// Register the created instance (thread-safe)
		c.mu.Lock()
		c.services[serviceType] = service
		c.mu.Unlock()

		return service, nil
	}

	c.mu.RUnlock()
	return nil, fmt.Errorf("service of type %v not registered", serviceType)
}

// ResolveByInterface resolves a service by interface type (supports polymorphism)
func (c *Container) ResolveByInterface(interfaceType reflect.Type) (interface{}, error) {
	if interfaceType == nil {
		return nil, fmt.Errorf("interfaceType cannot be nil")
	}
	if interfaceType.Kind() != reflect.Interface {
		return nil, fmt.Errorf("provided type %v is not an interface", interfaceType)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for serviceType, service := range c.services {
		serviceValue := reflect.ValueOf(service)
		if serviceValue.Type().Implements(interfaceType) {
			return service, nil
		}

		// Check if pointer to service implements the interface
		if reflect.PtrTo(serviceType).Implements(interfaceType) {
			if serviceValue.Kind() != reflect.Ptr {
				return reflect.New(serviceType).Interface(), nil
			}
			return service, nil
		}
	}

	return nil, fmt.Errorf("no service implements interface %v", interfaceType)
}

// GetService is a generic helper for resolving services with type safety
func GetService[T any](c *Container) (T, error) {
	var zero T
	serviceType := reflect.TypeOf(zero)

	service, err := c.Resolve(serviceType)
	if err != nil {
		return zero, err
	}

	if typedService, ok := service.(T); ok {
		return typedService, nil
	}

	return zero, fmt.Errorf("service is not of expected type %T", zero)
}

// GetAuthModule returns the auth module instance
func (c *Container) GetAuthModule() *auth.AuthModule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AuthModule
}

// GetFirestoreModule returns the Firestore module instance
func (c *Container) GetFirestoreModule() *firestore.FirestoreModule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.FirestoreModule
}

// GetRealtimeUsecase returns the unified realtime usecase from the Firestore module
// Following hexagonal architecture, this returns the domain interface, not implementation details
// Provides access to the enhanced, Firestore-compatible real-time functionality
func (c *Container) GetRealtimeUsecase() usecase.RealtimeUsecase {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.FirestoreModule == nil {
		return nil
	}

	return c.FirestoreModule.RealtimeUsecase
}

// HealthCheck performs health check on all registered services and infrastructure
func (c *Container) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check MongoDB connection (critical infrastructure dependency)
	if c.MongoDB != nil {
		if err := c.MongoDB.Client().Ping(ctx, nil); err != nil {
			return fmt.Errorf("MongoDB health check failed: %w", err)
		}
	}

	// Check auth module health (domain boundary)
	if c.AuthModule != nil {
		// Auth module health check would go here
		// For now, we assume it's healthy if it's initialized
	}

	// Check Firestore module health (primary domain)
	if c.FirestoreModule != nil {
		// Firestore module health check would go here
		// For now, we assume it's healthy if it's initialized
		// Additional check: Verify RealtimeUsecase is available
		if c.FirestoreModule.RealtimeUsecase == nil {
			return fmt.Errorf("firestore module initialized but RealtimeUsecase is nil")
		}
	}

	return nil
}

// Cleanup performs cleanup of registered services with proper shutdown order
// Follows dependency shutdown order: modules first, then infrastructure
func (c *Container) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	// Cleanup modules in reverse order of initialization (dependency order)
	if c.FirestoreModule != nil {
		c.FirestoreModule.Stop()
		c.FirestoreModule = nil
	}

	if c.AuthModule != nil {
		c.AuthModule.Stop()
		c.AuthModule = nil
	}

	// Cleanup generic services that implement the Cleanup interface
	for serviceType, service := range c.services {
		if cleaner, ok := service.(interface{ Cleanup(context.Context) error }); ok {
			if err := cleaner.Cleanup(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to cleanup service %v: %w", serviceType, err))
			}
		}
	}

	// Clear all services and factories
	c.services = make(map[reflect.Type]interface{})
	c.factories = make(map[reflect.Type]func() (interface{}, error))

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// NewFirestoreModule initializes the Firestore module components (deprecated - use InitializeFirestore)
// Maintained for backward compatibility but deprecated in favor of InitializeFirestore
func (c *Container) NewFirestoreModule() error { // Load Firestore configuration
	cfg, err := firestoreConfig.LoadConfig()
	if err != nil {
		c.Logger.Warn("Failed to load Firestore config, using defaults", "error", err)
		cfg = firestoreConfig.DefaultFirestoreConfig()
	}
	return c.InitializeFirestore(cfg)
}

// Close gracefully shuts down all services in the container with timeout
// Ensures proper resource cleanup following clean shutdown principles
func (c *Container) Close() error {
	fmt.Println("Closing DI Container resources...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Use time.Second constant
	defer cancel()

	if err := c.Cleanup(ctx); err != nil {
		fmt.Printf("Warning: cleanup errors occurred: %v\n", err)
		return err // Return error for caller to handle
	}

	fmt.Println("DI Container resources closed successfully.")
	return nil
}
