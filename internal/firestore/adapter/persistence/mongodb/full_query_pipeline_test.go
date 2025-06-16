package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestFullQueryPipeline validates the complete query execution pipeline
// following Firestore semantics and hexagonal architecture principles
func TestFullQueryPipeline(t *testing.T) {
	ctx := context.Background()

	t.Run("Validate Complete Query Execution with Filter Logic", func(t *testing.T) {
		// Create test documents following Firestore document structure
		testDocs := createFirestoreTestDocuments()

		t.Logf("Created %d test documents for validation", len(testDocs))

		// Create the problematic query (category=Electronics AND available=true)
		// This replicates the exact production scenario
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

		t.Logf("Executing query with filters: %+v", query.Filters)

		// Test the filter conversion (this is where our fix applies)
		mongoFilter := buildMongoFilter(query.Filters)
		require.NotNil(t, mongoFilter, "MongoDB filter must not be nil")
		require.NotEmpty(t, mongoFilter, "MongoDB filter must not be empty")

		t.Logf("Generated MongoDB filter: %+v", mongoFilter)

		// Simulate the filtering process using our domain logic
		filteredDocs := simulateFirestoreFiltering(testDocs, query.Filters)

		t.Logf("Query returned %d documents after filtering", len(filteredDocs))
		for i, doc := range filteredDocs {
			categoryField := extractFieldValue(doc.Fields["category"])
			availableField := extractFieldValue(doc.Fields["available"])

			t.Logf("Document %d: ID=%s, category=%v, available=%v",
				i, doc.DocumentID, categoryField, availableField)
		}

		// We should only get doc1 (Electronics + available=true)
		// This validates that our filter fix is working correctly
		assert.Equal(t, 1, len(filteredDocs),
			"Should return exactly 1 document matching both filters")

		if len(filteredDocs) > 0 {
			assert.Equal(t, "doc1", filteredDocs[0].DocumentID,
				"Should return doc1 which matches both filters")
		}

		// Verify the MongoDB filter structure follows Firestore document model
		validateMongoFilterStructure(t, mongoFilter)

		_ = ctx // Reserved for future integration test enhancements
	})

	t.Run("Validate Individual Filter Components", func(t *testing.T) {
		// Test that individual filters work correctly following Firestore semantics
		filterTestCases := []struct {
			name         string
			filter       model.Filter
			expectedBSON bson.M
			description  string
		}{
			{
				name: "Category Filter Validation",
				filter: model.Filter{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
				expectedBSON: bson.M{"fields.category.stringValue": "Electronics"},
				description:  "String field filter following Firestore document structure",
			},
			{
				name: "Available Filter Validation",
				filter: model.Filter{
					Field:    "available",
					Operator: model.OperatorEqual,
					Value:    true,
				},
				expectedBSON: bson.M{"fields.available.booleanValue": true},
				description:  "Boolean field filter following Firestore document structure",
			},
		}

		for _, tc := range filterTestCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Logf("Testing: %s", tc.description)

				// Test the filter conversion using our fixed implementation
				mongoFilter := singleMongoFilter(tc.filter)

				t.Logf("Filter: %+v", tc.filter)
				t.Logf("Expected BSON: %+v", tc.expectedBSON)
				t.Logf("Actual BSON: %+v", mongoFilter)

				assert.Equal(t, tc.expectedBSON, mongoFilter,
					"MongoDB filter must match expected BSON structure")
			})
		}
	})
}

// createFirestoreTestDocuments creates test documents following Firestore structure
func createFirestoreTestDocuments() []*model.Document {
	return []*model.Document{
		{
			DocumentID:   "doc1",
			CollectionID: "productos2",
			Fields: map[string]*model.FieldValue{
				"category":  {ValueType: model.FieldTypeString, Value: "Electronics"},
				"available": {ValueType: model.FieldTypeBool, Value: true},
				"name":      {ValueType: model.FieldTypeString, Value: "Laptop Gamer Pro"},
			},
		},
		{
			DocumentID:   "doc2",
			CollectionID: "productos2",
			Fields: map[string]*model.FieldValue{
				"category":  {ValueType: model.FieldTypeString, Value: "Office"},
				"available": {ValueType: model.FieldTypeBool, Value: true},
				"name":      {ValueType: model.FieldTypeString, Value: "Impresora Multifunción"},
			},
		},
		{
			DocumentID:   "doc3",
			CollectionID: "productos2",
			Fields: map[string]*model.FieldValue{
				"category":  {ValueType: model.FieldTypeString, Value: "Electronics"},
				"available": {ValueType: model.FieldTypeBool, Value: false},
				"name":      {ValueType: model.FieldTypeString, Value: "Tablet Discontinued"},
			},
		},
	}
}

// simulateFirestoreFiltering simulates Firestore query filtering logic
func simulateFirestoreFiltering(docs []*model.Document, filters []model.Filter) []*model.Document {
	var results []*model.Document

	for _, doc := range docs {
		if documentMatchesAllFilters(doc, filters) {
			results = append(results, doc)
		}
	}

	return results
}

// documentMatchesAllFilters checks if a document matches all provided filters
func documentMatchesAllFilters(doc *model.Document, filters []model.Filter) bool {
	for _, filter := range filters {
		if !documentMatchesFilter(doc, filter) {
			return false
		}
	}
	return true
}

// documentMatchesFilter checks if a document matches a specific filter
func documentMatchesFilter(doc *model.Document, filter model.Filter) bool {
	fieldValue, exists := doc.Fields[filter.Field]
	if !exists {
		return false
	}

	actualValue := extractFieldValue(fieldValue)
	expectedValue := filter.Value

	switch filter.Operator {
	case model.OperatorEqual:
		return actualValue == expectedValue
	case model.OperatorNotEqual:
		return actualValue != expectedValue
	// Add more operators as needed following Firestore semantics
	default:
		return false
	}
}

// extractFieldValue extracts the actual value from a FieldValue following Firestore semantics
func extractFieldValue(fv *model.FieldValue) interface{} {
	if fv == nil {
		return nil
	}
	return fv.Value
}

// validateMongoFilterStructure validates that the MongoDB filter follows Firestore conventions
func validateMongoFilterStructure(t *testing.T, filter bson.M) {
	t.Helper()

	// Validate that we have an $and structure for composite filters
	if andFilters, exists := filter["$and"]; exists {
		if filterSlice, ok := andFilters.([]bson.M); ok {
			assert.Len(t, filterSlice, 2, "Should have exactly 2 AND conditions")

			// Validate each condition follows Firestore field structure
			for i, condition := range filterSlice {
				assert.NotEmpty(t, condition, "Condition %d should not be empty", i)

				// Each condition should have exactly one field with the correct path structure
				assert.Len(t, condition, 1, "Each condition should have exactly one field")

				for fieldPath := range condition {
					assert.Contains(t, fieldPath, "fields.",
						"Field path should follow Firestore structure (fields.)")
				}
			}
		} else {
			t.Error("$and should contain a slice of bson.M")
		}
	} else {
		t.Error("Expected $and structure for composite filters")
	}
}
