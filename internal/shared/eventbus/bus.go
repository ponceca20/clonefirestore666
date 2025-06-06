package eventbus

import (
	"context"
	"sync"
)

// Event represents a generic event
type Event interface {
	Type() string
	Data() interface{}
}

// Handler defines the event handler function type
type Handler func(ctx context.Context, event Event) error

// EventBus represents an in-memory event bus
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewEventBus creates a new event bus instance
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe adds a handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type()]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// Unsubscribe removes all handlers for a specific event type
func (eb *EventBus) Unsubscribe(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, eventType)
}

// BasicEvent implements the Event interface
type BasicEvent struct {
	eventType string
	data      interface{}
}

// NewBasicEvent creates a new basic event
func NewBasicEvent(eventType string, data interface{}) Event {
	return &BasicEvent{
		eventType: eventType,
		data:      data,
	}
}

func (e *BasicEvent) Type() string {
	return e.eventType
}

func (e *BasicEvent) Data() interface{} {
	return e.data
}
