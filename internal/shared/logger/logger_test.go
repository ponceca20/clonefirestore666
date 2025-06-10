package logger

import (
	"context"
	"testing"

	"firestore-clone/internal/shared/contextkeys"

	"github.com/stretchr/testify/assert"
)

func TestLoggerInterface(t *testing.T) {
	// This is a placeholder test to ensure the logger package compiles and can be imported.
	// Real tests should mock or test actual logging behavior.
}

func TestLoggerInterface_Contract(t *testing.T) {
	var _ Logger = NewLogger()
	var _ Logger = NewLoggerWithConfig("info", "json")
}

func TestLogrusLogger_WithFieldsAndContext(t *testing.T) {
	logger := NewLogger()
	logger2 := logger.WithFields(map[string]interface{}{"foo": "bar"})
	assert.NotNil(t, logger2)
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextkeys.UserIDKey, "user1")
	logger3 := logger.WithContext(ctx)
	assert.NotNil(t, logger3)
}

func TestLogrusLogger_WithComponent(t *testing.T) {
	logger := NewLogger()
	logger2 := logger.WithComponent("test-component")
	assert.NotNil(t, logger2)
}
