package model

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Collection represents a collection in Firestore following the hierarchy:
// projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}
type Collection struct {
	// MongoDB internal ID
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`

	// Firestore hierarchy identifiers
	ProjectID    string `json:"projectID" bson:"project_id"`
	DatabaseID   string `json:"databaseID" bson:"database_id"`
	CollectionID string `json:"collectionID" bson:"collection_id"`

	// Collection path information
	Path       string `json:"path" bson:"path"`              // Full path: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}
	ParentPath string `json:"parentPath" bson:"parent_path"` // Parent document path for subcollections

	// Collection metadata
	DisplayName string `json:"displayName,omitempty" bson:"display_name,omitempty"`
	Description string `json:"description,omitempty" bson:"description,omitempty"`

	// Collection statistics
	DocumentCount int64 `json:"documentCount" bson:"document_count"`
	StorageSize   int64 `json:"storageSize" bson:"storage_size"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`

	// Collection state
	IsActive bool `json:"isActive" bson:"is_active"`

	// Indexing information
	Indexes []CollectionIndex `json:"indexes,omitempty" bson:"indexes,omitempty"`

	// Security rules applied to this collection
	SecurityRules string `json:"securityRules,omitempty" bson:"security_rules,omitempty"`
}

// CollectionIndex represents an index on a collection
type CollectionIndex struct {
	Name   string       `json:"name" bson:"name"`
	Fields []IndexField `json:"fields" bson:"fields"`
	State  IndexState   `json:"state" bson:"state"`
}

// IndexStatistics represents statistics for an index
type IndexStatistics struct {
	IndexID       string    `json:"indexId" bson:"indexId"`
	IndexName     string    `json:"indexName" bson:"indexName"`
	DocumentCount int64     `json:"documentCount" bson:"documentCount"`
	StorageSize   int64     `json:"storageSize" bson:"storageSize"`
	LastUsed      time.Time `json:"lastUsed" bson:"lastUsed"`
}

// GetResourceName returns the full resource name for this collection
func (c *Collection) GetResourceName() string {
	return c.Path
}

// GetParentDocumentPath returns the parent document path for subcollections
func (c *Collection) GetParentDocumentPath() string {
	return c.ParentPath
}

// IsSubcollection returns true if this is a subcollection
func (c *Collection) IsSubcollection() bool {
	return c.ParentPath != ""
}

// NewCollection creates a new collection with the given parameters
func NewCollection(projectID, databaseID, collectionID string) *Collection {
	now := time.Now()
	path := "projects/" + projectID + "/databases/" + databaseID + "/documents/" + collectionID

	return &Collection{
		ProjectID:     projectID,
		DatabaseID:    databaseID,
		CollectionID:  collectionID,
		Path:          path,
		ParentPath:    "",
		DocumentCount: 0,
		StorageSize:   0,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsActive:      true,
		Indexes:       []CollectionIndex{},
	}
}

// NewSubcollection creates a new subcollection under a parent document
func NewSubcollection(projectID, databaseID, parentDocPath, collectionID string) *Collection {
	now := time.Now()
	path := parentDocPath + "/" + collectionID

	return &Collection{
		ProjectID:     projectID,
		DatabaseID:    databaseID,
		CollectionID:  collectionID,
		Path:          path,
		ParentPath:    parentDocPath,
		DocumentCount: 0,
		StorageSize:   0,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsActive:      true,
		Indexes:       []CollectionIndex{},
	}
}

// Collection validation errors
var (
	ErrInvalidCollectionPath = errors.New("invalid collection path")
	ErrCollectionNotFound    = errors.New("collection not found")
	ErrCollectionExists      = errors.New("collection already exists")
	ErrInvalidIndexName      = errors.New("invalid index name")
)
