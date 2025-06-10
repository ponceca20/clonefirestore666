package auth_client

import (
	"context"
	"errors"
	"testing"
	"time"

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
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, projectID, user.ProjectID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		assert.WithinDuration(t, time.Now(), user.CreatedAt, time.Second)
	})

	t.Run("empty userID returns error", func(t *testing.T) {
		user, err := client.GetUserByID(ctx, "", projectID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound))
	})
}
