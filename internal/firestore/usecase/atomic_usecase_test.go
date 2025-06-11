//go:build !atomic_usecase_test_helpers

package usecase_test

import (
	"context"
	"testing"

	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicIncrement(t *testing.T) {
	uc := newTestFirestoreUsecase()
	resp, err := uc.AtomicIncrement(context.Background(), AtomicIncrementRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Field:        "count",
		IncrementBy:  int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.NewValue)
}

func TestAtomicArrayUnion(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.AtomicArrayUnion(context.Background(), AtomicArrayUnionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Field:        "arr",
		Elements:     []interface{}{1, 2, 3},
	})
	assert.NoError(t, err)
}

func TestAtomicArrayRemove(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.AtomicArrayRemove(context.Background(), AtomicArrayRemoveRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Field:        "arr",
		Elements:     []interface{}{1, 2, 3},
	})
	assert.NoError(t, err)
}

func TestAtomicServerTimestamp(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.AtomicServerTimestamp(context.Background(), AtomicServerTimestampRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Field:        "ts",
	})
	assert.NoError(t, err)
}
