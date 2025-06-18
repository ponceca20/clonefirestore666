package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newHexFirestoreUsecase crea un caso de uso con mocks centralizados y logger dummy.
func newHexFirestoreUsecase() usecase.FirestoreUsecaseInterface {
	return usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), // Mock repo hexagonal
		nil,                            // Security repo (no necesario aquí)
		nil,                            // Query engine (no necesario aquí)
		nil,                            // Projection service (no necesario aquí)
		&usecase.MockLogger{},          // Logger dummy
	)
}

func TestCreateIndex(t *testing.T) {
	uc := newHexFirestoreUsecase()
	ctx := context.Background()
	idx, err := uc.CreateIndex(ctx, usecase.CreateIndexRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		Index: model.Index{
			Name:       "idx1",
			Collection: "c1",
			Fields:     []model.IndexField{{Path: "f1", Order: model.IndexFieldOrderAscending}},
			State:      "READY",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "idx1", idx.Name)
	assert.Equal(t, "c1", idx.Collection)
	assert.Equal(t, model.IndexFieldOrderAscending, idx.Fields[0].Order)
}

func TestDeleteIndex(t *testing.T) {
	uc := newHexFirestoreUsecase()
	ctx := context.Background()
	err := uc.DeleteIndex(ctx, usecase.DeleteIndexRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		IndexName:  "idx1",
	})
	assert.NoError(t, err)
}

func TestListIndexes(t *testing.T) {
	uc := newHexFirestoreUsecase()
	ctx := context.Background()
	idxs, err := uc.ListIndexes(ctx, usecase.ListIndexesRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Len(t, idxs, 1)
	assert.Equal(t, "idx1", idxs[0].Name)
	assert.Equal(t, "c1", idxs[0].Collection)
	assert.Equal(t, model.IndexFieldOrderAscending, idxs[0].Fields[0].Order)
}
