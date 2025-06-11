package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCollection(t *testing.T) {
	uc := newTestFirestoreUsecase()
	coll, err := uc.CreateCollection(context.Background(), CreateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Equal(t, "c1", coll.CollectionID)
}

func TestGetCollection(t *testing.T) {
	uc := newTestFirestoreUsecase()
	coll, err := uc.GetCollection(context.Background(), GetCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Equal(t, "c1", coll.CollectionID)
}

func TestUpdateCollection(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.UpdateCollection(context.Background(), UpdateCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		Collection:   &model.Collection{CollectionID: "c1"},
	})
	assert.NoError(t, err)
}

func TestListCollections(t *testing.T) {
	uc := newTestFirestoreUsecase()
	colls, err := uc.ListCollections(context.Background(), ListCollectionsRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
	})
	require.NoError(t, err)
	assert.Len(t, colls, 1)
}

func TestDeleteCollection(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.DeleteCollection(context.Background(), DeleteCollectionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	assert.NoError(t, err)
}

func TestListSubcollections(t *testing.T) {
	uc := newTestFirestoreUsecase()
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
