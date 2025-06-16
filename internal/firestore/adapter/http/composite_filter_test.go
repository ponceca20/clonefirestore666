package http

import (
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertFirestoreFilter_SimpleFieldFilter(t *testing.T) {
	// Test simple field filter
	firestoreFilter := FirestoreFilter{
		FieldFilter: &FirestoreFieldFilter{
			Field: FirestoreFieldReference{FieldPath: "name"},
			Op:    "EQUAL",
			Value: map[string]interface{}{"stringValue": "test"},
		},
	}

	filters, err := convertFirestoreFilter(firestoreFilter)
	require.NoError(t, err)
	require.Len(t, filters, 1)

	filter := filters[0]
	assert.Equal(t, "name", filter.Field)
	assert.Equal(t, model.OperatorEqual, filter.Operator)
	assert.Equal(t, "test", filter.Value)
	assert.Empty(t, filter.Composite)
	assert.Empty(t, filter.SubFilters)
}

func TestConvertFirestoreFilter_CompositeFilterAND(t *testing.T) {
	// Test composite AND filter
	firestoreFilter := FirestoreFilter{
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
			}},
	}
	filters, err := convertFirestoreFilter(firestoreFilter)
	require.NoError(t, err)
	require.Len(t, filters, 2, "AND filters should be flattened into separate filters, got: %v", filters)

	// Check first filter
	assert.Equal(t, "price", filters[0].Field)
	assert.Equal(t, model.OperatorGreaterThanOrEqual, filters[0].Operator)
	assert.Equal(t, 50.0, filters[0].Value)
	assert.Empty(t, filters[0].Composite, "Should not be a composite filter")
	// Check second filter
	assert.Equal(t, "price", filters[1].Field)
	assert.Equal(t, model.OperatorLessThanOrEqual, filters[1].Operator)
	assert.Equal(t, 500.0, filters[1].Value)
	assert.Empty(t, filters[1].Composite, "Should not be a composite filter")
}

func TestConvertFirestoreFilter_CompositeFilterOR(t *testing.T) {
	// Test composite OR filter
	firestoreFilter := FirestoreFilter{
		CompositeFilter: &FirestoreCompositeFilter{
			Op: "OR",
			Filters: []FirestoreFilter{
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "category"},
						Op:    "EQUAL",
						Value: map[string]interface{}{"stringValue": "Electronics"},
					},
				},
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "category"},
						Op:    "EQUAL",
						Value: map[string]interface{}{"stringValue": "Peripherals"},
					},
				},
			},
		},
	}

	filters, err := convertFirestoreFilter(firestoreFilter)
	require.NoError(t, err)
	require.Len(t, filters, 1)

	filter := filters[0]
	assert.Equal(t, "or", filter.Composite)
	assert.Len(t, filter.SubFilters, 2)

	// Check first subfilter
	assert.Equal(t, "category", filter.SubFilters[0].Field)
	assert.Equal(t, model.OperatorEqual, filter.SubFilters[0].Operator)
	assert.Equal(t, "Electronics", filter.SubFilters[0].Value)

	// Check second subfilter
	assert.Equal(t, "category", filter.SubFilters[1].Field)
	assert.Equal(t, model.OperatorEqual, filter.SubFilters[1].Operator)
	assert.Equal(t, "Peripherals", filter.SubFilters[1].Value)
}

