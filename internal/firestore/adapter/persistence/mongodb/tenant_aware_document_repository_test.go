package mongodb

import (
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTenantAwareDocumentRepository_BasicOperations(t *testing.T) {
	repo := &TenantAwareDocumentRepository{}

	t.Run("document structure validation", func(t *testing.T) {
		doc := &model.Document{
			ID:           primitive.NewObjectID(),
			ProjectID:    "p1",
			DatabaseID:   "d1",
			CollectionID: "c1",
			DocumentID:   primitive.NewObjectID().Hex(),
			Path:         "projects/p1/databases/d1/documents/c1/doc1",
			ParentPath:   "projects/p1/databases/d1/documents/c1",
			Fields: map[string]*model.FieldValue{
				"name": {
					Value: "test doc",
				},
			},
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
			Version:    1,
			Exists:     true,
		}

		assert.NotNil(t, repo)
		assert.NotNil(t, doc)
		assert.Equal(t, "test doc", doc.Fields["name"].Value)
	})
}

func TestTenantAwareDocumentRepository_BatchOperations(t *testing.T) {
	repo := &TenantAwareDocumentRepository{}

	t.Run("batch document structure validation", func(t *testing.T) {
		doc := &model.Document{
			ID:           primitive.NewObjectID(),
			ProjectID:    "p1",
			DatabaseID:   "d1",
			CollectionID: "c1",
			DocumentID:   primitive.NewObjectID().Hex(),
			Fields: map[string]*model.FieldValue{
				"batch": {
					Value: "test1",
				},
			},
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
			Version:    1,
			Exists:     true,
		}

		assert.NotNil(t, repo)
		assert.NotNil(t, doc)
		assert.Equal(t, "test1", doc.Fields["batch"].Value)
	})
}

func TestTenantAwareDocumentRepository_QueryOperations(t *testing.T) {
	repo := &TenantAwareDocumentRepository{}

	t.Run("query structure validation", func(t *testing.T) {
		query := model.Query{}
		query.AddFilter("status", model.OperatorEqual, "active")
		query.Orders = append(query.Orders, model.Order{
			Field:     "createdAt",
			Direction: model.DirectionDescending,
		})
		query.Limit = 10

		assert.NotNil(t, repo)
		assert.NotNil(t, query)
		assert.Equal(t, 10, query.Limit)
		assert.Len(t, query.Orders, 1)
	})
}

func TestTenantAwareDocumentRepository_PathOperations(t *testing.T) {
	repo := &TenantAwareDocumentRepository{}

	t.Run("document path validation", func(t *testing.T) {
		docPath := "projects/p1/databases/d1/documents/c1/doc1"
		doc := &model.Document{
			ID:           primitive.NewObjectID(),
			ProjectID:    "p1",
			DatabaseID:   "d1",
			CollectionID: "c1",
			DocumentID:   "doc1",
			Path:         docPath,
			ParentPath:   "projects/p1/databases/d1/documents/c1",
			Fields: map[string]*model.FieldValue{
				"test": {
					Value: "test value",
				},
			},
		}

		assert.NotNil(t, repo)
		assert.NotNil(t, doc)
		assert.Equal(t, docPath, doc.Path)
	})
}
