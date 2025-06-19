package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
)

// mockQueryEngine implements QueryEngine interface for testing
// Following hexagonal architecture patterns - this is a test adapter
type mockQueryEngine struct {
	documents    []*model.Document
	shouldError  bool
	errorMessage string
	capabilities QueryCapabilities
}

// newMockQueryEngine creates a new mock query engine with sensible defaults
func newMockQueryEngine() *mockQueryEngine {
	return &mockQueryEngine{
		documents:   []*model.Document{},
		shouldError: false,
		capabilities: QueryCapabilities{
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

// ExecuteQuery implements QueryEngine interface
func (m *mockQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	// Simulate query execution
	if len(m.documents) == 0 {
		return []*model.Document{
			{
				DocumentID:   "doc1",
				CollectionID: "test-collection",
				ProjectID:    "test-project",
				DatabaseID:   "test-database",
				Path:         collectionPath + "/doc1",
			},
		}, nil
	}

	return m.documents, nil
}

// ExecuteQueryWithProjection implements QueryEngine interface
func (m *mockQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	// For testing, we just return the same documents but could filter fields based on projection
	docs, err := m.ExecuteQuery(ctx, collectionPath, query)
	if err != nil {
		return nil, err
	}

	// In a real implementation, we would apply the projection here
	// For the mock, we just return the documents as-is
	return docs, nil
}

// CountDocuments implements QueryEngine interface
func (m *mockQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	if m.shouldError {
		return 0, errors.New(m.errorMessage)
	}

	docs, err := m.ExecuteQuery(ctx, collectionPath, query)
	if err != nil {
		return 0, err
	}

	return int64(len(docs)), nil
}

// ValidateQuery implements QueryEngine interface
func (m *mockQueryEngine) ValidateQuery(query model.Query) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}

	// Basic validation - in real implementation this would be more comprehensive
	if query.Path == "" && query.CollectionID == "" {
		return errors.New("query must specify a collection")
	}

	// Validate filter count
	if len(query.Filters) > m.capabilities.MaxFilterCount {
		return errors.New("too many filters")
	}

	// Validate order by count
	if len(query.Orders) > m.capabilities.MaxOrderByCount {
		return errors.New("too many order by clauses")
	}

	return nil
}

// GetQueryCapabilities implements QueryEngine interface
func (m *mockQueryEngine) GetQueryCapabilities() QueryCapabilities {
	return m.capabilities
}

// ExecuteAggregationPipeline implements QueryEngine interface
func (m *mockQueryEngine) ExecuteAggregationPipeline(ctx context.Context, projectID, databaseID, collectionPath string, pipeline []interface{}) ([]map[string]interface{}, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	// For testing, return a simple aggregation result
	return []map[string]interface{}{
		{
			"_id":   "group1",
			"count": int64(5),
			"sum":   100.0,
			"avg":   20.0,
		},
	}, nil
}

// BuildMongoFilter implements QueryEngine interface
func (m *mockQueryEngine) BuildMongoFilter(filters []model.Filter) (interface{}, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	// For testing, return a simple filter representation
	if len(filters) == 0 {
		return map[string]interface{}{}, nil
	}

	// Build a simple mock filter for testing
	result := map[string]interface{}{}
	for _, filter := range filters {
		switch filter.Operator {
		case model.OperatorEqual:
			result[filter.Field] = filter.Value
		case model.OperatorGreaterThan:
			result[filter.Field] = map[string]interface{}{"$gt": filter.Value}
		case model.OperatorLessThan:
			result[filter.Field] = map[string]interface{}{"$lt": filter.Value}
		case model.OperatorArrayContains:
			result[filter.Field] = map[string]interface{}{"$in": []interface{}{filter.Value}}
		default:
			result[filter.Field] = filter.Value
		}
	}

	return result, nil
}

// Helper methods for testing

// withDocuments allows setting mock documents for testing
func (m *mockQueryEngine) withDocuments(docs []*model.Document) *mockQueryEngine {
	m.documents = docs
	return m
}

// withError allows setting error conditions for testing
func (m *mockQueryEngine) withError(message string) *mockQueryEngine {
	m.shouldError = true
	m.errorMessage = message
	return m
}

// withCapabilities allows customizing capabilities for testing
func (m *mockQueryEngine) withCapabilities(caps QueryCapabilities) *mockQueryEngine {
	m.capabilities = caps
	return m
}

func TestQueryEngine_InterfaceCompliance(t *testing.T) {
	var _ QueryEngine = &mockQueryEngine{}
}

func TestQueryEngine_ExecuteQuery(t *testing.T) {
	engine := &mockQueryEngine{}
	ctx := context.Background()
	result, err := engine.ExecuteQuery(ctx, "projects/p1/databases/d1/documents/c1", model.Query{})
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "doc1", result[0].DocumentID)
}

