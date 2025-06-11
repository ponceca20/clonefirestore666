package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginTransaction(t *testing.T) {
	uc := newTestFirestoreUsecase()
	txID, err := uc.BeginTransaction(context.Background(), "p1")
	require.NoError(t, err)
	assert.Contains(t, txID, "tx_")
}

func TestCommitTransaction(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.CommitTransaction(context.Background(), "p1", "tx_1")
	assert.NoError(t, err)
}

func TestRunBatchWrite(t *testing.T) {
	uc := newTestFirestoreUsecase()
	resp, err := uc.RunBatchWrite(context.Background(), BatchWriteRequest{
		ProjectID:  "p1",
		DatabaseID: "d1",
		Writes: []model.BatchWriteOperation{{
			Type:       "update",
			Path:       "projects/p1/databases/d1/documents/c1/doc1",
			DocumentID: "doc1",
			Data:       map[string]interface{}{"foo": "bar"},
		}},
	})
	require.NoError(t, err)
	assert.Len(t, resp.WriteResults, 1)
}

// Agrega un helper local para los tests de transacciones, usando los mocks centralizados.
func newTestFirestoreUsecase() FirestoreUsecaseInterface {
	return NewFirestoreUsecase(
		&MockFirestoreRepo{},
		nil, // securityRepo mock si es necesario
		nil, // queryEngine mock si es necesario
		&MockLogger{},
	)
}
