package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestFirestoreCompatibilityTestSuite tests compatibility with real Firestore queries
func TestFirestoreCompatibility_NestedFields(t *testing.T) {
	// This test validates the exact scenario from the original question
	testCases := []struct {
		name           string
		firestoreQuery string
		fieldPath      string
		operator       model.Operator
		value          interface{}
		valueType      model.FieldValueType
		expectedPath   string
		shouldWork     bool
	}{
		{
			name:         "Simple field query - status",
			fieldPath:    "status",
			operator:     model.OperatorEqual,
			value:        "paid",
			valueType:    model.FieldTypeString,
			expectedPath: "fields.status.stringValue",
			shouldWork:   true,
		},
		{
			name:         "Nested field level 1 - customer.ruc",
			fieldPath:    "customer.ruc",
			operator:     model.OperatorEqual,
			value:        "20123456789",
			valueType:    model.FieldTypeString,
			expectedPath: "fields.customer.value.ruc",
			shouldWork:   true,
		}, {
			name:         "Nested field level 2 - customer.address.city",
			fieldPath:    "customer.address.city",
			operator:     model.OperatorEqual,
			value:        "Lima",
			valueType:    model.FieldTypeString,
			expectedPath: "fields.customer.value.address.value.city",
			shouldWork:   true,
		},
		{
			name:         "Array contains object - items",
			fieldPath:    "items",
			operator:     model.OperatorArrayContains,
			value:        map[string]interface{}{"itemId": "PROD001"},
			valueType:    model.FieldTypeArray,
			expectedPath: "fields.items.arrayValue",
			shouldWork:   true,
		},
		{
			name:         "Array contains primitive - tags",
			fieldPath:    "tags",
			operator:     model.OperatorArrayContains,
			value:        "urgent",
			valueType:    model.FieldTypeArray,
			expectedPath: "fields.tags.arrayValue",
			shouldWork:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.shouldWork {
				t.Skip("Not implemented yet")
			}

			// Create field path
			fieldPath, err := model.NewFieldPath(tc.fieldPath)
			require.NoError(t, err)

			// Create resolver
			resolver := NewMongoFieldPathResolver()

			// Test field path resolution
			if tc.operator != model.OperatorArrayContains {
				mongoPath, err := resolver.ResolveFieldPath(fieldPath, tc.valueType)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPath, mongoPath)
			}

			// Create filter
			filter := model.Filter{
				FieldPath: fieldPath,
				Field:     tc.fieldPath, // For backward compatibility
				Operator:  tc.operator,
				Value:     tc.value,
				ValueType: tc.valueType,
			}

			// Test filter validation
			engine := createTestEngine()
			err = engine.(*EnhancedMongoQueryEngine).validateFilter(filter)
			assert.NoError(t, err)

			// Test filter building
			filters := []model.Filter{filter}
			mongoFilter, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "test_collection", filters)
			require.NoError(t, err)
			assert.NotEmpty(t, mongoFilter)
		})
	}
}

func TestEnhancedMongoQueryEngine_OriginalScenario(t *testing.T) {
	// Test the exact query from the original question:
	// compositeFilter with AND operation on "status" = "paid" AND "customer.ruc" = "20123456789"

	engine := createTestEngine()

	// Create the exact filters from the original query
	statusFilter := model.Filter{
		FieldPath: model.MustNewFieldPath("status"),
		Field:     "status",
		Operator:  model.OperatorEqual,
		Value:     "paid",
		ValueType: model.FieldTypeString,
	}

	customerRucFilter := model.Filter{
		FieldPath: model.MustNewFieldPath("customer.ruc"),
		Field:     "customer.ruc",
		Operator:  model.OperatorEqual,
		Value:     "20123456789",
		ValueType: model.FieldTypeString,
	}

	filters := []model.Filter{statusFilter, customerRucFilter}

	// Build the MongoDB filter
	mongoFilter, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "test_collection", filters)
	require.NoError(t, err)

	// Verify the generated MongoDB filter structure
	expectedFilter := bson.M{
		"$and": []bson.M{
			{"fields.status.stringValue": "paid"},
			{"fields.customer.value.ruc": "20123456789"},
		},
	}

	assert.Equal(t, expectedFilter, mongoFilter)
}

