package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/repository"

	"github.com/stretchr/testify/assert"
)

func TestSecurityRulesEngine_EvaluateAccess(t *testing.T) {
	engine := &SecurityRulesEngine{}

	t.Run("nil context", func(t *testing.T) {
		result, err := engine.EvaluateAccess(nil, "read", nil)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "nil context")
	})

	t.Run("nil security context", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.EvaluateAccess(ctx, "read", nil)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "nil security context")
	})

	t.Run("missing project or database ID", func(t *testing.T) {
		ctx := context.Background()
		secCtx := &repository.SecurityContext{
			ProjectID:  "", // empty project ID
			DatabaseID: "db1",
		}
		result, err := engine.EvaluateAccess(ctx, "read", secCtx)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "missing project ID")
	})
}
