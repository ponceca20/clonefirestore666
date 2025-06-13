package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockIndexCollection implementa la interfaz de colección de índices para pruebas
type MockIndexCollection struct {
	mock.Mock
}

func (m *MockIndexCollection) ListIndexes(ctx context.Context, opts interface{}) (mongo.Cursor, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(mongo.Cursor), args.Error(1)
}

func (m *MockIndexCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	args := m.Called(ctx, doc)
	return args.Get(0), args.Error(1)
}

func (m *MockIndexCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResult, error) {
	args := m.Called(ctx, filter)
	result := DeleteResult{DeletedCount: 1}
	return result, args.Error(1)
}

func (m *MockIndexCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	return UpdateResult{}, args.Error(1)
}

func (m *MockIndexCollection) Find(ctx context.Context, filter interface{}) (Cursor, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(Cursor), args.Error(1)
}

func (m *MockIndexCollection) FindOne(ctx context.Context, filter interface{}) SingleResult {
	args := m.Called(ctx, filter)
	return args.Get(0).(SingleResult)
}

func (m *MockIndexCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// MockDocumentCollection implementa la interfaz de colección de documentos para pruebas
type MockDocumentCollection struct {
	mock.Mock
}

type MockIndexManager struct {
	mock.Mock
}

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

func (m *MockDocumentCollection) Indexes() IndexManager {
	args := m.Called()
	if res := args.Get(0); res != nil {
		return res.(IndexManager)
	}
	return &MockIndexManager{}
}

func (m *MockDocumentCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// TestLogger implementa la interfaz Logger para pruebas
type TestLogger struct{}

func (l *TestLogger) Debug(args ...interface{})                              {}
func (l *TestLogger) Info(args ...interface{})                               {}
func (l *TestLogger) Warn(args ...interface{})                               {}
func (l *TestLogger) Error(args ...interface{})                              {}
func (l *TestLogger) Fatal(args ...interface{})                              {}
func (l *TestLogger) Debugf(format string, args ...interface{})              {}
func (l *TestLogger) Infof(format string, args ...interface{})               {}
func (l *TestLogger) Warnf(format string, args ...interface{})               {}
func (l *TestLogger) Errorf(format string, args ...interface{})              {}
func (l *TestLogger) Fatalf(format string, args ...interface{})              {}
func (l *TestLogger) WithFields(fields map[string]interface{}) logger.Logger { return l }
func (l *TestLogger) WithContext(ctx context.Context) logger.Logger          { return l }
func (l *TestLogger) WithComponent(component string) logger.Logger           { return l }

// TestIndexCursor implementa mongo.Cursor para pruebas de índices
type TestIndexCursor struct {
	results []bson.M
	pos     int
}

func NewTestIndexCursor(results []bson.M) *TestIndexCursor {
	return &TestIndexCursor{
		results: results,
		pos:     -1,
	}
}

func (c *TestIndexCursor) Next(ctx context.Context) bool {
	c.pos++
	return c.pos < len(c.results)
}

func (c *TestIndexCursor) Decode(val interface{}) error {
	if c.pos >= 0 && c.pos < len(c.results) {
		if v, ok := val.(*bson.M); ok {
			*v = c.results[c.pos]
		}
	}
	return nil
}

func (c *TestIndexCursor) Close(ctx context.Context) error {
	return nil
}

func (c *TestIndexCursor) Err() error {
	return nil
}

func (c *TestIndexCursor) ID() int64 {
	return 0
}

// createTestIndexOperations crea una instancia de IndexOperations para pruebas
func createTestIndexOperations() (*IndexOperations, *MockIndexCollection, *MockDocumentCollection) {
	mockIndexCol := new(MockIndexCollection)
	mockDocCol := new(MockDocumentCollection)
	testLogger := &TestLogger{}
	ops := NewIndexOperations(mockIndexCol, mockDocCol, testLogger)
	return ops, mockIndexCol, mockDocCol
}

func TestIndexOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateIndex", func(t *testing.T) {
		t.Run("with valid single field index", func(t *testing.T) {
			// Arrange
			ops, mockIndexCol, mockDocCol := createTestIndexOperations()
			mockIndexCol.On("CountDocuments", mock.Anything, mock.Anything).Return(int64(0), nil)
			mockIndexCol.On("InsertOne", mock.Anything, mock.Anything).Return("test_index", nil)
			mockIndexCol.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).Return(UpdateResult{}, nil)
			// Configurar el mock del IndexManager
			mockIndexManager := new(MockIndexManager)
			mockIndexManager.On("CreateOne", mock.Anything, mock.Anything).Return(nil, nil)
			mockDocCol.On("Indexes").Return(mockIndexManager)

			index := &model.CollectionIndex{
				Name: "test_index",
				Fields: []model.IndexField{
					{
						Path:  "name",
						Order: model.IndexFieldOrderAscending,
					},
				},
			}

			// Act
			err := ops.CreateIndex(ctx, "p1", "d1", "c1", index)

			// Assert
			assert.NoError(t, err)
			mockIndexCol.AssertExpectations(t)
		})

		t.Run("with compound index", func(t *testing.T) {
			// Arrange
			ops, mockIndexCol, mockDocCol := createTestIndexOperations()
			mockIndexCol.On("CountDocuments", mock.Anything, mock.Anything).Return(int64(0), nil)
			mockIndexCol.On("InsertOne", mock.Anything, mock.Anything).Return("test_compound_index", nil)
			mockIndexCol.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).Return(UpdateResult{}, nil)
			// Configurar el mock del IndexManager
			mockIndexManager := new(MockIndexManager)
			mockIndexManager.On("CreateOne", mock.Anything, mock.Anything).Return(nil, nil)
			mockDocCol.On("Indexes").Return(mockIndexManager)

			index := &model.CollectionIndex{
				Name: "test_compound_index",
				Fields: []model.IndexField{
					{
						Path:  "timestamp",
						Order: model.IndexFieldOrderDescending,
					},
					{
						Path:  "name",
						Order: model.IndexFieldOrderAscending,
					},
				},
			}

			// Act
			err := ops.CreateIndex(ctx, "p1", "d1", "c1", index)

			// Assert
			assert.NoError(t, err)
			mockIndexCol.AssertExpectations(t)
		})

		t.Run("validates input parameters", func(t *testing.T) {
			ops, _, _ := createTestIndexOperations()

			testCases := []struct {
				name    string
				project string
				db      string
				coll    string
				index   *model.CollectionIndex
				errMsg  string
			}{
				{"empty project ID", "", "d1", "c1", &model.CollectionIndex{}, "project ID"},
				{"empty database ID", "p1", "", "c1", &model.CollectionIndex{}, "database ID"},
				{"empty collection ID", "p1", "d1", "", &model.CollectionIndex{}, "collection ID"},
				{"nil index", "p1", "d1", "c1", nil, "index"},
				{"empty fields", "p1", "d1", "c1", &model.CollectionIndex{}, "fields"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := ops.CreateIndex(ctx, tc.project, tc.db, tc.coll, tc.index)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errMsg)
				})
			}
		})
	})

	t.Run("ListIndexes", func(t *testing.T) {
		t.Run("returns all indexes", func(t *testing.T) {
			// Arrange
			ops, mockIndexCol, mockDocCol := createTestIndexOperations()

			indexResults := []bson.M{
				{"name": "idx1", "key": bson.M{"field1": 1}},
				{"name": "idx2", "key": bson.M{"field2": -1}},
			}
			// Only expect Find, not ListIndexes
			mockIndexCol.On("Find", mock.Anything, mock.Anything).Return(NewTestIndexCursor(indexResults), nil)
			// Mock Indexes() para evitar error de mock
			mockIndexManager := new(MockIndexManager)
			mockDocCol.On("Indexes").Return(mockIndexManager)

			// Act
			results, err := ops.ListIndexes(ctx, "p1", "d1", "c1")

			// Assert
			assert.NoError(t, err)
			assert.Len(t, results, 2)
			mockIndexCol.AssertExpectations(t)
		})

		t.Run("validates input parameters", func(t *testing.T) {
			ops, _, _ := createTestIndexOperations()

			testCases := []struct {
				name    string
				project string
				db      string
				coll    string
				errMsg  string
			}{
				{"empty project ID", "", "d1", "c1", "project ID"},
				{"empty database ID", "p1", "", "c1", "database ID"},
				{"empty collection ID", "p1", "d1", "", "collection ID"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					_, err := ops.ListIndexes(ctx, tc.project, tc.db, tc.coll)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errMsg)
				})
			}
		})
	})

	t.Run("DeleteIndex", func(t *testing.T) {
		t.Run("deletes existing index", func(t *testing.T) {
			// Arrange
			ops, mockIndexCol, mockDocCol := createTestIndexOperations()
			mockIndexCol.On("FindOne", mock.Anything, mock.Anything).Return(MockSingleResult{})
			mockIndexCol.On("DeleteOne", mock.Anything, mock.Anything).Return(DeleteResult{DeletedCount: 1}, nil)
			// Mock Indexes() para evitar error de mock
			mockIndexManager := new(MockIndexManager)
			mockIndexManager.On("DropOne", mock.Anything, "test_index").Return(nil, nil)
			mockDocCol.On("Indexes").Return(mockIndexManager)

			// Act
			err := ops.DeleteIndex(ctx, "p1", "d1", "c1", "test_index")

			// Assert
			assert.NoError(t, err)
			mockIndexCol.AssertExpectations(t)
		})
	})
}