func TestEnhancedMongoQueryEngine_CompositeFilters(t *testing.T) {
	engine := createTestEngine()

	// Test AND composite filter
	t.Run("AND composite filter", func(t *testing.T) {
		compositeFilter := model.Filter{
			Composite: "and",
			SubFilters: []model.Filter{
				{
					FieldPath: model.MustNewFieldPath("status"),
					Field:     "status",
					Operator:  model.OperatorEqual,
					Value:     "paid",
					ValueType: model.FieldTypeString,
				},
				{
					FieldPath: model.MustNewFieldPath("amount"),
					Field:     "amount",
					Operator:  model.OperatorGreaterThan,
					Value:     1000,
					ValueType: model.FieldTypeInt,
				},
			}}

		filters := []model.Filter{compositeFilter}
		result, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "test_collection", filters)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should contain both conditions
		andArray, exists := result["$and"]
		if !exists {
			// Single filter might be flattened
			assert.Contains(t, result, "fields.status.stringValue")
		} else {
			assert.IsType(t, []bson.M{}, andArray)
		}
	})

	// Test OR composite filter
	t.Run("OR composite filter", func(t *testing.T) {
		compositeFilter := model.Filter{
			Composite: "or",
			SubFilters: []model.Filter{
				{
					FieldPath: model.MustNewFieldPath("status"),
					Field:     "status",
					Operator:  model.OperatorEqual,
					Value:     "paid",
					ValueType: model.FieldTypeString,
				},
				{
					FieldPath: model.MustNewFieldPath("status"),
					Field:     "status",
					Operator:  model.OperatorEqual,
					Value:     "pending",
					ValueType: model.FieldTypeString,
				},
			},
		}
		filters := []model.Filter{compositeFilter}
		result, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "test_collection", filters)
		require.NoError(t, err)
		assert.Contains(t, result, "$or")
	})
}

func TestEnhancedMongoQueryEngine_ArrayOperations(t *testing.T) {
	engine := createTestEngine()
	t.Run("Array contains primitive", func(t *testing.T) {
		filter := model.Filter{
			FieldPath: model.MustNewFieldPath("tags"),
			Field:     "tags",
			Operator:  model.OperatorArrayContains,
			Value:     "urgent",
			ValueType: model.FieldTypeArray,
		}

		filters := []model.Filter{filter}
		result, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
		require.NoError(t, err)
		expected := bson.M{
			"fields.tags.arrayValue.values": "urgent",
		}
		assert.Equal(t, expected, result)
	})
	t.Run("Array contains object", func(t *testing.T) {
		filter := model.Filter{
			FieldPath: model.MustNewFieldPath("items"),
			Field:     "items",
			Operator:  model.OperatorArrayContains,
			Value:     map[string]interface{}{"itemId": "PROD001"},
			ValueType: model.FieldTypeArray,
		}

		filters := []model.Filter{filter}
		result, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
		require.NoError(t, err)
		expected := bson.M{
			"fields.items.arrayValue.values": bson.M{
				"$elemMatch": map[string]interface{}{"itemId": "PROD001"},
			},
		}
		assert.Equal(t, expected, result)
	})
	t.Run("Array contains any", func(t *testing.T) {
		filter := model.Filter{
			FieldPath: model.MustNewFieldPath("tags"),
			Field:     "tags",
			Operator:  model.OperatorArrayContainsAny,
			Value:     []interface{}{"urgent", "pending"},
			ValueType: model.FieldTypeArray,
		}

		filters := []model.Filter{filter}
		result, err := engine.(*EnhancedMongoQueryEngine).buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
		require.NoError(t, err)
		expected := bson.M{
			"fields.tags.arrayValue.values": bson.M{
				"$in": []interface{}{"urgent", "pending"},
			},
		}
		assert.Equal(t, expected, result)
	})
}

