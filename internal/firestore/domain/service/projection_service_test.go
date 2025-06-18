package service

import (
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProjectionService(t *testing.T) {
	service := NewProjectionService()

	t.Run("ApplyProjection", func(t *testing.T) {
		t.Run("should return original documents when no projection fields", func(t *testing.T) {
			docs := createTestDocuments()
			result := service.ApplyProjection(docs, nil)
			assert.Equal(t, docs, result)

			result = service.ApplyProjection(docs, []string{})
			assert.Equal(t, docs, result)
		})

		t.Run("should filter fields based on projection", func(t *testing.T) {
			docs := createTestDocuments()
			projectionFields := []string{"name", "price"}

			result := service.ApplyProjection(docs, projectionFields)

			require.Len(t, result, len(docs))

			// Check first document
			doc := result[0]
			assert.Equal(t, docs[0].ID, doc.ID)
			assert.Equal(t, docs[0].DocumentID, doc.DocumentID)

			// Should only have projected fields
			assert.Len(t, doc.Fields, 2)
			assert.Contains(t, doc.Fields, "name")
			assert.Contains(t, doc.Fields, "price")
			assert.NotContains(t, doc.Fields, "description")
			assert.NotContains(t, doc.Fields, "stock")

			// Verify field values are preserved
			assert.Equal(t, docs[0].Fields["name"], doc.Fields["name"])
			assert.Equal(t, docs[0].Fields["price"], doc.Fields["price"])
		})

		t.Run("should handle single field projection", func(t *testing.T) {
			docs := createTestDocuments()
			projectionFields := []string{"name"}

			result := service.ApplyProjection(docs, projectionFields)

			require.Len(t, result, len(docs))
			doc := result[0]

			assert.Len(t, doc.Fields, 1)
			assert.Contains(t, doc.Fields, "name")
			assert.Equal(t, docs[0].Fields["name"], doc.Fields["name"])
		})

		t.Run("should handle non-existent fields gracefully", func(t *testing.T) {
			docs := createTestDocuments()
			projectionFields := []string{"nonexistent", "name"}

			result := service.ApplyProjection(docs, projectionFields)

			require.Len(t, result, len(docs))
			doc := result[0]

			// Should only include existing fields
			assert.Len(t, doc.Fields, 1)
			assert.Contains(t, doc.Fields, "name")
			assert.NotContains(t, doc.Fields, "nonexistent")
		})

		t.Run("should preserve document metadata", func(t *testing.T) {
			docs := createTestDocuments()
			projectionFields := []string{"name"}

			result := service.ApplyProjection(docs, projectionFields)

			require.Len(t, result, len(docs))
			original := docs[0]
			projected := result[0]

			// Verify all metadata is preserved
			assert.Equal(t, original.ID, projected.ID)
			assert.Equal(t, original.ProjectID, projected.ProjectID)
			assert.Equal(t, original.DatabaseID, projected.DatabaseID)
			assert.Equal(t, original.CollectionID, projected.CollectionID)
			assert.Equal(t, original.DocumentID, projected.DocumentID)
			assert.Equal(t, original.Path, projected.Path)
			assert.Equal(t, original.ParentPath, projected.ParentPath)
			assert.Equal(t, original.CreateTime, projected.CreateTime)
			assert.Equal(t, original.UpdateTime, projected.UpdateTime)
			assert.Equal(t, original.ReadTime, projected.ReadTime)
			assert.Equal(t, original.Version, projected.Version)
			assert.Equal(t, original.Exists, projected.Exists)
			assert.Equal(t, original.HasSubcollections, projected.HasSubcollections)
		})

		t.Run("should handle nil documents", func(t *testing.T) {
			result := service.ApplyProjection(nil, []string{"name"})
			assert.Nil(t, result)
		})

		t.Run("should handle empty documents list", func(t *testing.T) {
			result := service.ApplyProjection([]*model.Document{}, []string{"name"})
			assert.Empty(t, result)
		})
		t.Run("should handle document with nil fields", func(t *testing.T) {
			docs := []*model.Document{
				{
					ID:         primitive.NewObjectID(),
					DocumentID: "doc1",
					Fields:     nil,
				},
			}

			result := service.ApplyProjection(docs, []string{"name"})

			require.Len(t, result, 1)
			assert.Nil(t, result[0].Fields)
		})
	})

	t.Run("ValidateProjectionFields", func(t *testing.T) {
		t.Run("should validate empty field paths", func(t *testing.T) {
			err := service.ValidateProjectionFields([]string{"name", "", "price"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "field path cannot be empty")
		})

		t.Run("should accept valid field paths", func(t *testing.T) {
			err := service.ValidateProjectionFields([]string{"name", "price", "description"})
			assert.NoError(t, err)
		})

		t.Run("should handle nil fields", func(t *testing.T) {
			err := service.ValidateProjectionFields(nil)
			assert.NoError(t, err)
		})

		t.Run("should handle empty fields", func(t *testing.T) {
			err := service.ValidateProjectionFields([]string{})
			assert.NoError(t, err)
		})
	})

	t.Run("IsProjectionRequired", func(t *testing.T) {
		t.Run("should return false for nil fields", func(t *testing.T) {
			assert.False(t, service.IsProjectionRequired(nil))
		})

		t.Run("should return false for empty fields", func(t *testing.T) {
			assert.False(t, service.IsProjectionRequired([]string{}))
		})

		t.Run("should return true for non-empty fields", func(t *testing.T) {
			assert.True(t, service.IsProjectionRequired([]string{"name"}))
			assert.True(t, service.IsProjectionRequired([]string{"name", "price"}))
		})
	})
}

func TestProjectionServiceFunctionalHelpers(t *testing.T) {
	t.Run("normalizeFieldPaths", func(t *testing.T) {
		t.Run("should return copy of input", func(t *testing.T) {
			input := []string{"name", "price"}
			result := normalizeFieldPaths(input)

			assert.Equal(t, input, result)
			// Verify it's a copy, not the same slice
			result[0] = "modified"
			assert.NotEqual(t, input[0], result[0])
		})

		t.Run("should handle nil input", func(t *testing.T) {
			result := normalizeFieldPaths(nil)
			assert.Nil(t, result)
		})
	})
	t.Run("filterFields", func(t *testing.T) {
		t.Run("should filter fields correctly", func(t *testing.T) {
			originalFields := map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Test Product",
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     99.99,
				},
				"description": {
					ValueType: model.FieldTypeString,
					Value:     "Test Description",
				},
				"stock": {
					ValueType: model.FieldTypeInt,
					Value:     "10",
				},
			}

			projectionFields := []string{"name", "price"}
			result := filterFields(originalFields, projectionFields)

			assert.Len(t, result, 2)
			assert.Contains(t, result, "name")
			assert.Contains(t, result, "price")
			assert.NotContains(t, result, "description")
			assert.NotContains(t, result, "stock")
		})

		t.Run("should handle nil fields", func(t *testing.T) {
			result := filterFields(nil, []string{"name"})
			assert.Nil(t, result)
		})

		t.Run("should handle empty projection", func(t *testing.T) {
			originalFields := map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Test",
				},
			}
			result := filterFields(originalFields, []string{})
			assert.Empty(t, result)
		})
	})

	t.Run("getFieldByPath", func(t *testing.T) {
		fields := map[string]*model.FieldValue{
			"name": {
				ValueType: model.FieldTypeString,
				Value:     "Test Product",
			},
			"price": {
				ValueType: model.FieldTypeDouble,
				Value:     99.99,
			},
		}

		t.Run("should get existing field", func(t *testing.T) {
			value, exists := getFieldByPath(fields, "name")
			assert.True(t, exists)
			assert.Equal(t, fields["name"], value)
		})

		t.Run("should return false for non-existent field", func(t *testing.T) {
			value, exists := getFieldByPath(fields, "nonexistent")
			assert.False(t, exists)
			assert.Nil(t, value)
		})
	})

	t.Run("setFieldByPath", func(t *testing.T) {
		t.Run("should set field correctly", func(t *testing.T) {
			fields := make(map[string]*model.FieldValue)
			value := &model.FieldValue{
				ValueType: model.FieldTypeString,
				Value:     "Test",
			}

			setFieldByPath(fields, "name", value)

			assert.Contains(t, fields, "name")
			assert.Equal(t, value, fields["name"])
		})
	})
}

