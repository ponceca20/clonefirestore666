package mongodb

import (
	"firestore-clone/internal/firestore/domain/repository"
)

// TenantAwareCollectionRepository implements repository for dynamic collections per tenant/context

type TenantAwareCollectionRepository struct {
	manager repository.CollectionManager
}

func NewTenantAwareCollectionRepository(manager repository.CollectionManager) *TenantAwareCollectionRepository {
	return &TenantAwareCollectionRepository{manager: manager}
}

// Example: GetCollectionReference returns a collection reference for the given context
func (r *TenantAwareCollectionRepository) GetCollectionReference(projectID, databaseID, collectionPath string) (repository.CollectionReference, error) {
	return r.manager.GetOrCreateCollection(projectID, databaseID, collectionPath)
}

// Implement other repository methods as needed, using the dynamic collection reference
