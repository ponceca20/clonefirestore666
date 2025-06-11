package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"firestore-clone/internal/shared/logger"
)

// Event represents a generic event
type Event interface {
	Type() string
	Data() interface{}
	Timestamp() time.Time
	Source() string
}

// Handler defines the event handler function type
type Handler func(ctx context.Context, event Event) error

// EventBusInterface defines the contract for event bus implementations
type EventBusInterface interface {
	Subscribe(eventType string, handler Handler)
	Publish(ctx context.Context, event Event) error
	PublishAndForget(ctx context.Context, event Event)
	Unsubscribe(eventType string)
	GetSubscriberCount(eventType string) int
	GetEventTypes() []string
}

// EventBus represents an in-memory event bus with enhanced capabilities
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   logger.Logger
	config   BusConfig
}

// BusConfig holds configuration for the event bus
type BusConfig struct {
	AsyncProcessing bool
	MaxRetries      int
	RetryDelay      time.Duration
}

// DefaultBusConfig returns default configuration
func DefaultBusConfig() BusConfig {
	return BusConfig{
		AsyncProcessing: false,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}
}

// NewEventBus creates a new event bus instance
func NewEventBus(log logger.Logger) *EventBus {
	if log == nil {
		log = &noopLogger{}
	}
	return NewEventBusWithConfig(log, DefaultBusConfig())
}

// NewEventBusWithConfig creates a new event bus with custom configuration
func NewEventBusWithConfig(log logger.Logger, config BusConfig) *EventBus {
	if log == nil {
		log = &noopLogger{}
	}
	return &EventBus{
		handlers: make(map[string][]Handler),
		logger:   log,
		config:   config,
	}
}

// Subscribe adds a handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.logger.Debugf("Subscribed handler for event type: %s", eventType)
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type()]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		eb.logger.Debugf("No handlers found for event type: %s", event.Type())
		return nil
	}

	eb.logger.Debugf("Publishing event type: %s to %d handlers", event.Type(), len(handlers))

	if eb.config.AsyncProcessing {
		return eb.publishAsync(ctx, event, handlers)
	}

	return eb.publishSync(ctx, event, handlers)
}

// publishSync publishes events synchronously
func (eb *EventBus) publishSync(ctx context.Context, event Event, handlers []Handler) error {
	for i, handler := range handlers {
		if err := eb.executeHandler(ctx, event, handler, i); err != nil {
			return err
		}
	}
	return nil
}

// publishAsync publishes events asynchronously
func (eb *EventBus) publishAsync(ctx context.Context, event Event, handlers []Handler) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(handlers))

	for i, handler := range handlers {
		wg.Add(1)
		go func(h Handler, idx int) {
			defer wg.Done()
			if err := eb.executeHandler(ctx, event, h, idx); err != nil {
				errCh <- err
			}
		}(handler, i)
	}

	wg.Wait()
	close(errCh)

	// Collect any errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// executeHandler executes a handler with retry logic
func (eb *EventBus) executeHandler(ctx context.Context, event Event, handler Handler, handlerIndex int) error {
	var lastErr error

	for attempt := 0; attempt <= eb.config.MaxRetries; attempt++ {
		if attempt > 0 {
			eb.logger.Warnf("Retrying handler %d for event %s (attempt %d/%d)",
				handlerIndex, event.Type(), attempt+1, eb.config.MaxRetries+1)
			time.Sleep(eb.config.RetryDelay)
		}

		if err := handler(ctx, event); err != nil {
			lastErr = err
			eb.logger.Errorf("Handler %d failed for event %s: %v", handlerIndex, event.Type(), err)
			continue
		}

		// Success
		if attempt > 0 {
			eb.logger.Infof("Handler %d succeeded for event %s after %d retries",
				handlerIndex, event.Type(), attempt)
		}
		return nil
	}

	return fmt.Errorf("handler failed after %d attempts: %w", eb.config.MaxRetries+1, lastErr)
}

// PublishAndForget publishes an event asynchronously without waiting for completion
func (eb *EventBus) PublishAndForget(ctx context.Context, event Event) {
	go func() {
		if err := eb.Publish(ctx, event); err != nil {
			eb.logger.Errorf("Failed to publish event %s: %v", event.Type(), err)
		}
	}()
}

// Unsubscribe removes all handlers for a specific event type
func (eb *EventBus) Unsubscribe(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, eventType)
	eb.logger.Debugf("Unsubscribed all handlers for event type: %s", eventType)
}

// GetSubscriberCount returns the number of handlers for an event type
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventType])
}

// GetEventTypes returns all registered event types
func (eb *EventBus) GetEventTypes() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	types := make([]string, 0, len(eb.handlers))
	for eventType := range eb.handlers {
		types = append(types, eventType)
	}
	return types
}

// BasicEvent implements the Event interface
type BasicEvent struct {
	eventType string
	data      interface{}
	timestamp time.Time
	source    string
}

// NewBasicEvent creates a new basic event
func NewBasicEvent(eventType string, data interface{}) Event {
	return &BasicEvent{
		eventType: eventType,
		data:      data,
		timestamp: time.Now(),
		source:    "unknown",
	}
}

// NewBasicEventWithSource creates a new basic event with source
func NewBasicEventWithSource(eventType string, data interface{}, source string) Event {
	return &BasicEvent{
		eventType: eventType,
		data:      data,
		timestamp: time.Now(),
		source:    source,
	}
}

func (e *BasicEvent) Type() string {
	return e.eventType
}

func (e *BasicEvent) Data() interface{} {
	return e.data
}

func (e *BasicEvent) Timestamp() time.Time {
	return e.timestamp
}

func (e *BasicEvent) Source() string {
	return e.source
}

// Common event types for Firestore operations
const (
	EventTypeDocumentCreated       = "document.created"
	EventTypeDocumentUpdated       = "document.updated"
	EventTypeDocumentDeleted       = "document.deleted"
	EventTypeUserAuthenticated     = "user.authenticated"
	EventTypeUserLoggedOut         = "user.logged_out"
	EventTypeSecurityRuleViolation = "security.rule_violation"
)

// noopLogger implements logger.Logger but does nothing (for nil logger)
type noopLogger struct{}

func (n *noopLogger) Debug(args ...interface{})                 {}
func (n *noopLogger) Info(args ...interface{})                  {}
func (n *noopLogger) Warn(args ...interface{})                  {}
func (n *noopLogger) Error(args ...interface{})                 {}
func (n *noopLogger) Fatal(args ...interface{})                 {}
func (n *noopLogger) Debugf(format string, args ...interface{}) {}
func (n *noopLogger) Infof(format string, args ...interface{})  {}
func (n *noopLogger) Warnf(format string, args ...interface{})  {}
func (n *noopLogger) Errorf(format string, args ...interface{}) {}
func (n *noopLogger) Fatalf(format string, args ...interface{}) {}
func (n *noopLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return n
}
func (n *noopLogger) WithContext(ctx context.Context) logger.Logger {
	return n
}
func (n *noopLogger) WithComponent(component string) logger.Logger {
	return n
}
