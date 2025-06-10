package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDatabase_Compile(t *testing.T) {
	// Placeholder: Add real database model tests here
}

func TestDatabase_ModelFields(t *testing.T) {
	db := &Database{
		ID:                       primitive.NewObjectID(),
		ProjectID:                "p1",
		DatabaseID:               "d1",
		DisplayName:              "Test DB",
		LocationID:               "us-central1",
		Type:                     DatabaseTypeFirestoreNative,
		ConcurrencyMode:          ConcurrencyModeOptimistic,
		AppEngineIntegrationMode: AppEngineIntegrationEnabled,
		KeyPrefix:                "prefix_",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
		State:                    DatabaseStateActive,
		DocumentCount:            10,
		StorageSize:              100,
	}
	assert.Equal(t, "p1", db.ProjectID)
	assert.Equal(t, DatabaseTypeFirestoreNative, db.Type)
	assert.Equal(t, DatabaseStateActive, db.State)
	assert.Equal(t, int64(10), db.DocumentCount)
}
