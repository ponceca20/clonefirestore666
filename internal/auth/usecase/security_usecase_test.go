package usecase_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
)

type mockSecurityLogger struct{}

func (m *mockSecurityLogger) Debug(args ...interface{})                 {}
func (m *mockSecurityLogger) Info(args ...interface{})                  {}
func (m *mockSecurityLogger) Warn(args ...interface{})                  {}
func (m *mockSecurityLogger) Error(args ...interface{})                 {}
func (m *mockSecurityLogger) Fatal(args ...interface{})                 {}
func (m *mockSecurityLogger) Debugf(format string, args ...interface{}) {}
func (m *mockSecurityLogger) Infof(format string, args ...interface{})  {}
func (m *mockSecurityLogger) Warnf(format string, args ...interface{})  {}
func (m *mockSecurityLogger) Errorf(format string, args ...interface{}) {}
func (m *mockSecurityLogger) Fatalf(format string, args ...interface{}) {}
func (m *mockSecurityLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return m
}

func TestSecurityUsecase_ValidateRead_Success(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"

	// Act
	err := uc.ValidateRead(ctx, userID, path)

	// Assert
	assert.NoError(t, err, "ValidateRead should allow access for now")
}

func TestSecurityUsecase_ValidateWrite_Success(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"
	data := map[string]interface{}{"field": "value"}

	// Act
	err := uc.ValidateWrite(ctx, userID, path, data)

	// Assert
	assert.NoError(t, err, "ValidateWrite should allow access for now")
}

func TestSecurityUsecase_ValidateDelete_Success(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"

	// Act
	err := uc.ValidateDelete(ctx, userID, path)

	// Assert
	assert.NoError(t, err, "ValidateDelete should allow access for now")
}

func TestSecurityUsecase_ValidateOperations_WithEmptyUserID(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := ""
	path := "documents/doc1"
	data := map[string]interface{}{"field": "value"}

	// Act & Assert
	assert.NoError(t, uc.ValidateRead(ctx, userID, path))
	assert.NoError(t, uc.ValidateWrite(ctx, userID, path, data))
	assert.NoError(t, uc.ValidateDelete(ctx, userID, path))
}

func TestSecurityUsecase_ValidateOperations_WithEmptyPath(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := ""
	data := map[string]interface{}{"field": "value"}

	// Act & Assert
	assert.NoError(t, uc.ValidateRead(ctx, userID, path))
	assert.NoError(t, uc.ValidateWrite(ctx, userID, path, data))
	assert.NoError(t, uc.ValidateDelete(ctx, userID, path))
}

func TestSecurityUsecase_ValidateOperations_WithNilData(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"

	// Act
	err := uc.ValidateWrite(ctx, userID, path, nil)

	// Assert
	assert.NoError(t, err, "ValidateWrite should handle nil data")
}

func TestSecurityUsecase_ValidateOperations_WithCancelledContext(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	userID := "user-123"
	path := "documents/doc1"
	data := map[string]interface{}{"field": "value"}

	// Act & Assert - Should still work since current implementation doesn't check context
	assert.NoError(t, uc.ValidateRead(ctx, userID, path))
	assert.NoError(t, uc.ValidateWrite(ctx, userID, path, data))
	assert.NoError(t, uc.ValidateDelete(ctx, userID, path))
}

func TestSecurityUsecase_ValidateOperations_WithTimeout(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout

	userID := "user-123"
	path := "documents/doc1"
	data := map[string]interface{}{"field": "value"}

	// Act & Assert - Should still work since current implementation doesn't check context
	assert.NoError(t, uc.ValidateRead(ctx, userID, path))
	assert.NoError(t, uc.ValidateWrite(ctx, userID, path, data))
	assert.NoError(t, uc.ValidateDelete(ctx, userID, path))
}

func TestSecurityUsecase_ValidateOperations_ConcurrentAccess(t *testing.T) {
	// Arrange
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()

	// Act - Test concurrent access
	for i := 0; i < 10; i++ {
		go func(id int) {
			userID := "user-" + string(rune(id))
			path := "documents/doc" + string(rune(id))
			data := map[string]interface{}{"id": id}

			assert.NoError(t, uc.ValidateRead(ctx, userID, path))
			assert.NoError(t, uc.ValidateWrite(ctx, userID, path, data))
			assert.NoError(t, uc.ValidateDelete(ctx, userID, path))
		}(i)
	}
}

func BenchmarkSecurityUsecase_ValidateRead(b *testing.B) {
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.ValidateRead(ctx, userID, path)
	}
}

func BenchmarkSecurityUsecase_ValidateWrite(b *testing.B) {
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"
	data := map[string]interface{}{"field": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.ValidateWrite(ctx, userID, path, data)
	}
}

func BenchmarkSecurityUsecase_ValidateDelete(b *testing.B) {
	uc := usecase.NewSecurityUsecase(&mockSecurityLogger{})
	ctx := context.Background()
	userID := "user-123"
	path := "documents/doc1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.ValidateDelete(ctx, userID, path)
	}
}