// TestQueryEngine_CompleteTestSuite provides comprehensive testing for QueryEngine
func TestQueryEngine_CompleteTestSuite(t *testing.T) {
	t.Run("ExecuteQuery", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			engine := newMockQueryEngine()
			ctx := context.Background()

			query := model.Query{
				CollectionID: "facturas",
				Filters: []model.Filter{
					{
						Field:    "status",
						Operator: model.OperatorEqual,
						Value:    "paid",
					},
				},
			}

			result, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)

			assert.NoError(t, err)
			assert.NotEmpty(t, result)
		})

		t.Run("Error", func(t *testing.T) {
			engine := newMockQueryEngine().withError("database connection failed")
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}

			result, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "database connection failed")
		})

		t.Run("WithCustomDocuments", func(t *testing.T) {
			testDocs := []*model.Document{
				{
					DocumentID:   "INV001",
					CollectionID: "facturas",
					ProjectID:    "test-project",
					DatabaseID:   "test-database",
					Fields: map[string]*model.FieldValue{
						"status": {
							ValueType: model.FieldTypeString,
							Value:     "paid",
						},
					},
				},
				{
					DocumentID:   "INV002",
					CollectionID: "facturas",
					ProjectID:    "test-project",
					DatabaseID:   "test-database",
					Fields: map[string]*model.FieldValue{
						"status": {
							ValueType: model.FieldTypeString,
							Value:     "pending",
						},
					},
				},
			}

			engine := newMockQueryEngine().withDocuments(testDocs)
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}

			result, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)

			assert.NoError(t, err)
			assert.Len(t, result, 2)
			assert.Equal(t, "INV001", result[0].DocumentID)
			assert.Equal(t, "INV002", result[1].DocumentID)
		})
	})

	t.Run("ExecuteQueryWithProjection", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			engine := newMockQueryEngine()
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}
			projection := []string{"status", "customer.ruc"}

			result, err := engine.ExecuteQueryWithProjection(ctx, "projects/test/databases/db/documents/facturas", query, projection)

			assert.NoError(t, err)
			assert.NotEmpty(t, result)
		})

		t.Run("Error", func(t *testing.T) {
			engine := newMockQueryEngine().withError("projection not supported")
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}
			projection := []string{"status"}

			result, err := engine.ExecuteQueryWithProjection(ctx, "projects/test/databases/db/documents/facturas", query, projection)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	})

	t.Run("CountDocuments", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			testDocs := make([]*model.Document, 5)
			for i := 0; i < 5; i++ {
				testDocs[i] = &model.Document{
					DocumentID:   fmt.Sprintf("doc%d", i+1),
					CollectionID: "facturas",
				}
			}

			engine := newMockQueryEngine().withDocuments(testDocs)
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}

			count, err := engine.CountDocuments(ctx, "projects/test/databases/db/documents/facturas", query)

			assert.NoError(t, err)
			assert.Equal(t, int64(5), count)
		})

		t.Run("Error", func(t *testing.T) {
			engine := newMockQueryEngine().withError("count operation failed")
			ctx := context.Background()

			query := model.Query{CollectionID: "facturas"}

			count, err := engine.CountDocuments(ctx, "projects/test/databases/db/documents/facturas", query)

			assert.Error(t, err)
			assert.Equal(t, int64(0), count)
		})
	})

	t.Run("ValidateQuery", func(t *testing.T) {
		t.Run("ValidQuery", func(t *testing.T) {
			engine := newMockQueryEngine()

			query := model.Query{
				CollectionID: "facturas",
				Filters: []model.Filter{
					{
						Field:    "status",
						Operator: model.OperatorEqual,
						Value:    "paid",
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

		t.Run("EmptyCollectionAndPath", func(t *testing.T) {
			engine := newMockQueryEngine()

			query := model.Query{
				// Missing both Path and CollectionID
				Filters: []model.Filter{
					{
						Field:    "status",
						Operator: model.OperatorEqual,
						Value:    "paid",
					},
				},
			}

			err := engine.ValidateQuery(query)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "query must specify a collection")
		})

		t.Run("TooManyFilters", func(t *testing.T) {
			engine := newMockQueryEngine()

			// Create more filters than the maximum allowed
			filters := make([]model.Filter, 101) // MaxFilterCount is 100
			for i := 0; i < 101; i++ {
				filters[i] = model.Filter{
					Field:    fmt.Sprintf("field%d", i),
					Operator: model.OperatorEqual,
					Value:    "value",
				}
			}

			query := model.Query{
				CollectionID: "facturas",
				Filters:      filters,
			}

			err := engine.ValidateQuery(query)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "too many filters")
		})

		t.Run("TooManyOrderBy", func(t *testing.T) {
			engine := newMockQueryEngine()

			// Create more order clauses than the maximum allowed
			orders := make([]model.Order, 33) // MaxOrderByCount is 32
			for i := 0; i < 33; i++ {
				orders[i] = model.Order{
					Field:     fmt.Sprintf("field%d", i),
					Direction: model.DirectionAscending,
				}
			}

			query := model.Query{
				CollectionID: "facturas",
				Orders:       orders,
			}

			err := engine.ValidateQuery(query)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "too many order by clauses")
		})

		t.Run("ForcedError", func(t *testing.T) {
			engine := newMockQueryEngine().withError("validation failed")

			query := model.Query{CollectionID: "facturas"}

			err := engine.ValidateQuery(query)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "validation failed")
		})
	})

	t.Run("GetQueryCapabilities", func(t *testing.T) {
		t.Run("DefaultCapabilities", func(t *testing.T) {
			engine := newMockQueryEngine()

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
		})

		t.Run("CustomCapabilities", func(t *testing.T) {
			customCaps := QueryCapabilities{
				SupportsNestedFields:     false,
				SupportsArrayContains:    false,
				SupportsArrayContainsAny: false,
				SupportsCompositeFilters: true,
				SupportsOrderBy:          true,
				SupportsCursorPagination: false,
				SupportsOffsetPagination: true,
				SupportsProjection:       false,
				MaxFilterCount:           10,
				MaxOrderByCount:          5,
				MaxNestingDepth:          3,
			}

			engine := newMockQueryEngine().withCapabilities(customCaps)

			capabilities := engine.GetQueryCapabilities()

			assert.False(t, capabilities.SupportsNestedFields)
			assert.False(t, capabilities.SupportsArrayContains)
			assert.False(t, capabilities.SupportsArrayContainsAny)
			assert.True(t, capabilities.SupportsCompositeFilters)
			assert.True(t, capabilities.SupportsOrderBy)
			assert.False(t, capabilities.SupportsCursorPagination)
			assert.True(t, capabilities.SupportsOffsetPagination)
			assert.False(t, capabilities.SupportsProjection)
			assert.Equal(t, 10, capabilities.MaxFilterCount)
			assert.Equal(t, 5, capabilities.MaxOrderByCount)
			assert.Equal(t, 3, capabilities.MaxNestingDepth)
		})
	})
}

