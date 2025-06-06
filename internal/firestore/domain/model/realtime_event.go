package model

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
// This structure is sent to subscribed clients.
type RealtimeEvent struct {
	// Type of the event (e.g., created, updated, deleted).
	Type EventType `json:"type"`

	// Path to the document that changed, e.g., "users/userID123" or "collection/docID/subCollection/subDocID".
	Path string `json:"path"`

	// Data contains the document data associated with the event.
	// For 'deleted' events, this might be nil or just the ID.
	// For 'created' and 'updated' events, this will typically be the full document.
	Data map[string]interface{} `json:"data"`
}
