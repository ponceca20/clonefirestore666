package usecase_test

import (
	"context"
	"testing"

	. "firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
)

func TestAtomicIncrement_Valid(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	resp, err := uc.AtomicIncrement(ctx, AtomicIncrementRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "inventario",
		DocumentID:   "doc1",
		Field:        "stock",
		IncrementBy:  5,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil {
		assert.GreaterOrEqual(t, resp.NewValue, int64(5))
	}
}

func TestAtomicIncrement_InvalidField(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	_, err := uc.AtomicIncrement(ctx, AtomicIncrementRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "inventario",
		DocumentID:   "doc1",
		Field:        "",
		IncrementBy:  1,
	})
	assert.Error(t, err)
}

func TestAtomicArrayUnion_Valid(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicArrayUnion(ctx, AtomicArrayUnionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "roles",
		Elements:     []interface{}{"admin", "editor"},
	})
	assert.NoError(t, err)
}

func TestAtomicArrayUnion_EmptyElements(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicArrayUnion(ctx, AtomicArrayUnionRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "roles",
		Elements:     []interface{}{},
	})
	assert.Error(t, err)
}

func TestAtomicArrayRemove_Valid(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicArrayRemove(ctx, AtomicArrayRemoveRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "roles",
		Elements:     []interface{}{"admin"},
	})
	assert.NoError(t, err)
}

func TestAtomicArrayRemove_EmptyElements(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicArrayRemove(ctx, AtomicArrayRemoveRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "roles",
		Elements:     []interface{}{},
	})
	assert.Error(t, err)
}

func TestAtomicServerTimestamp_Valid(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicServerTimestamp(ctx, AtomicServerTimestampRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "lastLogin",
	})
	assert.NoError(t, err)
}

func TestAtomicServerTimestamp_InvalidField(t *testing.T) {
	uc := newTestFirestoreUsecase()
	ctx := context.Background()
	err := uc.AtomicServerTimestamp(ctx, AtomicServerTimestampRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "usuarios",
		DocumentID:   "user1",
		Field:        "",
	})
	assert.Error(t, err)
}
