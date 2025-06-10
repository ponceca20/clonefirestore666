package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextUtils_Compile(t *testing.T) {
	// Placeholder: Add real context utility tests here
}

func TestGetSetContextValues(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "tenant1")
	ctx = WithUserID(ctx, "user1")
	ctx = WithProjectID(ctx, "project1")
	ctx = WithDatabaseID(ctx, "db1")
	ctx = WithRequestID(ctx, "req1")
	ctx = WithUserEmail(ctx, "user@example.com")
	ctx = WithComponent(ctx, "componentA")
	ctx = WithOperation(ctx, "opX")

	tenantID, err := GetTenantIDFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "tenant1", tenantID)

	userID, err := GetUserIDFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "user1", userID)

	projectID, err := GetProjectIDFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "project1", projectID)

	dbID, err := GetDatabaseIDFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "db1", dbID)

	reqID, err := GetRequestIDFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "req1", reqID)

	email, err := GetUserEmailFromContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", email)

	assert.True(t, HasTenantID(ctx))
	assert.True(t, HasUserID(ctx))
	assert.True(t, HasProjectID(ctx))
	assert.True(t, HasDatabaseID(ctx))

	assert.Equal(t, "tenant1", GetTenantIDOrDefault(ctx, "default"))
	assert.Equal(t, "user1", GetUserIDOrDefault(ctx, "default"))
	assert.Equal(t, "project1", GetProjectIDOrDefault(ctx, "default"))
	assert.Equal(t, "db1", GetDatabaseIDOrDefault(ctx, "default"))
}

func TestContextUtils_MissingValues(t *testing.T) {
	ctx := context.Background()
	_, err := GetTenantIDFromContext(ctx)
	assert.Error(t, err)
	assert.Equal(t, "tenantID not found in context", err.Error())

	assert.Equal(t, "default", GetTenantIDOrDefault(ctx, "default"))
	assert.False(t, HasTenantID(ctx))
}
