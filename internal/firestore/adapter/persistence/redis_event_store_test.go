package persistence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRedisClient creates a Redis client for testing
func createTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           15, // Use a test database
		Password:     "",
		DialTimeout:  5 * time.Second, // Add connection timeout
		ReadTimeout:  3 * time.Second, // Add read timeout
		WriteTimeout: 3 * time.Second, // Add write timeout
	})
}

// TestRedisEventStore_StoreEvent tests storing events in Redis Streams
func TestRedisEventStore_StoreEvent(t *testing.T) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Skip if Redis is not available
	client := createTestRedisClient()

	// Test Redis connection with timeout
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}
	// Clean up test database before starting
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		client.FlushDB(cleanupCtx)
		client.Close()
	}()

	// Clear any existing data in the test stream
	testStreamKey := "projects/test-project/databases/test-db/documents/users/user123"
	client.Del(ctx, testStreamKey)

	// Initialize test components
	log := logger.NewLogger()
	store := NewRedisEventStore(client, log)

	// Create test event with consistent data types
	testEvent := model.RealtimeEvent{
		Type:           model.EventTypeAdded,
		FullPath:       testStreamKey,
		ProjectID:      "test-project",
		DatabaseID:     "test-db",
		DocumentPath:   "users/user123",
		Data:           map[string]interface{}{"name": "John Doe", "age": float64(30)}, // Use float64 for consistency
		OldData:        nil,
		Timestamp:      time.Now(),
		ResumeToken:    "test-token-001",
		SequenceNumber: 1,
		SubscriptionID: "sub-123",
	}

	// Test storing event
	err := store.StoreEvent(ctx, testEvent)
	require.NoError(t, err, "Failed to store event in Redis")

	// Verify event was stored by checking stream length
	streamLength, err := client.XLen(ctx, testEvent.FullPath).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), streamLength, "Stream should contain exactly one event")

	// Verify event data by reading from stream
	messages, err := client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{testEvent.FullPath, "0"},
		Count:   1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Len(t, messages[0].Messages, 1)

	msg := messages[0].Messages[0]
	assert.Equal(t, string(testEvent.Type), msg.Values["type"])
	assert.Equal(t, testEvent.FullPath, msg.Values["fullPath"])
	assert.Equal(t, testEvent.ProjectID, msg.Values["projectId"])
	assert.Equal(t, testEvent.DatabaseID, msg.Values["databaseId"])
	assert.Equal(t, testEvent.DocumentPath, msg.Values["documentPath"])

	// Verify serialized data
	var storedData map[string]interface{}
	err = json.Unmarshal([]byte(msg.Values["data"].(string)), &storedData)
	require.NoError(t, err)
	assert.Equal(t, testEvent.Data, storedData)
}

// TestRedisEventStore_GetEventsSince tests retrieving events from Redis Streams
func TestRedisEventStore_GetEventsSince(t *testing.T) {
	// Skip if Redis is not available
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}
	// Clean up test database
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		client.FlushDB(cleanupCtx)
		client.Close()
	}()

	// Initialize test components
	log := logger.NewLogger()
	store := NewRedisEventStore(client, log)

	// Create test path
	testPath := "projects/test-project/databases/test-db/documents/users/user123"

	// Clear any existing data in the test stream
	client.Del(ctx, testPath)
	// Store multiple events
	events := []model.RealtimeEvent{
		{
			Type:           model.EventTypeAdded,
			FullPath:       testPath,
			ProjectID:      "test-project",
			DatabaseID:     "test-db",
			DocumentPath:   "users/user123",
			Data:           map[string]interface{}{"name": "John Doe", "age": float64(30)},
			Timestamp:      time.Now(),
			ResumeToken:    "token-001",
			SequenceNumber: 1,
		},
		{
			Type:           model.EventTypeModified,
			FullPath:       testPath,
			ProjectID:      "test-project",
			DatabaseID:     "test-db",
			DocumentPath:   "users/user123",
			Data:           map[string]interface{}{"name": "John Doe", "age": float64(31)},
			OldData:        map[string]interface{}{"name": "John Doe", "age": float64(30)},
			Timestamp:      time.Now().Add(time.Second),
			ResumeToken:    "token-002",
			SequenceNumber: 2,
		},
		{
			Type:           model.EventTypeRemoved,
			FullPath:       testPath,
			ProjectID:      "test-project",
			DatabaseID:     "test-db",
			DocumentPath:   "users/user123",
			Data:           nil,
			Timestamp:      time.Now().Add(2 * time.Second),
			ResumeToken:    "token-003",
			SequenceNumber: 3,
		},
	}

	// Store all events
	for _, event := range events {
		err := store.StoreEvent(ctx, event)
		require.NoError(t, err, "Failed to store event")
	}
	// Test: Get all events (from beginning)
	t.Log("Testing GetEventsSince with empty resume token...")
	retrievedEvents, err := store.GetEventsSince(ctx, testPath, "")
	require.NoError(t, err)
	assert.Len(t, retrievedEvents, 3, "Should retrieve all 3 events")

	// Verify event types in order
	if len(retrievedEvents) >= 3 {
		assert.Equal(t, model.EventTypeAdded, retrievedEvents[0].Type)
		assert.Equal(t, model.EventTypeModified, retrievedEvents[1].Type)
		assert.Equal(t, model.EventTypeRemoved, retrievedEvents[2].Type)
	}

	// Test: Get events from non-existent path
	t.Log("Testing GetEventsSince with non-existent path...")
	emptyEvents, err := store.GetEventsSince(ctx, "non-existent-path", "")
	require.NoError(t, err)
	assert.Len(t, emptyEvents, 0, "Should return empty slice for non-existent path")
}

