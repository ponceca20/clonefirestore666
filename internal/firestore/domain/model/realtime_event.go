package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// EventType defines the type of real-time event.
type EventType string

const (
	// EventTypeAdded signifies a new document was added (Firestore naming)
	EventTypeAdded EventType = "added"
	// EventTypeModified signifies an existing document was modified (Firestore naming)
	EventTypeModified EventType = "modified"
	// EventTypeRemoved signifies a document was removed (Firestore naming)
	EventTypeRemoved EventType = "removed"
	// EventTypeHeartbeat is used for connection health checks
	EventTypeHeartbeat EventType = "heartbeat"
)

// ResumeToken represents a token that allows resuming a stream from a specific point
type ResumeToken string

// RealtimeEvent represents a real-time change to a document.
// This structure is sent to subscribed clients following Firestore real-time API structure.
type RealtimeEvent struct {
	// Type of the event (e.g., added, modified, removed).
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
	// For 'removed' events, this might be nil or just the ID.
	// For 'added' and 'modified' events, this will typically be the full document.
	Data map[string]interface{} `json:"data,omitempty"`

	// Timestamp when the event occurred (nanosecond precision for ordering)
	Timestamp time.Time `json:"timestamp"`

	// OldData contains the previous document data for update events
	OldData map[string]interface{} `json:"oldData,omitempty"`

	// ResumeToken allows clients to resume from this event
	ResumeToken ResumeToken `json:"resumeToken"`

	// SequenceNumber ensures ordering of events (monotonically increasing)
	SequenceNumber int64 `json:"sequenceNumber"`

	// SubscriptionID identifies which subscription this event belongs to
	SubscriptionID string `json:"subscriptionId,omitempty"`
}

// GenerateResumeToken creates a resume token based on timestamp and sequence
func (e *RealtimeEvent) GenerateResumeToken() ResumeToken {
	data := fmt.Sprintf("%d_%d_%s", e.Timestamp.UnixNano(), e.SequenceNumber, e.FullPath)
	hash := sha256.Sum256([]byte(data))
	return ResumeToken(hex.EncodeToString(hash[:16])) // Use first 16 bytes for shorter token
}

// SubscriptionID represents a unique identifier for a subscription
type SubscriptionID string

// SubscriptionRequest represents a client subscription request
type SubscriptionRequest struct {
	// Action can be "subscribe" or "unsubscribe"
	Action string `json:"action"`

	// SubscriptionID uniquely identifies this subscription within the connection
	SubscriptionID SubscriptionID `json:"subscriptionId"`

	// FullPath is the complete Firestore path to listen to
	FullPath string `json:"fullPath"`

	// IncludeMetadata whether to include metadata in events
	IncludeMetadata bool `json:"includeMetadata,omitempty"`

	// Query parameters for collection subscriptions
	Query *Query `json:"query,omitempty"`

	// ResumeToken allows resuming from a specific point in the stream
	ResumeToken ResumeToken `json:"resumeToken,omitempty"`

	// IncludeOldData whether to include old data in modification events
	IncludeOldData bool `json:"includeOldData,omitempty"`
}

// SubscriptionResponse represents the server's response to a subscription request
type SubscriptionResponse struct {
	// Type of response (subscription_confirmed, subscription_error, etc.)
	Type string `json:"type"`

	// SubscriptionID echoes back the subscription ID
	SubscriptionID SubscriptionID `json:"subscriptionId"`

	// Status indicates success or failure
	Status string `json:"status"`

	// Error message if subscription failed
	Error string `json:"error,omitempty"`

	// Data contains additional response data
	Data map[string]interface{} `json:"data,omitempty"`

	// ResumeToken for this subscription (if applicable)
	ResumeToken ResumeToken `json:"resumeToken,omitempty"`
}

// HeartbeatMessage represents a heartbeat ping/pong message
type HeartbeatMessage struct {
	Type      string    `json:"type"` // "ping" or "pong"
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data,omitempty"`
}

// WebSocketMessage represents the envelope for all WebSocket messages
type WebSocketMessage struct {
	Type           string                 `json:"type"`
	SubscriptionID SubscriptionID         `json:"subscriptionId,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// Constants for WebSocket message types
const (
	MessageTypeSubscribe             = "subscribe"
	MessageTypeUnsubscribe           = "unsubscribe"
	MessageTypeSubscriptionConfirmed = "subscription_confirmed"
	MessageTypeSubscriptionError     = "subscription_error"
	MessageTypeDocumentChange        = "document_change"
	MessageTypeHeartbeat             = "heartbeat"
	MessageTypePing                  = "ping"
	MessageTypePong                  = "pong"
	MessageTypeError                 = "error"
	MessageTypeConnectionClosed      = "connection_closed"
)
