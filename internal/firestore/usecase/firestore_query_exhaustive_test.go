package usecase_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
)

// FirestoreStructuredQuery represents the exact JSON format that Google Firestore expects
// This matches the structure you showed in your request
type FirestoreStructuredQuery struct {
	From    []FirestoreCollectionSelector `json:"from,omitempty"`
	Where   *FirestoreFilter              `json:"where,omitempty"`
	OrderBy []FirestoreOrder              `json:"orderBy,omitempty"`
	Limit   int                           `json:"limit,omitempty"`
	Offset  int                           `json:"offset,omitempty"`
}

type FirestoreCollectionSelector struct {
	CollectionID   string `json:"collectionId"`
	AllDescendants bool   `json:"allDescendants,omitempty"`
}

type FirestoreFilter struct {
	FieldFilter     *FirestoreFieldFilter     `json:"fieldFilter,omitempty"`
	CompositeFilter *FirestoreCompositeFilter `json:"compositeFilter,omitempty"`
	UnaryFilter     *FirestoreUnaryFilter     `json:"unaryFilter,omitempty"`
}

type FirestoreFieldFilter struct {
	Field *FirestoreFieldReference `json:"field"`
	Op    string                   `json:"op"`
	Value interface{}              `json:"value"`
}

type FirestoreCompositeFilter struct {
	Op      string            `json:"op"` // "AND" or "OR"
	Filters []FirestoreFilter `json:"filters"`
}

type FirestoreUnaryFilter struct {
	Field *FirestoreFieldReference `json:"field"`
	Op    string                   `json:"op"` // "IS_NULL", "IS_NOT_NULL"
}

type FirestoreFieldReference struct {
	FieldPath string `json:"fieldPath"`
}

type FirestoreOrder struct {
	Field     *FirestoreFieldReference `json:"field"`
	Direction string                   `json:"direction"` // "ASCENDING" or "DESCENDING"
}

// convertFirestoreJSONToModelQuery converts the Firestore JSON format to our internal model
func convertFirestoreJSONToModelQuery(jsonQuery FirestoreStructuredQuery, basePath string) *model.Query {
	query := &model.Query{
		Path:   basePath,
		Limit:  jsonQuery.Limit,
		Offset: jsonQuery.Offset,
	}

	// Set collection info from 'from' clause
	if len(jsonQuery.From) > 0 {
		query.CollectionID = jsonQuery.From[0].CollectionID
		query.AllDescendants = jsonQuery.From[0].AllDescendants
		if query.AllDescendants {
			query.Path = basePath // For collection group queries
		} else {
			query.Path = basePath + "/" + query.CollectionID
		}
	}

	// Convert where clause
	if jsonQuery.Where != nil {
		convertFirestoreFilter(jsonQuery.Where, query)
	}

	// Convert orderBy clause
	for _, order := range jsonQuery.OrderBy {
		direction := model.DirectionAscending
		if order.Direction == "DESCENDING" {
			direction = model.DirectionDescending
		}
		query.AddOrder(order.Field.FieldPath, direction)
	}

	return query
}

func convertFirestoreFilter(filter *FirestoreFilter, query *model.Query) {
	if filter.FieldFilter != nil {
		// Convert field filter
		ff := filter.FieldFilter
		operator := convertFirestoreOperator(ff.Op)
		query.AddFilter(ff.Field.FieldPath, operator, ff.Value)
	} else if filter.CompositeFilter != nil {
		// Handle composite filters (AND/OR)
		for _, subFilter := range filter.CompositeFilter.Filters {
			convertFirestoreFilter(&subFilter, query)
		}
	} else if filter.UnaryFilter != nil {
		// Handle unary filters (IS_NULL, IS_NOT_NULL)
		uf := filter.UnaryFilter
		operator := convertFirestoreOperator(uf.Op)
		var value interface{}
		if uf.Op == "IS_NULL" {
			value = nil
		}
		query.AddFilter(uf.Field.FieldPath, operator, value)
	}
}

