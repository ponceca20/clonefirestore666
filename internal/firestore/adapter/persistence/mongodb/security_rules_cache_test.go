package mongodb

import (
	"context"
	"sync"
	"testing"

	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSecurityRulesEngine_ClearCache(t *testing.T) {
	ctx := context.Background()

	// Connect to test MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
		return
	}
	defer client.Disconnect(ctx)

	// Use test database
	testDB := client.Database("firestore_cache_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	engine := NewSecurityRulesEngine(testDB, log).(*SecurityRulesEngine)

	// Test clearing cache
	engine.ClearCache("p1", "d1")

	// Test clearing all cache
	engine.ClearAllCache()
}

func TestSecurityRulesEngine_ClearCache_MinimalInit(t *testing.T) {
	// Test with minimal initialization for unit testing
	engine := &SecurityRulesEngine{
		rulesCache: make(map[string][]*CachedRule),
		cacheMu:    sync.RWMutex{},
		log:        logger.NewTestLogger(),
	}

	// This should not panic
	engine.ClearCache("p1", "d1")
	engine.ClearAllCache()
}
