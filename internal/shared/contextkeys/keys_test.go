//go:build unit
// +build unit

package contextkeys

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextKey_String(t *testing.T) {
	key := contextKey("testKey")
	assert.Equal(t, "firestore-clone context key testKey", key.String())
}

func TestContextKeys_Usage(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, UserIDKey, "user-123")
	ctx = context.WithValue(ctx, UserEmailKey, "user@example.com")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-abc")
	ctx = context.WithValue(ctx, RequestIDKey, "req-456")
	ctx = context.WithValue(ctx, ProjectIDKey, "project-789")
	ctx = context.WithValue(ctx, DatabaseIDKey, "db-xyz")
	ctx = context.WithValue(ctx, TokenKey, "token-foo")
	ctx = context.WithValue(ctx, ClaimsKey, "claims-bar")
	ctx = context.WithValue(ctx, ComponentKey, "component-logger")
	ctx = context.WithValue(ctx, OperationKey, "operation-read")

	assert.Equal(t, "user-123", ctx.Value(UserIDKey))
	assert.Equal(t, "user@example.com", ctx.Value(UserEmailKey))
	assert.Equal(t, "tenant-abc", ctx.Value(TenantIDKey))
	assert.Equal(t, "req-456", ctx.Value(RequestIDKey))
	assert.Equal(t, "project-789", ctx.Value(ProjectIDKey))
	assert.Equal(t, "db-xyz", ctx.Value(DatabaseIDKey))
	assert.Equal(t, "token-foo", ctx.Value(TokenKey))
	assert.Equal(t, "claims-bar", ctx.Value(ClaimsKey))
	assert.Equal(t, "component-logger", ctx.Value(ComponentKey))
	assert.Equal(t, "operation-read", ctx.Value(OperationKey))
}
