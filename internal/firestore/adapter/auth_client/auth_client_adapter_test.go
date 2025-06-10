package auth_client

import (
	"context"
	"errors"
	"testing"

	"firestore-clone/internal/firestore/domain/client"

	"github.com/stretchr/testify/assert"
)

func newSimpleAuthClient() client.AuthClient {
	return NewSimpleAuthClient()
}

func TestSimpleAuthClient_ValidateToken(t *testing.T) {
	client := newSimpleAuthClient()
	ctx := context.Background()

	t.Run("valid token returns userID", func(t *testing.T) {
		userID, err := client.ValidateToken(ctx, "sometoken123456")
		assert.NoError(t, err)
		assert.Contains(t, userID, "test-user-")
	})

	t.Run("empty token returns error", func(t *testing.T) {
		_, err := client.ValidateToken(ctx, "")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidToken))
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		_, err := client.ValidateToken(ctx, "invalid")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidToken) || err.Error() == "invalid token")
	})
}

func TestSimpleAuthClient_GetUserByID(t *testing.T) {
	client := newSimpleAuthClient()
	ctx := context.Background()
	userID := "user-123"
	projectID := "project-abc"

	t.Run("valid userID returns user", func(t *testing.T) {
		user, err := client.GetUserByID(ctx, userID, projectID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, "user@example.com", user.Email)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
	})

	t.Run("empty userID returns error", func(t *testing.T) {
		user, err := client.GetUserByID(ctx, "", projectID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound) || err.Error() == "user ID is empty")
	})
}