// TestQueryEngine_ContextHandling tests context cancellation and timeout handling
func TestQueryEngine_ContextHandling(t *testing.T) {
	t.Run("ContextCancellation", func(t *testing.T) {
		engine := newMockQueryEngine()
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context immediately
		cancel()

		query := model.Query{CollectionID: "facturas"}

		// Mock should handle context cancellation gracefully
		// In a real implementation, this would check ctx.Done()
		result, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)

		// Mock doesn't actually check context, but real implementation would
		assert.NoError(t, err) // Mock behavior
		assert.NotNil(t, result)
	})
}

// TestQueryEngine_ConcurrentAccess tests thread safety
func TestQueryEngine_ConcurrentAccess(t *testing.T) {
	engine := newMockQueryEngine()
	ctx := context.Background()
	query := model.Query{CollectionID: "facturas"}

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)
			results <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// TestQueryEngine_PerformanceBaseline provides baseline performance measurements
func TestQueryEngine_PerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	engine := newMockQueryEngine()
	ctx := context.Background()
	query := model.Query{CollectionID: "facturas"}

	const iterations = 1000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := engine.ExecuteQuery(ctx, "projects/test/databases/db/documents/facturas", query)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
	duration := time.Since(start)

	avgDuration := duration / iterations
	t.Logf("Average query execution time: %v", avgDuration)

	// Baseline expectation: mock should be very fast
	assert.Less(t, avgDuration, time.Millisecond, "Mock query engine should be very fast")
}

