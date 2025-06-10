package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealtimeEvent_Compile(t *testing.T) {
	// Placeholder: Add real realtime event model tests here
}

func TestRealtimeEvent_ModelFields(t *testing.T) {
	event := RealtimeEvent{
		Type:         EventTypeCreated,
		FullPath:     "projects/p1/databases/d1/documents/c1/doc1",
		ProjectID:    "p1",
		DatabaseID:   "d1",
		DocumentPath: "c1/doc1",
		Data:         map[string]interface{}{"field": "value"},
		Timestamp:    time.Now(),
		OldData:      map[string]interface{}{"field": "old"},
	}
	assert.Equal(t, EventTypeCreated, event.Type)
	assert.Equal(t, "p1", event.ProjectID)
	assert.Equal(t, "c1/doc1", event.DocumentPath)
	assert.Equal(t, "value", event.Data["field"])
	assert.Equal(t, "old", event.OldData["field"])
}

func TestSubscriptionRequest_ModelFields(t *testing.T) {
	req := SubscriptionRequest{
		Action:          "subscribe",
		FullPath:        "projects/p1/databases/d1/documents/c1",
		IncludeMetadata: true,
	}
	assert.Equal(t, "subscribe", req.Action)
	assert.True(t, req.IncludeMetadata)
}
