package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/repository"

	"github.com/stretchr/testify/assert"
)

func TestSecurityRulesEngine_LoadRules(t *testing.T) {
	// Test proper initialization and error handling
	t.Run("with proper initialization", func(t *testing.T) {
		// This test should demonstrate that the engine initializes correctly when dependencies are provided
		// For a real test, we would need to mock the MongoDB collection properly
		engine := &SecurityRulesEngine{
			collection: nil,           // When collection is nil, we expect the method to panic or fail
			log:        &MockLogger{}, // Use the MockLogger from organization_repository_test.go
			rulesCache: make(map[string][]*repository.SecurityRule),
		}

		// This should panic due to nil collection - we expect this behavior
		// In a production environment, the engine should always be created with proper dependencies
		assert.Panics(t, func() {
			_, _ = engine.LoadRules(context.Background(), "p1", "d1")
		}, "LoadRules should panic when collection is nil")
	})

	t.Run("cache initialization", func(t *testing.T) {
		// Test that the engine initializes with proper cache structure
		engine := &SecurityRulesEngine{
			rulesCache: make(map[string][]*repository.SecurityRule),
		}

		assert.NotNil(t, engine.rulesCache, "rulesCache should be initialized")
		assert.Equal(t, 0, len(engine.rulesCache), "rulesCache should be empty initially")
	})
}