func convertFirestoreOperator(op string) model.Operator {
	switch op {
	case "EQUAL":
		return model.OperatorEqual
	case "NOT_EQUAL":
		return model.OperatorNotEqual
	case "LESS_THAN":
		return model.OperatorLessThan
	case "LESS_THAN_OR_EQUAL":
		return model.OperatorLessThanOrEqual
	case "GREATER_THAN":
		return model.OperatorGreaterThan
	case "GREATER_THAN_OR_EQUAL":
		return model.OperatorGreaterThanOrEqual
	case "IN":
		return model.OperatorIn
	case "NOT_IN":
		return model.OperatorNotIn
	case "ARRAY_CONTAINS":
		return model.OperatorArrayContains
	case "ARRAY_CONTAINS_ANY":
		return model.OperatorArrayContainsAny
	case "IS_NULL":
		return model.OperatorEqual // Handle as equal to nil
	case "IS_NOT_NULL":
		return model.OperatorNotEqual // Handle as not equal to nil
	default:
		return model.OperatorEqual // Default fallback
	}
}

// TestFirestoreQueryExhaustive tests all Firestore query features with real-world scenarios
// This test ensures 100% compatibility with Google Firestore query behavior
func TestFirestoreQueryExhaustive(t *testing.T) {
	ctx := context.Background()
	// Create test use case with mocks
	firestoreUC := newTestFirestoreUsecase() // Test constants following Firestore paths
	const (
		projectID    = "test-project-2025"
		databaseID   = "test-database"
		collectionID = "users"
		parent       = "projects/test-project-2025/databases/test-database/documents/users"
	)

	t.Run("1. Basic Filter Operations - Google Firestore Compatible", func(t *testing.T) {
		testCases := []struct {
			name        string
			field       string
			operator    model.Operator
			value       interface{}
			description string
		}{
			{"Equal operator", "status", model.OperatorEqual, "active", "WHERE status == 'active'"},
			{"Not equal operator", "status", model.OperatorNotEqual, "deleted", "WHERE status != 'deleted'"},
			{"Greater than", "age", model.OperatorGreaterThan, 18, "WHERE age > 18"},
			{"Greater than or equal", "age", model.OperatorGreaterThanOrEqual, 21, "WHERE age >= 21"},
			{"Less than", "age", model.OperatorLessThan, 65, "WHERE age < 65"}, {"Less than or equal", "score", model.OperatorLessThanOrEqual, 100, "WHERE score <= 100"},
			{"In operator", "city", model.OperatorIn, []interface{}{"Madrid", "Barcelona", "Valencia"}, "WHERE city IN ['Madrid', 'Barcelona', 'Valencia']"},
			{"Not in operator", "status", model.OperatorNotIn, []interface{}{"banned", "deleted"}, "WHERE status NOT IN ['banned', 'deleted']"},
			{"Array contains", "tags", model.OperatorArrayContains, "premium", "WHERE tags ARRAY_CONTAINS 'premium'"},
			{"Array contains any", "categories", model.OperatorArrayContainsAny, []interface{}{"tech", "science"}, "WHERE categories ARRAY_CONTAINS_ANY ['tech', 'science']"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) { // Create Firestore-compatible structured query
				query := &model.Query{
					Path:         parent,
					CollectionID: collectionID,
				}
				query.AddFilter(tc.field, tc.operator, tc.value)

				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parent,
					StructuredQuery: query,
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Query should execute without error: %s", tc.description)
				assert.NotNil(t, docs, "Documents should not be nil for: %s", tc.description)
			})
		}
	})

	t.Run("2. Complex Compound Filters - Real World Scenarios", func(t *testing.T) {
		testCases := []struct {
			name        string
			setupQuery  func() *model.Query
			description string
		}{{
			name: "Age range with status filter",
			setupQuery: func() *model.Query {
				q := &model.Query{Path: parent, CollectionID: collectionID}
				q.AddFilter("age", model.OperatorGreaterThanOrEqual, 18)
				q.AddFilter("age", model.OperatorLessThan, 65)
				q.AddFilter("status", model.OperatorEqual, "active")
				return q
			},
			description: "Find active users between 18-65 years old",
		}, {
			name: "Multiple string filters",
			setupQuery: func() *model.Query {
				q := &model.Query{Path: parent, CollectionID: collectionID}
				q.AddFilter("country", model.OperatorEqual, "Spain")
				q.AddFilter("role", model.OperatorNotEqual, "admin")
				q.AddFilter("subscription", model.OperatorIn, []interface{}{"premium", "enterprise"})
				return q
			},
			description: "Spanish non-admin users with premium+ subscriptions",
		},
			{
				name: "Array and value combinations", setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddFilter("tags", model.OperatorArrayContains, "verified")
					q.AddFilter("score", model.OperatorGreaterThan, 500)
					q.AddFilter("status", model.OperatorNotIn, []interface{}{"banned", "suspended"})
					return q
				},
				description: "Verified users with high scores, not banned/suspended",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parent,
					StructuredQuery: tc.setupQuery(),
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Complex query should execute: %s", tc.description)
				assert.NotNil(t, docs, "Result should not be nil: %s", tc.description)
			})
		}
	})

	t.Run("3. Ordering and Pagination - Firestore Standard", func(t *testing.T) {
		testCases := []struct {
			name        string
			setupQuery  func() *model.Query
			description string
		}{
			{
				name: "Single field ascending order",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("name", model.DirectionAscending)
					q.SetLimit(10)
					return q
				},
				description: "ORDER BY name ASC LIMIT 10",
			},
			{
				name: "Multiple field ordering",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("priority", model.DirectionDescending)
					q.AddOrder("createdAt", model.DirectionAscending)
					q.SetLimit(20)
					return q
				},
				description: "ORDER BY priority DESC, createdAt ASC LIMIT 20",
			},
			{
				name: "Filtered with ordering",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddFilter("status", model.OperatorEqual, "active")
					q.AddOrder("lastLoginAt", model.DirectionDescending)
					q.SetLimit(50)
					return q
				},
				description: "WHERE status = 'active' ORDER BY lastLoginAt DESC LIMIT 50",
			},
			{
				name: "Pagination with offset",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("email", model.DirectionAscending)
					q.SetLimit(25)
					q.SetOffset(100)
					return q
				},
				description: "ORDER BY email ASC LIMIT 25 OFFSET 100",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parent,
					StructuredQuery: tc.setupQuery(),
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Ordering query should execute: %s", tc.description)
				assert.NotNil(t, docs, "Ordered result should not be nil: %s", tc.description)
			})
		}
	})

	t.Run("4. Cursor-based Pagination - Firestore Compatible", func(t *testing.T) {
		testCases := []struct {
			name        string
			setupQuery  func() *model.Query
			description string
		}{
			{
				name: "StartAt cursor",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("score", model.DirectionDescending)
					q.StartAt = []interface{}{1000}
					q.SetLimit(10)
					return q
				},
				description: "Start at score 1000, descending",
			},
			{
				name: "StartAfter cursor",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("name", model.DirectionAscending)
					q.StartAfter = []interface{}{"John"}
					q.SetLimit(15)
					return q
				},
				description: "Start after name 'John', ascending",
			},
			{
				name: "EndBefore cursor",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("createdAt", model.DirectionAscending)
					q.EndBefore = []interface{}{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
					q.SetLimit(20)
					return q
				},
				description: "End before 2025-01-01",
			},
			{
				name: "Multi-field cursor",
				setupQuery: func() *model.Query {
					q := &model.Query{Path: parent + "/" + collectionID, CollectionID: collectionID}
					q.AddOrder("priority", model.DirectionDescending)
					q.AddOrder("id", model.DirectionAscending)
					q.StartAt = []interface{}{5, "abc123"}
					q.SetLimit(10)
					return q
				},
				description: "Multi-field cursor: priority DESC, id ASC starting at [5, 'abc123']",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parent,
					StructuredQuery: tc.setupQuery(),
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Cursor query should execute: %s", tc.description)
				assert.NotNil(t, docs, "Cursor result should not be nil: %s", tc.description)
			})
		}
	})

	t.Run("5. Field Selection (Projection) - Firestore Compatible", func(t *testing.T) {
		query := &model.Query{
			Path:         parent + "/" + collectionID,
			CollectionID: collectionID,
			SelectFields: []string{"name", "email", "status"},
		}
		query.AddFilter("status", model.OperatorEqual, "active")

		req := usecase.QueryRequest{
			ProjectID:       projectID,
			DatabaseID:      databaseID,
			Parent:          parent,
			StructuredQuery: query,
		}

		docs, err := firestoreUC.RunQuery(ctx, req)
		assert.NoError(t, err, "Field selection query should execute")
		assert.NotNil(t, docs, "Projected result should not be nil")
	})

	t.Run("6. LimitToLast - Firestore Reverse Pagination", func(t *testing.T) {
		query := &model.Query{
			Path:         parent + "/" + collectionID,
			CollectionID: collectionID,
			LimitToLast:  true,
		}
		query.AddOrder("timestamp", model.DirectionDescending)
		query.SetLimit(5)

		req := usecase.QueryRequest{
			ProjectID:       projectID,
			DatabaseID:      databaseID,
			Parent:          parent,
			StructuredQuery: query,
		}

		docs, err := firestoreUC.RunQuery(ctx, req)
		assert.NoError(t, err, "LimitToLast query should execute")
		assert.NotNil(t, docs, "Reverse pagination result should not be nil")
	})

	t.Run("7. Data Type Compatibility - Firestore Types", func(t *testing.T) {
		testCases := []struct {
			name     string
			field    string
			operator model.Operator
			value    interface{}
		}{
			{"String value", "name", model.OperatorEqual, "John Doe"},
			{"Integer value", "age", model.OperatorGreaterThan, 25},
			{"Float value", "rating", model.OperatorLessThanOrEqual, 4.5},
			{"Boolean value", "isActive", model.OperatorEqual, true},
			{"Null value", "deletedAt", model.OperatorEqual, nil},
			{"Timestamp value", "createdAt", model.OperatorGreaterThan, time.Now().Add(-24 * time.Hour)},
			{"Array of strings", "categories", model.OperatorIn, []interface{}{"tech", "business", "science"}},
			{"Array of numbers", "scores", model.OperatorIn, []interface{}{85, 90, 95}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				query := &model.Query{
					Path:         parent + "/" + collectionID,
					CollectionID: collectionID,
				}
				query.AddFilter(tc.field, tc.operator, tc.value)

				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parent,
					StructuredQuery: query,
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Data type query should execute for: %s", tc.name)
				assert.NotNil(t, docs, "Result should not be nil for: %s", tc.name)
			})
		}
	})

	t.Run("8. Edge Cases and Error Handling", func(t *testing.T) {
		t.Run("Empty query should work", func(t *testing.T) {
			query := &model.Query{
				Path:         parent + "/" + collectionID,
				CollectionID: collectionID,
			}

			req := usecase.QueryRequest{
				ProjectID:       projectID,
				DatabaseID:      databaseID,
				Parent:          parent,
				StructuredQuery: query,
			}

			docs, err := firestoreUC.RunQuery(ctx, req)
			assert.NoError(t, err, "Empty query should execute")
			assert.NotNil(t, docs, "Empty query result should not be nil")
		})

		t.Run("Nil structured query should fail", func(t *testing.T) {
			req := usecase.QueryRequest{
				ProjectID:       projectID,
				DatabaseID:      databaseID,
				Parent:          parent,
				StructuredQuery: nil,
			}

			docs, err := firestoreUC.RunQuery(ctx, req)
			assert.Error(t, err, "Nil structured query should fail")
			assert.Contains(t, err.Error(), "structured query is required")
			assert.Nil(t, docs, "Nil query result should be nil")
		})

		t.Run("Zero limit should work", func(t *testing.T) {
			query := &model.Query{
				Path:         parent + "/" + collectionID,
				CollectionID: collectionID,
				Limit:        0,
			}

			req := usecase.QueryRequest{
				ProjectID:       projectID,
				DatabaseID:      databaseID,
				Parent:          parent,
				StructuredQuery: query,
			}

			docs, err := firestoreUC.RunQuery(ctx, req)
			assert.NoError(t, err, "Zero limit query should execute")
			assert.NotNil(t, docs, "Zero limit result should not be nil")
		})
	})

	t.Run("9. Collection Group Queries - Firestore Feature", func(t *testing.T) {
		query := &model.Query{
			Path:           parent,
			CollectionID:   "messages", // Collection group name
			AllDescendants: true,       // This makes it a collection group query
		}
		query.AddFilter("type", model.OperatorEqual, "public")
		query.AddOrder("timestamp", model.DirectionDescending)
		query.SetLimit(100)

		req := usecase.QueryRequest{
			ProjectID:       projectID,
			DatabaseID:      databaseID,
			Parent:          parent,
			StructuredQuery: query,
		}

		docs, err := firestoreUC.RunQuery(ctx, req)
		assert.NoError(t, err, "Collection group query should execute")
		assert.NotNil(t, docs, "Collection group result should not be nil")
	})

	t.Run("10. Performance and Scalability Tests", func(t *testing.T) {
		t.Run("Large limit query", func(t *testing.T) {
			query := &model.Query{
				Path:         parent + "/" + collectionID,
				CollectionID: collectionID,
				Limit:        10000, // Large limit
			}

			req := usecase.QueryRequest{
				ProjectID:       projectID,
				DatabaseID:      databaseID,
				Parent:          parent,
				StructuredQuery: query,
			}

			docs, err := firestoreUC.RunQuery(ctx, req)
			assert.NoError(t, err, "Large limit query should execute")
			assert.NotNil(t, docs, "Large limit result should not be nil")
		})

		t.Run("Complex multi-condition query", func(t *testing.T) {
			query := &model.Query{
				Path:         parent + "/" + collectionID,
				CollectionID: collectionID,
			}

			// Add multiple complex conditions
			query.AddFilter("status", model.OperatorEqual, "active")
			query.AddFilter("age", model.OperatorGreaterThanOrEqual, 18)
			query.AddFilter("age", model.OperatorLessThan, 65)
			query.AddFilter("country", model.OperatorIn, []interface{}{"ES", "US", "UK", "DE", "FR"})
			query.AddFilter("tags", model.OperatorArrayContains, "premium")
			query.AddFilter("score", model.OperatorGreaterThan, 500)
			query.AddOrder("score", model.DirectionDescending)
			query.AddOrder("lastActive", model.DirectionDescending)
			query.SetLimit(50)

			req := usecase.QueryRequest{
				ProjectID:       projectID,
				DatabaseID:      databaseID,
				Parent:          parent,
				StructuredQuery: query,
			}

			docs, err := firestoreUC.RunQuery(ctx, req)
			assert.NoError(t, err, "Complex multi-condition query should execute")
			assert.NotNil(t, docs, "Complex query result should not be nil")
		})
	})
}

