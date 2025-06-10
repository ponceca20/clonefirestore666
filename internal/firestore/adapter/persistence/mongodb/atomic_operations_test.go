package mongodb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mockCollection implements CollectionUpdater for testing
type mockCollection struct{ mock.Mock }

func (m *mockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func TestAtomicOperations_Compile(t *testing.T) {
	// Placeholder: Add real atomic operations tests here
}

func TestAtomicOperations_AtomicIncrement(t *testing.T) {
	col := new(mockCollection)
	atomicOps := NewAtomicOperations(col)
	ctx := context.Background()

	col.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{MatchedCount: 1}, nil)

	err := atomicOps.AtomicIncrement(ctx, "proj", "db", "coll", "doc", "field", 5)
	assert.NoError(t, err)
	col.AssertCalled(t, "UpdateOne", mock.Anything, mock.Anything, mock.Anything)
}

func TestAtomicOperations_AtomicIncrement_DocumentNotFound(t *testing.T) {
	col := new(mockCollection)
	atomicOps := NewAtomicOperations(col)
	ctx := context.Background()
	col.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{MatchedCount: 0}, nil)

	err := atomicOps.AtomicIncrement(ctx, "proj", "db", "coll", "doc", "field", 5)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDocumentNotFound))
}

func TestAtomicOperations_AtomicIncrement_Error(t *testing.T) {
	col := new(mockCollection)
	atomicOps := NewAtomicOperations(col)
	ctx := context.Background()
	col.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).Return((*mongo.UpdateResult)(nil), errors.New("db error"))

	err := atomicOps.AtomicIncrement(ctx, "proj", "db", "coll", "doc", "field", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ... Add similar tests for other atomic methods as needed ...
