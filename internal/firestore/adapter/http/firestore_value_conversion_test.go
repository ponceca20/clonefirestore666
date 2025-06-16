// Package http provides HTTP adapter tests for Firestore value conversion
// following hexagonal architecture principles.
//
// This file contains comprehensive tests for:
// - Firestore typed value conversion (booleanValue, stringValue, etc.)
// - Query structure conversion from Firestore REST API format
// - Composite filter handling (AND/OR operations)
// - Edge cases and error scenarios
// - Performance benchmarks
//
// These tests ensure the HTTP adapter layer correctly translates
// Firestore REST API requests to internal domain models.
package http

import (
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFirestoreValueConversion_TypedValues tests the conversion of Firestore typed values
// following hexagonal architecture principles for Firestore value handling
func TestFirestoreValueConversion_TypedValues(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "BooleanValue_True",
			input:    map[string]interface{}{"booleanValue": true},
			expected: true,
		},
		{
			name:     "BooleanValue_False",
			input:    map[string]interface{}{"booleanValue": false},
			expected: false,
		},
		{
			name:     "StringValue_Basic",
			input:    map[string]interface{}{"stringValue": "test"},
			expected: "test",
		},
		{
			name:     "IntegerValue_AsString",
			input:    map[string]interface{}{"integerValue": "123"},
			expected: int64(123),
		},
		{
			name:     "IntegerValue_AsNumber",
			input:    map[string]interface{}{"integerValue": 456},
			expected: 456,
		},
		{
			name:     "DoubleValue_Basic",
			input:    map[string]interface{}{"doubleValue": 3.14},
			expected: 3.14,
		},
		{
			name:     "NullValue_Standard",
			input:    map[string]interface{}{"nullValue": "NULL_VALUE"},
			expected: nil,
		},
		{
			name: "ArrayValue_Mixed",
			input: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{"stringValue": "item1"},
						map[string]interface{}{"booleanValue": true},
						map[string]interface{}{"integerValue": "42"},
					},
				},
			},
			expected: []interface{}{"item1", true, int64(42)},
		},
		{
			name:     "PlainValue_String",
			input:    "plainString",
			expected: "plainString",
		},
		{
			name:     "PlainValue_Number",
			input:    42,
			expected: 42,
		},
		{
			name:     "NilValue",
			input:    nil,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertFirestoreValue(tc.input)

			// Special handling for array comparison using assert for better error messages
			if expectedArray, ok := tc.expected.([]interface{}); ok {
				resultArray, ok := result.([]interface{})
				require.True(t, ok, "Expected result to be array, got %T", result)
				assert.Equal(t, expectedArray, resultArray, "Array values should match")
				return
			}

			assert.Equal(t, tc.expected, result, "Converted value should match expected")
		})
	}
}

// TestFirestoreQueryConversion_WithTypedValues tests the conversion of Firestore queries with typed values
// ensuring proper handling following Firestore API specification
func TestFirestoreQueryConversion_WithTypedValues(t *testing.T) { // Test case that mimics the actual Firestore JSON structure with typed values
	// This follows the Firestore REST API specification for structured queries
	firestoreQuery := FirestoreStructuredQuery{
		From: []FirestoreCollectionSelector{
			{CollectionID: "products"},
		},
		Where: &FirestoreFilter{
			FieldFilter: &FirestoreFieldFilter{
				Field: FirestoreFieldReference{FieldPath: "available"},
				Op:    "EQUAL",
				Value: map[string]interface{}{"booleanValue": true}, // Typed Firestore value
			},
		},
		Limit: 10,
	}

	query, err := convertFirestoreJSONToModelQuery(firestoreQuery)
	require.NoError(t, err, "Should convert Firestore query without error")
	require.NotNil(t, query, "Converted query should not be nil")

	// Validate collection mapping
	assert.Equal(t, "products", query.CollectionID, "Collection ID should be mapped correctly")

	// Validate filters conversion
	require.Len(t, query.Filters, 1, "Should have exactly one filter")

	filter := query.Filters[0]
	assert.Equal(t, "available", filter.Field, "Field name should be mapped correctly")
	assert.Equal(t, model.OperatorEqual, filter.Operator, "Operator should be mapped correctly")
	assert.Equal(t, true, filter.Value, "Typed value should be extracted correctly")

	// Validate query options
	assert.Equal(t, 10, query.Limit, "Limit should be mapped correctly")
}

// TestFirestoreValueConversion_EdgeCases tests edge cases and error scenarios
// for Firestore value conversion following defensive programming principles
func TestFirestoreValueConversion_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "EmptyMap",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "MapValue_Nested",
			input: map[string]interface{}{
				"mapValue": map[string]interface{}{
					"fields": map[string]interface{}{
						"name": map[string]interface{}{"stringValue": "test"},
						"age":  map[string]interface{}{"integerValue": "25"},
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"age":  int64(25),
			},
		},
		{
			name: "ArrayValue_Empty",
			input: map[string]interface{}{
				"arrayValue": map[string]interface{}{
					"values": []interface{}{},
				},
			},
			expected: []interface{}{},
		}, {
			name:     "TimestampValue",
			input:    map[string]interface{}{"timestampValue": "2023-01-01T00:00:00Z"},
			expected: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "ReferenceValue",
			input:    map[string]interface{}{"referenceValue": "projects/test/databases/(default)/documents/users/123"},
			expected: "projects/test/databases/(default)/documents/users/123",
		},
		{
			name: "GeoPointValue",
			input: map[string]interface{}{
				"geoPointValue": map[string]interface{}{
					"latitude":  40.7128,
					"longitude": -74.0060,
				},
			},
			expected: map[string]interface{}{
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
		},
		{
			name:     "BytesValue",
			input:    map[string]interface{}{"bytesValue": "SGVsbG8gV29ybGQ="},
			expected: "SGVsbG8gV29ybGQ=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertFirestoreValue(tc.input)
			assert.Equal(t, tc.expected, result, "Edge case should be handled correctly")
		})
	}
}

