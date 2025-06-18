package usecase_test

import (
	"context"
	"errors"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Utiliza los mocks centralizados para mantener la arquitectura hexagonal y el código limpio.
func newTestFirestoreUsecaseWithMocks(repo repository.FirestoreRepository) FirestoreUsecaseInterface {
	return NewFirestoreUsecase(repo, nil, nil, nil, &MockLogger{})
}

// MockRepository especializado para pruebas de colecciones
// Solo sobreescribe los métodos necesarios para cada test

type collectionRepoMock struct {
	repository.FirestoreRepository
	CreateCollectionFn   func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error
	GetCollectionFn      func(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error)
	UpdateCollectionFn   func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error
	ListCollectionsFn    func(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error)
	DeleteCollectionFn   func(ctx context.Context, projectID, databaseID, collectionID string) error
	GetDocumentFn        func(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error)
	ListSubcollectionsFn func(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error)
}

func (m *collectionRepoMock) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	if m.CreateCollectionFn != nil {
		return m.CreateCollectionFn(ctx, projectID, databaseID, collection)
	}
	return nil
}
func (m *collectionRepoMock) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	if m.GetCollectionFn != nil {
		return m.GetCollectionFn(ctx, projectID, databaseID, collectionID)
	}
	return &model.Collection{CollectionID: collectionID}, nil
}
func (m *collectionRepoMock) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	if m.UpdateCollectionFn != nil {
		return m.UpdateCollectionFn(ctx, projectID, databaseID, collection)
	}
	return nil
}
func (m *collectionRepoMock) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	if m.ListCollectionsFn != nil {
		return m.ListCollectionsFn(ctx, projectID, databaseID)
	}
	return []*model.Collection{}, nil
}
func (m *collectionRepoMock) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	if m.DeleteCollectionFn != nil {
		return m.DeleteCollectionFn(ctx, projectID, databaseID, collectionID)
	}
	return nil
}
func (m *collectionRepoMock) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	if m.GetDocumentFn != nil {
		return m.GetDocumentFn(ctx, projectID, databaseID, collectionID, documentID)
	}
	return &model.Document{ID: primitive.NewObjectID()}, nil
}
func (m *collectionRepoMock) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	if m.ListSubcollectionsFn != nil {
		return m.ListSubcollectionsFn(ctx, projectID, databaseID, collectionID, documentID)
	}
	return []string{}, nil
}
func (m *collectionRepoMock) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	// Simula la existencia del proyecto para los tests
	if projectID == "" {
		return nil, errors.New("projectID required")
	}
	return &model.Project{ProjectID: projectID}, nil
}
func (m *collectionRepoMock) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	// Simula la existencia de la base de datos para los tests
	if projectID == "" || databaseID == "" {
		return nil, errors.New("projectID y databaseID requeridos")
	}
	return &model.Database{DatabaseID: databaseID}, nil
}

func TestCreateCollection_Success(t *testing.T) {
	repo := &collectionRepoMock{
		CreateCollectionFn: func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
			return nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	coll, err := uc.CreateCollection(context.Background(), CreateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Equal(t, "c1", coll.CollectionID)
}

func TestCreateCollection_Error(t *testing.T) {
	repo := &collectionRepoMock{
		CreateCollectionFn: func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
			return errors.New("fail create")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	_, err := uc.CreateCollection(context.Background(), CreateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	assert.Error(t, err)
}

func TestGetCollection_Success(t *testing.T) {
	repo := &collectionRepoMock{
		GetCollectionFn: func(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
			return &model.Collection{CollectionID: collectionID}, nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	coll, err := uc.GetCollection(context.Background(), GetCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Equal(t, "c1", coll.CollectionID)
}

func TestGetCollection_Error(t *testing.T) {
	repo := &collectionRepoMock{
		GetCollectionFn: func(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
			return nil, errors.New("fail get")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	_, err := uc.GetCollection(context.Background(), GetCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	assert.Error(t, err)
}

func TestUpdateCollection_Success(t *testing.T) {
	repo := &collectionRepoMock{
		UpdateCollectionFn: func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
			return nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	err := uc.UpdateCollection(context.Background(), UpdateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		Collection:   &model.Collection{CollectionID: "c1"},
	})
	assert.NoError(t, err)
}

func TestUpdateCollection_Error(t *testing.T) {
	repo := &collectionRepoMock{
		UpdateCollectionFn: func(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
			return errors.New("fail update")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	err := uc.UpdateCollection(context.Background(), UpdateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		Collection:   &model.Collection{CollectionID: "c1"},
	})
	assert.Error(t, err)
}

func TestListCollections_Success(t *testing.T) {
	repo := &collectionRepoMock{
		ListCollectionsFn: func(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
			return []*model.Collection{{CollectionID: "c1"}}, nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	colls, err := uc.ListCollections(context.Background(), ListCollectionsRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
	})
	require.NoError(t, err)
	assert.Len(t, colls, 1)
}

func TestListCollections_Error(t *testing.T) {
	repo := &collectionRepoMock{
		ListCollectionsFn: func(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
			return nil, errors.New("fail list")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	_, err := uc.ListCollections(context.Background(), ListCollectionsRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
	})
	assert.Error(t, err)
}

func TestDeleteCollection_Success(t *testing.T) {
	repo := &collectionRepoMock{
		DeleteCollectionFn: func(ctx context.Context, projectID, databaseID, collectionID string) error {
			return nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	err := uc.DeleteCollection(context.Background(), DeleteCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	assert.NoError(t, err)
}

func TestDeleteCollection_Error(t *testing.T) {
	repo := &collectionRepoMock{
		DeleteCollectionFn: func(ctx context.Context, projectID, databaseID, collectionID string) error {
			return errors.New("fail delete")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	err := uc.DeleteCollection(context.Background(), DeleteCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	assert.Error(t, err)
}

func TestListSubcollections_Success(t *testing.T) {
	objID := primitive.NewObjectID()
	repo := &collectionRepoMock{
		GetDocumentFn: func(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
			return &model.Document{ID: objID}, nil
		},
		ListSubcollectionsFn: func(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
			return []string{"sub1"}, nil
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	subs, err := uc.ListSubcollections(context.Background(), ListSubcollectionsRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "sub1", subs[0].ID)
}

func TestListSubcollections_ParentNotFound(t *testing.T) {
	repo := &collectionRepoMock{
		GetDocumentFn: func(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
			return nil, errors.New("not found")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	_, err := uc.ListSubcollections(context.Background(), ListSubcollectionsRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	assert.Error(t, err)
}

func TestListSubcollections_ListError(t *testing.T) {
	objID := primitive.NewObjectID()
	repo := &collectionRepoMock{
		GetDocumentFn: func(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
			return &model.Document{ID: objID}, nil
		},
		ListSubcollectionsFn: func(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
			return nil, errors.New("fail list subcollections")
		},
	}
	uc := newTestFirestoreUsecaseWithMocks(repo)
	_, err := uc.ListSubcollections(context.Background(), ListSubcollectionsRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	assert.Error(t, err)
}
