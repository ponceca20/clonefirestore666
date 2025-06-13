package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirestorePathCollectionStrategy(t *testing.T) {
	strategy := &FirestorePathCollectionStrategy{}
	name := strategy.CollectionName("p1", "d1", "inventario_productos")
	assert.Equal(t, "docs_p1_d1_inventario_productos", name)
}

func TestOptimizedCollectionStrategy(t *testing.T) {
	strategy := &OptimizedCollectionStrategy{}
	name := strategy.CollectionName("p1", "d1", "ventas_pedidos")
	assert.Equal(t, "opt_p1_d1_ventas_pedidos", name)
}

func TestInMemoryCollectionManager_Caching(t *testing.T) {
	factory := NewDefaultCollectionFactory(&FirestorePathCollectionStrategy{})
	manager := NewInMemoryCollectionManager(factory)
	ref1, _ := manager.GetOrCreateCollection("p1", "d1", "c1")
	ref2, _ := manager.GetOrCreateCollection("p1", "d1", "c1")
	assert.Equal(t, ref1, ref2)
}
