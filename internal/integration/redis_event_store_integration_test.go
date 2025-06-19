package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"firestore-clone/internal/firestore/adapter/persistence"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRedisClient creates a Redis client for integration testing
func createTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           14, // Use a different test database for integration tests
		Password:     "",
		DialTimeout:  5 * time.Second, // Add connection timeout
		ReadTimeout:  3 * time.Second, // Add read timeout
		WriteTimeout: 3 * time.Second, // Add write timeout
	})
}

// TestRedisEventStoreIntegration tests Redis integration with RealtimeUsecase
func TestRedisEventStoreIntegration(t *testing.T) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Skip if Redis is not available
	client := createTestRedisClient()

	// Test connection with timeout
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for integration testing:", err)
	}

	// Clean up test database
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		client.FlushDB(cleanupCtx)
		client.Close()
	}()

	// Initialize components
	log := logger.NewLogger()
	redisEventStore := persistence.NewRedisEventStore(client, log)
	realtimeUsecase := usecase.NewRealtimeUsecaseWithEventStore(log, redisEventStore)

	// Test data
	subscriberID := "test-subscriber-001"
	subscriptionID := model.SubscriptionID("test-subscription-001")
	firestorePath := "projects/test-project/databases/test-db/documents/users/user123"

	// Create event channel to receive events
	eventChannel := make(chan model.RealtimeEvent, 100)

	// Subscribe to events
	subscribeReq := usecase.SubscribeRequest{
		SubscriberID:   subscriberID,
		SubscriptionID: subscriptionID,
		FirestorePath:  firestorePath,
		EventChannel:   eventChannel,
		ResumeToken:    "",
		Query:          nil,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata:   true,
			IncludeOldData:    true,
			HeartbeatInterval: 30 * time.Second,
		},
	}

	subscribeResp, err := realtimeUsecase.Subscribe(ctx, subscribeReq)
	require.NoError(t, err, "Failed to subscribe")
	assert.Equal(t, subscriptionID, subscribeResp.SubscriptionID)
	assert.True(t, subscribeResp.InitialSnapshot, "Should be initial snapshot")

	// Test publishing events
	events := []model.RealtimeEvent{
		{
			Type:         model.EventTypeAdded,
			FullPath:     firestorePath,
			ProjectID:    "test-project",
			DatabaseID:   "test-db",
			DocumentPath: "users/user123",
			Data:         map[string]interface{}{"name": "John Doe", "age": 30},
			Timestamp:    time.Now(),
		},
		{
			Type:         model.EventTypeModified,
			FullPath:     firestorePath,
			ProjectID:    "test-project",
			DatabaseID:   "test-db",
			DocumentPath: "users/user123",
			Data:         map[string]interface{}{"name": "John Doe", "age": 31},
			OldData:      map[string]interface{}{"name": "John Doe", "age": 30},
			Timestamp:    time.Now().Add(time.Second),
		},
		{
			Type:         model.EventTypeRemoved,
			FullPath:     firestorePath,
			ProjectID:    "test-project",
			DatabaseID:   "test-db",
			DocumentPath: "users/user123",
			Data:         nil,
			Timestamp:    time.Now().Add(2 * time.Second),
		},
	}

	// Publish events and collect received events
	var receivedEvents []model.RealtimeEvent
	var mu sync.Mutex

	// Start goroutine to collect events
	done := make(chan bool)
	go func() {
		defer close(done)
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-eventChannel:
				if event.Type != model.EventTypeHeartbeat {
					mu.Lock()
					receivedEvents = append(receivedEvents, event)
					mu.Unlock()

					// Stop after receiving all expected events
					if len(receivedEvents) >= len(events) {
						return
					}
				}
			case <-timeout:
				t.Error("Timeout waiting for events")
				return
			}
		}
	}()

	// Publish events
	for _, event := range events {
		err := realtimeUsecase.PublishEvent(ctx, event)
		require.NoError(t, err, "Failed to publish event")
		time.Sleep(100 * time.Millisecond) // Small delay between events
	}

	// Wait for all events to be received
	<-done

	// Verify received events
	mu.Lock()
	defer mu.Unlock()

	require.Len(t, receivedEvents, len(events), "Should receive all published events")

	for i, expectedEvent := range events {
		receivedEvent := receivedEvents[i]
		assert.Equal(t, expectedEvent.Type, receivedEvent.Type)
		assert.Equal(t, expectedEvent.FullPath, receivedEvent.FullPath)
		assert.Equal(t, expectedEvent.ProjectID, receivedEvent.ProjectID)
		assert.Equal(t, expectedEvent.DatabaseID, receivedEvent.DatabaseID)
		assert.Equal(t, expectedEvent.DocumentPath, receivedEvent.DocumentPath)
		assert.Equal(t, expectedEvent.Data, receivedEvent.Data)
		assert.Equal(t, expectedEvent.OldData, receivedEvent.OldData)

		// Verify that Redis-generated fields are present
		assert.NotEmpty(t, receivedEvent.ResumeToken, "Resume token should be set")
		assert.Greater(t, receivedEvent.SequenceNumber, int64(0), "Sequence number should be positive")
		assert.Equal(t, string(subscriptionID), receivedEvent.SubscriptionID, "Subscription ID should match")
	}

	// Test event persistence and retrieval
	storedEvents, err := realtimeUsecase.GetEventsSince(ctx, firestorePath, "")
	require.NoError(t, err, "Failed to get stored events")
	assert.Len(t, storedEvents, len(events), "Should have all events stored in Redis")

	// Test resume functionality
	if len(storedEvents) > 1 {
		resumeToken := storedEvents[0].ResumeToken
		eventsFromResume, err := realtimeUsecase.GetEventsSince(ctx, firestorePath, resumeToken)
		require.NoError(t, err, "Failed to get events from resume token")
		assert.Len(t, eventsFromResume, len(events)-1, "Should get events after resume token")
	}

	// Cleanup subscription
	unsubscribeReq := usecase.UnsubscribeRequest{
		SubscriberID:   subscriberID,
		SubscriptionID: subscriptionID,
	}
	err = realtimeUsecase.Unsubscribe(ctx, unsubscribeReq)
	require.NoError(t, err, "Failed to unsubscribe")
}