// BenchmarkQueryEngine_ExecuteQuery benchmarks the query execution
func BenchmarkQueryEngine_ExecuteQuery(b *testing.B) {
	engine := newMockQueryEngine()
	ctx := context.Background()
	query := model.Query{CollectionID: "facturas"}
	collectionPath := "projects/test/databases/db/documents/facturas"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteQuery(ctx, collectionPath, query)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkQueryEngine_CountDocuments benchmarks the count operation
func BenchmarkQueryEngine_CountDocuments(b *testing.B) {
	engine := newMockQueryEngine()
	ctx := context.Background()
	query := model.Query{CollectionID: "facturas"}
	collectionPath := "projects/test/databases/db/documents/facturas"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.CountDocuments(ctx, collectionPath, query)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkQueryEngine_ExecuteAggregationPipeline benchmarks the aggregation pipeline execution
func BenchmarkQueryEngine_ExecuteAggregationPipeline(b *testing.B) {
	engine := newMockQueryEngine()
	ctx := context.Background()
	pipeline := []interface{}{
		map[string]interface{}{
			"$group": map[string]interface{}{
				"_id":   "$category",
				"count": map[string]interface{}{"$sum": 1},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteAggregationPipeline(ctx, "test-project", "test-db", "facturas", pipeline)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkQueryEngine_BuildMongoFilter benchmarks the filter building
func BenchmarkQueryEngine_BuildMongoFilter(b *testing.B) {
	engine := newMockQueryEngine()
	filters := []model.Filter{
		{
			Field:    "status",
			Operator: model.OperatorEqual,
			Value:    "paid",
		},
		{
			Field:    "amount",
			Operator: model.OperatorGreaterThan,
			Value:    100.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.BuildMongoFilter(filters)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// TestQueryEngine_ExecuteAggregationPipeline tests the aggregation pipeline functionality
func TestQueryEngine_ExecuteAggregationPipeline(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		engine := newMockQueryEngine()
		ctx := context.Background()

		pipeline := []interface{}{
			map[string]interface{}{
				"$match": map[string]interface{}{
					"status": "paid",
				},
			},
			map[string]interface{}{
				"$group": map[string]interface{}{
					"_id":   "$category",
					"count": map[string]interface{}{"$sum": 1},
					"total": map[string]interface{}{"$sum": "$amount"},
				},
			},
		}

		result, err := engine.ExecuteAggregationPipeline(ctx, "test-project", "test-db", "facturas", pipeline)

		assert.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result[0], "_id")
		assert.Contains(t, result[0], "count")
	})

	t.Run("Error", func(t *testing.T) {
		engine := newMockQueryEngine().withError("aggregation failed")
		ctx := context.Background()

		pipeline := []interface{}{
			map[string]interface{}{"$match": map[string]interface{}{"status": "paid"}},
		}

		result, err := engine.ExecuteAggregationPipeline(ctx, "test-project", "test-db", "facturas", pipeline)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "aggregation failed")
	})

	t.Run("EmptyPipeline", func(t *testing.T) {
		engine := newMockQueryEngine()
		ctx := context.Background()

		pipeline := []interface{}{}

		result, err := engine.ExecuteAggregationPipeline(ctx, "test-project", "test-db", "facturas", pipeline)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestQueryEngine_BuildMongoFilter tests the MongoDB filter building functionality
func TestQueryEngine_BuildMongoFilter(t *testing.T) {
	t.Run("NoFilters", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		expectedFilter := map[string]interface{}{}
		assert.Equal(t, expectedFilter, result)
	})

	t.Run("SingleEqualFilter", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{
			{
				Field:    "status",
				Operator: model.OperatorEqual,
				Value:    "paid",
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		expectedFilter := map[string]interface{}{
			"status": "paid",
		}
		assert.Equal(t, expectedFilter, result)
	})

	t.Run("GreaterThanFilter", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{
			{
				Field:    "amount",
				Operator: model.OperatorGreaterThan,
				Value:    100.0,
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		expectedFilter := map[string]interface{}{
			"amount": map[string]interface{}{"$gt": 100.0},
		}
		assert.Equal(t, expectedFilter, result)
	})

	t.Run("LessThanFilter", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{
			{
				Field:    "amount",
				Operator: model.OperatorLessThan,
				Value:    1000.0,
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		expectedFilter := map[string]interface{}{
			"amount": map[string]interface{}{"$lt": 1000.0},
		}
		assert.Equal(t, expectedFilter, result)
	})

	t.Run("ArrayContainsFilter", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{
			{
				Field:    "tags",
				Operator: model.OperatorArrayContains,
				Value:    "important",
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		expectedFilter := map[string]interface{}{
			"tags": map[string]interface{}{"$in": []interface{}{"important"}},
		}
		assert.Equal(t, expectedFilter, result)
	})

	t.Run("MultipleFilters", func(t *testing.T) {
		engine := newMockQueryEngine()
		filters := []model.Filter{
			{
				Field:    "status",
				Operator: model.OperatorEqual,
				Value:    "paid",
			},
			{
				Field:    "amount",
				Operator: model.OperatorGreaterThan,
				Value:    100.0,
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		assert.Equal(t, "paid", resultMap["status"])
		assert.Equal(t, map[string]interface{}{"$gt": 100.0}, resultMap["amount"])
	})

	t.Run("Error", func(t *testing.T) {
		engine := newMockQueryEngine().withError("filter build failed")
		filters := []model.Filter{
			{
				Field:    "status",
				Operator: model.OperatorEqual,
				Value:    "paid",
			},
		}

		result, err := engine.BuildMongoFilter(filters)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "filter build failed")
	})
}
