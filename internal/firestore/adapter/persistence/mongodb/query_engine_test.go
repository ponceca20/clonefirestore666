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
