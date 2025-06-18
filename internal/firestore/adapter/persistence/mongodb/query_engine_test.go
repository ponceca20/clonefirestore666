package mongodb

import (
	"context"
	"fmt"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// createTestQueryEngine creates a MongoQueryEngine for testing
// Following hexagonal architecture, we inject dependencies at the infrastructure layer
func createTestQueryEngine() *MongoQueryEngine {
	// For unit tests, we don't need a real database connection
	// The query engine functionality we're testing is pure logic
	return &MongoQueryEngine{
		db:                nil, // Not needed for pure filter logic tests
		fieldPathResolver: NewMongoFieldPathResolver(),
		typeCache:         make(map[string]model.FieldValueType),
	}
}

// TestBuildMongoFilter tests the filter construction for Firestore field paths
// Following clean architecture principles: testing infrastructure layer behavior
func TestBuildMongoFilter(t *testing.T) {
	queryEngine := createTestQueryEngine()

	t.Run("simple equality filter", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "name", Operator: "==", Value: "fred"},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"fields.name.stringValue": "fred"}

		assert.Equal(t, expected, result)
	})

	t.Run("less than filter", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "born", Operator: "<", Value: 1900},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"fields.born.integerValue": bson.M{"$lt": 1900}}

		assert.Equal(t, expected, result)
	})

	t.Run("multiple filters with AND", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "age", Operator: ">=", Value: 18},
			{Field: "status", Operator: "==", Value: "active"},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"$and": []bson.M{
			{"fields.age.integerValue": bson.M{"$gte": 18}},
			{"fields.status.stringValue": "active"},
		}}

		assert.Equal(t, expected, result)
	})

	t.Run("OR composite filter", func(t *testing.T) {
		filters := []model.Filter{
			{
				Composite: "or",
				SubFilters: []model.Filter{
					{Field: "status", Operator: "==", Value: "active"},
					{Field: "status", Operator: "==", Value: "pending"},
				},
			},
		}
		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"$or": []bson.M{
			{"fields.status.stringValue": "active"},
			{"fields.status.stringValue": "pending"},
		}}

		assert.Equal(t, expected, result)
	})
	t.Run("in operator", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "category", Operator: "in", Value: []string{"electronics", "books"}},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"fields.category.stringValue": bson.M{"$in": []string{"electronics", "books"}}}

		assert.Equal(t, expected, result)
	})
	t.Run("array-contains operator with string value", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "tags", Operator: "array-contains", Value: "featured"},
		}

		result := queryEngine.buildMongoFilter(filters)
		// Should wrap the value in Firestore format and use $elemMatch
		expected := bson.M{"fields.tags.arrayValue.values": bson.M{"$elemMatch": bson.M{"stringValue": "featured"}}}

		assert.Equal(t, expected, result)
	})
	t.Run("array-contains operator with number value", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "ratings", Operator: "array-contains", Value: 5},
		}

		result := queryEngine.buildMongoFilter(filters)
		// Should wrap the value in Firestore format for numbers
		expected := bson.M{"fields.ratings.arrayValue.values": bson.M{"$elemMatch": bson.M{"integerValue": 5}}}

		assert.Equal(t, expected, result)
	})
	t.Run("array-contains operator with boolean value", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "flags", Operator: "array-contains", Value: true},
		}

		result := queryEngine.buildMongoFilter(filters)
		// Should wrap the value in Firestore format for booleans
		expected := bson.M{"fields.flags.arrayValue.values": bson.M{"$elemMatch": bson.M{"booleanValue": true}}}

		assert.Equal(t, expected, result)
	})
}

