package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// DummyEvent implements Event for testing
type DummyEvent struct {
	typeStr   string
	data      interface{}
	timestamp time.Time
	source    string
}

func (e *DummyEvent) Type() string         { return e.typeStr }
func (e *DummyEvent) Data() interface{}    { return e.data }
func (e *DummyEvent) Timestamp() time.Time { return e.timestamp }
func (e *DummyEvent) Source() string       { return e.source }

func TestEventBus_Compile(t *testing.T) {
	// Placeholder: Add real event bus tests here
}

func TestEventBus_SubscribePublish(t *testing.T) {
	bus := NewEventBus(nil)
	var called bool
	bus.Subscribe("test", func(ctx context.Context, event Event) error {
		called = true
		assert.Equal(t, "test", event.Type())
		return nil
	})
	err := bus.Publish(context.Background(), &DummyEvent{typeStr: "test", timestamp: time.Now()})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestEventBus_AsyncPublish(t *testing.T) {
	bus := NewEventBusWithConfig(&noopLogger{}, BusConfig{AsyncProcessing: true})
	ch := make(chan struct{})
	bus.Subscribe("async", func(ctx context.Context, event Event) error {
		ch <- struct{}{}
		return nil
	})
	_ = bus.Publish(context.Background(), &DummyEvent{typeStr: "async", timestamp: time.Now()})
	select {
	case <-ch:
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for async event")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(nil)
	bus.Subscribe("ev", func(ctx context.Context, event Event) error { return nil })
	assert.Equal(t, 1, bus.GetSubscriberCount("ev"))
	bus.Unsubscribe("ev")
	assert.Equal(t, 0, bus.GetSubscriberCount("ev"))
}

func TestEventBus_GetEventTypes(t *testing.T) {
	bus := NewEventBus(nil)
	bus.Subscribe("a", func(ctx context.Context, event Event) error { return nil })
	bus.Subscribe("b", func(ctx context.Context, event Event) error { return nil })
	types := bus.GetEventTypes()
	assert.Contains(t, types, "a")
	assert.Contains(t, types, "b")
}

func TestEventBus_PublishAndForget(t *testing.T) {
	bus := NewEventBus(nil)
	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe("forget", func(ctx context.Context, event Event) error {
		wg.Done()
		return nil
	})
	bus.PublishAndForget(context.Background(), &DummyEvent{typeStr: "forget", timestamp: time.Now()})
	wait := make(chan struct{})
	go func() {
		wg.Wait()
		close(wait)
	}()
	select {
	case <-wait:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for PublishAndForget")
	}
}
