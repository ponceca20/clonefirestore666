package usecase_test

import (
	"context"
	"testing"

	. "firestore-clone/internal/firestore/domain/model"
	usecase "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIndex(t *testing.T) {
	uc := newTestFirestoreUsecase()
	idx, err := uc.CreateIndex(context.Background(), usecase.CreateIndexRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		Index: Index{
			Name:       "idx1",
			Collection: "c1",
			Fields:     []IndexField{{Path: "f1", Order: IndexFieldOrderAscending}},
			State:      "READY",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "idx1", idx.Name)
}

func TestDeleteIndex(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.DeleteIndex(context.Background(), usecase.DeleteIndexRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		IndexName:  "idx1",
	})
	assert.NoError(t, err)
}

func TestListIndexes(t *testing.T) {
	uc := newTestFirestoreUsecase()
	idxs, err := uc.ListIndexes(context.Background(), usecase.ListIndexesRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Len(t, idxs, 1)
	assert.Equal(t, "idx1", idxs[0].Name)
}
