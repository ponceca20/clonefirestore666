package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDocument_ModelFields(t *testing.T) {
	doc := Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Path:         "projects/p1/databases/d1/documents/c1/doc1",
		ParentPath:   "projects/p1/databases/d1/documents/c1",
		Fields: map[string]*FieldValue{
			"field": {ValueType: FieldTypeString, Value: "value"},
		},
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
		ReadTime:          time.Now(),
		Version:           1,
		ETag:              "etag1",
		Exists:            true,
		HasSubcollections: true,
		Subcollections:    []string{"sub1"},
	}
	assert.Equal(t, "p1", doc.ProjectID)
	assert.Equal(t, "doc1", doc.DocumentID)
	assert.True(t, doc.Exists)
	assert.Equal(t, FieldTypeString, doc.Fields["field"].ValueType)
}

func TestFieldValue_Types(t *testing.T) {
	fv := FieldValue{ValueType: FieldTypeInt, Value: 42}
	assert.Equal(t, FieldTypeInt, fv.ValueType)
	assert.Equal(t, 42, fv.Value)
}