// TestRedisEventStoreResilience tests Redis resilience scenarios
func TestRedisEventStoreResilience(t *testing.T) {
	// Skip if Redis is not available
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for resilience testing:", err)
	}

	// Clean up test database
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Initialize components
	log := logger.NewLogger()
	redisEventStore := persistence.NewRedisEventStore(client, log)

	// Test storing events when Redis is available
	testEvent := model.RealtimeEvent{
		Type:           model.EventTypeAdded,
		FullPath:       "projects/test/databases/test/documents/test/123",
		ProjectID:      "test",
		DatabaseID:     "test",
		DocumentPath:   "test/123",
		Data:           map[string]interface{}{"test": "data"},
		Timestamp:      time.Now(),
		SequenceNumber: 1,
	}

	err := redisEventStore.StoreEvent(ctx, testEvent)
	require.NoError(t, err, "Should store event successfully when Redis is available")

	// Test retrieving events
	events, err := redisEventStore.GetEventsSince(ctx, testEvent.FullPath, "")
	require.NoError(t, err, "Should retrieve events successfully")
	assert.Len(t, events, 1, "Should have one stored event")

	// Test event count
	count := redisEventStore.GetEventCount(testEvent.FullPath)
	assert.Equal(t, 1, count, "Should have correct event count")

	// Test cleanup
	err = redisEventStore.CleanupOldEvents(ctx, time.Hour)
	require.NoError(t, err, "Should cleanup successfully")
}

