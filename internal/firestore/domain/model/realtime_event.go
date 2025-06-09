package model

import "time"

// EventType defines the type of real-time event.
type EventType string

const (
	// EventTypeCreated signifies a new document was created.
	EventTypeCreated EventType = "created"
	// EventTypeUpdated signifies an existing document was updated.
	EventTypeUpdated EventType = "updated"
	// EventTypeDeleted signifies a document was deleted.
	EventTypeDeleted EventType = "deleted"
)

// RealtimeEvent represents a real-time change to a document.
// This structure is sent to subscribed clients following Firestore real-time API structure.
type RealtimeEvent struct {
	// Type of the event (e.g., created, updated, deleted).
	Type EventType `json:"type"`

	// FullPath is the complete Firestore path: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{DOCUMENT_PATH}
	FullPath string `json:"fullPath"`

	// ProjectID is the project identifier
	ProjectID string `json:"projectId"`

	// DatabaseID is the database identifier
	DatabaseID string `json:"databaseId"`

	// DocumentPath is the relative path within the database (e.g., "users/user123")
	DocumentPath string `json:"documentPath"`

	// Data contains the document data associated with the event.
	// For 'deleted' events, this might be nil or just the ID.
	// For 'created' and 'updated' events, this will typically be the full document.
	Data map[string]interface{} `json:"data,omitempty"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// OldData contains the previous document data for update events
	OldData map[string]interface{} `json:"oldData,omitempty"`
}

// SubscriptionRequest represents a client subscription request
type SubscriptionRequest struct {
	// Action can be "subscribe" or "unsubscribe"
	Action string `json:"action"`

	// FullPath is the complete Firestore path to listen to
	FullPath string `json:"fullPath"`

	// IncludeMetadata whether to include metadata in events
	IncludeMetadata bool `json:"includeMetadata,omitempty"`

	// Query parameters for collection subscriptions
	Query *Query `json:"query,omitempty"`
}
