package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BatchWriteOperation represents a single operation in a batch write
type BatchWriteOperation struct {
	Type         BatchOperationType `json:"type" bson:"type"`
	DocumentID   string             `json:"documentId" bson:"documentId"`
	Path         string             `json:"path" bson:"path"` // Full Firestore path
	Data         map[string]any     `json:"data,omitempty" bson:"data,omitempty"`
	Mask         []string           `json:"mask,omitempty" bson:"mask,omitempty"` // For updates
	Precondition *Precondition      `json:"precondition,omitempty" bson:"precondition,omitempty"`
}

// BatchOperationType defines the type of batch operation
type BatchOperationType string

const (
	BatchOperationTypeCreate BatchOperationType = "create"
	BatchOperationTypeUpdate BatchOperationType = "update"
	BatchOperationTypeDelete BatchOperationType = "delete"
	BatchOperationTypeSet    BatchOperationType = "set"
)

// BatchWriteRequest represents a batch write request
type BatchWriteRequest struct {
	ProjectID  string                `json:"projectId" bson:"projectId"`
	DatabaseID string                `json:"databaseId" bson:"databaseId"`
	Operations []BatchWriteOperation `json:"operations" bson:"operations"`
	Labels     map[string]string     `json:"labels,omitempty" bson:"labels,omitempty"`
}

// BatchWriteResponse represents the response from a batch write operation
type BatchWriteResponse struct {
	WriteResults []WriteResult `json:"writeResults" bson:"writeResults"`
	Status       []Status      `json:"status" bson:"status"`
}

// WriteResult represents the result of a single write operation
type WriteResult struct {
	UpdateTime time.Time `json:"updateTime" bson:"updateTime"`
	Transform  []any     `json:"transform,omitempty" bson:"transform,omitempty"`
}

// Status represents the status of an operation
type Status struct {
	Code    int32  `json:"code" bson:"code"`
	Message string `json:"message" bson:"message"`
}

// Index represents a Firestore index
type Index struct {
	ID         string             `json:"id" bson:"_id,omitempty"`
	ObjectID   primitive.ObjectID `json:"-" bson:"objectId,omitempty"`
	ProjectID  string             `json:"projectId" bson:"projectId"`
	DatabaseID string             `json:"databaseId" bson:"databaseId"`
	Name       string             `json:"name" bson:"name"`
	Collection string             `json:"collection" bson:"collection"`
	Fields     []IndexField       `json:"fields" bson:"fields"`
	State      IndexState         `json:"state" bson:"state"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// IndexField represents a field in an index
type IndexField struct {
	Path  string          `json:"path" bson:"path"`
	Order IndexFieldOrder `json:"order" bson:"order"`
	Mode  IndexFieldMode  `json:"mode,omitempty" bson:"mode,omitempty"`
}

// IndexFieldOrder defines the order of an index field
type IndexFieldOrder string

const (
	IndexFieldOrderAscending  IndexFieldOrder = "ASCENDING"
	IndexFieldOrderDescending IndexFieldOrder = "DESCENDING"
)

// IndexFieldMode defines the mode of an index field
type IndexFieldMode string

const (
	IndexFieldModeArray IndexFieldMode = "ARRAY_CONTAINS"
)

// IndexState defines the state of an index
type IndexState string

const (
	IndexStateCreating    IndexState = "CREATING"
	IndexStateReady       IndexState = "READY"
	IndexStateNeedsRepair IndexState = "NEEDS_REPAIR"
	IndexStateError       IndexState = "ERROR"
)

// Subcollection represents metadata about a subcollection
type Subcollection struct {
	ID            string `json:"id" bson:"id"`
	Path          string `json:"path" bson:"path"`
	DocumentCount int64  `json:"documentCount" bson:"documentCount"`
}