// TestRedisEventStore_GetEventCount tests event counting functionality
func TestRedisEventStore_GetEventCount(t *testing.T) {
	// Skip if Redis is not available
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Clean up test database
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Initialize test components
	log := logger.NewLogger()
	store := NewRedisEventStore(client, log)

	// Test initial count (should be 0)
	count := store.GetEventCount("projects/test-project/databases/test-db/documents/users/user123")
	assert.Equal(t, 0, count, "Initial count should be 0")

	// Store some events
	testPath := "projects/test-project/databases/test-db/documents/users/user123"
	for i := 0; i < 5; i++ {
		event := model.RealtimeEvent{
			Type:           model.EventTypeAdded,
			FullPath:       testPath,
			ProjectID:      "test-project",
			DatabaseID:     "test-db",
			DocumentPath:   "users/user123",
			Data:           map[string]interface{}{"count": i},
			Timestamp:      time.Now(),
			SequenceNumber: int64(i + 1),
		}
		err := store.StoreEvent(ctx, event)
		require.NoError(t, err)
	}

	// Test count after storing events
	count = store.GetEventCount(testPath)
	assert.Equal(t, 5, count, "Count should be 5 after storing 5 events")

	// Test total count across all streams
	totalCount := store.GetEventCount("")
	assert.Equal(t, 5, totalCount, "Total count should be 5")
}

// TestRedisEventStore_CleanupOldEvents tests cleanup functionality
func TestRedisEventStore_CleanupOldEvents(t *testing.T) {
	// Skip if Redis is not available
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Clean up test database
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Initialize test components
	log := logger.NewLogger()
	store := NewRedisEventStore(client, log)

	// Create a stream with many events to test trimming
	testPath := "projects/test-project/databases/test-db/documents/users/user123"

	// Store more than 10000 events to trigger trimming
	for i := 0; i < 15000; i++ {
		event := model.RealtimeEvent{
			Type:           model.EventTypeAdded,
			FullPath:       testPath,
			ProjectID:      "test-project",
			DatabaseID:     "test-db",
			DocumentPath:   "users/user123",
			Data:           map[string]interface{}{"count": i},
			Timestamp:      time.Now(),
			SequenceNumber: int64(i + 1),
		}
		err := store.StoreEvent(ctx, event)
		require.NoError(t, err)
	}

	// Verify initial count
	initialCount := store.GetEventCount(testPath)
	assert.Equal(t, 15000, initialCount, "Should have 15000 events initially")

	// Run cleanup
	err := store.CleanupOldEvents(ctx, time.Hour)
	require.NoError(t, err)

	// Verify count after cleanup (should be trimmed to 10000)
	finalCount := store.GetEventCount(testPath)
	assert.Equal(t, 10000, finalCount, "Should have 10000 events after cleanup")
}

// TestRedisEventStore_ParseEventFromMessage tests message parsing
func TestRedisEventStore_ParseEventFromMessage(t *testing.T) {
	log := logger.NewLogger()
	store := NewRedisEventStore(nil, log)
	// Create test Redis message
	testData := map[string]interface{}{
		"name": "John Doe",
		"age":  float64(30), // Use float64 to match JSON unmarshaling behavior
	}

	testOldData := map[string]interface{}{
		"name": "Jane Doe",
		"age":  float64(25), // Use float64 to match JSON unmarshaling behavior
	}

	dataBytes, _ := json.Marshal(testData)
	oldDataBytes, _ := json.Marshal(testOldData)

	msg := redis.XMessage{
		ID: "1234567890123-0",
		Values: map[string]interface{}{
			"type":           "modified",
			"fullPath":       "projects/test/databases/test/documents/users/123",
			"projectId":      "test",
			"databaseId":     "test",
			"documentPath":   "users/123",
			"data":           string(dataBytes),
			"oldData":        string(oldDataBytes),
			"timestamp":      "1640995200000000000",
			"resumeToken":    "token-123",
			"sequenceNumber": "42",
			"subscriptionId": "sub-456",
		},
	}

	// Parse the message
	event, err := store.parseEventFromMessage(msg)
	require.NoError(t, err)

	// Verify parsed event
	assert.Equal(t, model.EventTypeModified, event.Type)
	assert.Equal(t, "projects/test/databases/test/documents/users/123", event.FullPath)
	assert.Equal(t, "test", event.ProjectID)
	assert.Equal(t, "test", event.DatabaseID)
	assert.Equal(t, "users/123", event.DocumentPath)
	assert.Equal(t, testData, event.Data)
	assert.Equal(t, testOldData, event.OldData)
	assert.Equal(t, model.ResumeToken("token-123"), event.ResumeToken)
	assert.Equal(t, int64(42), event.SequenceNumber)
	assert.Equal(t, "sub-456", event.SubscriptionID)

	// Verify timestamp parsing
	expectedTime := time.Unix(0, 1640995200000000000)
	assert.Equal(t, expectedTime, event.Timestamp)
}
