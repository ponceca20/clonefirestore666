package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_ModelFields(t *testing.T) {
	q := Query{
		Path:           "projects/p1/databases/d1/documents/c1",
		CollectionID:   "c1",
		Filters:        []Filter{{Field: "age", Operator: OperatorGreaterThan, Value: 18}},
		Orders:         []Order{{Field: "name", Direction: "ASCENDING"}},
		Limit:          10,
		Offset:         2,
		SelectFields:   []string{"name", "age"},
		AllDescendants: true,
		LimitToLast:    false,
	}
	assert.Equal(t, "c1", q.CollectionID)
	assert.Equal(t, 10, q.Limit)
	assert.True(t, q.AllDescendants)
	assert.Equal(t, "age", q.Filters[0].Field)
	assert.Equal(t, OperatorGreaterThan, q.Filters[0].Operator)
	assert.Equal(t, "name", q.Orders[0].Field)
}

func TestFilter_Composite(t *testing.T) {
	f := Filter{
		Composite: "and",
		SubFilters: []Filter{
			{Field: "active", Operator: OperatorEqual, Value: true},
			{Field: "age", Operator: OperatorGreaterThan, Value: 18},
		},
	}
	assert.Equal(t, "and", f.Composite)
	assert.Len(t, f.SubFilters, 2)
}
