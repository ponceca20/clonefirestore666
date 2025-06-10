package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"firestore-clone/internal/auth"
	"firestore-clone/internal/auth/config"
	"firestore-clone/internal/firestore"
	authClientAdapter "firestore-clone/internal/firestore/adapter/auth_client"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

// Container represents a dependency injection container with proper lifecycle management
type Container struct {
	mu        sync.RWMutex
	services  map[reflect.Type]interface{}
	factories map[reflect.Type]func() (interface{}, error)
	// Module instances
	AuthModule      *auth.AuthModule
	FirestoreModule *firestore.FirestoreModule
	// Database connections
	MongoDB *mongo.Database
	// Configuration
	AuthConfig *config.Config
	// Logger
	Logger logger.Logger
}

// NewContainer creates a new DI container with Firestore-compatible service initialization
func NewContainer() *Container {
	return &Container{
		services:  make(map[reflect.Type]interface{}),
		factories: make(map[reflect.Type]func() (interface{}, error)),
	}
}

// InitializeAuth initializes the authentication module with proper Firestore project support
func (c *Container) InitializeAuth(mongoDB *mongo.Database, authConfig *config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store references
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

// InitializeFirestore initializes the Firestore module with auth integration
func (c *Container) InitializeFirestore() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.AuthModule == nil {
		return fmt.Errorf("auth module must be initialized before Firestore module")
	}

	// Check if MongoDB is initialized
	if c.MongoDB == nil {
		return fmt.Errorf("MongoDB must be initialized before Firestore module")
	}

	// Initialize logger if not already initialized
	if c.Logger == nil {
		c.Logger = logger.NewLogger()
	}
	// Create integrated auth client using the auth module
	authUsecase := c.AuthModule.GetUsecase()
	tokenSvc := c.AuthModule.GetTokenService()
	authClient := authClientAdapter.NewAuthClientAdapter(authUsecase, tokenSvc)
	// Initialize Firestore module with auth integration
	// NewFirestoreModule expects: authClient, logger, mongoClient, masterDB
	mongoClient := c.MongoDB.Client()
	masterDB := c.MongoDB // Use the same database as master for simplicity
	firestoreModule, err := firestore.NewFirestoreModule(authClient, c.Logger, mongoClient, masterDB)
	if err != nil {
		return fmt.Errorf("failed to create Firestore module: %w", err)
	}

	c.FirestoreModule = firestoreModule
	return nil
}

// Register registers a service instance
func (c *Container) Register(service interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	serviceType := reflect.TypeOf(service)
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	c.services[serviceType] = service
	return nil
}

// RegisterFactory registers a factory function for a service
func (c *Container) RegisterFactory(serviceType reflect.Type, factory func() (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories[serviceType] = factory
	return nil
}

// Resolve resolves a service by type
func (c *Container) Resolve(serviceType reflect.Type) (interface{}, error) {
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
			return nil, fmt.Errorf("failed to create service: %w", err)
		}

		// Register the created instance
		c.mu.Lock()
		c.services[serviceType] = service
		c.mu.Unlock()

		return service, nil
	}

	c.mu.RUnlock()
	return nil, fmt.Errorf("service of type %v not registered", serviceType)
}

// ResolveByInterface resolves a service by interface type
func (c *Container) ResolveByInterface(interfaceType reflect.Type) (interface{}, error) {
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

// GetService is a generic helper for resolving services
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

// HealthCheck performs health check on all registered services
func (c *Container) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check MongoDB connection
	if c.MongoDB != nil {
		if err := c.MongoDB.Client().Ping(ctx, nil); err != nil {
			return fmt.Errorf("MongoDB health check failed: %w", err)
		}
	}

	// Check auth module health
	if c.AuthModule != nil {
		// Auth module health check would go here
		// For now, we assume it's healthy if it's initialized
	}

	// Check Firestore module health
	if c.FirestoreModule != nil {
		// Firestore module health check would go here
		// For now, we assume it's healthy if it's initialized
	}

	return nil
}

// Cleanup performs cleanup of registered services with proper shutdown order
func (c *Container) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	// Cleanup modules in reverse order of initialization
	if c.FirestoreModule != nil {
		c.FirestoreModule.Stop()
		c.FirestoreModule = nil
	}

	if c.AuthModule != nil {
		c.AuthModule.Stop()
		c.AuthModule = nil
	}

	// Cleanup generic services
	for _, service := range c.services {
		if cleaner, ok := service.(interface{ Cleanup(context.Context) error }); ok {
			if err := cleaner.Cleanup(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to cleanup service: %w", err))
			}
		}
	}

	// Clear all services
	c.services = make(map[reflect.Type]interface{})
	c.factories = make(map[reflect.Type]func() (interface{}, error))

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// NewFirestoreModule initializes the Firestore module components (deprecated - use InitializeFirestore)
func (c *Container) NewFirestoreModule() error {
	return c.InitializeFirestore()
}

// Close gracefully shuts down all services in the container with timeout
func (c *Container) Close() error {
	fmt.Println("Closing DI Container resources...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*1000000000) // 30 seconds
	defer cancel()

	if err := c.Cleanup(ctx); err != nil {
		fmt.Printf("Warning: cleanup errors occurred: %v\n", err)
	}

	fmt.Println("DI Container resources closed.")
	return nil
}
