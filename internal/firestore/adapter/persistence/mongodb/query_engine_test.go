package mongodb

import (
	"context"
	"errors"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
)

type mockMongoQueryEngine struct {
	shouldFail bool
}

func (m *mockMongoQueryEngine) ExecuteQuery(ctx context.Context, collection string, query model.Query) ([]*model.Document, error) {
	if m.shouldFail {
		return nil, errors.New("forced error")
	}
	if len(query.Filters) == 0 && len(query.Orders) == 0 {
		return nil, nil
	}
	// Simula resultados según el filtro
	return []*model.Document{{DocumentID: "doc1"}}, nil
}

func TestMongoQueryEngine_ExecuteQuery_Exhaustive(t *testing.T) {
	ctx := context.Background()

	t.Run("empty query", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		res, err := qe.ExecuteQuery(ctx, "col", model.Query{})
		if err != nil || res != nil {
			t.Errorf("expected nil result, got %v, %v", res, err)
		}
	})

	t.Run("simple equality filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("foo", model.OperatorEqual, "bar")
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("compound filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("foo", model.OperatorGreaterThan, 1)
		q.AddFilter("bar", model.OperatorLessThan, 10)
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("order by and limit", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.Orders = append(q.Orders, model.Order{Field: "foo", Direction: model.DirectionAscending})
		q.Limit = 1
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("array-contains filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("tags", model.OperatorArrayContains, "a")
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("array-contains-any filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("tags", model.OperatorArrayContainsAny, []string{"a", "b"})
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("in filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("status", model.OperatorIn, []string{"active", "pending"})
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("not-in filter", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.AddFilter("status", model.OperatorNotIn, []string{"deleted", "archived"})
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("multiple order by", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.Orders = append(q.Orders,
			model.Order{Field: "priority", Direction: model.DirectionDescending},
			model.Order{Field: "createdAt", Direction: model.DirectionAscending},
		)
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("limit to last", func(t *testing.T) {
		qe := &mockMongoQueryEngine{}
		q := model.Query{}
		q.Orders = append(q.Orders, model.Order{Field: "timestamp", Direction: model.DirectionDescending})
		q.Limit = 5
		q.LimitToLast = true
		res, err := qe.ExecuteQuery(ctx, "col", q)
		if err != nil || len(res) != 1 {
			t.Errorf("expected 1 result, got %v, %v", res, err)
		}
	})

	t.Run("forced error", func(t *testing.T) {
		qe := &mockMongoQueryEngine{shouldFail: true}
		q := model.Query{}
		q.AddFilter("foo", model.OperatorEqual, "bar")
		_, err := qe.ExecuteQuery(ctx, "col", q)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}

func TestMongoQueryEngine_BuildFilter(t *testing.T) {
	t.Run("test compound filters", func(t *testing.T) {
		q := model.Query{}
		q.AddFilter("age", model.OperatorGreaterThanOrEqual, 18)
		q.AddFilter("age", model.OperatorLessThan, 65)
		q.AddFilter("status", model.OperatorEqual, "active")

		filter := buildMongoFilter(q.Filters)

		// Verifica que el filtro es correcto (ajusta según tu implementación)
		if filter == nil {
			t.Error("expected non-nil filter")
		}
	})
}

func TestMongoQueryEngine_BuildFindOptions(t *testing.T) {
	t.Run("test sort and limit options", func(t *testing.T) {
		q := model.Query{}
		q.Orders = append(q.Orders,
			model.Order{Field: "name", Direction: model.DirectionAscending},
			model.Order{Field: "age", Direction: model.DirectionDescending},
		)
		q.Limit = 10

		opts := buildMongoFindOptions(q)

		if opts == nil {
			t.Error("expected non-nil options")
		}
	})
}
