package model

import "time"

// Document represents a document in Firestore.
type Document struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	// Path is the full path to the document, e.g., "users/userId/posts/postId"
	Path string
}

// FieldValue represents special server-side values like ServerTimestamp.
type FieldValue string

const (
	// ServerTimestamp is a sentinel value to set a field to the server's timestamp.
	ServerTimestamp FieldValue = "ServerTimestamp"
)

// GeoPoint represents a geographical point.
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Reference represents a reference to another document.
type Reference string

// WriteOperationType defines the type of a write operation in a batch.
type WriteOperationType string

const (
	WriteTypeCreate WriteOperationType = "CREATE"
	WriteTypeUpdate WriteOperationType = "UPDATE"
	WriteTypeDelete WriteOperationType = "DELETE"
	// WriteTypeSet WriteOperationType = "SET" // Create or overwrite
)

// WriteOperation represents a single operation in a batch write.
type WriteOperation struct {
	Type WriteOperationType     `json:"type"`
	Path string                 `json:"path"`           // Full document path
	Data map[string]interface{} `json:"data,omitempty"` // Used for Create and Update
}
