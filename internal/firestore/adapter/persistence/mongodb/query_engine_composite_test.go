package mongodb

import (
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestBuildMongoFilter_SimpleFilter(t *testing.T) {
	filters := []model.Filter{
		{
			Field:    "name",
			Operator: model.OperatorEqual,
			Value:    "test",
		},
	}

	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{"name": "test"}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_MultipleSimpleFilters(t *testing.T) {
	filters := []model.Filter{
		{
			Field:    "name",
			Operator: model.OperatorEqual,
			Value:    "test",
		},
		{
			Field:    "price",
			Operator: model.OperatorGreaterThan,
			Value:    100.0,
		},
	}
	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"price": bson.M{"$gt": 100.0}},
		},
	}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_CompositeAND(t *testing.T) {
	filters := []model.Filter{
		{
			Composite: "and",
			SubFilters: []model.Filter{
				{
					Field:    "price",
					Operator: model.OperatorGreaterThanOrEqual,
					Value:    50.0,
				},
				{
					Field:    "price",
					Operator: model.OperatorLessThanOrEqual,
					Value:    500.0,
				},
			},
		},
	}

	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{
		"$and": []bson.M{
			{"price": bson.M{"$gte": 50.0}},
			{"price": bson.M{"$lte": 500.0}},
		},
	}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_CompositeOR(t *testing.T) {
	filters := []model.Filter{
		{
			Composite: "or",
			SubFilters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Peripherals",
				},
			},
		},
	}
	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{
		"$or": []bson.M{
			{"category": "Electronics"},
			{"category": "Peripherals"},
		},
	}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_NestedComposite(t *testing.T) {
	// Test AND filter containing OR filter
	filters := []model.Filter{
		{
			Composite: "and",
			SubFilters: []model.Filter{
				{
					Field:    "available",
					Operator: model.OperatorEqual,
					Value:    true,
				},
				{
					Composite: "or",
					SubFilters: []model.Filter{
						{
							Field:    "brand",
							Operator: model.OperatorEqual,
							Value:    "TechMaster",
						},
						{
							Field:    "brand",
							Operator: model.OperatorEqual,
							Value:    "MobileGenius",
						},
					},
				},
			},
		},
	}
	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{
		"$and": []bson.M{
			{"available": true},
			{
				"$or": []bson.M{
					{"brand": "TechMaster"},
					{"brand": "MobileGenius"},
				},
			},
		},
	}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_MixedCompositeAndSimple(t *testing.T) {
	// Test query with both composite and simple filters
	filters := []model.Filter{
		{
			Field:    "available",
			Operator: model.OperatorEqual,
			Value:    true,
		},
		{
			Composite: "or",
			SubFilters: []model.Filter{
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Electronics",
				},
				{
					Field:    "category",
					Operator: model.OperatorEqual,
					Value:    "Peripherals",
				},
			},
		},
	}
	filter := BuildSimpleMongoFilter(filters)
	expected := bson.M{
		"$and": []bson.M{
			{"available": true},
			{
				"$or": []bson.M{
					{"category": "Electronics"},
					{"category": "Peripherals"},
				},
			},
		},
	}
	assert.Equal(t, expected, filter)
}

func TestBuildMongoFilter_WithCursor(t *testing.T) {
	filters := []model.Filter{
		{
			Field:    "name",
			Operator: model.OperatorEqual,
			Value:    "test",
		}}
	// Test the filter building with cursor
	baseFilter := BuildSimpleMongoFilter(filters)
	cursorFilter := bson.M{"name": bson.M{"$gt": "cursor_value"}}

	// Test mergeFiltersWithAnd function
	result := mergeFiltersWithAnd(baseFilter, cursorFilter)

	expected := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"name": bson.M{"$gt": "cursor_value"}},
		},
	}
	assert.Equal(t, expected, result)
}

func TestMergeFiltersWithAnd_BothSimple(t *testing.T) {
	filter1 := bson.M{"name": "test"}
	filter2 := bson.M{"price": bson.M{"$gt": 100}}

	result := mergeFiltersWithAnd(filter1, filter2)
	expected := bson.M{
		"$and": []bson.M{filter1, filter2},
	}
	assert.Equal(t, expected, result)
}

func TestMergeFiltersWithAnd_FirstIsAnd(t *testing.T) {
	filter1 := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"category": "Electronics"},
		},
	}
	filter2 := bson.M{"price": bson.M{"$gt": 100}}

	result := mergeFiltersWithAnd(filter1, filter2)
	expected := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"category": "Electronics"},
			{"price": bson.M{"$gt": 100}},
		},
	}
	assert.Equal(t, expected, result)
}

func TestMergeFiltersWithAnd_SecondIsAnd(t *testing.T) {
	filter1 := bson.M{"name": "test"}
	filter2 := bson.M{
		"$and": []bson.M{
			{"category": "Electronics"},
			{"price": bson.M{"$gt": 100}},
		},
	}

	result := mergeFiltersWithAnd(filter1, filter2)
	expected := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"category": "Electronics"},
			{"price": bson.M{"$gt": 100}},
		},
	}
	assert.Equal(t, expected, result)
}

func TestMergeFiltersWithAnd_BothAreAnd(t *testing.T) {
	filter1 := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"available": true},
		},
	}
	filter2 := bson.M{
		"$and": []bson.M{
			{"category": "Electronics"},
			{"price": bson.M{"$gt": 100}},
		},
	}

	result := mergeFiltersWithAnd(filter1, filter2)
	expected := bson.M{
		"$and": []bson.M{
			{"name": "test"},
			{"available": true},
			{"category": "Electronics"},
			{"price": bson.M{"$gt": 100}},
		},
	}
	assert.Equal(t, expected, result)
}

func TestBuildFieldFilter_AllOperators(t *testing.T) {
	testCases := []struct {
		operator model.Operator
		value    interface{}
		expected bson.M
	}{
		{model.OperatorEqual, "test", bson.M{"field": "test"}},
		{model.OperatorNotEqual, "test", bson.M{"field": bson.M{"$ne": "test"}}},
		{model.OperatorLessThan, 100, bson.M{"field": bson.M{"$lt": 100}}},
		{model.OperatorLessThanOrEqual, 100, bson.M{"field": bson.M{"$lte": 100}}},
		{model.OperatorGreaterThan, 100, bson.M{"field": bson.M{"$gt": 100}}},
		{model.OperatorGreaterThanOrEqual, 100, bson.M{"field": bson.M{"$gte": 100}}},
		{model.OperatorArrayContains, "item", bson.M{"field": "item"}},
		{model.OperatorArrayContainsAny, []string{"a", "b"}, bson.M{"field": bson.M{"$in": []string{"a", "b"}}}},
		{model.OperatorIn, []string{"a", "b"}, bson.M{"field": bson.M{"$in": []string{"a", "b"}}}},
		{model.OperatorNotIn, []string{"a", "b"}, bson.M{"field": bson.M{"$nin": []string{"a", "b"}}}},
	}

	for _, tc := range testCases {
		t.Run(string(tc.operator), func(t *testing.T) {
			result := BuildSimpleFieldFilter("field", tc.operator, tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}
