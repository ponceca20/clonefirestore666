package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Crea un DocumentRepository con mocks para los tests
func newTestDocumentRepositoryForCollections() *DocumentRepository {
	return &DocumentRepository{
		collectionsCol: &mockCollection{},
		documentsCol:   &mockCollection{},
		logger:         &usecase.MockLogger{}, // Use the existing MockLogger from usecase package
	}
}

// En los tests, usa el adaptador para crear el repo compatible
func TestCollectionOperations_CreateCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	col := &model.Collection{ID: primitive.NewObjectID(), CollectionID: "c1"}
	_ = ops.CreateCollection(ctx, "p1", "d1", col)
}

func TestCollectionOperations_GetCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_, _ = ops.GetCollection(ctx, "p1", "d1", "c1")
}

func TestCollectionOperations_UpdateCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	col := &model.Collection{ID: primitive.NewObjectID(), CollectionID: "c1"}
	_ = ops.UpdateCollection(ctx, "p1", "d1", col)
}

func TestCollectionOperations_DeleteCollection(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_ = ops.DeleteCollection(ctx, "p1", "d1", "c1")
}

func TestCollectionOperations_ListCollections(t *testing.T) {
	repo := newTestDocumentRepositoryForCollections()
	ops := NewCollectionOperations(repo)
	ctx := context.Background()
	_, _ = ops.ListCollections(ctx, "p1", "d1")
}

// En los tests, reemplaza NewCollectionOperations(repo) por NewCollectionOperations((*DocumentRepository)(repo)) si es necesario,
// o adapta el constructor para aceptar la interfaz en vez de *DocumentRepository.
