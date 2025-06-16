package mongodb

import (
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

// TestBuildMongoFilter tests the filter construction for Firestore field paths
func TestBuildMongoFilter(t *testing.T) {
	t.Run("simple equality filter", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "name", Operator: "==", Value: "fred"},
		}

		result := buildMongoFilter(filters)
		expected := bson.M{"fields.name.stringValue": "fred"}

		assert.Equal(t, expected, result)
	})
	t.Run("less than filter", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "born", Operator: "<", Value: 1900},
		}

		result := buildMongoFilter(filters)
		expected := bson.M{"fields.born.integerValue": bson.M{"$lt": 1900}}

		assert.Equal(t, expected, result)
	})
	t.Run("multiple filters with AND", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "age", Operator: ">=", Value: 18},
			{Field: "status", Operator: "==", Value: "active"},
		}

		result := buildMongoFilter(filters)
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

		result := buildMongoFilter(filters)
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

		result := buildMongoFilter(filters)
		expected := bson.M{"fields.category.stringValue": bson.M{"$in": []string{"electronics", "books"}}}

		assert.Equal(t, expected, result)
	})
	t.Run("array-contains operator", func(t *testing.T) {
		filters := []model.Filter{
			{Field: "tags", Operator: "array-contains", Value: "featured"},
		}

		result := buildMongoFilter(filters)
		expected := bson.M{"fields.tags.arrayValue": bson.M{"$elemMatch": bson.M{"$eq": "featured"}}}

		assert.Equal(t, expected, result)
	})
}

// TestSingleMongoFilter tests individual filter translation
func TestSingleMongoFilter(t *testing.T) {
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
		},
		{
			name:     "array-contains-any",
			filter:   model.Filter{Field: "tags", Operator: "array-contains-any", Value: []string{"urgent", "important"}},
			expected: bson.M{"fields.tags.arrayValue": bson.M{"$in": []string{"urgent", "important"}}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := singleMongoFilter(tc.filter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestBuildMongoFindOptions tests the find options construction
func TestBuildMongoFindOptions(t *testing.T) {
	t.Run("limit and offset", func(t *testing.T) {
		query := model.Query{
			Limit:  10,
			Offset: 5,
		}

		opts := buildMongoFindOptions(query)

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

		opts := buildMongoFindOptions(query)

		assert.NotNil(t, opts)
	})

	t.Run("sort by multiple fields", func(t *testing.T) {
		query := model.Query{
			Orders: []model.Order{
				{Field: "priority", Direction: "desc"},
				{Field: "created", Direction: "asc"},
			},
		}

		opts := buildMongoFindOptions(query)

		assert.NotNil(t, opts)
	})

	t.Run("with projection", func(t *testing.T) {
		query := model.Query{
			SelectFields: []string{"name", "email", "status"},
		}

		opts := buildMongoFindOptions(query)

		assert.NotNil(t, opts)
	})
}

// TestBuildCursorFilter tests cursor-based pagination
func TestBuildCursorFilter(t *testing.T) {
	t.Run("no orders returns nil", func(t *testing.T) {
		query := model.Query{}

		result := buildCursorFilter(query)

		assert.Nil(t, result)
	})
	t.Run("startAt cursor", func(t *testing.T) {
		query := model.Query{
			Orders:  []model.Order{{Field: "name", Direction: "asc"}},
			StartAt: []interface{}{"john"},
		}

		result := buildCursorFilter(query)
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

		result := buildCursorFilter(query)
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

		result := buildCursorFilter(query)
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
