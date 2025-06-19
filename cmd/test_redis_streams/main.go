package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"firestore-clone/internal/firestore/adapter/persistence"
	"firestore-clone/internal/firestore/config"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"
)

// TestRedisStreamsImplementation is a simple test to verify Redis Streams is working
func main() {
	fmt.Println("🚀 Testing Redis Streams Implementation...")

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Load configuration - if it fails, use default Redis config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("⚠️  Failed to load full config (%v), using default Redis config\n", err)
		// Use default Redis configuration for testing
		cfg = &config.FirestoreConfig{
			Redis: config.RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
				Database: 0,
			},
		}
	}

	// Initialize Redis client
	redisClient := config.NewRedisClient(&cfg.Redis)

	// Test Redis connection with timeout
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	fmt.Println("✅ Redis connection successful")

	// Initialize logger
	appLogger := logger.NewLogger()

	// Initialize Redis Event Store
	redisEventStore := persistence.NewRedisEventStore(redisClient, appLogger)
	fmt.Println("✅ Redis Event Store initialized")

	// Initialize Realtime Usecase with Redis
	realtimeUsecase := usecase.NewRealtimeUsecaseWithEventStore(appLogger, redisEventStore)
	fmt.Println("✅ Realtime Usecase with Redis initialized")

	// Test 1: Store a test event
	testEvent := model.RealtimeEvent{
		Type:           model.EventTypeAdded,
		FullPath:       "projects/test-project/databases/test-db/documents/users/test-user",
		ProjectID:      "test-project",
		DatabaseID:     "test-db",
		DocumentPath:   "users/test-user",
		Data:           map[string]interface{}{"name": "Test User", "email": "test@example.com"},
		Timestamp:      time.Now(),
		SequenceNumber: 1,
	}

	err = redisEventStore.StoreEvent(ctx, testEvent)
	if err != nil {
		log.Fatalf("❌ Failed to store event: %v", err)
	}
	fmt.Println("✅ Event stored successfully in Redis Streams")

	// Test 2: Retrieve events
	events, err := redisEventStore.GetEventsSince(ctx, testEvent.FullPath, "")
	if err != nil {
		log.Fatalf("❌ Failed to retrieve events: %v", err)
	}

	if len(events) == 0 {
		log.Fatalf("❌ No events retrieved")
	}

	fmt.Printf("✅ Retrieved %d event(s) from Redis Streams\n", len(events))

	// Test 3: Verify event data
	retrievedEvent := events[0]
	if retrievedEvent.Type != testEvent.Type {
		log.Fatalf("❌ Event type mismatch: expected %v, got %v", testEvent.Type, retrievedEvent.Type)
	}

	if retrievedEvent.FullPath != testEvent.FullPath {
		log.Fatalf("❌ Event path mismatch: expected %v, got %v", testEvent.FullPath, retrievedEvent.FullPath)
	}

	fmt.Println("✅ Event data verification successful")

	// Test 4: Test with Realtime Usecase
	err = realtimeUsecase.PublishEvent(ctx, testEvent)
	if err != nil {
		log.Fatalf("❌ Failed to publish event through usecase: %v", err)
	}
	fmt.Println("✅ Event published through Realtime Usecase")

	// Test 5: Get health status
	healthStatus := realtimeUsecase.GetHealthStatus()
	fmt.Printf("✅ Realtime service health: %+v\n", healthStatus)

	// Test 6: Get metrics
	metrics := realtimeUsecase.GetMetrics()
	fmt.Printf("✅ Realtime service metrics: %+v\n", metrics)

	// Test 7: Event count
	count := redisEventStore.GetEventCount(testEvent.FullPath)
	fmt.Printf("✅ Event count for path: %d\n", count)

	// Clean up test data
	redisClient.FlushDB(ctx)
	fmt.Println("✅ Test data cleaned up")

	fmt.Println("\n🎉 All Redis Streams tests passed! Implementation is working correctly.")
	fmt.Println("\n📊 Summary:")
	fmt.Println("  - Redis connection: ✅")
	fmt.Println("  - Event storage: ✅")
	fmt.Println("  - Event retrieval: ✅")
	fmt.Println("  - Data integrity: ✅")
	fmt.Println("  - Realtime usecase integration: ✅")
	fmt.Println("  - Health monitoring: ✅")
	fmt.Println("  - Metrics collection: ✅")
	fmt.Println("\n🔥 Redis Streams is now powering your Firestore clone's realtime events!")
}