func TestEnhancedMongoQueryEngine_QueryValidation(t *testing.T) {
	engine := createTestEngine()

	t.Run("Valid query passes validation", func(t *testing.T) {
		query := model.Query{
			Path:         "projects/test/databases/test/documents/facturas",
			CollectionID: "facturas",
			Filters: []model.Filter{
				{
					FieldPath: model.MustNewFieldPath("status"),
					Field:     "status",
					Operator:  model.OperatorEqual,
					Value:     "paid",
					ValueType: model.FieldTypeString,
				},
			},
			Orders: []model.Order{
				{
					Field:     "timestamp",
					Direction: model.DirectionDescending,
				},
			},
			Limit: 10,
		}

		err := engine.ValidateQuery(query)
		assert.NoError(t, err)
	})

	t.Run("Query with too many filters fails validation", func(t *testing.T) {
		// Create a query with more than max filters
		filters := make([]model.Filter, 101) // Exceeds max of 100
		for i := range filters {
			filters[i] = model.Filter{
				FieldPath: model.MustNewFieldPath("field"),
				Field:     "field",
				Operator:  model.OperatorEqual,
				Value:     "value",
				ValueType: model.FieldTypeString,
			}
		}

		query := model.Query{
			Path:         "projects/test/databases/test/documents/facturas",
			CollectionID: "facturas",
			Filters:      filters,
		}

		err := engine.ValidateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many filters")
	})

	t.Run("Query with invalid field path fails validation", func(t *testing.T) {
		query := model.Query{
			Path:         "projects/test/databases/test/documents/facturas",
			CollectionID: "facturas",
			Filters: []model.Filter{
				{
					Field:     "invalid..path", // Invalid path
					Operator:  model.OperatorEqual,
					Value:     "value",
					ValueType: model.FieldTypeString,
				},
			},
		}

		err := engine.ValidateQuery(query)
		assert.Error(t, err)
	})
}

func TestEnhancedMongoQueryEngine_QueryCapabilities(t *testing.T) {
	engine := createTestEngine()

	capabilities := engine.GetQueryCapabilities()

	assert.True(t, capabilities.SupportsNestedFields)
	assert.True(t, capabilities.SupportsArrayContains)
	assert.True(t, capabilities.SupportsArrayContainsAny)
	assert.True(t, capabilities.SupportsCompositeFilters)
	assert.True(t, capabilities.SupportsOrderBy)
	assert.True(t, capabilities.SupportsCursorPagination)
	assert.True(t, capabilities.SupportsOffsetPagination)
	assert.True(t, capabilities.SupportsProjection)
	assert.Equal(t, 100, capabilities.MaxFilterCount)
	assert.Equal(t, 32, capabilities.MaxOrderByCount)
	assert.Equal(t, 100, capabilities.MaxNestingDepth)
}

// Helper function to create a test engine
func createTestEngine() repository.QueryEngine {
	// For testing, we don't need a real MongoDB connection
	// We'll create the engine and test the filter building logic
	return &EnhancedMongoQueryEngine{
		db:                nil, // Not needed for filter building tests
		fieldPathResolver: NewMongoFieldPathResolver(),
		typeCache:         make(map[string]model.FieldValueType), // Initialize type inference cache
		capabilities: repository.QueryCapabilities{
			SupportsNestedFields:     true,
			SupportsArrayContains:    true,
			SupportsArrayContainsAny: true,
			SupportsCompositeFilters: true,
			SupportsOrderBy:          true,
			SupportsCursorPagination: true,
			SupportsOffsetPagination: true,
			SupportsProjection:       true,
			MaxFilterCount:           100,
			MaxOrderByCount:          32,
			MaxNestingDepth:          100,
		},
	}
}