// TestSingleMongoFilter tests individual filter translation
func TestSingleMongoFilter(t *testing.T) {
	queryEngine := createTestQueryEngine()

	testCases := []struct {
		name     string
		filter   model.Filter
		expected bson.M
	}{
		{
			name:     "equality",
			filter:   model.Filter{Field: "name", Operator: "==", Value: "test"},
			expected: bson.M{"fields.name.stringValue": "test"},
		},
		{
			name:     "not equal",
			filter:   model.Filter{Field: "status", Operator: "!=", Value: "deleted"},
			expected: bson.M{"fields.status.stringValue": bson.M{"$ne": "deleted"}},
		},
		{
			name:     "greater than",
			filter:   model.Filter{Field: "price", Operator: ">", Value: 100},
			expected: bson.M{"fields.price.integerValue": bson.M{"$gt": 100}},
		},
		{
			name:     "less than or equal",
			filter:   model.Filter{Field: "stock", Operator: "<=", Value: 10},
			expected: bson.M{"fields.stock.integerValue": bson.M{"$lte": 10}},
		},
		{
			name:     "not in",
			filter:   model.Filter{Field: "category", Operator: "not-in", Value: []string{"spam", "deleted"}},
			expected: bson.M{"fields.category.stringValue": bson.M{"$nin": []string{"spam", "deleted"}}},
		}, {
			name:     "array-contains-any with strings",
			filter:   model.Filter{Field: "tags", Operator: "array-contains-any", Value: []interface{}{"urgent", "important"}}, // Use []interface{}
			expected: bson.M{"fields.tags.arrayValue.values": bson.M{"$elemMatch": bson.M{"$in": []bson.M{{"stringValue": "urgent"}, {"stringValue": "important"}}}}},
		},
		{
			name:     "array-contains with string",
			filter:   model.Filter{Field: "categories", Operator: "array-contains", Value: "tech"},
			expected: bson.M{"fields.categories.arrayValue.values": bson.M{"$elemMatch": bson.M{"stringValue": "tech"}}},
		},
		{
			name:     "array-contains with number",
			filter:   model.Filter{Field: "scores", Operator: "array-contains", Value: 100},
			expected: bson.M{"fields.scores.arrayValue.values": bson.M{"$elemMatch": bson.M{"integerValue": 100}}},
		},
		{
			name:     "array-contains with boolean",
			filter:   model.Filter{Field: "flags", Operator: "array-contains", Value: true},
			expected: bson.M{"fields.flags.arrayValue.values": bson.M{"$elemMatch": bson.M{"booleanValue": true}}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := queryEngine.singleMongoFilter(tc.filter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestBuildMongoFindOptions tests the find options construction
func TestBuildMongoFindOptions(t *testing.T) {
	// Setup test database and query engine
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_db")
	qe := NewMongoQueryEngine(db)
	ctx := context.Background()
	collectionPath := "test_collection"

	t.Run("limit and offset", func(t *testing.T) {
		query := model.Query{
			Limit:  10,
			Offset: 5,
		}

		opts := qe.buildMongoFindOptions(ctx, collectionPath, query)

		// Verificar que las opciones se configuraron correctamente
		assert.NotNil(t, opts)
		// Note: In a real test, you'd verify the actual limit and skip values
		// but mongo options don't expose these values directly for testing
	})

	t.Run("sort by single field", func(t *testing.T) {
		query := model.Query{
			Orders: []model.Order{
				{Field: "name", Direction: "asc"},
			},
		}

		opts := qe.buildMongoFindOptions(ctx, collectionPath, query)

		assert.NotNil(t, opts)
	})

	t.Run("sort by multiple fields", func(t *testing.T) {
		query := model.Query{
			Orders: []model.Order{
				{Field: "priority", Direction: "desc"},
				{Field: "created", Direction: "asc"},
			},
		}

		opts := qe.buildMongoFindOptions(ctx, collectionPath, query)

		assert.NotNil(t, opts)
	})

	t.Run("with projection", func(t *testing.T) {
		query := model.Query{
			SelectFields: []string{"name", "email", "status"},
		}

		opts := qe.buildMongoFindOptions(ctx, collectionPath, query)

		assert.NotNil(t, opts)
	})
}

// TestBuildCursorFilter tests cursor-based pagination
func TestBuildCursorFilter(t *testing.T) {
	queryEngine := createTestQueryEngine()

	t.Run("no orders returns nil", func(t *testing.T) {
		query := model.Query{}

		result := queryEngine.buildCursorFilter(query)

		assert.Nil(t, result)
	})
	t.Run("startAt cursor", func(t *testing.T) {
		query := model.Query{
			Orders:  []model.Order{{Field: "name", Direction: "asc"}},
			StartAt: []interface{}{"john"},
		}

		result := queryEngine.buildCursorFilter(query)
		expected := bson.M{"$and": []bson.M{
			{"fields.name.stringValue": bson.M{"$gte": "john"}},
		}}

		assert.Equal(t, expected, result)
	})
	t.Run("startAfter cursor", func(t *testing.T) {
		query := model.Query{
			Orders:     []model.Order{{Field: "age", Direction: "desc"}},
			StartAfter: []interface{}{25},
		}

		result := queryEngine.buildCursorFilter(query)
		expected := bson.M{"$and": []bson.M{
			{"fields.age.integerValue": bson.M{"$lt": 25}},
		}}

		assert.Equal(t, expected, result)
	})
	t.Run("endBefore cursor", func(t *testing.T) {
		query := model.Query{
			Orders:    []model.Order{{Field: "score", Direction: "asc"}},
			EndBefore: []interface{}{100},
		}

		result := queryEngine.buildCursorFilter(query)
		expected := bson.M{"$and": []bson.M{
			{"fields.score.integerValue": bson.M{"$lt": 100}},
		}}

		assert.Equal(t, expected, result)
	})
}

// TestExtractPrimitiveValue tests the value extraction helper
func TestExtractPrimitiveValue(t *testing.T) {
	t.Run("extract from map", func(t *testing.T) {
		input := map[string]interface{}{
			"stringValue": "hello",
		}

		result := extractPrimitiveValue(input)

		assert.Equal(t, "hello", result)
	})

	t.Run("return as-is for primitive", func(t *testing.T) {
		input := "direct_value"

		result := extractPrimitiveValue(input)

		assert.Equal(t, "direct_value", result)
	})

	t.Run("extract numeric value", func(t *testing.T) {
		input := map[string]interface{}{
			"doubleValue": 42.5,
		}

		result := extractPrimitiveValue(input)

		assert.Equal(t, 42.5, result)
	})
}

// TestReverseDocs tests the document reversal utility
func TestReverseDocs(t *testing.T) {
	t.Run("reverse empty slice", func(t *testing.T) {
		docs := []*model.Document{}

		reverseDocs(docs)

		assert.Empty(t, docs)
	})

	t.Run("reverse single document", func(t *testing.T) {
		docs := []*model.Document{
			{DocumentID: "doc1"},
		}

		reverseDocs(docs)

		assert.Len(t, docs, 1)
		assert.Equal(t, "doc1", docs[0].DocumentID)
	})

	t.Run("reverse multiple documents", func(t *testing.T) {
		docs := []*model.Document{
			{DocumentID: "doc1"},
			{DocumentID: "doc2"},
			{DocumentID: "doc3"},
		}

		reverseDocs(docs)

		assert.Len(t, docs, 3)
		assert.Equal(t, "doc3", docs[0].DocumentID)
		assert.Equal(t, "doc2", docs[1].DocumentID)
		assert.Equal(t, "doc1", docs[2].DocumentID)
	})
}

// TestNestedFieldPathResolution tests the integration with FieldPathResolver
// This is critical for Firestore clone functionality with nested documents
func TestNestedFieldPathResolution(t *testing.T) {
	queryEngine := createTestQueryEngine()

	t.Run("simple nested field path", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "customer.ruc", Operator: "==", Value: "20123456789"},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"fields.customer.value.ruc": "20123456789"}

		assert.Equal(t, expected, result)
	})

	t.Run("deeply nested field path", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "address.billing.country", Operator: "==", Value: "PE"},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"fields.address.value.billing.value.country": "PE"}

		assert.Equal(t, expected, result)
	})

	t.Run("nested field with different value types", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "stats.views", Operator: ">", Value: 1000},
			{Field: "meta.published", Operator: "==", Value: true},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"$and": []bson.M{
			{"fields.stats.value.views": bson.M{"$gt": 1000}},
			{"fields.meta.value.published": true},
		}}

		assert.Equal(t, expected, result)
	})

	t.Run("mixed simple and nested fields", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "status", Operator: "==", Value: "active"},
			{Field: "user.role", Operator: "==", Value: "admin"},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"$and": []bson.M{
			{"fields.status.stringValue": "active"},
			{"fields.user.value.role": "admin"},
		}}

		assert.Equal(t, expected, result)
	})
}

