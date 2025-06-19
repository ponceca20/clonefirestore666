package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
)

type mockDatabaseProvider struct {
	store *MockDocumentStore
}

// Ahora retorna la interfaz CollectionInterface
func (m *mockDatabaseProvider) Collection(name string) CollectionInterface {
	return &MockCollectionWithStore{store: m.store}
}

func (m *mockDatabaseProvider) Client() interface{} {
	return nil
}

// Ajusta el constructor para aceptar la interfaz
func setupAtomicOpsTest() (*AtomicOperations, *MockDocumentStore) {
	store := NewMockDocumentStore()
	dbProvider := &mockDatabaseProvider{store: store}
	atomicOps := NewAtomicOperations(dbProvider)
	return atomicOps, store
}

func TestAtomicOperations(t *testing.T) {
	atomicOps, _ := setupAtomicOpsTest()
	ctx := context.Background()

	t.Run("AtomicIncrement - campo numérico", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "users", "doc1", "counter", 1)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicArrayUnion - agregar elementos únicos", func(t *testing.T) {
		values := []*model.FieldValue{{ValueType: model.FieldTypeString, Value: "item1"}, {ValueType: model.FieldTypeString, Value: "item2"}}
		err := atomicOps.AtomicArrayUnion(ctx, "p1", "d1", "users", "doc1", "tags", values)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicArrayRemove - eliminar elementos", func(t *testing.T) {
		values := []*model.FieldValue{{ValueType: model.FieldTypeString, Value: "item1"}}
		err := atomicOps.AtomicArrayRemove(ctx, "p1", "d1", "users", "doc1", "tags", values)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicServerTimestamp - establecer hora del servidor", func(t *testing.T) {
		err := atomicOps.AtomicServerTimestamp(ctx, "p1", "d1", "users", "doc1", "lastUpdated")
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicDelete - eliminar campos", func(t *testing.T) {
		fields := []string{"obsoleteField", "temporaryField"}
		err := atomicOps.AtomicDelete(ctx, "p1", "d1", "users", "doc1", fields)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicSetIfEmpty - establecer valor predeterminado", func(t *testing.T) {
		value := &model.FieldValue{ValueType: model.FieldTypeString, Value: "default value"}
		err := atomicOps.AtomicSetIfEmpty(ctx, "p1", "d1", "users", "doc1", "status", value)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicMaximum - establecer valor máximo", func(t *testing.T) {
		err := atomicOps.AtomicMaximum(ctx, "p1", "d1", "users", "doc1", "highScore", 1000)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})

	t.Run("AtomicMinimum - establecer valor mínimo", func(t *testing.T) {
		err := atomicOps.AtomicMinimum(ctx, "p1", "d1", "users", "doc1", "lowScore", 0)
		assert.Error(t, err, "debe fallar si el documento no existe")
	})
}

func TestAtomicOperations_Validation(t *testing.T) {
	atomicOps, _ := setupAtomicOpsTest()
	ctx := context.Background()

	t.Run("valida el ID del proyecto", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "", "d1", "users", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project ID")
	})

	t.Run("valida el ID de la base de datos", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "", "users", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database ID")
	})

	t.Run("valida el ID de la colección", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "", "doc1", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection ID")
	})

	t.Run("valida el ID del documento", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "users", "", "field", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document ID")
	})

	t.Run("valida el nombre del campo", func(t *testing.T) {
		err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "users", "doc1", "", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field")
	})
}

func TestAtomicOperations_EdgeCases(t *testing.T) {
	atomicOps, _ := setupAtomicOpsTest()
	ctx := context.Background()

	t.Run("maneja valor de campo nulo", func(t *testing.T) {
		err := atomicOps.AtomicSetIfEmpty(ctx, "p1", "d1", "users", "doc1", "field", nil)
		assert.Error(t, err)
	})

	t.Run("maneja arreglos de valores vacíos", func(t *testing.T) {
		err := atomicOps.AtomicArrayUnion(ctx, "p1", "d1", "users", "doc1", "array", nil)
		assert.Error(t, err)
	})

	t.Run("maneja lista de campos vacía", func(t *testing.T) {
		err := atomicOps.AtomicDelete(ctx, "p1", "d1", "users", "doc1", []string{})
		assert.Error(t, err)
	})
}
