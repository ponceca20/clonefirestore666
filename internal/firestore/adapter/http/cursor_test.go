package http

import (
	"firestore-clone/internal/firestore/domain/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertFirestoreJSONToModelQuery_CursorPagination(t *testing.T) {
	tests := []struct {
		name     string
		query    FirestoreStructuredQuery
		expected func(*testing.T, *model.Query)
	}{
		{
			name: "startAfter with timestamp",
			query: FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "productos2"}},
				OrderBy: []FirestoreOrder{
					{
						Field:     FirestoreFieldReference{FieldPath: "fechaFabricacion"},
						Direction: "DESCENDING",
					},
				},
				StartAfter: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{
							"timestampValue": "2025-04-01T06:00:00-05:00",
						},
					},
				},
				Limit: 10,
			},
			expected: func(t *testing.T, query *model.Query) {
				assert.Equal(t, "productos2", query.CollectionID)
				assert.Len(t, query.Orders, 1)
				assert.Equal(t, "fechaFabricacion", query.Orders[0].Field)
				assert.Equal(t, model.DirectionDescending, query.Orders[0].Direction)
				assert.Equal(t, 10, query.Limit)

				require.Len(t, query.StartAfter, 1)
				// The timestamp should be parsed as a time.Time
				assert.IsType(t, time.Time{}, query.StartAfter[0])
			},
		},
		{
			name: "startAt with string value",
			query: FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "users"}},
				OrderBy: []FirestoreOrder{
					{
						Field:     FirestoreFieldReference{FieldPath: "name"},
						Direction: "ASCENDING",
					},
				},
				StartAt: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{
							"stringValue": "Alice",
						},
					},
				},
				Limit: 5,
			},
			expected: func(t *testing.T, query *model.Query) {
				assert.Equal(t, "users", query.CollectionID)
				assert.Len(t, query.Orders, 1)
				assert.Equal(t, "name", query.Orders[0].Field)
				assert.Equal(t, model.DirectionAscending, query.Orders[0].Direction)
				assert.Equal(t, 5, query.Limit)

				require.Len(t, query.StartAt, 1)
				assert.Equal(t, "Alice", query.StartAt[0])
			},
		},
		{
			name: "endBefore with integer value",
			query: FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "products"}},
				OrderBy: []FirestoreOrder{
					{
						Field:     FirestoreFieldReference{FieldPath: "price"},
						Direction: "ASCENDING",
					},
				},
				EndBefore: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{
							"integerValue": "100",
						},
					},
				},
				Limit: 20,
			},
			expected: func(t *testing.T, query *model.Query) {
				assert.Equal(t, "products", query.CollectionID)
				assert.Len(t, query.Orders, 1)
				assert.Equal(t, "price", query.Orders[0].Field)
				assert.Equal(t, model.DirectionAscending, query.Orders[0].Direction)
				assert.Equal(t, 20, query.Limit)

				require.Len(t, query.EndBefore, 1)
				assert.Equal(t, int64(100), query.EndBefore[0])
			},
		},
		{
			name: "multi-field orderBy with startAfter",
			query: FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "products"}},
				OrderBy: []FirestoreOrder{
					{
						Field:     FirestoreFieldReference{FieldPath: "price"},
						Direction: "DESCENDING",
					},
					{
						Field:     FirestoreFieldReference{FieldPath: "name"},
						Direction: "ASCENDING",
					},
				},
				StartAfter: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{"doubleValue": 299.99},
						map[string]interface{}{"stringValue": "Product A"},
					},
				},
				Limit: 15,
			},
			expected: func(t *testing.T, query *model.Query) {
				assert.Equal(t, "products", query.CollectionID)
				assert.Len(t, query.Orders, 2)

				// First order: price DESC
				assert.Equal(t, "price", query.Orders[0].Field)
				assert.Equal(t, model.DirectionDescending, query.Orders[0].Direction)

				// Second order: name ASC
				assert.Equal(t, "name", query.Orders[1].Field)
				assert.Equal(t, model.DirectionAscending, query.Orders[1].Direction)

				assert.Equal(t, 15, query.Limit)

				require.Len(t, query.StartAfter, 2)
				assert.Equal(t, 299.99, query.StartAfter[0])
				assert.Equal(t, "Product A", query.StartAfter[1])
			},
		},
		{
			name: "all cursor types",
			query: FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "test"}},
				OrderBy: []FirestoreOrder{
					{
						Field:     FirestoreFieldReference{FieldPath: "id"},
						Direction: "ASCENDING",
					},
				},
				StartAt: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{"stringValue": "start"},
					},
				},
				StartAfter: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{"stringValue": "startAfter"},
					},
				},
				EndAt: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{"stringValue": "end"},
					},
				},
				EndBefore: &FirestoreCursor{
					Values: []interface{}{
						map[string]interface{}{"stringValue": "endBefore"},
					},
				},
				Limit: 10,
			},
			expected: func(t *testing.T, query *model.Query) {
				assert.Equal(t, "test", query.CollectionID)
				assert.Equal(t, 10, query.Limit)

				require.Len(t, query.StartAt, 1)
				assert.Equal(t, "start", query.StartAt[0])

				require.Len(t, query.StartAfter, 1)
				assert.Equal(t, "startAfter", query.StartAfter[0])

				require.Len(t, query.EndAt, 1)
				assert.Equal(t, "end", query.EndAt[0])

				require.Len(t, query.EndBefore, 1)
				assert.Equal(t, "endBefore", query.EndBefore[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := convertFirestoreJSONToModelQuery(tt.query)
			require.NoError(t, err)
			require.NotNil(t, query)

			tt.expected(t, query)
		})
	}
}

func TestConvertFirestoreCursorValues(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name:     "nil values",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty values",
			input:    []interface{}{},
			expected: []interface{}{},
		},
		{
			name: "mixed typed values",
			input: []interface{}{
				map[string]interface{}{"stringValue": "test"},
				map[string]interface{}{"integerValue": "42"},
				map[string]interface{}{"doubleValue": 3.14},
				map[string]interface{}{"booleanValue": true},
			},
			expected: []interface{}{"test", int64(42), 3.14, true},
		},
		{
			name: "timestamp value",
			input: []interface{}{
				map[string]interface{}{"timestampValue": "2025-04-01T06:00:00-05:00"},
			},
			expected: func() []interface{} {
				t, _ := time.Parse(time.RFC3339, "2025-04-01T06:00:00-05:00")
				return []interface{}{t}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertFirestoreCursorValues(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
