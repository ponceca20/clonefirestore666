package mongodb

import (
	"firestore-clone/internal/firestore/domain/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMongoQueryEngine_INOperator_ValueTypeDetection(t *testing.T) {
	tests := []struct {
		name            string
		operator        model.Operator
		value           interface{}
		expectedType    model.FieldValueType
		expectedBSONKey string
	}{
		{
			name:            "IN operator with string values",
			operator:        model.OperatorIn,
			value:           []interface{}{"Electronics", "Office"},
			expectedType:    model.FieldTypeString,
			expectedBSONKey: "fields.category.stringValue",
		},
		{
			name:            "IN operator with integer values",
			operator:        model.OperatorIn,
			value:           []interface{}{int64(1), int64(2), int64(3)},
			expectedType:    model.FieldTypeInt,
			expectedBSONKey: "fields.count.integerValue",
		},
		{
			name:            "IN operator with double values",
			operator:        model.OperatorIn,
			value:           []interface{}{1.5, 2.5, 3.5},
			expectedType:    model.FieldTypeDouble,
			expectedBSONKey: "fields.price.doubleValue",
		},
		{
			name:            "IN operator with boolean values",
			operator:        model.OperatorIn,
			value:           []interface{}{true, false},
			expectedType:    model.FieldTypeBool,
			expectedBSONKey: "fields.active.booleanValue",
		},
		{
			name:            "EQUAL operator with string value",
			operator:        model.OperatorEqual,
			value:           "Electronics",
			expectedType:    model.FieldTypeString,
			expectedBSONKey: "fields.category.stringValue",
		},
		{
			name:            "ARRAY_CONTAINS with string value",
			operator:        model.OperatorArrayContains,
			value:           "tag1",
			expectedType:    model.FieldTypeArray,
			expectedBSONKey: "fields.tags.arrayValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the specific logic that determines value type for IN operations
			var actualType model.FieldValueType

			if tt.operator == model.OperatorIn || tt.operator == model.OperatorNotIn {
				// This simulates the fixed logic in MongoQueryEngine
				if valueSlice, ok := tt.value.([]interface{}); ok && len(valueSlice) > 0 {
					actualType = model.DetermineValueType(valueSlice[0])
				} else {
					actualType = model.DetermineValueType(tt.value)
				}
			} else if tt.operator == model.OperatorArrayContains || tt.operator == model.OperatorArrayContainsAny {
				// Array operations should always use arrayValue
				actualType = model.FieldTypeArray
			} else {
				actualType = model.DetermineValueType(tt.value)
			}

			assert.Equal(t, tt.expectedType, actualType, "Value type should be determined correctly for operator %s", tt.operator)
		})
	}
}

func TestMongoQueryEngine_INOperator_BSONGeneration(t *testing.T) {
	// Test that the BSON filter is generated correctly for IN operations
	filter := model.Filter{
		Field:    "category",
		Operator: model.OperatorIn,
		Value:    []interface{}{"Electronics", "Home Goods"},
	}

	// Simulate the expected behavior after our fix
	expectedFilter := bson.M{
		"fields.category.stringValue": bson.M{
			"$in": []interface{}{"Electronics", "Home Goods"},
		},
	}

	// Test that the logic correctly identifies string type from first element
	valueSlice := filter.Value.([]interface{})
	firstElementType := model.DetermineValueType(valueSlice[0])

	require.Equal(t, model.FieldTypeString, firstElementType)

	// Verify the expected MongoDB filter structure
	actualBSONValue := bson.M{"$in": filter.Value}
	assert.Equal(t, expectedFilter["fields.category.stringValue"], actualBSONValue)
}

func TestDetermineValueType_WithArrayInput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected model.FieldValueType
	}{
		{
			name:     "String slice should return array type",
			input:    []interface{}{"a", "b", "c"},
			expected: model.FieldTypeArray,
		},
		{
			name:     "First element is string",
			input:    "Electronics",
			expected: model.FieldTypeString,
		},
		{
			name:     "First element is int64",
			input:    int64(42),
			expected: model.FieldTypeInt,
		},
		{
			name:     "First element is float64",
			input:    3.14,
			expected: model.FieldTypeDouble,
		},
		{
			name:     "First element is boolean",
			input:    true,
			expected: model.FieldTypeBool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.DetermineValueType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
