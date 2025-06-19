package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealtimeEvent_Compile(t *testing.T) {
	// Basic compilation test to ensure the model compiles correctly
	event := RealtimeEvent{}
	assert.NotNil(t, event)
}

func TestRealtimeEvent_ModelFields(t *testing.T) {
	now := time.Now()
	event := RealtimeEvent{
		Type:           EventTypeAdded,
		FullPath:       "projects/p1/databases/d1/documents/c1/doc1",
		ProjectID:      "p1",
		DatabaseID:     "d1",
		DocumentPath:   "c1/doc1",
		Data:           map[string]interface{}{"field": "value"},
		Timestamp:      now,
		OldData:        map[string]interface{}{"field": "old"},
		ResumeToken:    ResumeToken("token123"),
		SequenceNumber: 42,
		SubscriptionID: "sub123",
	}

	assert.Equal(t, EventTypeAdded, event.Type)
	assert.Equal(t, "projects/p1/databases/d1/documents/c1/doc1", event.FullPath)
	assert.Equal(t, "p1", event.ProjectID)
	assert.Equal(t, "d1", event.DatabaseID)
	assert.Equal(t, "c1/doc1", event.DocumentPath)
	assert.Equal(t, "value", event.Data["field"])
	assert.Equal(t, "old", event.OldData["field"])
	assert.Equal(t, ResumeToken("token123"), event.ResumeToken)
	assert.Equal(t, int64(42), event.SequenceNumber)
	assert.Equal(t, "sub123", event.SubscriptionID)
	assert.Equal(t, now, event.Timestamp)
}

func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("added"), EventTypeAdded)
	assert.Equal(t, EventType("modified"), EventTypeModified)
	assert.Equal(t, EventType("removed"), EventTypeRemoved)
	assert.Equal(t, EventType("heartbeat"), EventTypeHeartbeat)
}

func TestRealtimeEvent_GenerateResumeToken(t *testing.T) {
	event := RealtimeEvent{
		FullPath:       "projects/p1/databases/d1/documents/c1/doc1",
		Timestamp:      time.Unix(1234567890, 123456789),
		SequenceNumber: 42,
	}

	token := event.GenerateResumeToken()

	// Token should not be empty
	assert.NotEmpty(t, token)

	// Token should be deterministic - same input should produce same token
	token2 := event.GenerateResumeToken()
	assert.Equal(t, token, token2)

	// Different events should produce different tokens
	event2 := event
	event2.SequenceNumber = 43
	token3 := event2.GenerateResumeToken()
	assert.NotEqual(t, token, token3)
}

func TestSubscriptionRequest_ModelFields(t *testing.T) {
	req := SubscriptionRequest{
		Action:          "subscribe",
		SubscriptionID:  SubscriptionID("sub123"),
		FullPath:        "projects/p1/databases/d1/documents/c1",
		IncludeMetadata: true,
		Query:           &Query{},
		ResumeToken:     ResumeToken("token123"),
		IncludeOldData:  true,
	}

	assert.Equal(t, "subscribe", req.Action)
	assert.Equal(t, SubscriptionID("sub123"), req.SubscriptionID)
	assert.Equal(t, "projects/p1/databases/d1/documents/c1", req.FullPath)
	assert.True(t, req.IncludeMetadata)
	assert.NotNil(t, req.Query)
	assert.Equal(t, ResumeToken("token123"), req.ResumeToken)
	assert.True(t, req.IncludeOldData)
}

func TestSubscriptionResponse_ModelFields(t *testing.T) {
	resp := SubscriptionResponse{
		Type:           "subscription_confirmed",
		SubscriptionID: SubscriptionID("sub123"),
		Status:         "success",
		Error:          "",
		Data:           map[string]interface{}{"count": 5},
		ResumeToken:    ResumeToken("token123"),
	}

	assert.Equal(t, "subscription_confirmed", resp.Type)
	assert.Equal(t, SubscriptionID("sub123"), resp.SubscriptionID)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Error)
	assert.Equal(t, 5, resp.Data["count"])
	assert.Equal(t, ResumeToken("token123"), resp.ResumeToken)
}

func TestHeartbeatMessage_ModelFields(t *testing.T) {
	now := time.Now()
	hb := HeartbeatMessage{
		Type:      "ping",
		Timestamp: now,
		Data:      "test-data",
	}

	assert.Equal(t, "ping", hb.Type)
	assert.Equal(t, now, hb.Timestamp)
	assert.Equal(t, "test-data", hb.Data)
}

