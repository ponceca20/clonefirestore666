package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollectionUpdater implementa la interfaz para operaciones atómicas en pruebas
type MockCollectionUpdater struct{}

func (m *MockCollectionUpdater) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	// Simular una actualización exitosa para pruebas
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func TestAtomicOperations(t *testing.T) {
	// Configuración común para las pruebas
	mockCol := &MockCollectionUpdater{}
	atomicOps := NewAtomicOperations(mockCol)
	ctx := context.Background()

	t.Run("AtomicIncrement - campo numérico", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "c1", "doc1", "counter", 1)
		assert.NoError(t, err)
	})

	t.Run("AtomicArrayUnion - agregar elementos únicos", func(t *testing.T) {
		values := []*model.FieldValue{
			{
				ValueType: model.FieldTypeString,
				Value:     "item1",
			},
			{
				ValueType: model.FieldTypeString,
				Value:     "item2",
			},
		}

		err := atomicOps.AtomicArrayUnion(ctx, "p1", "d1", "c1", "doc1", "tags", values)
		assert.NoError(t, err)
	})

	t.Run("AtomicArrayRemove - eliminar elementos", func(t *testing.T) {
		values := []*model.FieldValue{
			{
				ValueType: model.FieldTypeString,
				Value:     "item1",
			},
		}

		err := atomicOps.AtomicArrayRemove(ctx, "p1", "d1", "c1", "doc1", "tags", values)
		assert.NoError(t, err)
	})

	t.Run("AtomicServerTimestamp - establecer hora del servidor", func(t *testing.T) {
		err := atomicOps.AtomicServerTimestamp(ctx, "p1", "d1", "c1", "doc1", "lastUpdated")
		assert.NoError(t, err)
	})

	t.Run("AtomicDelete - eliminar campos", func(t *testing.T) {
		fields := []string{"obsoleteField", "temporaryField"}
		err := atomicOps.AtomicDelete(ctx, "p1", "d1", "c1", "doc1", fields)
		assert.NoError(t, err)
	})

	t.Run("AtomicSetIfEmpty - establecer valor predeterminado", func(t *testing.T) {
		value := &model.FieldValue{
			ValueType: model.FieldTypeString,
			Value:     "default value",
		}

		err := atomicOps.AtomicSetIfEmpty(ctx, "p1", "d1", "c1", "doc1", "status", value)
		assert.NoError(t, err)
	})

	t.Run("AtomicMaximum - establecer valor máximo", func(t *testing.T) {
		err := atomicOps.AtomicMaximum(ctx, "p1", "d1", "c1", "doc1", "highScore", 1000)
		assert.NoError(t, err)
	})

	t.Run("AtomicMinimum - establecer valor mínimo", func(t *testing.T) {
		err := atomicOps.AtomicMinimum(ctx, "p1", "d1", "c1", "doc1", "lowScore", 0)
		assert.NoError(t, err)
	})
}

func TestAtomicOperations_Validation(t *testing.T) {
	mockCol := &MockCollectionUpdater{}
	atomicOps := NewAtomicOperations(mockCol)
	ctx := context.Background()

	t.Run("valida el ID del proyecto", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "", "d1", "c1", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project ID")
	})

	t.Run("valida el ID de la base de datos", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "", "c1", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database ID")
	})

	t.Run("valida el ID de la colección", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection ID")
	})

	t.Run("valida el ID del documento", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "c1", "", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document ID")
	})

	t.Run("valida el nombre del campo", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "c1", "doc1", "", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field")
	})
}

func TestAtomicOperations_EdgeCases(t *testing.T) {
	mockCol := &MockCollectionUpdater{}
	atomicOps := NewAtomicOperations(mockCol)
	ctx := context.Background()

	t.Run("maneja valor de campo nulo", func(t *testing.T) {
		err := atomicOps.AtomicSetIfEmpty(ctx, "p1", "d1", "c1", "doc1", "field", nil)
		assert.Error(t, err)
	})

	t.Run("maneja arreglos de valores vacíos", func(t *testing.T) {
		err := atomicOps.AtomicArrayUnion(ctx, "p1", "d1", "c1", "doc1", "array", nil)
		assert.Error(t, err)
	})

	t.Run("maneja lista de campos vacía", func(t *testing.T) {
		err := atomicOps.AtomicDelete(ctx, "p1", "d1", "c1", "doc1", []string{})
		assert.Error(t, err)
	})
}
