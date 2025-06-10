package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
)

// Mock DocumentCollection para pruebas
// Solo implementa los métodos requeridos
type mockDocumentCollection struct{}

func (m *mockDocumentCollection) Indexes() IndexManager {
	return nil
}
func (m *mockDocumentCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return 0, nil
}

// Mock Logger para pruebas
// Implementa solo los métodos usados (pueden ser vacíos)
type mockLogger struct{}

func (m *mockLogger) Info(args ...interface{})                               {}
func (m *mockLogger) Error(args ...interface{})                              {}
func (m *mockLogger) Warn(args ...interface{})                               {}
func (m *mockLogger) Debug(args ...interface{})                              {}
func (m *mockLogger) Fatal(args ...interface{})                              {}
func (m *mockLogger) Infof(format string, args ...interface{})               {}
func (m *mockLogger) Errorf(format string, args ...interface{})              {}
func (m *mockLogger) Warnf(format string, args ...interface{})               {}
func (m *mockLogger) Debugf(format string, args ...interface{})              {}
func (m *mockLogger) Fatalf(format string, args ...interface{})              {}
func (m *mockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *mockLogger) WithComponent(component string) logger.Logger           { return m }

// Elimina la definición de mockCollection, mockDeleteResult, mockUpdateResult, mockCursor y mockSingleResult de este archivo.
// Usa los mocks compartidos del otro archivo de test.

func TestIndexOperations_CreateIndex(t *testing.T) {
	col := NewIndexCollectionAdapter(&mockCollection{})
	docCol := &mockDocumentCollection{}
	logger := &mockLogger{}
	ops := NewIndexOperations(col, docCol, logger)
	err := ops.CreateIndex(context.Background(), "p1", "d1", "c1", nil)
	assert.Error(t, err) // Should error due to nil index
}
