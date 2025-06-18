package service

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueryEngine is a mock implementation of QueryEngine for testing
type MockQueryEngine struct {
	mock.Mock
}

func (m *MockQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	args := m.Called(ctx, collectionPath, query)
	return args.Get(0).([]*model.Document), args.Error(1)
}

func (m *MockQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	args := m.Called(ctx, collectionPath, query, projection)
	return args.Get(0).([]*model.Document), args.Error(1)
}

func (m *MockQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	args := m.Called(ctx, collectionPath, query)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueryEngine) ValidateQuery(query model.Query) error {
	args := m.Called(query)
	return args.Error(0)
}

func (m *MockQueryEngine) GetQueryCapabilities() repository.QueryCapabilities {
	args := m.Called()
	return args.Get(0).(repository.QueryCapabilities)
}

func TestDocumentBasedSchemaInferenceService(t *testing.T) {
	tests := []struct {
		name          string
		fieldPath     string
		mockDocuments []*model.Document
		expectedType  model.FieldValueType
	}{{
		name:      "infer double type from price field",
		fieldPath: "price",
		mockDocuments: []*model.Document{
			{
				Fields: map[string]*model.FieldValue{
					"price": {
						ValueType: model.FieldTypeDouble,
						Value:     1800.0,
					},
				},
			},
		},
		expectedType: model.FieldTypeDouble,
	},
		{
			name:      "infer string type from name field",
			fieldPath: "name",
			mockDocuments: []*model.Document{
				{
					Fields: map[string]*model.FieldValue{
						"name": {
							ValueType: model.FieldTypeString,
							Value:     "Laptop Gamer Pro",
						},
					},
				},
			},
			expectedType: model.FieldTypeString,
		},
		{
			name:      "infer boolean type from available field",
			fieldPath: "available",
			mockDocuments: []*model.Document{
				{
					Fields: map[string]*model.FieldValue{
						"available": {
							ValueType: model.FieldTypeBool,
							Value:     true,
						},
					},
				},
			},
			expectedType: model.FieldTypeBool,
		},
		{
			name:          "default to string when no documents found",
			fieldPath:     "nonexistent",
			mockDocuments: []*model.Document{},
			expectedType:  model.FieldTypeString,
		}, {
			name:      "default to string when field not found",
			fieldPath: "nonexistent",
			mockDocuments: []*model.Document{
				{
					Fields: map[string]*model.FieldValue{
						"other": {
							ValueType: model.FieldTypeString,
							Value:     "value",
						},
					},
				},
			},
			expectedType: model.FieldTypeString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockEngine := new(MockQueryEngine)
			mockEngine.On("ExecuteQuery", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.mockDocuments, nil)

			// Create service
			service := NewDocumentBasedSchemaInferenceService(mockEngine)

			// Test
			resultType, err := service.InferFieldType(context.Background(), "test_collection", tt.fieldPath)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedType, resultType)

			// Verify mock was called
			mockEngine.AssertExpectations(t)
		})
	}
}

func TestExtractPrimitiveValue(t *testing.T) {
	service := &DocumentBasedSchemaInferenceService{}

	tests := []struct {
		name          string
		fieldData     interface{}
		expectedValue interface{}
	}{
		{
			name: "extract string value",
			fieldData: map[string]interface{}{
				"stringValue": "test string",
			},
			expectedValue: "test string",
		},
		{
			name: "extract double value",
			fieldData: map[string]interface{}{
				"doubleValue": 123.45,
			},
			expectedValue: 123.45,
		},
		{
			name: "extract boolean value",
			fieldData: map[string]interface{}{
				"booleanValue": true,
			},
			expectedValue: true,
		},
		{
			name: "extract integer value",
			fieldData: map[string]interface{}{
				"integerValue": 42,
			},
			expectedValue: 42,
		},
		{
			name:          "return nil for nil input",
			fieldData:     nil,
			expectedValue: nil,
		},
		{
			name:          "return value as-is if not in Firestore format",
			fieldData:     "direct value",
			expectedValue: "direct value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractPrimitiveValue(tt.fieldData)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}
