package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"firestore-clone/internal/firestore"
	authClientAdapter "firestore-clone/internal/firestore/adapter/auth_client"
	"firestore-clone/internal/shared/logger"
)

// Container represents a dependency injection container
type Container struct {
	mu        sync.RWMutex
	services  map[reflect.Type]interface{}
	factories map[reflect.Type]func() (interface{}, error)
	// Firestore Module Components
	// FirestoreRepository firestoreDomainRepo.FirestoreRepository
	// QueryEngine firestoreDomainRepo.QueryEngine
	// SecurityRulesEngine firestoreDomainRepo.SecurityRulesEngine
	// FirestoreAuthClient firestoreDomainRepo.AuthClient // The client Firestore uses to talk to Auth
	// FirestoreUsecase firestoreUseCase.FirestoreUsecase
	// RealtimeUsecase firestoreUseCase.RealtimeUsecase
	// SecurityUsecase firestoreUseCase.SecurityUsecase
	FirestoreModule *firestore.FirestoreModule
}

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services:  make(map[reflect.Type]interface{}),
		factories: make(map[reflect.Type]func() (interface{}, error)),
	}
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

// Cleanup performs cleanup of registered services
func (c *Container) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, service := range c.services {
		if cleaner, ok := service.(interface{ Cleanup(context.Context) error }); ok {
			if err := cleaner.Cleanup(ctx); err != nil {
				return fmt.Errorf("failed to cleanup service: %w", err)
			}
		}
	}

	// Clear all services
	c.services = make(map[reflect.Type]interface{})
	c.factories = make(map[reflect.Type]func() (interface{}, error))

	return nil
}

// NewFirestoreModule initializes the Firestore module components.
func (c *Container) NewFirestoreModule() error {
	authClient := authClientAdapter.NewSimpleAuthClient()
	log := logger.NewDefaultLogger()

	fm, err := firestore.NewFirestoreModule(authClient, log)
	if err != nil {
		return fmt.Errorf("failed to create Firestore module: %w", err)
	}
	c.FirestoreModule = fm

	return nil
}

// Close gracefully shuts down all services in the container.
func (c *Container) Close() error {
	fmt.Println("Closing DI Container resources...")
	// Call Stop() on modules if they have cleanup tasks
	if c.FirestoreModule != nil {
		c.FirestoreModule.Stop()
	}
	fmt.Println("DI Container resources closed.")
	return nil
}
