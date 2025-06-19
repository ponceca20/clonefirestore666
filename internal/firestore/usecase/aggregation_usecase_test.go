package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAggregationRequest(t *testing.T) {
	// Create a mock usecase for testing
	uc := &FirestoreUsecase{}

	tests := []struct {
		name        string
		req         AggregationQueryRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid basic count request",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "total_count",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid count and sum request",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "total_count",
							Count: &CountAggregation{},
						},
						{
							Alias: "total_stock",
							Sum: &FieldAggregation{
								Field: FieldReference{FieldPath: "stock"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid request with groupBy",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					GroupBy: []GroupByField{
						{FieldPath: "category"},
					},
					Aggregations: []AggregationFunction{
						{
							Alias: "count_per_category",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid request with extended operators",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "min_price",
							Min: &FieldAggregation{
								Field: FieldReference{FieldPath: "price"},
							},
						},
						{
							Alias: "max_price",
							Max: &FieldAggregation{
								Field: FieldReference{FieldPath: "price"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing projectID",
			req: AggregationQueryRequest{
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "total_count",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "projectID is required",
		},
		{
			name: "missing databaseID",
			req: AggregationQueryRequest{
				ProjectID: "test-project",
				Parent:    "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "total_count",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "databaseID is required",
		},
		{
			name: "missing aggregations",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{},
				},
			},
			expectError: true,
			errorMsg:    "at least one aggregation is required",
		},
		{
			name: "too many aggregations",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{Alias: "agg1", Count: &CountAggregation{}},
						{Alias: "agg2", Count: &CountAggregation{}},
						{Alias: "agg3", Count: &CountAggregation{}},
						{Alias: "agg4", Count: &CountAggregation{}},
						{Alias: "agg5", Count: &CountAggregation{}},
						{Alias: "agg6", Count: &CountAggregation{}}, // Too many
					},
				},
			},
			expectError: true,
			errorMsg:    "maximum 5 aggregations allowed",
		},
		{
			name: "duplicate alias",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "duplicate",
							Count: &CountAggregation{},
						},
						{
							Alias: "duplicate",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate alias: duplicate",
		},
		{
			name: "aggregation with multiple types",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "invalid",
							Count: &CountAggregation{},
							Sum: &FieldAggregation{
								Field: FieldReference{FieldPath: "price"},
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "aggregation 'invalid' must have exactly one aggregation type",
		},
		{
			name: "sum without field",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					Aggregations: []AggregationFunction{
						{
							Alias: "invalid_sum",
							Sum: &FieldAggregation{
								Field: FieldReference{FieldPath: ""}, // Empty field path
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "sum aggregation 'invalid_sum' must specify a field",
		},
		{
			name: "groupBy without field path",
			req: AggregationQueryRequest{
				ProjectID:  "test-project",
				DatabaseID: "test-database",
				Parent:     "projects/test-project/databases/test-database/documents",
				StructuredAggregationQuery: &StructuredAggregationQuery{
					GroupBy: []GroupByField{
						{FieldPath: ""}, // Empty field path
					},
					Aggregations: []AggregationFunction{
						{
							Alias: "count",
							Count: &CountAggregation{},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "groupBy at index 0 must specify a fieldPath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.validateAggregationRequest(tt.req)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildFieldPath(t *testing.T) {
	uc := &FirestoreUsecase{}

	tests := []struct {
		name      string
		fieldPath string
		expected  string
	}{
		{
			name:      "simple field",
			fieldPath: "price",
			expected:  "fields.price.doubleValue",
		},
		{
			name:      "field with underscore",
			fieldPath: "stock_count",
			expected:  "fields.stock_count.doubleValue",
		},
		{
			name:      "nested-like field name",
			fieldPath: "product.category",
			expected:  "fields.product.category.doubleValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.buildFieldPath(tt.fieldPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFirestoreValue(t *testing.T) {
	uc := &FirestoreUsecase{}

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "int32 value",
			input:    int32(42),
			expected: map[string]interface{}{"integerValue": "42"},
		},
		{
			name:     "int64 value",
			input:    int64(1000),
			expected: map[string]interface{}{"integerValue": "1000"},
		},
		{
			name:     "float64 value",
			input:    123.45,
			expected: map[string]interface{}{"doubleValue": 123.45},
		},
		{
			name:     "string value",
			input:    "test string",
			expected: map[string]interface{}{"stringValue": "test string"},
		},
		{
			name:     "boolean true",
			input:    true,
			expected: map[string]interface{}{"booleanValue": true},
		},
		{
			name:     "boolean false",
			input:    false,
			expected: map[string]interface{}{"booleanValue": false},
		},
		{
			name:     "unknown type",
			input:    []string{"test"},
			expected: map[string]interface{}{"stringValue": "[test]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.formatFirestoreValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildGroupStage(t *testing.T) {
	uc := &FirestoreUsecase{}

	tests := []struct {
		name     string
		aggQuery *StructuredAggregationQuery
		expected map[string]interface{}
	}{
		{
			name: "simple count without groupBy",
			aggQuery: &StructuredAggregationQuery{
				Aggregations: []AggregationFunction{
					{
						Alias: "total_count",
						Count: &CountAggregation{},
					},
				},
			},
			expected: map[string]interface{}{
				"_id":         nil,
				"total_count": map[string]interface{}{"$sum": 1},
			},
		}, {
			name: "count with single groupBy",
			aggQuery: &StructuredAggregationQuery{
				GroupBy: []GroupByField{
					{FieldPath: "category"},
				},
				Aggregations: []AggregationFunction{
					{
						Alias: "count_per_category",
						Count: &CountAggregation{},
					},
				},
			},
			expected: map[string]interface{}{
				"_id":                "$fields.category.stringValue",
				"count_per_category": map[string]interface{}{"$sum": 1},
			},
		},
		{
			name: "multiple aggregations with groupBy",
			aggQuery: &StructuredAggregationQuery{
				GroupBy: []GroupByField{
					{FieldPath: "category"},
				},
				Aggregations: []AggregationFunction{
					{
						Alias: "count",
						Count: &CountAggregation{},
					},
					{
						Alias: "total_price",
						Sum: &FieldAggregation{
							Field: FieldReference{FieldPath: "price"},
						},
					},
					{
						Alias: "avg_price",
						Avg: &FieldAggregation{
							Field: FieldReference{FieldPath: "price"},
						},
					},
					{
						Alias: "min_price",
						Min: &FieldAggregation{
							Field: FieldReference{FieldPath: "price"},
						},
					},
					{
						Alias: "max_price",
						Max: &FieldAggregation{
							Field: FieldReference{FieldPath: "price"},
						},
					},
				},
			}, expected: map[string]interface{}{
				"_id":         "$fields.category.stringValue",
				"count":       map[string]interface{}{"$sum": 1},
				"total_price": map[string]interface{}{"$sum": map[string]interface{}{"$ifNull": []interface{}{"$fields.price.doubleValue", map[string]interface{}{"$ifNull": []interface{}{"$fields.price.integerValue", 0}}}}},
				"avg_price":   map[string]interface{}{"$avg": map[string]interface{}{"$ifNull": []interface{}{"$fields.price.doubleValue", map[string]interface{}{"$ifNull": []interface{}{"$fields.price.integerValue", 0}}}}},
				"min_price":   map[string]interface{}{"$min": "$fields.price.doubleValue"},
				"max_price":   map[string]interface{}{"$max": "$fields.price.doubleValue"},
			},
		},
		{
			name: "multiple groupBy fields",
			aggQuery: &StructuredAggregationQuery{
				GroupBy: []GroupByField{
					{FieldPath: "category"},
					{FieldPath: "brand"},
				},
				Aggregations: []AggregationFunction{
					{
						Alias: "count",
						Count: &CountAggregation{},
					},
				},
			}, expected: map[string]interface{}{
				"_id": map[string]interface{}{
					"category": "$fields.category.stringValue",
					"brand":    "$fields.brand.stringValue",
				},
				"count": map[string]interface{}{"$sum": 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := uc.buildGroupStage(tt.aggQuery)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildProjectStage(t *testing.T) {
	uc := &FirestoreUsecase{}

	tests := []struct {
		name     string
		aggQuery *StructuredAggregationQuery
		expected map[string]interface{}
	}{
		{
			name: "simple count without groupBy",
			aggQuery: &StructuredAggregationQuery{
				Aggregations: []AggregationFunction{
					{
						Alias: "total_count",
						Count: &CountAggregation{},
					},
				},
			},
			expected: map[string]interface{}{
				"_id":         0,
				"total_count": "$total_count",
			},
		},
		{
			name: "with single groupBy",
			aggQuery: &StructuredAggregationQuery{
				GroupBy: []GroupByField{
					{FieldPath: "category"},
				},
				Aggregations: []AggregationFunction{
					{
						Alias: "count",
						Count: &CountAggregation{},
					},
				},
			},
			expected: map[string]interface{}{
				"_id":      0,
				"category": "$_id",
				"count":    "$count",
			},
		},
		{
			name: "with multiple groupBy fields",
			aggQuery: &StructuredAggregationQuery{
				GroupBy: []GroupByField{
					{FieldPath: "category"},
					{FieldPath: "brand"},
				},
				Aggregations: []AggregationFunction{
					{
						Alias: "count",
						Count: &CountAggregation{},
					},
					{
						Alias: "total_price",
						Sum: &FieldAggregation{
							Field: FieldReference{FieldPath: "price"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"_id":         0,
				"category":    "$_id.category",
				"brand":       "$_id.brand",
				"count":       "$count",
				"total_price": "$total_price",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := uc.buildProjectStage(tt.aggQuery)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
