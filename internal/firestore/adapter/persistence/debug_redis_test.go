package persistence

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisEventStore_GetEventsSince_Debug is a simplified debug version
func TestRedisEventStore_GetEventsSince_Debug(t *testing.T) {
	// Skip if Redis is not available
	client := createTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Clean up test database
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		client.FlushDB(cleanupCtx)
		client.Close()
	}()

	// Initialize test components
	log := logger.NewLogger()
	store := NewRedisEventStore(client, log)

	// Create test path
	testPath := "projects/debug-test/databases/test-db/documents/users/debug123"

	// Clear any existing data in the test stream
	client.Del(ctx, testPath)

	// Store a single simple event
	testEvent := model.RealtimeEvent{
		Type:           model.EventTypeAdded,
		FullPath:       testPath,
		ProjectID:      "debug-test",
		DatabaseID:     "test-db",
		DocumentPath:   "users/debug123",
		Data:           map[string]interface{}{"name": "Debug User"},
		Timestamp:      time.Now(),
		ResumeToken:    "debug-token-001",
		SequenceNumber: 1,
	}

	// Store the event
	t.Log("Storing test event...")
	err := store.StoreEvent(ctx, testEvent)
	require.NoError(t, err, "Failed to store event")

	// Try to retrieve the event
	t.Log("Retrieving events...")
	retrievedEvents, err := store.GetEventsSince(ctx, testPath, "")
	require.NoError(t, err, "Failed to retrieve events")

	t.Logf("Retrieved %d events", len(retrievedEvents))
	assert.Len(t, retrievedEvents, 1, "Should retrieve exactly 1 event")

	if len(retrievedEvents) > 0 {
		t.Logf("First event type: %v", retrievedEvents[0].Type)
		t.Logf("First event path: %s", retrievedEvents[0].FullPath)
		assert.Equal(t, model.EventTypeAdded, retrievedEvents[0].Type)
		assert.Equal(t, testPath, retrievedEvents[0].FullPath)
	}
}
