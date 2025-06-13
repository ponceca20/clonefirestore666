package mongodb

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
)

// DocumentRepositoryAdapter es un adaptador limpio para la implementación hexagonal principal.
type DocumentRepositoryAdapter struct {
	repo FirestoreHexRepository // Usa la interfaz hexagonal principal
}

// FirestoreHexRepository define los métodos mínimos esperados del repositorio hexagonal
// (puedes expandir según tus necesidades)
type FirestoreHexRepository interface {
	GetDocumentByPath(ctx context.Context, firestorePath string) (*model.Document, error)
	SetDocumentByPath(ctx context.Context, firestorePath string, doc *model.Document) error
}

func NewDocumentRepositoryAdapter(repo FirestoreHexRepository) *DocumentRepositoryAdapter {
	return &DocumentRepositoryAdapter{repo: repo}
}

// GetDocument delega en el repositorio hexagonal
func (a *DocumentRepositoryAdapter) GetDocument(ctx context.Context, firestorePath string) (*model.Document, error) {
	return a.repo.GetDocumentByPath(ctx, firestorePath)
}

// SetDocument delega en el repositorio hexagonal
func (a *DocumentRepositoryAdapter) SetDocument(ctx context.Context, firestorePath string, doc *model.Document) error {
	return a.repo.SetDocumentByPath(ctx, firestorePath, doc)
}

// Nota: Si necesitas métodos adicionales, agrégalos siguiendo el mismo patrón.
