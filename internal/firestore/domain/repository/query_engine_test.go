package repository

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
)

type mockQueryEngine struct{}

func (m *mockQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	return []*model.Document{{DocumentID: "doc1"}}, nil
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
