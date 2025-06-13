package mongodb

import (
	"firestore-clone/internal/firestore/usecase"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantAwareCollectionRepository_GetCollectionReference(t *testing.T) {
	factory := usecase.NewDefaultCollectionFactory(&usecase.FirestorePathCollectionStrategy{})
	manager := usecase.NewInMemoryCollectionManager(factory)
	repo := NewTenantAwareCollectionRepository(manager)
	ref, err := repo.GetCollectionReference("p1", "d1", "inventario_productos")
	assert.NoError(t, err)
	assert.Equal(t, "docs_p1_d1_inventario_productos", ref.Name())
}
