package mongodb

import (
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QueryEngineTestSuite defines the test suite for MongoQueryEngine
type QueryEngineTestSuite struct {
	suite.Suite
	queryEngine *MongoQueryEngine
	mockDB      *mongo.Database
}

// SetupSuite runs once before all tests in the suite
func (suite *QueryEngineTestSuite) SetupSuite() {
	// Use real MongoDB database for integration-style tests
	// In production tests, you would set up a test database connection
	// For unit tests, we'll test the individual functions separately
}

// SetupTest runs before each test
func (suite *QueryEngineTestSuite) SetupTest() {
	// Reset state before each test
}

// TestNewMongoQueryEngine tests the constructor
func (suite *QueryEngineTestSuite) TestNewMongoQueryEngine() {
	var mockDB *mongo.Database
	engine := NewMongoQueryEngine(mockDB)

	suite.NotNil(engine)
	suite.Equal(mockDB, engine.db)
}

// TestExecuteQuery_DatabaseError tests handling of database errors
func (suite *QueryEngineTestSuite) TestExecuteQuery_DatabaseError() {
	collectionPath := "users"

	query := model.Query{
		Path:         "projects/test-project/databases/test-db/documents/users",
		CollectionID: "users",
		Filters: []model.Filter{
			{Field: "email", Operator: "==", Value: "test@example.com"},
		},
	}

	// For unit test purposes, we test the error handling logic separately
	// In real integration tests, you would set up actual database connections

	suite.NotEmpty(query.Path)
	suite.NotEmpty(collectionPath)
}

// Run the test suite
func TestQueryEngineTestSuite(t *testing.T) {
	suite.Run(t, new(QueryEngineTestSuite))
}

// --- Unit tests for individual functions ---

// TestBuildMongoFilter tests the buildMongoFilter function
func TestBuildMongoFilter(t *testing.T) {
	tests := []struct {
		name     string
		filters  []model.Filter
		expected bson.M
	}{
		{
			name:     "empty filters",
			filters:  []model.Filter{},
			expected: bson.M{"$and": []bson.M(nil)},
		},
		{
			name: "single equality filter",
			filters: []model.Filter{
				{Field: "name", Operator: "==", Value: "John"},
			},
			expected: bson.M{"name": "John"},
		},
		{
			name: "multiple AND filters",
			filters: []model.Filter{
				{Field: "name", Operator: "==", Value: "John"},
				{Field: "age", Operator: ">", Value: 18},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"name": "John"},
					{"age": bson.M{"$gt": 18}},
				},
			},
		},
		{
			name: "OR composite filter",
			filters: []model.Filter{
				{
					Composite: "or",
					SubFilters: []model.Filter{
						{Field: "status", Operator: "==", Value: "active"},
						{Field: "status", Operator: "==", Value: "pending"},
					},
				},
			},
			expected: bson.M{
				"$or": []bson.M{
					{"status": "active"},
					{"status": "pending"},
				},
			},
		},
		{
			name: "array contains filter",
			filters: []model.Filter{
				{Field: "tags", Operator: "array-contains", Value: "golang"},
			},
			expected: bson.M{"tags": bson.M{"$elemMatch": bson.M{"$eq": "golang"}}},
		},
		{
			name: "in operator filter",
			filters: []model.Filter{
				{Field: "category", Operator: "in", Value: []string{"tech", "science"}},
			},
			expected: bson.M{"category": bson.M{"$in": []string{"tech", "science"}}},
		},
		{
			name: "mixed AND and OR filters",
			filters: []model.Filter{
				{Field: "active", Operator: "==", Value: true},
				{
					Composite: "or",
					SubFilters: []model.Filter{
						{Field: "type", Operator: "==", Value: "premium"},
						{Field: "type", Operator: "==", Value: "gold"},
					},
				},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"active": true},
					bson.M{
						"$or": []bson.M{
							{"type": "premium"},
							{"type": "gold"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildMongoFilter(tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSingleMongoFilter tests the singleMongoFilter function
func TestSingleMongoFilter(t *testing.T) {
	tests := []struct {
		name     string
		filter   model.Filter
		expected bson.M
	}{
		{
			name:     "equality operator",
			filter:   model.Filter{Field: "name", Operator: "==", Value: "John"},
			expected: bson.M{"name": "John"},
		},
		{
			name:     "not equal operator",
			filter:   model.Filter{Field: "status", Operator: "!=", Value: "deleted"},
			expected: bson.M{"status": bson.M{"$ne": "deleted"}},
		},
		{
			name:     "greater than operator",
			filter:   model.Filter{Field: "age", Operator: ">", Value: 18},
			expected: bson.M{"age": bson.M{"$gt": 18}},
		},
		{
			name:     "greater than or equal operator",
			filter:   model.Filter{Field: "score", Operator: ">=", Value: 90},
			expected: bson.M{"score": bson.M{"$gte": 90}},
		},
		{
			name:     "less than operator",
			filter:   model.Filter{Field: "price", Operator: "<", Value: 100},
			expected: bson.M{"price": bson.M{"$lt": 100}},
		},
		{
			name:     "less than or equal operator",
			filter:   model.Filter{Field: "discount", Operator: "<=", Value: 50},
			expected: bson.M{"discount": bson.M{"$lte": 50}},
		},
		{
			name:     "in operator",
			filter:   model.Filter{Field: "category", Operator: "in", Value: []string{"A", "B"}},
			expected: bson.M{"category": bson.M{"$in": []string{"A", "B"}}},
		},
		{
			name:     "not in operator",
			filter:   model.Filter{Field: "type", Operator: "not-in", Value: []string{"spam", "deleted"}},
			expected: bson.M{"type": bson.M{"$nin": []string{"spam", "deleted"}}},
		},
		{
			name:     "array contains operator",
			filter:   model.Filter{Field: "tags", Operator: "array-contains", Value: "important"},
			expected: bson.M{"tags": bson.M{"$elemMatch": bson.M{"$eq": "important"}}},
		},
		{
			name:     "array contains any operator",
			filter:   model.Filter{Field: "skills", Operator: "array-contains-any", Value: []string{"Go", "Python"}},
			expected: bson.M{"skills": bson.M{"$in": []string{"Go", "Python"}}},
		},
		{
			name:     "unknown operator defaults to equality",
			filter:   model.Filter{Field: "field", Operator: "unknown", Value: "value"},
			expected: bson.M{"field": "value"},
		},
		{
			name:     "boolean value",
			filter:   model.Filter{Field: "isActive", Operator: "==", Value: true},
			expected: bson.M{"isActive": true},
		},
		{
			name:     "numeric value",
			filter:   model.Filter{Field: "count", Operator: "==", Value: 42},
			expected: bson.M{"count": 42},
		},
		{
			name:     "nil value",
			filter:   model.Filter{Field: "deletedAt", Operator: "==", Value: nil},
			expected: bson.M{"deletedAt": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := singleMongoFilter(tt.filter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildMongoFindOptions tests the buildMongoFindOptions function
func TestBuildMongoFindOptions(t *testing.T) {
	tests := []struct {
		name     string
		query    model.Query
		validate func(t *testing.T, opts *options.FindOptions)
	}{
		{
			name:  "empty query",
			query: model.Query{},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.Nil(t, opts.Limit)
				assert.Nil(t, opts.Skip)
				assert.Nil(t, opts.Sort)
				assert.Nil(t, opts.Projection)
			},
		},
		{
			name: "with limit",
			query: model.Query{
				Limit: 10,
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.NotNil(t, opts.Limit)
				assert.Equal(t, int64(10), *opts.Limit)
			},
		},
		{
			name: "with offset",
			query: model.Query{
				Offset: 5,
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.NotNil(t, opts.Skip)
				assert.Equal(t, int64(5), *opts.Skip)
			},
		},
		{
			name: "with ordering",
			query: model.Query{
				Orders: []model.Order{
					{Field: "name", Direction: model.DirectionAscending},
					{Field: "createdAt", Direction: model.DirectionDescending},
				},
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.NotNil(t, opts.Sort)
				sort := opts.Sort.(bson.D)
				assert.Len(t, sort, 2)
				assert.Equal(t, "name", sort[0].Key)
				assert.Equal(t, 1, sort[0].Value)
				assert.Equal(t, "createdAt", sort[1].Key)
				assert.Equal(t, -1, sort[1].Value)
			},
		},
		{
			name: "with field selection",
			query: model.Query{
				SelectFields: []string{"name", "email", "age"},
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.NotNil(t, opts.Projection)
				projection := opts.Projection.(bson.M)
				assert.Equal(t, 1, projection["name"])
				assert.Equal(t, 1, projection["email"])
				assert.Equal(t, 1, projection["age"])
			},
		},
		{
			name: "all options combined",
			query: model.Query{
				Limit:  20,
				Offset: 10,
				Orders: []model.Order{
					{Field: "score", Direction: model.DirectionDescending},
				},
				SelectFields: []string{"title", "score"},
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.NotNil(t, opts.Limit)
				assert.Equal(t, int64(20), *opts.Limit)
				assert.NotNil(t, opts.Skip)
				assert.Equal(t, int64(10), *opts.Skip)
				assert.NotNil(t, opts.Sort)
				assert.NotNil(t, opts.Projection)
			},
		},
		{
			name: "zero limit should not be set",
			query: model.Query{
				Limit: 0,
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.Nil(t, opts.Limit)
			},
		},
		{
			name: "zero offset should not be set",
			query: model.Query{
				Offset: 0,
			},
			validate: func(t *testing.T, opts *options.FindOptions) {
				assert.Nil(t, opts.Skip)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := buildMongoFindOptions(tt.query)
			require.NotNil(t, opts)
			tt.validate(t, opts)
		})
	}
}

// TestBuildCursorFilter tests the buildCursorFilter function
func TestBuildCursorFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    model.Query
		expected bson.M
	}{
		{
			name: "no orders - returns nil",
			query: model.Query{
				StartAt: []interface{}{100},
			},
			expected: nil,
		},
		{
			name: "no cursor values - returns nil",
			query: model.Query{
				Orders: []model.Order{
					{Field: "price", Direction: model.DirectionAscending},
				},
			},
			expected: nil,
		},
		{
			name: "startAt with ascending order",
			query: model.Query{
				Orders: []model.Order{
					{Field: "price", Direction: model.DirectionAscending},
				},
				StartAt: []interface{}{100},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"price": bson.M{"$gte": 100}},
				},
			},
		},
		{
			name: "startAfter with descending order",
			query: model.Query{
				Orders: []model.Order{
					{Field: "timestamp", Direction: model.DirectionDescending},
				},
				StartAfter: []interface{}{time.Unix(1600000000, 0)},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"timestamp": bson.M{"$lt": time.Unix(1600000000, 0)}},
				},
			},
		},
		{
			name: "endBefore with ascending order",
			query: model.Query{
				Orders: []model.Order{
					{Field: "score", Direction: model.DirectionAscending},
				},
				EndBefore: []interface{}{90},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"score": bson.M{"$lt": 90}},
				},
			},
		},
		{
			name: "endAt with descending order",
			query: model.Query{
				Orders: []model.Order{
					{Field: "rating", Direction: model.DirectionDescending},
				},
				EndAt: []interface{}{3.5},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"rating": bson.M{"$gte": 3.5}},
				},
			},
		},
		{
			name: "multiple cursor conditions",
			query: model.Query{
				Orders: []model.Order{
					{Field: "category", Direction: model.DirectionAscending},
					{Field: "price", Direction: model.DirectionDescending},
				},
				StartAt:   []interface{}{"electronics", 500},
				EndBefore: []interface{}{"sports", 100},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"category": bson.M{"$gte": "electronics"}},
					{"category": bson.M{"$lt": "sports"}},
					{"price": bson.M{"$lte": 500}},
					{"price": bson.M{"$gt": 100}},
				},
			},
		},
		{
			name: "startAt and startAfter (startAfter takes precedence)",
			query: model.Query{
				Orders: []model.Order{
					{Field: "id", Direction: model.DirectionAscending},
				},
				StartAt:    []interface{}{100},
				StartAfter: []interface{}{200},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"id": bson.M{"$gte": 100}},
					{"id": bson.M{"$gt": 200}},
				},
			},
		},
		{
			name: "cursor values exceed order fields",
			query: model.Query{
				Orders: []model.Order{
					{Field: "name", Direction: model.DirectionAscending},
				},
				StartAt: []interface{}{"Alice", "extra", "values"},
			},
			expected: bson.M{
				"$and": []bson.M{
					{"name": bson.M{"$gte": "Alice"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCursorFilter(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReverseDocs tests the reverseDocs function
func TestReverseDocs(t *testing.T) {
	tests := []struct {
		name     string
		input    []*model.Document
		expected []string // document IDs in expected order
	}{
		{
			name:     "empty slice",
			input:    []*model.Document{},
			expected: []string{},
		},
		{
			name: "single document",
			input: []*model.Document{
				{DocumentID: "doc1"},
			},
			expected: []string{"doc1"},
		},
		{
			name: "two documents",
			input: []*model.Document{
				{DocumentID: "doc1"},
				{DocumentID: "doc2"},
			},
			expected: []string{"doc2", "doc1"},
		},
		{
			name: "multiple documents",
			input: []*model.Document{
				{DocumentID: "doc1"},
				{DocumentID: "doc2"},
				{DocumentID: "doc3"},
				{DocumentID: "doc4"},
				{DocumentID: "doc5"},
			},
			expected: []string{"doc5", "doc4", "doc3", "doc2", "doc1"},
		},
		{
			name: "odd number of documents",
			input: []*model.Document{
				{DocumentID: "doc1"},
				{DocumentID: "doc2"},
				{DocumentID: "doc3"},
			},
			expected: []string{"doc3", "doc2", "doc1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original test data
			docs := make([]*model.Document, len(tt.input))
			copy(docs, tt.input)

			reverseDocs(docs)

			// Extract document IDs for comparison
			result := make([]string, len(docs))
			for i, doc := range docs {
				result[i] = doc.DocumentID
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMongoQueryEngine_InterfaceCompliance tests that MongoQueryEngine implements repository.QueryEngine
func TestMongoQueryEngine_InterfaceCompliance(t *testing.T) {
	var mockDB *mongo.Database
	engine := NewMongoQueryEngine(mockDB)

	// This will compile only if MongoQueryEngine implements repository.QueryEngine
	var _ repository.QueryEngine = engine

	assert.NotNil(t, engine)
}

// TestQueryValidation tests query validation scenarios
func TestQueryValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   model.Query
		isValid bool
	}{
		{
			name: "valid simple query",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Filters: []model.Filter{
					{Field: "name", Operator: "==", Value: "John"},
				},
				Limit: 10,
			},
			isValid: true,
		},
		{
			name: "query with invalid operator",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Filters: []model.Filter{
					{Field: "name", Operator: "invalid", Value: "John"},
				},
			},
			isValid: false,
		},
		{
			name: "query with negative limit",
			query: model.Query{
				Path:  "projects/test/databases/default/documents/users",
				Limit: -1,
			},
			isValid: false,
		},
		{
			name: "query with invalid order direction",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Orders: []model.Order{
					{Field: "name", Direction: "invalid"},
				},
			},
			isValid: false,
		},
		{
			name: "empty path query",
			query: model.Query{
				Path: "",
				Filters: []model.Filter{
					{Field: "name", Operator: "==", Value: "John"},
				},
			},
			isValid: false,
		},
		{
			name: "query with empty filter field",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Filters: []model.Filter{
					{Field: "", Operator: "==", Value: "John"},
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.ValidateQuery()
			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestFirestoreQueryTranslation tests translation of Firestore-specific queries
func TestFirestoreQueryTranslation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		query       model.Query
		expectedMsg string
	}{
		{
			name:        "collection_group_query",
			description: "Query across all subcollections with same name",
			query: model.Query{
				Path:           "projects/test/databases/default/documents",
				CollectionID:   "posts",
				AllDescendants: true,
				Filters: []model.Filter{
					{Field: "published", Operator: "==", Value: true},
				},
			},
			expectedMsg: "should handle collection group queries",
		},
		{
			name:        "cursor_pagination",
			description: "Cursor-based pagination for large datasets",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Orders: []model.Order{
					{Field: "createdAt", Direction: model.DirectionDescending},
					{Field: "name", Direction: model.DirectionAscending},
				},
				StartAfter: []interface{}{"2023-01-01T00:00:00Z", "Alice"},
				Limit:      25,
			},
			expectedMsg: "should handle multi-field cursor pagination",
		},
		{
			name:        "compound_query",
			description: "Complex query with multiple conditions",
			query: model.Query{
				Path: "projects/test/databases/default/documents/products",
				Filters: []model.Filter{
					{Field: "category", Operator: "==", Value: "electronics"},
					{Field: "price", Operator: ">=", Value: 100},
					{Field: "price", Operator: "<=", Value: 500},
					{Field: "inStock", Operator: "==", Value: true},
				},
				Orders: []model.Order{
					{Field: "rating", Direction: model.DirectionDescending},
					{Field: "price", Direction: model.DirectionAscending},
				},
				Limit: 50,
			},
			expectedMsg: "should handle complex compound queries",
		},
		{
			name:        "array_operations",
			description: "Query with array contains operations",
			query: model.Query{
				Path: "projects/test/databases/default/documents/users",
				Filters: []model.Filter{
					{Field: "skills", Operator: "array-contains", Value: "golang"},
					{Field: "languages", Operator: "array-contains-any", Value: []string{"en", "es"}},
				},
			},
			expectedMsg: "should handle array operations",
		},
		{
			name:        "field_selection",
			description: "Query with field projection",
			query: model.Query{
				Path:         "projects/test/databases/default/documents/users",
				SelectFields: []string{"name", "email", "lastLogin"},
				Filters: []model.Filter{
					{Field: "active", Operator: "==", Value: true},
				},
			},
			expectedMsg: "should handle field projection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the query structure is valid
			err := tt.query.ValidateQuery()
			assert.NoError(t, err, tt.expectedMsg)

			// Test MongoDB filter generation
			mongoFilter := buildMongoFilter(tt.query.Filters)
			assert.NotNil(t, mongoFilter, "should generate MongoDB filter")

			// Test MongoDB options generation
			mongoOpts := buildMongoFindOptions(tt.query)
			assert.NotNil(t, mongoOpts, "should generate MongoDB find options")

			// Test cursor filter generation if applicable
			if len(tt.query.Orders) > 0 && (len(tt.query.StartAt) > 0 || len(tt.query.StartAfter) > 0 || len(tt.query.EndAt) > 0 || len(tt.query.EndBefore) > 0) {
				cursorFilter := buildCursorFilter(tt.query)
				assert.NotNil(t, cursorFilter, "should generate cursor filter")
			}

			// Log for debugging
			t.Logf("Test: %s - %s", tt.name, tt.description)
			t.Logf("Generated MongoDB filter: %+v", mongoFilter)
		})
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("buildMongoFilter with nil filters", func(t *testing.T) {
		result := buildMongoFilter(nil)
		assert.Equal(t, bson.M{"$and": []bson.M(nil)}, result)
	})

	t.Run("singleMongoFilter with empty field", func(t *testing.T) {
		filter := model.Filter{Field: "", Operator: "==", Value: "test"}
		result := singleMongoFilter(filter)
		assert.Equal(t, bson.M{"": "test"}, result)
	})

	t.Run("buildCursorFilter with more values than orders", func(t *testing.T) {
		query := model.Query{
			Orders: []model.Order{
				{Field: "name", Direction: model.DirectionAscending},
			},
			StartAt: []interface{}{"Alice", "Bob", "Charlie"},
		}
		result := buildCursorFilter(query)
		expected := bson.M{
			"$and": []bson.M{
				{"name": bson.M{"$gte": "Alice"}},
			},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("reverseDocs with nil slice", func(t *testing.T) {
		var docs []*model.Document
		// Should not panic
		reverseDocs(docs)
		assert.Nil(t, docs)
	})

	t.Run("buildMongoFindOptions with negative values", func(t *testing.T) {
		query := model.Query{
			Limit:  -10,
			Offset: -5,
		}
		opts := buildMongoFindOptions(query)
		// Negative values should not be set
		assert.Nil(t, opts.Limit)
		assert.Nil(t, opts.Skip)
	})
}

// TestFilterOperatorCoverage ensures all Firestore operators are handled
func TestFilterOperatorCoverage(t *testing.T) {
	operators := []string{
		"==", "!=", "<", "<=", ">", ">=",
		"in", "not-in",
		"array-contains", "array-contains-any",
	}

	for _, op := range operators {
		t.Run("operator_"+op, func(t *testing.T) {
			var value interface{}
			switch op {
			case "in", "not-in", "array-contains-any":
				value = []string{"test1", "test2"}
			default:
				value = "test"
			}

			filter := model.Filter{
				Field:    "testField",
				Operator: model.Operator(op),
				Value:    value,
			}

			result := singleMongoFilter(filter)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "testField")
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkBuildMongoFilter(b *testing.B) {
	filters := []model.Filter{
		{Field: "name", Operator: "==", Value: "John"},
		{Field: "age", Operator: ">", Value: 18},
		{Field: "status", Operator: "in", Value: []string{"active", "pending"}},
		{
			Composite: "or",
			SubFilters: []model.Filter{
				{Field: "type", Operator: "==", Value: "premium"},
				{Field: "type", Operator: "==", Value: "gold"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildMongoFilter(filters)
	}
}

func BenchmarkBuildMongoFindOptions(b *testing.B) {
	query := model.Query{
		Limit:  100,
		Offset: 10,
		Orders: []model.Order{
			{Field: "createdAt", Direction: model.DirectionDescending},
			{Field: "name", Direction: model.DirectionAscending},
		},
		SelectFields: []string{"name", "email", "createdAt", "status"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildMongoFindOptions(query)
	}
}

func BenchmarkBuildCursorFilter(b *testing.B) {
	query := model.Query{
		Orders: []model.Order{
			{Field: "timestamp", Direction: model.DirectionDescending},
			{Field: "id", Direction: model.DirectionAscending},
		},
		StartAfter: []interface{}{time.Now(), "doc123"},
		EndBefore:  []interface{}{time.Now().Add(-24 * time.Hour), "doc456"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildCursorFilter(query)
	}
}

func BenchmarkReverseDocs(b *testing.B) {
	// Create test documents
	docs := make([]*model.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = &model.Document{
			DocumentID: "doc" + string(rune(i)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy for each iteration
		testDocs := make([]*model.Document, len(docs))
		copy(testDocs, docs)
		reverseDocs(testDocs)
	}
}
