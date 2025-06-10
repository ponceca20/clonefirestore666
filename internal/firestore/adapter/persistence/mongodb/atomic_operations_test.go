package mongodb

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mock minimal compatible con CollectionUpdater
// para pruebas unitarias limpias y sin dependencias externas.
type mockCollectionUpdater struct{}

func (m *mockCollectionUpdater) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func TestAtomicOperations_AtomicIncrement(t *testing.T) {
	mockCol := &mockCollectionUpdater{}
	atomicOps := NewAtomicOperations(mockCol)
	ctx := context.Background()

	err := atomicOps.AtomicIncrement(ctx, "p1", "d1", "c1", "doc1", "field", 1)
	if err != nil {
		t.Fatalf("AtomicIncrement failed: %v", err)
	}
}

// Puedes agregar tests similares para AtomicArrayUnion, AtomicArrayRemove, etc.