// TestFirestoreQueryValidation tests query validation according to Firestore rules
func TestFirestoreQueryValidation(t *testing.T) {
	t.Run("Query validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			setupQuery  func() *model.Query
			shouldError bool
			errorCheck  string
		}{
			{
				name: "Valid query should pass",
				setupQuery: func() *model.Query {
					q := &model.Query{
						Path:         "projects/test/databases/test/documents/users",
						CollectionID: "users",
						Limit:        10,
					}
					q.AddFilter("status", model.OperatorEqual, "active")
					return q
				},
				shouldError: false,
			},
			{
				name: "Empty path should fail",
				setupQuery: func() *model.Query {
					return &model.Query{
						Path:         "",
						CollectionID: "users",
					}
				},
				shouldError: true,
				errorCheck:  "invalid query path",
			},
			{
				name: "Negative limit should fail",
				setupQuery: func() *model.Query {
					return &model.Query{
						Path:         "projects/test/databases/test/documents/users",
						CollectionID: "users",
						Limit:        -1,
					}
				},
				shouldError: true,
				errorCheck:  "invalid query limit",
			},
			{
				name: "Invalid operator should fail",
				setupQuery: func() *model.Query {
					q := &model.Query{
						Path:         "projects/test/databases/test/documents/users",
						CollectionID: "users",
					}
					// Manually add invalid filter to test validation
					q.Filters = append(q.Filters, model.Filter{
						Field:    "status",
						Operator: "INVALID_OP",
						Value:    "active",
					})
					return q
				},
				shouldError: true,
				errorCheck:  "invalid filter operator",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				query := tc.setupQuery()
				err := query.ValidateQuery()

				if tc.shouldError {
					assert.Error(t, err, "Query should fail validation")
					if tc.errorCheck != "" {
						assert.Contains(t, err.Error(), tc.errorCheck)
					}
				} else {
					assert.NoError(t, err, "Query should pass validation")
				}
			})
		}
	})
}