// Helper function to create test documents
func createTestDocuments() []*model.Document {
	now := time.Now()

	return []*model.Document{
		{
			ID:           primitive.NewObjectID(),
			ProjectID:    "test-project",
			DatabaseID:   "test-database",
			CollectionID: "products",
			DocumentID:   "doc1",
			Path:         "projects/test-project/databases/test-database/documents/products/doc1",
			ParentPath:   "projects/test-project/databases/test-database/documents/products",
			Fields: map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Test Product 1",
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     99.99,
				},
				"description": {
					ValueType: model.FieldTypeString,
					Value:     "A test product",
				},
				"stock": {
					ValueType: model.FieldTypeInt,
					Value:     "10",
				},
			},
			CreateTime:        now,
			UpdateTime:        now,
			ReadTime:          now,
			Version:           1,
			Exists:            true,
			HasSubcollections: false,
		},
		{
			ID:           primitive.NewObjectID(),
			ProjectID:    "test-project",
			DatabaseID:   "test-database",
			CollectionID: "products",
			DocumentID:   "doc2",
			Path:         "projects/test-project/databases/test-database/documents/products/doc2",
			ParentPath:   "projects/test-project/databases/test-database/documents/products",
			Fields: map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Test Product 2",
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     149.99,
				},
				"description": {
					ValueType: model.FieldTypeString,
					Value:     "Another test product",
				},
				"stock": {
					ValueType: model.FieldTypeInt,
					Value:     "5",
				},
			},
			CreateTime:        now,
			UpdateTime:        now,
			ReadTime:          now,
			Version:           1,
			Exists:            true,
			HasSubcollections: false,
		},
	}
}
