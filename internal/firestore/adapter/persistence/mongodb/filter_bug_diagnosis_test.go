package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestFilterBugDiagnosis validates the specific filter conversion issue
// following Firestore's query semantics and hexagonal architecture principles
func TestFilterBugDiagnosis(t *testing.T) {
	ctx := context.Background()

	// Create query engine instance for testing, following hexagonal architecture
	queryEngine := NewMongoQueryEngine(nil) // MongoDB connection not needed for filter logic testing

	t.Run("Diagnose Category AND Available Filter Bug", func(t *testing.T) {
		// This reproduces the exact query that's failing in production
		// Following Firestore's composite filter structure
		query := model.Query{
			CollectionID: "productos2",
			Filters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
				{
					Field:    "available",
					Operator: model.OperatorEqual,
					Value:    true,
				},
			},
		}

		t.Logf("Query filters: %+v", query.Filters)

		// Test filter conversion using internal MongoDB adapter
		mongoFilter := queryEngine.buildMongoFilter(query.Filters)
		require.NotNil(t, mongoFilter, "MongoDB filter must not be nil")

		t.Logf("Generated MongoDB filter: %+v", mongoFilter)

		// Expected MongoDB filter should follow Firestore document structure
		expectedFilter := bson.M{
			"$and": []bson.M{
				{"fields.category.stringValue": "Electronics"},
				{"fields.available.booleanValue": true},
			},
		}

		// Validate BSON marshaling for debugging
		actualBson, err := bson.Marshal(mongoFilter)
		require.NoError(t, err, "MongoDB filter must be valid BSON")

		expectedBson, err := bson.Marshal(expectedFilter)
		require.NoError(t, err, "Expected filter must be valid BSON")

		t.Logf("Expected BSON: %s", string(expectedBson))
		t.Logf("Actual BSON: %s", string(actualBson))

		// Validate filter is properly constructed
		assert.NotEmpty(t, mongoFilter, "MongoDB filter should not be empty")
		// Test individual filter conversion following single responsibility principle
		for i, filter := range query.Filters {
			t.Logf("Testing filter %d: Field=%s, Operator=%s, Value=%v",
				i, filter.Field, filter.Operator, filter.Value)

			individualFilter := queryEngine.SingleMongoFilter(filter)
			require.NotEmpty(t, individualFilter, "Individual filter should not be empty")

			t.Logf("Individual MongoDB filter %d: %+v", i, individualFilter)
		}

		_ = ctx // Prepare for future integration tests
	})
	t.Run("Validate Operator Mapping Contract", func(t *testing.T) {
		// Test that our model operators comply with Firestore API standards
		operatorTestCases := []struct {
			name           string
			modelOperator  model.Operator
			expectedString string
		}{
			{"Equal", model.OperatorEqual, "=="},
			{"NotEqual", model.OperatorNotEqual, "!="},
			{"LessThan", model.OperatorLessThan, "<"},
			{"LessThanOrEqual", model.OperatorLessThanOrEqual, "<="},
			{"GreaterThan", model.OperatorGreaterThan, ">"},
			{"GreaterThanOrEqual", model.OperatorGreaterThanOrEqual, ">="},
		}

		for _, tc := range operatorTestCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Logf("Validating operator mapping: %s -> %s", tc.modelOperator, tc.expectedString)
				assert.Equal(t, tc.expectedString, string(tc.modelOperator),
					"Operator mapping must match Firestore API specification")
			})
		}
	})
	t.Run("Validate Field Path Resolution", func(t *testing.T) {
		// Test field path resolution following Firestore document structure using the FieldPathResolver
		fieldPathTestCases := []struct {
			field    string
			value    interface{}
			expected string
		}{
			{"category", "Electronics", "fields.category.stringValue"},
			{"available", true, "fields.available.booleanValue"},
			{"price", 99.99, "fields.price.doubleValue"},
			{"stock", int64(50), "fields.stock.integerValue"},
		}

		for _, tc := range fieldPathTestCases {
			t.Run(tc.field, func(t *testing.T) {
				// Create a FieldPath and determine value type using the new approach
				fieldPath, err := model.NewFieldPath(tc.field)
				require.NoError(t, err, "Field path creation should not fail")

				valueType := model.DetermineValueType(tc.value)

				// Use the FieldPathResolver to get the MongoDB path
				mongoPath, err := queryEngine.fieldPathResolver.ResolveFieldPath(fieldPath, valueType)
				require.NoError(t, err, "Field path resolution should not fail")
				t.Logf("Field: %s, Value: %v (%T), ValueType: %s -> Path: %s",
					tc.field, tc.value, tc.value, valueType, mongoPath)
				assert.Equal(t, tc.expected, mongoPath,
					"Field path must follow Firestore document structure")
			})
		}
	})

	t.Run("Simulate Real Filter Problem", func(t *testing.T) {
		// Create the exact filter that should work following Firestore semantics
		filter := model.Filter{
			Field:    "category",
			Operator: model.OperatorEqual,
			Value:    "Electronics",
		}

		t.Logf("Original filter: %+v", filter)

		// Convert using our adapter function
		mongoFilter := queryEngine.SingleMongoFilter(filter)
		t.Logf("Converted MongoDB filter: %+v", mongoFilter)

		// Validate filter is not empty (this would indicate our bug)
		require.NotEmpty(t, mongoFilter,
			"CRITICAL BUG: MongoDB filter is empty! This explains why no filters are being applied.")

		// Also test with boolean value following Firestore type system
		boolFilter := model.Filter{
			Field:    "available",
			Operator: model.OperatorEqual,
			Value:    true,
		}

		t.Logf("Boolean filter: %+v", boolFilter)

		mongoBoolFilter := queryEngine.SingleMongoFilter(boolFilter)
		t.Logf("Converted boolean MongoDB filter: %+v", mongoBoolFilter)

		require.NotEmpty(t, mongoBoolFilter,
			"CRITICAL BUG: Boolean MongoDB filter is empty!")
	})
}