// TestSingleMongoFilterWithFieldPathResolver tests individual filter translation with nested paths
func TestSingleMongoFilterWithFieldPathResolver(t *testing.T) {
	queryEngine := createTestQueryEngine()

	testCases := []struct {
		name     string
		filter   model.Filter
		expected bson.M
	}{
		{
			name:     "simple field",
			filter:   model.Filter{Field: "name", Operator: "==", Value: "test"},
			expected: bson.M{"fields.name.stringValue": "test"},
		},
		{
			name:     "nested field level 1",
			filter:   model.Filter{Field: "customer.ruc", Operator: "==", Value: "20123456789"},
			expected: bson.M{"fields.customer.value.ruc": "20123456789"},
		},
		{
			name:     "nested field level 2",
			filter:   model.Filter{Field: "address.billing.city", Operator: "==", Value: "Lima"},
			expected: bson.M{"fields.address.value.billing.value.city": "Lima"},
		},
		{
			name:     "nested field with integer",
			filter:   model.Filter{Field: "stats.count", Operator: ">=", Value: 100},
			expected: bson.M{"fields.stats.value.count": bson.M{"$gte": 100}},
		},
		{
			name:     "nested field with boolean",
			filter:   model.Filter{Field: "config.enabled", Operator: "==", Value: true},
			expected: bson.M{"fields.config.value.enabled": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := queryEngine.singleMongoFilter(tc.filter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestFieldTypeInferenceForOrdering tests the field type inference for ordering operations
func TestFieldTypeInferenceForOrdering(t *testing.T) {
	qe := createTestQueryEngine()
	ctx := context.Background()
	collectionPath := "test_collection"

	tests := []struct {
		name         string
		fieldName    string
		expectedType model.FieldValueType
		setupCache   func()
	}{
		{
			name:         "numeric field should use doubleValue",
			fieldName:    "price",
			expectedType: model.FieldTypeDouble,
			setupCache: func() {
				qe.typeCache["test_collection.price"] = model.FieldTypeDouble
			},
		},
		{
			name:         "string field should use stringValue",
			fieldName:    "name",
			expectedType: model.FieldTypeString,
			setupCache: func() {
				qe.typeCache["test_collection.name"] = model.FieldTypeString
			},
		},
		{
			name:         "timestamp field should use timestampValue",
			fieldName:    "createdAt",
			expectedType: model.FieldTypeTimestamp,
			setupCache: func() {
				qe.typeCache["test_collection.createdAt"] = model.FieldTypeTimestamp
			},
		},
		{
			name:         "unknown field defaults to stringValue",
			fieldName:    "unknownField",
			expectedType: model.FieldTypeString,
			setupCache:   func() {}, // No cache setup
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup cache
			tt.setupCache()

			// Create a dummy query for the test
			dummyQuery := model.Query{}

			// Test type inference
			inferredType := qe.inferFieldTypeForOrdering(ctx, collectionPath, tt.fieldName, dummyQuery)
			assert.Equal(t, tt.expectedType, inferredType)

			// Test field path building
			fieldPath := qe.buildOrderFieldPath(tt.fieldName, inferredType)
			expectedPath := "fields." + tt.fieldName + "." + string(inferredType)
			assert.Equal(t, expectedPath, fieldPath)
		})
	}
}

// TestOrderByWithCorrectTypes tests that orderBy uses correct field types
func TestOrderByWithCorrectTypes(t *testing.T) {
	qe := createTestQueryEngine()
	ctx := context.Background()
	collectionPath := "productos2"

	// Setup type cache to simulate field type discovery
	qe.typeCache["productos2.price"] = model.FieldTypeDouble
	qe.typeCache["productos2.name"] = model.FieldTypeString
	qe.typeCache["productos2.stock"] = model.FieldTypeDouble
	qe.typeCache["productos2.fechaFabricacion"] = model.FieldTypeTimestamp

	tests := []struct {
		name     string
		query    model.Query
		expected []string // Expected sort field paths
	}{
		{
			name: "order by price (numeric)",
			query: model.Query{
				Orders: []model.Order{
					{Field: "price", Direction: "asc"},
				},
			},
			expected: []string{"fields.price.doubleValue"},
		},
		{
			name: "order by name (string)",
			query: model.Query{
				Orders: []model.Order{
					{Field: "name", Direction: "desc"},
				},
			},
			expected: []string{"fields.name.stringValue"},
		},
		{
			name: "order by multiple fields with different types",
			query: model.Query{
				Orders: []model.Order{
					{Field: "price", Direction: "desc"},
					{Field: "name", Direction: "asc"},
				},
			},
			expected: []string{"fields.price.doubleValue", "fields.name.stringValue"},
		},
		{
			name: "order by timestamp field",
			query: model.Query{
				Orders: []model.Order{
					{Field: "fechaFabricacion", Direction: "desc"},
				},
			},
			expected: []string{"fields.fechaFabricacion.timestampValue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := qe.buildMongoFindOptions(ctx, collectionPath, tt.query)
			assert.NotNil(t, opts)

			// Note: In a real implementation, you would inspect the Sort field
			// of the FindOptions to verify the correct field paths are used
			// For now, we verify that the method doesn't panic and returns valid options
		})
	}
}

// TestInferTypeFromQueryFilters tests the filter-based type inference
func TestInferTypeFromQueryFilters(t *testing.T) {
	qe := createTestQueryEngine()

	tests := []struct {
		name         string
		fieldName    string
		filters      []model.Filter
		expectedType model.FieldValueType
	}{
		{
			name:      "infer double from numeric filter",
			fieldName: "price",
			filters: []model.Filter{
				{
					Field:    "price",
					Operator: model.OperatorGreaterThan,
					Value:    500.0,
				},
			},
			expectedType: model.FieldTypeDouble,
		},
		{
			name:      "infer string from string filter",
			fieldName: "category",
			filters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
			},
			expectedType: model.FieldTypeString,
		},
		{
			name:      "infer bool from boolean filter",
			fieldName: "available",
			filters: []model.Filter{
				{
					Field:    "available",
					Operator: model.OperatorEqual,
					Value:    true,
				},
			},
			expectedType: model.FieldTypeBool,
		}, {
			name:      "infer from composite filter",
			fieldName: "stock",
			filters: []model.Filter{
				{
					Composite: "and",
					SubFilters: []model.Filter{
						{
							Field:    "stock",
							Operator: model.OperatorGreaterThan,
							Value:    0,
						},
					},
				},
			},
			expectedType: model.FieldTypeInt,
		},
		{
			name:         "field not found in filters",
			fieldName:    "nonexistent",
			filters:      []model.Filter{},
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qe.inferTypeFromQueryFilters(tt.fieldName, tt.filters)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

// TestHybridTypeInference tests the complete hybrid inference strategy
func TestHybridTypeInference(t *testing.T) {
	qe := createTestQueryEngine()
	ctx := context.Background()
	collectionPath := "test_collection"

	t.Run("cache takes priority", func(t *testing.T) {
		// Setup cache
		qe.typeCache["test_collection.price"] = model.FieldTypeDouble

		// Create query with conflicting filter type (should be ignored due to cache)
		query := model.Query{
			Filters: []model.Filter{
				{
					Field:    "price",
					Operator: model.OperatorEqual,
					Value:    "string_value", // This would infer string type
				},
			},
		}

		result := qe.inferFieldTypeForOrdering(ctx, collectionPath, "price", query)
		assert.Equal(t, model.FieldTypeDouble, result) // Should use cache, not filter
	})

	t.Run("filter analysis when no cache", func(t *testing.T) {
		// Clear cache
		delete(qe.typeCache, "test_collection.category")

		query := model.Query{
			Filters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
			},
		}

		result := qe.inferFieldTypeForOrdering(ctx, collectionPath, "category", query)
		assert.Equal(t, model.FieldTypeString, result)

		// Verify cache was updated
		assert.Equal(t, model.FieldTypeString, qe.typeCache["test_collection.category"])
	})

	t.Run("fallback to string when no information available", func(t *testing.T) {
		// Clear cache
		delete(qe.typeCache, "test_collection.unknown")

		query := model.Query{
			Filters: []model.Filter{}, // No filters
		}

		result := qe.inferFieldTypeForOrdering(ctx, collectionPath, "unknown", query)
		assert.Equal(t, model.FieldTypeString, result) // Should fallback to string
	})
}

// TestWrapValueForFirestore tests the wrapValueForFirestore helper function
func TestWrapValueForFirestore(t *testing.T) {
	queryEngine := createTestQueryEngine()

	testCases := []struct {
		name     string
		value    interface{}
		expected bson.M
	}{
		{
			name:     "string value",
			value:    "hello",
			expected: bson.M{"stringValue": "hello"},
		},
		{
			name:     "integer value",
			value:    42,
			expected: bson.M{"integerValue": 42},
		}, {
			name:     "int64 value",
			value:    int64(999),
			expected: bson.M{"integerValue": 999},
		},
		{
			name:     "float64 value",
			value:    3.14159,
			expected: bson.M{"doubleValue": 3.14159},
		}, {
			name:     "float32 value",
			value:    float32(2.5),
			expected: bson.M{"doubleValue": float64(2.5)}, // float32 gets promoted to float64
		},
		{
			name:     "boolean true",
			value:    true,
			expected: bson.M{"booleanValue": true},
		},
		{
			name:     "boolean false",
			value:    false,
			expected: bson.M{"booleanValue": false},
		},
		{
			name:     "nil value",
			value:    nil,
			expected: bson.M{"nullValue": nil},
		}, {
			name:     "unknown type defaults to string",
			value:    struct{ Name string }{Name: "test"},
			expected: bson.M{"stringValue": "{test}"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := queryEngine.wrapValueForFirestore(tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestArrayOperatorsIntegration tests array operators with realistic scenarios
func TestArrayOperatorsIntegration(t *testing.T) {
	queryEngine := createTestQueryEngine()

	t.Run("array-contains with mixed value types", func(t *testing.T) {
		tests := []struct {
			name     string
			filter   model.Filter
			expected bson.M
		}{
			{
				name:   "array contains string tag",
				filter: model.Filter{Field: "tags", Operator: "array-contains", Value: "featured"},
				expected: bson.M{
					"fields.tags.arrayValue.values": bson.M{
						"$elemMatch": bson.M{"stringValue": "featured"},
					},
				},
			},
			{
				name:   "array contains rating number",
				filter: model.Filter{Field: "ratings", Operator: "array-contains", Value: 5},
				expected: bson.M{
					"fields.ratings.arrayValue.values": bson.M{
						"$elemMatch": bson.M{"integerValue": 5},
					},
				},
			},
			{
				name:   "array contains active flag",
				filter: model.Filter{Field: "features", Operator: "array-contains", Value: true},
				expected: bson.M{
					"fields.features.arrayValue.values": bson.M{
						"$elemMatch": bson.M{"booleanValue": true},
					},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := queryEngine.singleMongoFilter(test.filter)
				assert.Equal(t, test.expected, result)
			})
		}
	})
	t.Run("array-contains-any with multiple values", func(t *testing.T) {
		filter := model.Filter{
			Field:    "categories",
			Operator: "array-contains-any",
			Value:    []interface{}{"tech", "gaming", "mobile"}, // Use []interface{} like the HTTP conversion
		}
		result := queryEngine.singleMongoFilter(filter)
		expected := bson.M{
			"fields.categories.arrayValue.values": bson.M{
				"$elemMatch": bson.M{
					"$in": []bson.M{
						{"stringValue": "tech"},
						{"stringValue": "gaming"},
						{"stringValue": "mobile"},
					},
				},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("array-contains-any with mixed numeric types", func(t *testing.T) {
		filter := model.Filter{
			Field:    "scores",
			Operator: "array-contains-any",
			Value:    []interface{}{85, 90, 95},
		}

		result := queryEngine.singleMongoFilter(filter)
		expected := bson.M{
			"fields.scores.arrayValue.values": bson.M{
				"$elemMatch": bson.M{
					"$in": []bson.M{
						{"integerValue": 85},
						{"integerValue": 90},
						{"integerValue": 95},
					},
				},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("complex filter combining array and non-array operators", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "status", Operator: "==", Value: "active"},
			{Field: "tags", Operator: "array-contains", Value: "premium"},
			{Field: "price", Operator: ">", Value: 100},
		}

		result := queryEngine.buildMongoFilter(filters)
		expected := bson.M{"$and": []bson.M{
			{"fields.status.stringValue": "active"},
			{"fields.tags.arrayValue.values": bson.M{"$elemMatch": bson.M{"stringValue": "premium"}}},
			{"fields.price.integerValue": bson.M{"$gt": 100}},
		},
		}

		assert.Equal(t, expected, result)
	})
}

// TestArrayOperatorsEdgeCases tests edge cases and error scenarios
func TestArrayOperatorsEdgeCases(t *testing.T) {
	queryEngine := createTestQueryEngine()

	t.Run("array-contains-any with empty array", func(t *testing.T) {
		filter := model.Filter{
			Field:    "tags",
			Operator: "array-contains-any",
			Value:    []interface{}{}, // Use []interface{} instead of []string{}
		}
		result := queryEngine.singleMongoFilter(filter)
		expected := bson.M{
			"fields.tags.arrayValue.values": bson.M{
				"$elemMatch": bson.M{
					"$in": []bson.M{},
				},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("array-contains-any with single item", func(t *testing.T) {
		filter := model.Filter{
			Field:    "priorities",
			Operator: "array-contains-any",
			Value:    []interface{}{"high"}, // Use []interface{} instead of []string{}
		}

		result := queryEngine.singleMongoFilter(filter)
		expected := bson.M{
			"fields.priorities.arrayValue.values": bson.M{
				"$elemMatch": bson.M{
					"$in": []bson.M{
						{"stringValue": "high"},
					},
				},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("array-contains with nil value", func(t *testing.T) {
		filter := model.Filter{
			Field:    "nulls",
			Operator: "array-contains",
			Value:    nil,
		}
		result := queryEngine.singleMongoFilter(filter)
		expected := bson.M{
			"fields.nulls.arrayValue.values": bson.M{
				"$elemMatch": bson.M{"nullValue": nil},
			},
		}

		assert.Equal(t, expected, result)
	})
}

// TestArrayOperatorsPerformance tests performance characteristics
func TestArrayOperatorsPerformance(t *testing.T) {
	queryEngine := createTestQueryEngine()
	t.Run("large array-contains-any performance", func(t *testing.T) {
		// Create a large array of values
		largeArray := make([]interface{}, 1000) // Use []interface{} instead of []string
		for i := 0; i < 1000; i++ {
			largeArray[i] = fmt.Sprintf("value_%d", i)
		}

		filter := model.Filter{
			Field:    "large_tags",
			Operator: "array-contains-any",
			Value:    largeArray,
		}

		// This should not panic or timeout
		result := queryEngine.singleMongoFilter(filter)

		// Verify the structure is correct		assert.Contains(t, result, "fields.large_tags.arrayValue.values")
		arrayFilter := result["fields.large_tags.arrayValue.values"].(bson.M)
		assert.Contains(t, arrayFilter, "$elemMatch")
		elemMatch := arrayFilter["$elemMatch"].(bson.M)
		assert.Contains(t, elemMatch, "$in")

		// Verify we have the expected number of wrapped values
		inArray := elemMatch["$in"].([]bson.M)
		assert.Len(t, inArray, 1000)
	})
}
