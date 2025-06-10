package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProject_Compile(t *testing.T) {
	// Placeholder: Add real project model tests here
}

func TestProject_ModelFields(t *testing.T) {
	id := primitive.NewObjectID()
	created := time.Now()
	updated := created.Add(time.Hour)
	pr := &Project{
		ID:            id,
		ProjectID:     "p1",
		DisplayName:   "Test Project",
		LocationID:    "us-central1",
		CreatedAt:     created,
		UpdatedAt:     updated,
		OwnerEmail:    "owner@example.com",
		Collaborators: []string{"collab1@example.com"},
		State:         ProjectStateActive,
	}
	assert.Equal(t, "p1", pr.ProjectID)
	assert.Equal(t, "Test Project", pr.DisplayName)
	assert.Equal(t, ProjectStateActive, pr.State)
	assert.Equal(t, "owner@example.com", pr.OwnerEmail)
	assert.Contains(t, pr.Collaborators, "collab1@example.com")
}

func TestProjectResources_ModelFields(t *testing.T) {
	res := &ProjectResources{
		Name:          "projects/p1",
		ProjectNumber: "123456",
		StorageQuota:  1000,
		DocumentQuota: 100,
		RequestQuota:  10000,
		StorageUsed:   10,
		DocumentsUsed: 2,
		RequestsToday: 5,
	}
	assert.Equal(t, "projects/p1", res.Name)
	assert.Equal(t, int64(1000), res.StorageQuota)
	assert.Equal(t, int64(2), res.DocumentsUsed)
}
