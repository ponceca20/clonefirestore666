package mongodb

import (
	"context"
	"errors"
	"firestore-clone/internal/firestore/domain/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockIndexCollection implementa IndexCollection para pruebas
// MockDocumentCollection implementa DocumentCollection para pruebas
// MockIndexManager implementa IndexManager para pruebas

type MockIndexCollection struct{ mock.Mock }

func (m *MockIndexCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockIndexCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	args := m.Called(ctx, doc)
	return args.Get(0), args.Error(1)
}
func (m *MockIndexCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResult, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(DeleteResult), args.Error(1)
}
func (m *MockIndexCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	return args.Get(0).(UpdateResult), args.Error(1)
}
func (m *MockIndexCollection) Find(ctx context.Context, filter interface{}) (Cursor, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(Cursor), args.Error(1)
}
func (m *MockIndexCollection) FindOne(ctx context.Context, filter interface{}) SingleResult {
	args := m.Called(ctx, filter)
	return args.Get(0).(SingleResult)
}

type MockDocumentCollection struct{ mock.Mock }

func (m *MockDocumentCollection) Indexes() IndexManager {
	args := m.Called()
	return args.Get(0).(IndexManager)
}
func (m *MockDocumentCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

type MockIndexManager struct{ mock.Mock }

func (m *MockIndexManager) CreateOne(ctx context.Context, model interface{}) (interface{}, error) {
	args := m.Called(ctx, model)
	return args.Get(0), args.Error(1)
}
func (m *MockIndexManager) DropOne(ctx context.Context, name string) (interface{}, error) {
	args := m.Called(ctx, name)
	return args.Get(0), args.Error(1)
}
func (m *MockIndexManager) ListSpecifications(ctx context.Context) ([]IndexSpec, error) {
	args := m.Called(ctx)
	return args.Get(0).([]IndexSpec), args.Error(1)
}

// MockSingleResult implements SingleResult for testing
type MockSingleResult struct{ mock.Mock }

func (m *MockSingleResult) Decode(val interface{}) error {
	args := m.Called(val)
	return args.Error(0)
}

// MockCursor implements Cursor for testing
type MockCursor struct{ mock.Mock }

func (m *MockCursor) Next(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockCursor) Decode(val interface{}) error {
	args := m.Called(val)
	return args.Error(0)
}

func (m *MockCursor) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCursor) Err() error {
	args := m.Called()
	return args.Error(0)
}

// --- Pruebas unitarias ---
func TestIndexOperations_CreateIndex_Success(t *testing.T) {
	ctx := context.Background()
	mockIdxCol := new(MockIndexCollection)
	mockDocCol := new(MockDocumentCollection)
	mockIdxMgr := new(MockIndexManager)
	mockLogger := new(MockLogger)
	indexOps := NewIndexOperations(mockIdxCol, mockDocCol, mockLogger)

	// Setup mocks
	mockIdxCol.On("CountDocuments", ctx, mock.Anything).Return(int64(0), nil)
	mockIdxCol.On("InsertOne", ctx, mock.Anything).Return(nil, nil)
	mockDocCol.On("Indexes").Return(mockIdxMgr)
	mockIdxMgr.On("CreateOne", ctx, mock.Anything).Return(nil, nil)
	mockIdxCol.On("UpdateOne", ctx, mock.Anything, mock.Anything).Return(UpdateResult{}, nil)

	index := &model.CollectionIndex{Name: "idx1"}
	err := indexOps.CreateIndex(ctx, "p", "d", "c", index)
	assert.NoError(t, err)
	mockIdxCol.AssertExpectations(t)
	mockDocCol.AssertExpectations(t)
	mockIdxMgr.AssertExpectations(t)
}

func TestIndexOperations_CreateIndex_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	mockIdxCol := new(MockIndexCollection)
	mockDocCol := new(MockDocumentCollection)
	mockLogger := new(MockLogger)
	indexOps := NewIndexOperations(mockIdxCol, mockDocCol, mockLogger)

	mockIdxCol.On("CountDocuments", ctx, mock.Anything).Return(int64(1), nil)
	index := &model.CollectionIndex{Name: "idx1"}
	err := indexOps.CreateIndex(ctx, "p", "d", "c", index)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrIndexAlreadyExists))
}

func TestIndexOperations_DeleteIndex_NotFound(t *testing.T) {
	ctx := context.Background()
	mockIdxCol := new(MockIndexCollection)
	mockDocCol := new(MockDocumentCollection)
	mockLogger := new(MockLogger)
	mockSingleResult := new(MockSingleResult)
	indexOps := NewIndexOperations(mockIdxCol, mockDocCol, mockLogger)

	// Mock FindOne to return ErrNoDocuments (which should return ErrIndexNotFound)
	mockIdxCol.On("FindOne", ctx, mock.Anything).Return(mockSingleResult)
	mockSingleResult.On("Decode", mock.Anything).Return(mongo.ErrNoDocuments)

	err := indexOps.DeleteIndex(ctx, "p", "d", "c", "id1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrIndexNotFound))
}

func TestIndexOperations_ListIndexes_Error(t *testing.T) {
	ctx := context.Background()
	mockIdxCol := new(MockIndexCollection)
	mockDocCol := new(MockDocumentCollection)
	mockLogger := new(MockLogger)
	indexOps := NewIndexOperations(mockIdxCol, mockDocCol, mockLogger)

	errList := errors.New("db error")
	mockCursor := new(MockCursor)
	mockIdxCol.On("Find", ctx, mock.Anything).Return(mockCursor, errList)
	_, err := indexOps.ListIndexes(ctx, "p", "d", "c")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}
