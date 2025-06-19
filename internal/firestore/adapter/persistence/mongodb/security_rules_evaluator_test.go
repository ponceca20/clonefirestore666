package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
)

func TestSecurityRulesEngine_EvaluateAccess(t *testing.T) {
	// Create a properly initialized SecurityRulesEngine for testing
	engine := &SecurityRulesEngine{
		rulesCache: make(map[string][]*CachedRule),
		log:        logger.NewTestLogger(),
		// Note: collection is nil, which will cause LoadRules to fail gracefully
	}

	t.Run("nil security context", func(t *testing.T) {
		ctx := context.TODO()
		result, err := engine.EvaluateAccess(ctx, "read", nil)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, err.Error(), "securityContext is required")
		assert.Contains(t, result.Reason, "securityContext is required")
	})

	t.Run("empty project ID", func(t *testing.T) {
		ctx := context.TODO()
		secCtx := &repository.SecurityContext{
			ProjectID:  "", // empty project ID
			DatabaseID: "db1",
			Path:       "/test/doc1",
		}
		result, err := engine.EvaluateAccess(ctx, "read", secCtx)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, err.Error(), "projectID is required")
		assert.Contains(t, result.Reason, "projectID is required")
	})

	t.Run("empty database ID", func(t *testing.T) {
		ctx := context.TODO()
		secCtx := &repository.SecurityContext{
			ProjectID:  "proj1",
			DatabaseID: "", // empty database ID
			Path:       "/test/doc1",
		}
		result, err := engine.EvaluateAccess(ctx, "read", secCtx)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, err.Error(), "databaseID is required")
		assert.Contains(t, result.Reason, "databaseID is required")
	})
}