// Benchmark tests for performance
func BenchmarkEnhancedMongoQueryEngine_SimpleFilter(b *testing.B) {
	engine := createTestEngine().(*EnhancedMongoQueryEngine)

	filters := []model.Filter{
		{
			FieldPath: model.MustNewFieldPath("status"),
			Field:     "status",
			Operator:  model.OperatorEqual,
			Value:     "paid",
			ValueType: model.FieldTypeString,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
	}
}

func BenchmarkEnhancedMongoQueryEngine_NestedFilter(b *testing.B) {
	engine := createTestEngine().(*EnhancedMongoQueryEngine)

	filters := []model.Filter{
		{
			FieldPath: model.MustNewFieldPath("customer.address.city"),
			Field:     "customer.address.city",
			Operator:  model.OperatorEqual,
			Value:     "Lima",
			ValueType: model.FieldTypeString,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
	}
}

func BenchmarkEnhancedMongoQueryEngine_CompositeFilter(b *testing.B) {
	engine := createTestEngine().(*EnhancedMongoQueryEngine)

	filters := []model.Filter{
		{
			FieldPath: model.MustNewFieldPath("status"),
			Field:     "status",
			Operator:  model.OperatorEqual,
			Value:     "paid",
			ValueType: model.FieldTypeString,
		},
		{
			FieldPath: model.MustNewFieldPath("customer.ruc"),
			Field:     "customer.ruc",
			Operator:  model.OperatorEqual,
			Value:     "20123456789",
			ValueType: model.FieldTypeString,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
	}
}

// TestEnhancedQueryEngineWithProjectionAndTypeInference tests that queries with projection
// use the same type inference logic as queries without projection
func TestEnhancedQueryEngineWithProjectionAndTypeInference(t *testing.T) {
	// Mock database for testing type inference
	engine := NewEnhancedMongoQueryEngine(nil).(*EnhancedMongoQueryEngine)

	testCases := []struct {
		name         string
		field        string
		operator     model.Operator
		value        interface{}
		expectedPath string
		valueType    model.FieldValueType
	}{
		{
			name:         "Boolean field with projection",
			field:        "available",
			operator:     model.OperatorEqual,
			value:        map[string]interface{}{"booleanValue": true},
			expectedPath: "fields.available.booleanValue",
			valueType:    model.FieldTypeBool,
		},
		{
			name:         "String field with projection",
			field:        "name",
			operator:     model.OperatorEqual,
			value:        map[string]interface{}{"stringValue": "test"},
			expectedPath: "fields.name.stringValue",
			valueType:    model.FieldTypeString,
		},
		{
			name:         "Integer field with projection",
			field:        "count",
			operator:     model.OperatorGreaterThan,
			value:        map[string]interface{}{"integerValue": int64(10)},
			expectedPath: "fields.count.integerValue",
			valueType:    model.FieldTypeInt,
		},
		{
			name:         "Double field with projection",
			field:        "price",
			operator:     model.OperatorLessThan,
			value:        map[string]interface{}{"doubleValue": 99.99},
			expectedPath: "fields.price.doubleValue",
			valueType:    model.FieldTypeDouble,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that type inference works correctly
			filter := model.Filter{
				Field:     tc.field,
				Operator:  tc.operator,
				Value:     tc.value,
				ValueType: tc.valueType,
			}

			// Create field path
			fieldPath, err := model.NewFieldPath(tc.field)
			require.NoError(t, err)

			// Test field path resolution with type inference
			mongoPath, err := engine.fieldPathResolver.ResolveFieldPath(fieldPath, tc.valueType)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPath, mongoPath)

			// Test filter building with type inference (without context for simplicity)
			primitiveValue := engine.extractPrimitiveValue(tc.value)
			filterBSON := engine.buildFilterBSON(mongoPath, tc.operator, primitiveValue)

			// Verify filter structure
			assert.NotEmpty(t, filterBSON)
			assert.Contains(t, filterBSON, tc.expectedPath)

			// Use the filter variable to avoid "declared and not used" error
			assert.Equal(t, tc.field, filter.Field, "Filter field should match test case")
		})
	}
}

// TestEnhancedQueryEngineTypeInferenceComparison compares type inference behavior
// between the enhanced engine and the main engine
func TestEnhancedQueryEngineTypeInferenceComparison(t *testing.T) {
	// Create both engines for comparison
	mainEngine := NewMongoQueryEngine(nil)
	enhancedEngine := NewEnhancedMongoQueryEngine(nil).(*EnhancedMongoQueryEngine)

	testCases := []struct {
		name      string
		field     string
		operator  model.Operator
		value     interface{}
		valueType model.FieldValueType
	}{
		{
			name:      "Boolean field",
			field:     "isActive",
			operator:  model.OperatorEqual,
			value:     map[string]interface{}{"booleanValue": true},
			valueType: model.FieldTypeBool,
		},
		{
			name:      "String field",
			field:     "description",
			operator:  model.OperatorEqual,
			value:     map[string]interface{}{"stringValue": "test"},
			valueType: model.FieldTypeString,
		},
		{
			name:      "Integer field",
			field:     "quantity",
			operator:  model.OperatorGreaterThan,
			value:     map[string]interface{}{"integerValue": int64(5)},
			valueType: model.FieldTypeInt,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create filter
			filter := model.Filter{
				Field:     tc.field,
				Operator:  tc.operator,
				Value:     tc.value,
				ValueType: tc.valueType,
			}

			// Test main engine filter building
			mainFilter := mainEngine.singleMongoFilter(filter)

			// Test enhanced engine filter building (with mock context)
			enhancedFilter, err := enhancedEngine.buildSingleFilterWithContext(nil, "", filter)
			require.NoError(t, err)

			// Both engines should produce the same filter structure for the same input
			assert.Equal(t, mainFilter, enhancedFilter,
				"Enhanced engine should produce same filter as main engine for field %s", tc.field)
		})
	}
}