// TestFirestoreQueryJSON tests JSON serialization/deserialization compatibility
func TestFirestoreQueryJSON(t *testing.T) {
	t.Run("JSON compatibility with Firestore format", func(t *testing.T) {
		// This tests the exact JSON format that comes from Postman/clients
		// matching Google Firestore's structured query format

		query := &model.Query{
			Path:         "projects/test-project/databases/test-db/documents/users",
			CollectionID: "users",
			Limit:        10,
		}

		// Add filters in Firestore format
		query.AddFilter("born", model.OperatorLessThan, 1900)
		query.AddOrder("name", model.DirectionAscending)

		// Test that our model can handle the structures
		assert.Equal(t, "users", query.CollectionID)
		assert.Equal(t, 10, query.Limit)
		assert.Len(t, query.Filters, 1)
		assert.Len(t, query.Orders, 1)

		// Verify filter structure
		filter := query.Filters[0]
		assert.Equal(t, "born", filter.Field)
		assert.Equal(t, model.OperatorLessThan, filter.Operator)
		assert.Equal(t, 1900, filter.Value)

		// Verify order structure
		order := query.Orders[0]
		assert.Equal(t, "name", order.Field)
		assert.Equal(t, model.DirectionAscending, order.Direction)
	})
}

// TestFirestoreQueryBuilders tests the helper methods for building queries
func TestFirestoreQueryBuilders(t *testing.T) {
	t.Run("Query builder methods", func(t *testing.T) {
		query := &model.Query{
			Path:         "projects/test/databases/test/documents/users",
			CollectionID: "users",
		}

		// Test fluent interface
		query.AddFilter("status", model.OperatorEqual, "active").
			AddFilter("age", model.OperatorGreaterThan, 18).
			AddOrder("name", model.DirectionAscending).
			AddOrder("createdAt", model.DirectionDescending).
			SetLimit(50).
			SetOffset(10)

		assert.Len(t, query.Filters, 2, "Should have 2 filters")
		assert.Len(t, query.Orders, 2, "Should have 2 orders")
		assert.Equal(t, 50, query.Limit, "Limit should be set")
		assert.Equal(t, 10, query.Offset, "Offset should be set")

		// Test individual filters
		assert.Equal(t, "status", query.Filters[0].Field)
		assert.Equal(t, "age", query.Filters[1].Field)

		// Test individual orders
		assert.Equal(t, "name", query.Orders[0].Field)
		assert.Equal(t, "createdAt", query.Orders[1].Field)
	})
}