// TestFirestoreFilterConversion_CompositeFilters tests the conversion of composite filters
// ensuring proper handling of AND/OR operations following Firestore query specification
func TestFirestoreFilterConversion_CompositeFilters(t *testing.T) {
	testCases := []struct {
		name           string
		input          FirestoreFilter
		expectedCount  int
		expectedFields []string
		expectError    bool
	}{
		{
			name: "SimpleFieldFilter",
			input: FirestoreFilter{
				FieldFilter: &FirestoreFieldFilter{
					Field: FirestoreFieldReference{FieldPath: "name"},
					Op:    "EQUAL",
					Value: map[string]interface{}{"stringValue": "test"},
				},
			},
			expectedCount:  1,
			expectedFields: []string{"name"},
			expectError:    false,
		},
		{
			name: "CompositeFilter_AND",
			input: FirestoreFilter{
				CompositeFilter: &FirestoreCompositeFilter{
					Op: "AND",
					Filters: []FirestoreFilter{
						{
							FieldFilter: &FirestoreFieldFilter{
								Field: FirestoreFieldReference{FieldPath: "price"},
								Op:    "GREATER_THAN_OR_EQUAL",
								Value: map[string]interface{}{"doubleValue": 50.0},
							},
						},
						{
							FieldFilter: &FirestoreFieldFilter{
								Field: FirestoreFieldReference{FieldPath: "price"},
								Op:    "LESS_THAN_OR_EQUAL",
								Value: map[string]interface{}{"doubleValue": 500.0},
							},
						},
					},
				},
			},
			expectedCount:  2,
			expectedFields: []string{"price", "price"},
			expectError:    false,
		},
		{
			name: "CompositeFilter_OR",
			input: FirestoreFilter{
				CompositeFilter: &FirestoreCompositeFilter{
					Op: "OR",
					Filters: []FirestoreFilter{
						{
							FieldFilter: &FirestoreFieldFilter{
								Field: FirestoreFieldReference{FieldPath: "category"},
								Op:    "EQUAL",
								Value: map[string]interface{}{"stringValue": "electronics"},
							},
						},
						{
							FieldFilter: &FirestoreFieldFilter{
								Field: FirestoreFieldReference{FieldPath: "category"},
								Op:    "EQUAL",
								Value: map[string]interface{}{"stringValue": "books"},
							},
						},
					},
				},
			},
			expectedCount:  1, // OR creates one composite filter with sub-filters
			expectedFields: []string{},
			expectError:    false,
		},
		{
			name: "UnsupportedOperator",
			input: FirestoreFilter{
				FieldFilter: &FirestoreFieldFilter{
					Field: FirestoreFieldReference{FieldPath: "name"},
					Op:    "INVALID_OP",
					Value: map[string]interface{}{"stringValue": "test"},
				},
			},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "EmptyFilter",
			input:         FirestoreFilter{},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := convertFirestoreFilter(tc.input)

			if tc.expectError {
				assert.Error(t, err, "Should return error for invalid input")
				return
			}

			require.NoError(t, err, "Should convert filter without error")
			assert.Len(t, result, tc.expectedCount, "Should return expected number of filters")

			// For non-OR filters, validate field names
			if tc.input.CompositeFilter == nil || tc.input.CompositeFilter.Op != "OR" {
				for i, expectedField := range tc.expectedFields {
					if i < len(result) {
						assert.Equal(t, expectedField, result[i].Field, "Field should match expected")
					}
				}
			}

			// For OR filters, validate composite structure
			if tc.input.CompositeFilter != nil && tc.input.CompositeFilter.Op == "OR" {
				require.Len(t, result, 1, "OR should create one composite filter")
				assert.Equal(t, "or", result[0].Composite, "Should be marked as OR composite")
				assert.NotEmpty(t, result[0].SubFilters, "Should have sub-filters")
			}
		})
	}
}

// BenchmarkFirestoreValueConversion benchmarks the performance of Firestore value conversion
// to ensure it meets performance requirements for high-throughput scenarios
func BenchmarkFirestoreValueConversion(b *testing.B) {
	testValues := []interface{}{
		map[string]interface{}{"stringValue": "test"},
		map[string]interface{}{"booleanValue": true},
		map[string]interface{}{"doubleValue": 3.14},
		map[string]interface{}{
			"arrayValue": map[string]interface{}{
				"values": []interface{}{
					map[string]interface{}{"stringValue": "item1"},
					map[string]interface{}{"booleanValue": true},
					map[string]interface{}{"integerValue": "42"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, value := range testValues {
			convertFirestoreValue(value)
		}
	}
}

// BenchmarkFirestoreFilterConversion benchmarks the performance of filter conversion
// critical for query processing performance
func BenchmarkFirestoreFilterConversion(b *testing.B) {
	filter := FirestoreFilter{
		CompositeFilter: &FirestoreCompositeFilter{
			Op: "AND",
			Filters: []FirestoreFilter{
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "price"},
						Op:    "GREATER_THAN_OR_EQUAL",
						Value: map[string]interface{}{"doubleValue": 50.0},
					},
				},
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "price"},
						Op:    "LESS_THAN_OR_EQUAL",
						Value: map[string]interface{}{"doubleValue": 500.0},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = convertFirestoreFilter(filter)
	}
}