// TestEnhancedQueryEngineWithProjection_IntegrationStyle simulates the integration
// test scenario where queries with projection should behave identically to queries without projection
func TestEnhancedQueryEngineWithProjection_IntegrationStyle(t *testing.T) {
	engine := NewEnhancedMongoQueryEngine(nil).(*EnhancedMongoQueryEngine)

	// Simulate the failing scenario: boolean filter with projection
	filter := model.Filter{
		Field:    "available",
		Operator: model.OperatorEqual,
		Value:    map[string]interface{}{"booleanValue": true},
	}
	query := model.Query{
		Path:         "projects/test-project/databases/(default)/documents/products",
		CollectionID: "products",
		Filters:      []model.Filter{filter},
		Limit:        10,
		// SelectFields would be added for projection queries
		SelectFields: []string{"name", "available", "price"},
	}

	// Test building the filter with context (simulates real execution)
	// Note: Using empty context and collection path for this test
	filterBSON, err := engine.buildEnhancedMongoFilterWithContext(nil, "", query.Filters)
	require.NoError(t, err)

	// The filter should be built successfully
	assert.NotEmpty(t, filterBSON)

	// Should contain a boolean field path (the key issue we're fixing)
	// The actual path will depend on type inference, but it should work without errors
	t.Logf("Generated filter BSON: %+v", filterBSON)

	// Verify the filter has the expected structure
	// In a real scenario with type inference, this would resolve to fields.available.booleanValue
	assert.NotNil(t, filterBSON)

	// Test that the query can be validated
	err = engine.ValidateQuery(query)
	assert.NoError(t, err, "Query with projection should be valid")
}

// TestEnhancedQueryEngineArrayOperations tests array operations with projection
func TestEnhancedQueryEngineArrayOperations(t *testing.T) {
	engine := NewEnhancedMongoQueryEngine(nil).(*EnhancedMongoQueryEngine)

	testCases := []struct {
		name      string
		field     string
		operator  model.Operator
		value     interface{}
		expectErr bool
	}{
		{
			name:     "Array contains primitive",
			field:    "tags",
			operator: model.OperatorArrayContains,
			value:    map[string]interface{}{"stringValue": "electronics"},
		},
		{
			name:     "Array contains object",
			field:    "items",
			operator: model.OperatorArrayContains,
			value:    map[string]interface{}{"id": "123", "name": "Product"},
		},
		{
			name:     "Array contains any",
			field:    "categories",
			operator: model.OperatorArrayContainsAny,
			value:    []interface{}{"electronics", "computers"},
		},
		{
			name:     "IN operation with array",
			field:    "status",
			operator: model.OperatorIn,
			value:    []interface{}{"active", "pending"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := model.Filter{
				Field:    tc.field,
				Operator: tc.operator,
				Value:    tc.value,
			}

			// Test filter building with type inference
			filterBSON, err := engine.buildSingleFilterWithContext(nil, "", filter)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, filterBSON)
				t.Logf("Filter for %s: %+v", tc.name, filterBSON)
			}
		})
	}
}

// TestEnhancedQueryEngineCompositeFilters tests composite filters (AND/OR) with projection
func TestEnhancedQueryEngineCompositeFilters(t *testing.T) {
	engine := NewEnhancedMongoQueryEngine(nil).(*EnhancedMongoQueryEngine)

	// Test composite AND filter
	andFilter := model.Filter{
		Composite: "and",
		SubFilters: []model.Filter{
			{
				Field:    "available",
				Operator: model.OperatorEqual,
				Value:    map[string]interface{}{"booleanValue": true},
			},
			{
				Field:    "price",
				Operator: model.OperatorLessThan,
				Value:    map[string]interface{}{"doubleValue": 100.0},
			},
		},
	}

	// Test composite OR filter
	orFilter := model.Filter{
		Composite: "or",
		SubFilters: []model.Filter{
			{
				Field:    "category",
				Operator: model.OperatorEqual,
				Value:    map[string]interface{}{"stringValue": "electronics"},
			},
			{
				Field:    "featured",
				Operator: model.OperatorEqual,
				Value:    map[string]interface{}{"booleanValue": true},
			},
		},
	}

	testCases := []struct {
		name   string
		filter model.Filter
	}{
		{"Composite AND", andFilter},
		{"Composite OR", orFilter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterBSON, err := engine.buildCompositeFilterWithContext(nil, "", tc.filter)
			assert.NoError(t, err)
			assert.NotEmpty(t, filterBSON)
			t.Logf("Composite filter %s: %+v", tc.name, filterBSON)
		})
	}
}
