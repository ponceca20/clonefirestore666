package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDatabase(t *testing.T) {
	uc := newTestFirestoreUsecase()
	db, err := uc.CreateDatabase(context.Background(), CreateDatabaseRequest{
		ProjectID: "p1",
		Database:  &model.Database{DatabaseID: "d1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "d1", db.DatabaseID)
}

func TestGetDatabase(t *testing.T) {
	uc := newTestFirestoreUsecase()
	db, err := uc.GetDatabase(context.Background(), GetDatabaseRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
	})
	require.NoError(t, err)
	assert.Equal(t, "d1", db.DatabaseID)
}

func TestUpdateDatabase(t *testing.T) {
	uc := newTestFirestoreUsecase()
	db, err := uc.UpdateDatabase(context.Background(), UpdateDatabaseRequest{
		ProjectID: "p1",
		Database:  &model.Database{DatabaseID: "d1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "d1", db.DatabaseID)
}

func TestDeleteDatabase(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.DeleteDatabase(context.Background(), DeleteDatabaseRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
	})
	assert.NoError(t, err)
}

func TestListDatabases(t *testing.T) {
	uc := newTestFirestoreUsecase()
	dbs, err := uc.ListDatabases(context.Background(), ListDatabasesRequest{
		ProjectID: "p1",
	})
	require.NoError(t, err)
	assert.Len(t, dbs, 1)
	assert.Equal(t, "d1", dbs[0].DatabaseID)
}