// TestRedisEventStorePerformance tests Redis performance characteristics
func TestRedisEventStorePerformance(t *testing.T) {
	// Skip if Redis is not available or not in performance testing mode
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for performance testing:", err)
	}

	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Clean up test database
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Initialize components
	log := logger.NewLogger()
	redisEventStore := persistence.NewRedisEventStore(client, log)

	// Performance test: Store many events
	numEvents := 1000
	firestorePath := "projects/perf-test/databases/test/documents/perf/test"

	start := time.Now()

	for i := 0; i < numEvents; i++ {
		event := model.RealtimeEvent{
			Type:           model.EventTypeAdded,
			FullPath:       firestorePath,
			ProjectID:      "perf-test",
			DatabaseID:     "test",
			DocumentPath:   "perf/test",
			Data:           map[string]interface{}{"index": i, "data": "test-data"},
			Timestamp:      time.Now(),
			SequenceNumber: int64(i + 1),
		}

		err := redisEventStore.StoreEvent(ctx, event)
		require.NoError(t, err, "Failed to store event %d", i)
	}

	storeTime := time.Since(start)
	t.Logf("Stored %d events in %v (%.2f events/sec)",
		numEvents, storeTime, float64(numEvents)/storeTime.Seconds())

	// Performance test: Retrieve events
	start = time.Now()
	events, err := redisEventStore.GetEventsSince(ctx, firestorePath, "")
	require.NoError(t, err, "Failed to retrieve events")
	retrieveTime := time.Since(start)

	assert.Len(t, events, numEvents, "Should retrieve all stored events")
	t.Logf("Retrieved %d events in %v (%.2f events/sec)",
		len(events), retrieveTime, float64(len(events))/retrieveTime.Seconds())

	// Verify event ordering and content
	for i, event := range events {
		assert.Equal(t, int64(i+1), event.SequenceNumber, "Events should be in order")
		assert.Equal(t, i, int(event.Data["index"].(float64)), "Event data should be correct")
	}
}

// BenchmarkRedisEventStore_StoreEvent benchmarks event storage
func BenchmarkRedisEventStore_StoreEvent(b *testing.B) {
	client := createTestRedisClient()
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available for benchmarking")
	}

	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	log := logger.NewLogger()
	store := persistence.NewRedisEventStore(client, log)

	event := model.RealtimeEvent{
		Type:           model.EventTypeAdded,
		FullPath:       "projects/bench/databases/test/documents/bench/test",
		ProjectID:      "bench",
		DatabaseID:     "test",
		DocumentPath:   "bench/test",
		Data:           map[string]interface{}{"benchmark": true, "iteration": 0},
		Timestamp:      time.Now(),
		SequenceNumber: 1,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.SequenceNumber = int64(i)
		event.Data["iteration"] = i
		err := store.StoreEvent(ctx, event)
		if err != nil {
			b.Fatal("Failed to store event:", err)
		}
	}
}

// BenchmarkRedisEventStore_GetEventsSince benchmarks event retrieval
func BenchmarkRedisEventStore_GetEventsSince(b *testing.B) {
	client := createTestRedisClient()
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available for benchmarking")
	}

	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	log := logger.NewLogger()
	store := persistence.NewRedisEventStore(client, log)

	// Pre-populate with events
	firestorePath := "projects/bench/databases/test/documents/bench/test"
	for i := 0; i < 100; i++ {
		event := model.RealtimeEvent{
			Type:           model.EventTypeAdded,
			FullPath:       firestorePath,
			ProjectID:      "bench",
			DatabaseID:     "test",
			DocumentPath:   "bench/test",
			Data:           map[string]interface{}{"iteration": i},
			Timestamp:      time.Now(),
			SequenceNumber: int64(i),
		}
		store.StoreEvent(ctx, event)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.GetEventsSince(ctx, firestorePath, "")
		if err != nil {
			b.Fatal("Failed to get events:", err)
		}
	}
}