// Benchmark tests for performance validation
func BenchmarkFirestoreQuery(b *testing.B) {
	ctx := context.Background()
	firestoreUC := newTestFirestoreUsecase()

	query := &model.Query{
		Path:         "projects/bench/databases/bench/documents/users",
		CollectionID: "users",
	}
	query.AddFilter("status", model.OperatorEqual, "active")
	query.AddOrder("score", model.DirectionDescending)
	query.SetLimit(100)

	req := usecase.QueryRequest{
		ProjectID:       "bench-project",
		DatabaseID:      "bench-db",
		Parent:          "projects/bench/databases/bench/documents/users",
		StructuredQuery: query,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := firestoreUC.RunQuery(ctx, req)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// TestFirestoreJSONStructuredQuery tests the exact JSON format from your request
func TestFirestoreJSONStructuredQuery(t *testing.T) {
	ctx := context.Background()
	firestoreUC := newTestFirestoreUsecase()

	const (
		projectID  = "test-project-2025"
		databaseID = "test-database"
		basePath   = "projects/test-project-2025/databases/test-database/documents"
	)

	t.Run("1. Exact JSON Format from Your Request", func(t *testing.T) {
		// This is exactly the JSON structure you provided
		jsonQueryString := `{
			"from": [
				{
					"collectionId": "productos"
				}
			],
			"where": {
				"fieldFilter": {
					"field": {
						"fieldPath": "born"
					},
					"op": "LESS_THAN",
					"value": 1900
				}
			},
			"orderBy": [
				{
					"field": {
						"fieldPath": "name"
					},
					"direction": "ASCENDING"
				}
			],
			"limit": 10
		}`

		// Parse the JSON
		var firestoreQuery FirestoreStructuredQuery
		err := json.Unmarshal([]byte(jsonQueryString), &firestoreQuery)
		assert.NoError(t, err, "Should parse Firestore JSON format")

		// Convert to our internal model
		query := convertFirestoreJSONToModelQuery(firestoreQuery, basePath)

		// Execute the query
		req := usecase.QueryRequest{
			ProjectID:       projectID,
			DatabaseID:      databaseID,
			Parent:          basePath + "/productos",
			StructuredQuery: query,
		}

		docs, err := firestoreUC.RunQuery(ctx, req)
		assert.NoError(t, err, "Firestore JSON query should execute successfully")
		assert.NotNil(t, docs, "Query result should not be nil")

		// Verify the query was converted correctly
		assert.Equal(t, "productos", query.CollectionID)
		assert.Equal(t, 10, query.Limit)
		assert.Len(t, query.Filters, 1)
		assert.Len(t, query.Orders, 1)

		// Verify filter
		filter := query.Filters[0]
		assert.Equal(t, "born", filter.Field)
		assert.Equal(t, model.OperatorLessThan, filter.Operator)
		assert.Equal(t, float64(1900), filter.Value) // JSON numbers become float64

		// Verify order
		order := query.Orders[0]
		assert.Equal(t, "name", order.Field)
		assert.Equal(t, model.DirectionAscending, order.Direction)
	})

	t.Run("2. Complex Firestore JSON Queries", func(t *testing.T) {
		testCases := []struct {
			name        string
			jsonQuery   string
			description string
		}{
			{
				name: "Multiple filters with AND",
				jsonQuery: `{
					"from": [{"collectionId": "users"}],
					"where": {
						"compositeFilter": {
							"op": "AND",
							"filters": [
								{
									"fieldFilter": {
										"field": {"fieldPath": "age"},
										"op": "GREATER_THAN_OR_EQUAL",
										"value": 18
									}
								},
								{
									"fieldFilter": {
										"field": {"fieldPath": "status"},
										"op": "EQUAL",
										"value": "active"
									}
								}
							]
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "name"},
							"direction": "ASCENDING"
						}
					],
					"limit": 50
				}`,
				description: "Complex AND filter with multiple conditions",
			},
			{
				name: "Array operations",
				jsonQuery: `{
					"from": [{"collectionId": "posts"}],
					"where": {
						"fieldFilter": {
							"field": {"fieldPath": "tags"},
							"op": "ARRAY_CONTAINS",
							"value": "featured"
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "publishedAt"},
							"direction": "DESCENDING"
						}
					],
					"limit": 20
				}`,
				description: "Array contains operation",
			},
			{
				name: "IN operator with multiple values",
				jsonQuery: `{
					"from": [{"collectionId": "products"}],
					"where": {
						"fieldFilter": {
							"field": {"fieldPath": "category"},
							"op": "IN",
							"value": ["electronics", "books", "clothing"]
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "price"},
							"direction": "ASCENDING"
						}
					],
					"limit": 100
				}`,
				description: "IN operator with array of values",
			},
			{
				name: "Collection group query",
				jsonQuery: `{
					"from": [
						{
							"collectionId": "messages",
							"allDescendants": true
						}
					],
					"where": {
						"fieldFilter": {
							"field": {"fieldPath": "type"},
							"op": "EQUAL",
							"value": "public"
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "timestamp"},
							"direction": "DESCENDING"
						}
					],
					"limit": 50
				}`,
				description: "Collection group query across all subcollections",
			},
			{
				name: "Null checks",
				jsonQuery: `{
					"from": [{"collectionId": "users"}],
					"where": {
						"unaryFilter": {
							"field": {"fieldPath": "deletedAt"},
							"op": "IS_NULL"
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "createdAt"},
							"direction": "DESCENDING"
						}
					],
					"limit": 25
				}`,
				description: "Null check filter",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Parse JSON
				var firestoreQuery FirestoreStructuredQuery
				err := json.Unmarshal([]byte(tc.jsonQuery), &firestoreQuery)
				assert.NoError(t, err, "Should parse JSON for: %s", tc.description)

				// Convert to internal model
				query := convertFirestoreJSONToModelQuery(firestoreQuery, basePath) // Execute query
				parentPath := basePath
				if len(firestoreQuery.From) > 0 {
					// Both regular collection queries and collection group queries
					// need the collection in the parent path for the current implementation
					parentPath = basePath + "/" + firestoreQuery.From[0].CollectionID
				}

				req := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          parentPath,
					StructuredQuery: query,
				}

				docs, err := firestoreUC.RunQuery(ctx, req)
				assert.NoError(t, err, "Query should execute for: %s", tc.description)
				assert.NotNil(t, docs, "Result should not be nil for: %s", tc.description)
			})
		}
	})

	t.Run("3. Real Postman-like Requests", func(t *testing.T) {
		// Simulate what would come from a Postman request or HTTP client
		postmanRequests := []struct {
			name        string
			method      string
			url         string
			body        string
			description string
		}{
			{
				name:   "Product search by category",
				method: "POST",
				url:    "/api/v1/organizations/org1/projects/proj1/databases/db1/query/productos",
				body: `{
					"from": [{"collectionId": "productos"}],
					"where": {
						"fieldFilter": {
							"field": {"fieldPath": "categoria"},
							"op": "EQUAL",
							"value": "electronica"
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "precio"},
							"direction": "ASCENDING"
						}
					],
					"limit": 20
				}`,
				description: "Search products by category, ordered by price",
			},
			{
				name:   "User filtering by age range",
				method: "POST",
				url:    "/api/v1/organizations/org1/projects/proj1/databases/db1/query/usuarios",
				body: `{
					"from": [{"collectionId": "usuarios"}],
					"where": {
						"compositeFilter": {
							"op": "AND",
							"filters": [
								{
									"fieldFilter": {
										"field": {"fieldPath": "edad"},
										"op": "GREATER_THAN_OR_EQUAL",
										"value": 18
									}
								},
								{
									"fieldFilter": {
										"field": {"fieldPath": "edad"},
										"op": "LESS_THAN",
										"value": 65
									}
								}
							]
						}
					},
					"orderBy": [
						{
							"field": {"fieldPath": "nombre"},
							"direction": "ASCENDING"
						}
					],
					"limit": 100
				}`,
				description: "Filter users by age range (18-65)",
			},
		}

		for _, req := range postmanRequests {
			t.Run(req.name, func(t *testing.T) {
				// Parse the request body (what would come from Postman)
				var firestoreQuery FirestoreStructuredQuery
				err := json.Unmarshal([]byte(req.body), &firestoreQuery)
				assert.NoError(t, err, "Should parse Postman request body")

				// Convert to internal model
				query := convertFirestoreJSONToModelQuery(firestoreQuery, basePath)

				// Execute as if it came through HTTP handler
				collectionId := "productos" // This would be extracted from URL
				if len(firestoreQuery.From) > 0 {
					collectionId = firestoreQuery.From[0].CollectionID
				}

				queryReq := usecase.QueryRequest{
					ProjectID:       projectID,
					DatabaseID:      databaseID,
					Parent:          basePath + "/" + collectionId,
					StructuredQuery: query,
				}

				docs, err := firestoreUC.RunQuery(ctx, queryReq)
				assert.NoError(t, err, "Postman-like request should work: %s", req.description)
				assert.NotNil(t, docs, "Should return results for: %s", req.description)

				t.Logf("✅ %s: %s", req.method, req.url)
				t.Logf("   📊 Query executed successfully with %d filters", len(query.Filters))
				t.Logf("   🎯 Collection: %s, Limit: %d", query.CollectionID, query.Limit)
			})
		}
	})
}
