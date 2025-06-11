package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Usa el mock centralizado de logger y el helper newTestFirestoreUsecase si aplica.
func newTestRealtimeUsecase(t *testing.T) RealtimeUsecase {
	return NewRealtimeUsecase(&MockLogger{})
}

func TestRealtimeUsecase_SubscribeUnsubscribe(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID1 := "client1"
	// Usar path completo
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)

	err := rtu.Subscribe(ctx, subscriberID1, path1, eventChan1)
	require.NoError(t, err, "Subscribe should not return an error")

	// Unsubscribe
	err = rtu.Unsubscribe(ctx, subscriberID1, path1)
	require.NoError(t, err, "Unsubscribe should not return an error")
	// Check if channel is closed (optional, depends on implementation detail, current Unsubscribe does not close it)
	// _, ok := <-eventChan1
	// assert.False(t, ok, "Event channel should be closed after unsubscribe by the component managing the channel (e.g. ws_handler)")
	// Note: RealtimeUsecase itself doesn't close the channel; the subscriber (ws_handler) is responsible.

	// Try to unsubscribe again (should be graceful)
	err = rtu.Unsubscribe(ctx, subscriberID1, path1)
	require.NoError(t, err, "Unsubscribing a non-existent subscription should be graceful")

	// Try to unsubscribe a non-existent client from an existing path (if any other client is subscribed)
	// or a non-existent path
	err = rtu.Unsubscribe(ctx, "nonExistentClient", path1)
	require.NoError(t, err, "Unsubscribing a non-existent client should be graceful")

	err = rtu.Unsubscribe(ctx, subscriberID1, "projects/test-project/databases/test-db/documents/nonExistentPath")
	require.NoError(t, err, "Unsubscribing from a non-existent path should be graceful")
}

func TestRealtimeUsecase_PublishEvent_SingleSubscriber(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID1 := "client1"
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)
	rtu.Subscribe(ctx, subscriberID1, path1, eventChan1)

	event := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err := rtu.PublishEvent(ctx, event)
	require.NoError(t, err)

	select {
	case receivedEvent := <-eventChan1:
		assert.Equal(t, event, receivedEvent, "Received event does not match published event")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestRealtimeUsecase_PublishEvent_MultipleSubscribersSamePath(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)
	eventChan2 := make(chan model.RealtimeEvent, 1)
	rtu.Subscribe(ctx, "client1", path1, eventChan1)
	rtu.Subscribe(ctx, "client2", path1, eventChan2)

	event := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	rtu.PublishEvent(ctx, event)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		select {
		case receivedEvent := <-eventChan1:
			assert.Equal(t, event, receivedEvent)
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Client1 timed out waiting for event")
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case receivedEvent := <-eventChan2:
			assert.Equal(t, event, receivedEvent)
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Client2 timed out waiting for event")
		}
	}()
	wg.Wait()
}

func TestRealtimeUsecase_PublishEvent_DifferentPaths(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	eventChan1 := make(chan model.RealtimeEvent, 1)
	eventChan2 := make(chan model.RealtimeEvent, 1) // For a different path
	path1 := "projects/test-project/databases/test-db/documents/path1"
	path2 := "projects/test-project/databases/test-db/documents/path2"
	rtu.Subscribe(ctx, "client1", path1, eventChan1)
	rtu.Subscribe(ctx, "client2", path2, eventChan2)

	eventForPath1 := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "path1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	rtu.PublishEvent(ctx, eventForPath1)

	// Check path1 receives it
	select {
	case receivedEvent := <-eventChan1:
		assert.Equal(t, eventForPath1, receivedEvent)
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Client1 (path1) timed out waiting for event")
	}

	// Check path2 does not receive it
	select {
	case receivedEvent := <-eventChan2:
		t.Fatalf("Client2 (path2) received event meant for path1: %+v", receivedEvent)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event
	}
}

func TestRealtimeUsecase_PublishEvent_NoSubscribers(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	event := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     "projects/test-project/databases/test-db/documents/path/none",
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "path/none",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err := rtu.PublishEvent(ctx, event)
	assert.NoError(t, err, "Publishing to a path with no subscribers should not error")
}

func TestRealtimeUsecase_Subscribe_OverwriteSubscription(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID := "client1"
	path := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChanOld := make(chan model.RealtimeEvent, 1)
	eventChanNew := make(chan model.RealtimeEvent, 1)

	// First subscription
	err := rtu.Subscribe(ctx, subscriberID, path, eventChanOld)
	require.NoError(t, err)
	// Second subscription to the same path by the same client, with a new channel
	err = rtu.Subscribe(ctx, subscriberID, path, eventChanNew)
	require.NoError(t, err) // Current implementation overwrites

	event := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-database",
		DocumentPath: "docs/test_doc",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event)
	require.NoError(t, err)

	// New channel should receive the event
	select {
	case receivedEvent := <-eventChanNew:
		assert.Equal(t, event, receivedEvent, "New channel should receive the event")
	case <-time.After(50 * time.Millisecond):
		t.Fatal("New channel timed out waiting for event")
	}

	// Old channel should not receive the event (as it was overwritten)
	select {
	case ev := <-eventChanOld:
		t.Fatalf("Old channel received an event unexpectedly: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// Expected behavior: old channel gets nothing
	}
}

func TestRealtimeUsecase_PublishEvent_ChannelFull(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID := "client_slow"
	path := "projects/test-project/databases/test-db/documents/docs/doc_slow"
	// Unbuffered channel to simulate immediate blocking
	eventChan := make(chan model.RealtimeEvent)
	err := rtu.Subscribe(ctx, subscriberID, path, eventChan)
	require.NoError(t, err)

	event := model.RealtimeEvent{
		Type:         model.EventTypeUpdated,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-database",
		DocumentPath: "docs/doc_slow",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}

	// Publish. Since the channel is unbuffered and no one is reading,
	// the send attempt in PublishEvent should hit the 'default' case in the select.
	err = rtu.PublishEvent(ctx, event)
	require.NoError(t, err, "PublishEvent itself should not error if a client channel is full")

	// Try to read, but it should be empty as the send should have been skipped
	select {
	case ev := <-eventChan:
		t.Fatalf("Event was received on unbuffered/blocked channel, but should have been dropped: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// Expected: event was dropped for this subscriber
	}
}