func TestConvertFirestoreFilter_NestedCompositeFilter(t *testing.T) {
	// Test nested composite filter (AND containing OR)
	firestoreFilter := FirestoreFilter{
		CompositeFilter: &FirestoreCompositeFilter{
			Op: "AND",
			Filters: []FirestoreFilter{
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "available"},
						Op:    "EQUAL",
						Value: map[string]interface{}{"booleanValue": true},
					},
				},
				{
					CompositeFilter: &FirestoreCompositeFilter{
						Op: "OR",
						Filters: []FirestoreFilter{
							{
								FieldFilter: &FirestoreFieldFilter{
									Field: FirestoreFieldReference{FieldPath: "brand"},
									Op:    "EQUAL",
									Value: map[string]interface{}{"stringValue": "TechMaster"},
								},
							},
							{
								FieldFilter: &FirestoreFieldFilter{
									Field: FirestoreFieldReference{FieldPath: "brand"},
									Op:    "EQUAL",
									Value: map[string]interface{}{"stringValue": "MobileGenius"},
								},
							},
						},
					},
				},
			},
		},
	}
	filters, err := convertFirestoreFilter(firestoreFilter)
	require.NoError(t, err)
	require.Len(t, filters, 2, "AND should be flattened into separate filters")

	// Check first filter (simple field filter)
	assert.Equal(t, "available", filters[0].Field)
	assert.Equal(t, model.OperatorEqual, filters[0].Operator)
	assert.Equal(t, true, filters[0].Value)
	assert.Empty(t, filters[0].Composite, "Should not be a composite filter")

	// Check second filter (OR composite filter)
	assert.Equal(t, "or", filters[1].Composite)
	assert.Len(t, filters[1].SubFilters, 2)

	// Check OR sub-filters
	assert.Equal(t, "brand", filters[1].SubFilters[0].Field)
	assert.Equal(t, model.OperatorEqual, filters[1].SubFilters[0].Operator)
	assert.Equal(t, "TechMaster", filters[1].SubFilters[0].Value)

	assert.Equal(t, "brand", filters[1].SubFilters[1].Field)
	assert.Equal(t, model.OperatorEqual, filters[1].SubFilters[1].Operator)
	assert.Equal(t, "MobileGenius", filters[1].SubFilters[1].Value)
}

func TestConvertFirestoreFilter_ErrorCases(t *testing.T) {
	// Test empty filter
	firestoreFilter := FirestoreFilter{}
	_, err := convertFirestoreFilter(firestoreFilter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter must have either fieldFilter or compositeFilter")

	// Test composite filter with invalid operator
	firestoreFilter = FirestoreFilter{
		CompositeFilter: &FirestoreCompositeFilter{
			Op: "INVALID",
			Filters: []FirestoreFilter{
				{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "name"},
						Op:    "EQUAL",
						Value: map[string]interface{}{"stringValue": "test"},
					},
				},
			},
		},
	}
	_, err = convertFirestoreFilter(firestoreFilter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported composite filter operator")

	// Test composite filter with no filters
	firestoreFilter = FirestoreFilter{
		CompositeFilter: &FirestoreCompositeFilter{
			Op:      "AND",
			Filters: []FirestoreFilter{},
		},
	}
	_, err = convertFirestoreFilter(firestoreFilter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "composite filter must have at least one sub-filter")
}

func TestMapFirestoreOperator(t *testing.T) {
	testCases := []struct {
		input    string
		expected model.Operator
	}{
		{"EQUAL", model.OperatorEqual},
		{"NOT_EQUAL", model.OperatorNotEqual},
		{"LESS_THAN", model.OperatorLessThan},
		{"LESS_THAN_OR_EQUAL", model.OperatorLessThanOrEqual},
		{"GREATER_THAN", model.OperatorGreaterThan},
		{"GREATER_THAN_OR_EQUAL", model.OperatorGreaterThanOrEqual},
		{"ARRAY_CONTAINS", model.OperatorArrayContains},
		{"ARRAY_CONTAINS_ANY", model.OperatorArrayContainsAny},
		{"IN", model.OperatorIn},
		{"NOT_IN", model.OperatorNotIn},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapFirestoreOperator(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}

	// Test unknown operator returns the input string cast to Operator
	result := mapFirestoreOperator("UNKNOWN_OP")
	assert.Equal(t, model.Operator("UNKNOWN_OP"), result)
}

func TestConvertFirestoreValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "String value",
			input:    map[string]interface{}{"stringValue": "test"},
			expected: "test",
		},
		{
			name:     "Double value",
			input:    map[string]interface{}{"doubleValue": 123.45},
			expected: 123.45,
		},
		{
			name:     "Boolean value",
			input:    map[string]interface{}{"booleanValue": true},
			expected: true,
		},
		{
			name:     "Integer value as string",
			input:    map[string]interface{}{"integerValue": "42"},
			expected: int64(42),
		},
		{
			name:     "Integer value as number",
			input:    map[string]interface{}{"integerValue": 42},
			expected: 42,
		},
		{
			name:     "Null value",
			input:    map[string]interface{}{"nullValue": "NULL_VALUE"},
			expected: nil,
		},
		{
			name:     "Array value",
			input:    map[string]interface{}{"arrayValue": map[string]interface{}{"values": []interface{}{map[string]interface{}{"stringValue": "item1"}, map[string]interface{}{"stringValue": "item2"}}}},
			expected: []interface{}{"item1", "item2"},
		},
		{
			name:     "Direct value (no wrapper)",
			input:    "direct_string",
			expected: "direct_string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertFirestoreValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