// TestFirestoreProductionQueryReplication validates the exact query structure from the failing request
// This test replicates the production scenario to identify the root cause
func TestFirestoreProductionQueryReplication(t *testing.T) {
	ctx := context.Background()

	// Create query engine instance for testing, following hexagonal architecture
	queryEngine := NewMongoQueryEngine(nil) // MongoDB connection not needed for filter logic testing

	t.Run("Replicate Exact Production Query", func(t *testing.T) {
		// This simulates the exact query structure from the JSON request body
		// Following Firestore's composite filter specification
		query := model.Query{
			CollectionID: "productos2",
			Filters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
				{
					Field:    "available",
					Operator: model.OperatorEqual,
					Value:    true,
				},
			},
		}

		t.Logf("=== TESTING EXACT PRODUCTION QUERY ===")
		t.Logf("Query: %+v", query)
		t.Logf("Number of filters: %d", len(query.Filters))

		// Build the MongoDB filter using the adapter
		mongoFilter := queryEngine.buildMongoFilter(query.Filters)
		require.NotNil(t, mongoFilter,
			"CRITICAL BUG: MongoDB filter is nil! This is why all documents are being returned.")
		require.NotEmpty(t, mongoFilter,
			"CRITICAL BUG: MongoDB filter is empty! This is why all documents are being returned.")

		t.Logf("Generated MongoDB filter: %+v", mongoFilter)

		// Convert to BSON for inspection and validation
		bsonBytes, err := bson.Marshal(mongoFilter)
		require.NoError(t, err, "MongoDB filter must be valid BSON")

		t.Logf("BSON representation: %s", string(bsonBytes))

		// Validate the filter structure follows Firestore document model
		// Expected result should filter to only Electronics that are available
		// If this test passes but the server still returns all docs,
		// the bug is in the database query execution, not filter conversion

		// Additional validation: ensure filter has the correct structure
		if andFilter, ok := mongoFilter["$and"]; ok {
			if andSlice, ok := andFilter.([]bson.M); ok {
				assert.Len(t, andSlice, 2, "Should have exactly 2 AND conditions")

				// Validate each condition follows Firestore field structure
				for i, condition := range andSlice {
					assert.NotEmpty(t, condition, "Condition %d should not be empty", i)
					t.Logf("Condition %d: %+v", i, condition)
				}
			}
		}

		_ = ctx // Reserved for future integration test enhancements
	})
}
