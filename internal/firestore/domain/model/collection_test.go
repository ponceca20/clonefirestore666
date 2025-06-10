package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCollection_Compile(t *testing.T) {
	// Placeholder: Add real collection model tests here
}

func TestCollection_ModelFields(t *testing.T) {
	col := &Collection{
		ID:            primitive.NewObjectID(),
		ProjectID:     "p1",
		DatabaseID:    "d1",
		CollectionID:  "c1",
		Path:          "projects/p1/databases/d1/documents/c1",
		ParentPath:    "projects/p1/databases/d1/documents",
		DisplayName:   "Test Collection",
		Description:   "desc",
		DocumentCount: 5,
		StorageSize:   100,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		IsActive:      true,
		Indexes:       []CollectionIndex{{Name: "idx", State: IndexStateReady}},
		SecurityRules: "allow read;",
	}
	assert.Equal(t, "p1", col.ProjectID)
	assert.Equal(t, "c1", col.CollectionID)
	assert.True(t, col.IsActive)
	assert.Equal(t, "idx", col.Indexes[0].Name)
	assert.Equal(t, IndexStateReady, col.Indexes[0].State)
}