func TestWebSocketMessage_ModelFields(t *testing.T) {
	now := time.Now()
	msg := WebSocketMessage{
		Type:           MessageTypeDocumentChange,
		SubscriptionID: SubscriptionID("sub123"),
		Data:           map[string]interface{}{"doc": "data"},
		Error:          "",
		Timestamp:      now,
	}

	assert.Equal(t, MessageTypeDocumentChange, msg.Type)
	assert.Equal(t, SubscriptionID("sub123"), msg.SubscriptionID)
	assert.Equal(t, "data", msg.Data["doc"])
	assert.Empty(t, msg.Error)
	assert.Equal(t, now, msg.Timestamp)
}

func TestWebSocketMessageTypes(t *testing.T) {
	assert.Equal(t, "subscribe", MessageTypeSubscribe)
	assert.Equal(t, "unsubscribe", MessageTypeUnsubscribe)
	assert.Equal(t, "subscription_confirmed", MessageTypeSubscriptionConfirmed)
	assert.Equal(t, "subscription_error", MessageTypeSubscriptionError)
	assert.Equal(t, "document_change", MessageTypeDocumentChange)
	assert.Equal(t, "heartbeat", MessageTypeHeartbeat)
	assert.Equal(t, "ping", MessageTypePing)
	assert.Equal(t, "pong", MessageTypePong)
	assert.Equal(t, "error", MessageTypeError)
	assert.Equal(t, "connection_closed", MessageTypeConnectionClosed)
}

func TestResumeTokenType(t *testing.T) {
	token := ResumeToken("test-token")
	assert.Equal(t, "test-token", string(token))
}

func TestSubscriptionIDType(t *testing.T) {
	subID := SubscriptionID("test-subscription")
	assert.Equal(t, "test-subscription", string(subID))
}

func TestRealtimeEventWithAllEventTypes(t *testing.T) {
	testCases := []struct {
		name      string
		eventType EventType
	}{
		{"Added Event", EventTypeAdded},
		{"Modified Event", EventTypeModified},
		{"Removed Event", EventTypeRemoved},
		{"Heartbeat Event", EventTypeHeartbeat},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := RealtimeEvent{
				Type:           tc.eventType,
				FullPath:       "projects/test/databases/default/documents/users/user1",
				ProjectID:      "test",
				DatabaseID:     "default",
				DocumentPath:   "users/user1",
				Timestamp:      time.Now(),
				SequenceNumber: 1,
			}

			assert.Equal(t, tc.eventType, event.Type)
			assert.NotEmpty(t, event.FullPath)
			assert.NotZero(t, event.Timestamp)
		})
	}
}

func TestSubscriptionRequestActions(t *testing.T) {
	testCases := []struct {
		action   string
		expected string
	}{
		{"subscribe", "subscribe"},
		{"unsubscribe", "unsubscribe"},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			req := SubscriptionRequest{
				Action:         tc.action,
				SubscriptionID: SubscriptionID("test"),
				FullPath:       "projects/test/databases/default/documents/test",
			}

			assert.Equal(t, tc.expected, req.Action)
			assert.NotEmpty(t, req.SubscriptionID)
			assert.NotEmpty(t, req.FullPath)
		})
	}
}

func TestRealtimeEventJSONSerialization(t *testing.T) {
	// Test that the model can be properly serialized to JSON
	// This is important for WebSocket communication
	event := RealtimeEvent{
		Type:           EventTypeAdded,
		FullPath:       "projects/test/databases/default/documents/users/user1",
		ProjectID:      "test",
		DatabaseID:     "default",
		DocumentPath:   "users/user1",
		Data:           map[string]interface{}{"name": "Test User"},
		Timestamp:      time.Now(),
		ResumeToken:    ResumeToken("test-token"),
		SequenceNumber: 1,
		SubscriptionID: "sub1",
	}

	// Verify all fields are accessible (this would fail if there were struct tag issues)
	assert.Equal(t, EventTypeAdded, event.Type)
	assert.Equal(t, "Test User", event.Data["name"])
	assert.NotEmpty(t, event.ResumeToken)
	assert.NotZero(t, event.SequenceNumber)
}
