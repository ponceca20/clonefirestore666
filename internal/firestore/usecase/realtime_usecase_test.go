package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LegacyMockLogger implements the logger interface for testing
type LegacyMockLogger struct{}

func (m *LegacyMockLogger) Debug(args ...interface{})                              {}
func (m *LegacyMockLogger) Info(args ...interface{})                               {}
func (m *LegacyMockLogger) Warn(args ...interface{})                               {}
func (m *LegacyMockLogger) Error(args ...interface{})                              {}
func (m *LegacyMockLogger) Fatal(args ...interface{})                              {}
func (m *LegacyMockLogger) Debugf(format string, args ...interface{})              {}
func (m *LegacyMockLogger) Infof(format string, args ...interface{})               {}
func (m *LegacyMockLogger) Warnf(format string, args ...interface{})               {}
func (m *LegacyMockLogger) Errorf(format string, args ...interface{})              {}
func (m *LegacyMockLogger) Fatalf(format string, args ...interface{})              {}
func (m *LegacyMockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *LegacyMockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *LegacyMockLogger) WithComponent(component string) logger.Logger           { return m }

// newTestRealtimeUsecase creates a new realtime usecase for testing
func newTestRealtimeUsecase(t *testing.T) usecase.RealtimeUsecase {
	return usecase.NewRealtimeUsecase(&LegacyMockLogger{})
}

func TestRealtimeUsecase_SubscribeUnsubscribe(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID1 := "client1"
	subscriptionID1 := model.SubscriptionID("sub1")
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)

	// Subscribe using new request structure
	subscribeReq := usecase.SubscribeRequest{
		SubscriberID:   subscriberID1,
		SubscriptionID: subscriptionID1,
		FirestorePath:  path1,
		EventChannel:   eventChan1,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	}

	resp, err := rtu.Subscribe(ctx, subscribeReq)
	require.NoError(t, err, "Subscribe should not return an error")
	assert.NotNil(t, resp)
	assert.Equal(t, subscriptionID1, resp.SubscriptionID)
	assert.True(t, resp.InitialSnapshot)
	assert.NotZero(t, resp.CreatedAt)

	// Unsubscribe using new request structure
	unsubscribeReq := usecase.UnsubscribeRequest{
		SubscriberID:   subscriberID1,
		SubscriptionID: subscriptionID1,
	}
	err = rtu.Unsubscribe(ctx, unsubscribeReq)
	require.NoError(t, err, "Unsubscribe should not return an error")

	// Try to unsubscribe again (should be graceful)
	err = rtu.Unsubscribe(ctx, unsubscribeReq)
	require.NoError(t, err, "Unsubscribing a non-existent subscription should be graceful")

	// Try to unsubscribe a non-existent client
	nonExistentReq := usecase.UnsubscribeRequest{
		SubscriberID:   "nonExistentClient",
		SubscriptionID: subscriptionID1,
	}
	err = rtu.Unsubscribe(ctx, nonExistentReq)
	require.NoError(t, err, "Unsubscribing a non-existent client should be graceful")
}

func TestRealtimeUsecase_PublishEvent_SingleSubscriber(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID1 := "client1"
	subscriptionID1 := model.SubscriptionID("sub1")
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)

	resp, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   subscriberID1,
		SubscriptionID: subscriptionID1,
		FirestorePath:  path1,
		EventChannel:   eventChan1,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)

	event := model.RealtimeEvent{
		Type:         model.EventTypeModified,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event)
	require.NoError(t, err)

	select {
	case receivedEvent := <-eventChan1:
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.FullPath, receivedEvent.FullPath)
		assert.Equal(t, event.Data, receivedEvent.Data)
		// Enhanced features - resume token and sequence number should be populated
		assert.NotEmpty(t, receivedEvent.ResumeToken)
		assert.NotZero(t, receivedEvent.SequenceNumber)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestRealtimeUsecase_PublishEvent_MultipleSubscribersSamePath(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan1 := make(chan model.RealtimeEvent, 1)
	eventChan2 := make(chan model.RealtimeEvent, 1)

	// Subscribe first client
	resp1, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   "client1",
		SubscriptionID: model.SubscriptionID("sub1"),
		FirestorePath:  path1,
		EventChannel:   eventChan1,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	// Subscribe second client
	resp2, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   "client2",
		SubscriptionID: model.SubscriptionID("sub2"),
		FirestorePath:  path1,
		EventChannel:   eventChan2,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp2)

	event := model.RealtimeEvent{
		Type:         model.EventTypeModified,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case receivedEvent := <-eventChan1:
			assert.Equal(t, event.Type, receivedEvent.Type)
			assert.Equal(t, event.FullPath, receivedEvent.FullPath)
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Client1 timed out waiting for event")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case receivedEvent := <-eventChan2:
			assert.Equal(t, event.Type, receivedEvent.Type)
			assert.Equal(t, event.FullPath, receivedEvent.FullPath)
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Client2 timed out waiting for event")
		}
	}()

	wg.Wait()
}

func TestRealtimeUsecase_PublishEvent_DifferentPaths(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	eventChan1 := make(chan model.RealtimeEvent, 1)
	eventChan2 := make(chan model.RealtimeEvent, 1)
	path1 := "projects/test-project/databases/test-db/documents/path1"
	path2 := "projects/test-project/databases/test-db/documents/path2"

	// Subscribe first client
	resp1, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   "client1",
		SubscriptionID: model.SubscriptionID("sub1"),
		FirestorePath:  path1,
		EventChannel:   eventChan1,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	// Subscribe second client
	resp2, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   "client2",
		SubscriptionID: model.SubscriptionID("sub2"),
		FirestorePath:  path2,
		EventChannel:   eventChan2,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp2)

	eventForPath1 := model.RealtimeEvent{
		Type:         model.EventTypeModified,
		FullPath:     path1,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "path1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, eventForPath1)
	require.NoError(t, err)

	// Check path1 receives it
	select {
	case receivedEvent := <-eventChan1:
		assert.Equal(t, eventForPath1.Type, receivedEvent.Type)
		assert.Equal(t, eventForPath1.FullPath, receivedEvent.FullPath)
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Client1 (path1) timed out waiting for event")
	}

	// Check path2 does not receive it
	select {
	case receivedEvent := <-eventChan2:
		t.Fatalf("Client2 (path2) received event meant for path1: %+v", receivedEvent)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event
	}
}

func TestRealtimeUsecase_PublishEvent_NoSubscribers(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	event := model.RealtimeEvent{
		Type:         model.EventTypeModified,
		FullPath:     "projects/test-project/databases/test-db/documents/path/none",
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "path/none",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err := rtu.PublishEvent(ctx, event)
	assert.NoError(t, err, "Publishing to a path with no subscribers should not error")
}

func TestRealtimeUsecase_EnhancedFeatures(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID := "client1"
	subscriptionID := model.SubscriptionID("sub1")
	path := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan := make(chan model.RealtimeEvent, 1)

	// Test subscription with resume token
	resumeToken := model.ResumeToken("test-resume-token")
	resp, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   subscriberID,
		SubscriptionID: subscriptionID,
		FirestorePath:  path,
		EventChannel:   eventChan,
		ResumeToken:    resumeToken,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, subscriptionID, resp.SubscriptionID)

	// Test subscriber count
	count := rtu.GetSubscriberCount(path)
	assert.Equal(t, 1, count)

	// Test heartbeat
	err = rtu.SendHeartbeat(ctx)
	require.NoError(t, err)

	// Test permission validation
	err = rtu.ValidatePermissions(ctx, subscriberID, func(subscriberID, path string) error {
		// Mock permission validator that always passes
		return nil
	})
	require.NoError(t, err)

	// Test getting events since resume token
	events, err := rtu.GetEventsSince(ctx, path, resumeToken)
	require.NoError(t, err)
	assert.NotNil(t, events)

	// Test unsubscribe all
	err = rtu.UnsubscribeAll(ctx, subscriberID)
	require.NoError(t, err)

	// Verify subscriber count is 0
	count = rtu.GetSubscriberCount(path)
	assert.Equal(t, 0, count)
}

func TestRealtimeUsecase_MultipleSubscriptionsPerClient(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	subscriberID := "client1"
	path1 := "projects/test-project/databases/test-db/documents/docs/doc1"
	path2 := "projects/test-project/databases/test-db/documents/docs/doc2"
	eventChan1 := make(chan model.RealtimeEvent, 1)
	eventChan2 := make(chan model.RealtimeEvent, 1)

	// Subscribe to multiple paths
	resp1, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   subscriberID,
		SubscriptionID: model.SubscriptionID("sub1"),
		FirestorePath:  path1,
		EventChannel:   eventChan1,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	resp2, err := rtu.Subscribe(ctx, usecase.SubscribeRequest{
		SubscriberID:   subscriberID,
		SubscriptionID: model.SubscriptionID("sub2"),
		FirestorePath:  path2,
		EventChannel:   eventChan2,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
			IncludeOldData:  true,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp2)

	// Verify subscriber counts
	assert.Equal(t, 1, rtu.GetSubscriberCount(path1))
	assert.Equal(t, 1, rtu.GetSubscriberCount(path2))

	// Unsubscribe all should remove both
	err = rtu.UnsubscribeAll(ctx, subscriberID)
	require.NoError(t, err)

	assert.Equal(t, 0, rtu.GetSubscriberCount(path1))
	assert.Equal(t, 0, rtu.GetSubscriberCount(path2))
}

func TestRealtimeUsecase_MatchesQuery_AdvancedFiltering(t *testing.T) {
	rtu := newTestRealtimeUsecase(t)
	ctx := context.Background()
	path := "projects/test-project/databases/test-db/documents/docs/doc1"
	eventChan := make(chan model.RealtimeEvent, 1)

	// Filtro simple: key == "value"
	filter := model.Filter{Field: "key", Operator: model.OperatorEqual, Value: "value"}
	query := &model.Query{Filters: []model.Filter{filter}}

	subReq := usecase.SubscribeRequest{
		SubscriberID:   "client1",
		SubscriptionID: model.SubscriptionID("sub1"),
		FirestorePath:  path,
		EventChannel:   eventChan,
		Query:          query,
	}
	_, err := rtu.Subscribe(ctx, subReq)
	require.NoError(t, err)

	event := model.RealtimeEvent{
		Type:         model.EventTypeAdded,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event)
	require.NoError(t, err)
	select {
	case received := <-eventChan:
		assert.Equal(t, event.Data, received.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No event received for matching filter")
	}

	// Filtro compuesto: (key == "value" AND num > 10)
	filter2 := model.Filter{Field: "num", Operator: model.OperatorGreaterThan, Value: 10}
	comp := model.Filter{Composite: "and", SubFilters: []model.Filter{filter, filter2}}
	query2 := &model.Query{Filters: []model.Filter{comp}}
	eventChan2 := make(chan model.RealtimeEvent, 1)
	subReq2 := usecase.SubscribeRequest{
		SubscriberID:   "client2",
		SubscriptionID: model.SubscriptionID("sub2"),
		FirestorePath:  path,
		EventChannel:   eventChan2,
		Query:          query2,
	}
	_, err = rtu.Subscribe(ctx, subReq2)
	require.NoError(t, err)
	event2 := model.RealtimeEvent{
		Type:         model.EventTypeAdded,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value", "num": 20},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event2)
	require.NoError(t, err)
	select {
	case received := <-eventChan2:
		assert.Equal(t, event2.Data, received.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No event received for AND composite filter")
	}

	// Filtro OR: (key == "other" OR num == 20)
	or1 := model.Filter{Field: "key", Operator: model.OperatorEqual, Value: "other"}
	or2 := model.Filter{Field: "num", Operator: model.OperatorEqual, Value: 20}
	orComp := model.Filter{Composite: "or", SubFilters: []model.Filter{or1, or2}}
	query3 := &model.Query{Filters: []model.Filter{orComp}}
	eventChan3 := make(chan model.RealtimeEvent, 1)
	subReq3 := usecase.SubscribeRequest{
		SubscriberID:   "client3",
		SubscriptionID: model.SubscriptionID("sub3"),
		FirestorePath:  path,
		EventChannel:   eventChan3,
		Query:          query3,
	}
	_, err = rtu.Subscribe(ctx, subReq3)
	require.NoError(t, err)
	event3 := model.RealtimeEvent{
		Type:         model.EventTypeAdded,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "no", "num": 20},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event3)
	require.NoError(t, err)
	select {
	case received := <-eventChan3:
		assert.Equal(t, event3.Data, received.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No event received for OR composite filter")
	}

	// Proyección de campos: selectFields
	query4 := &model.Query{Filters: []model.Filter{filter}, SelectFields: []string{"key"}}
	eventChan4 := make(chan model.RealtimeEvent, 1)
	subReq4 := usecase.SubscribeRequest{
		SubscriberID:   "client4",
		SubscriptionID: model.SubscriptionID("sub4"),
		FirestorePath:  path,
		EventChannel:   eventChan4,
		Query:          query4,
	}
	_, err = rtu.Subscribe(ctx, subReq4)
	require.NoError(t, err)
	event4 := model.RealtimeEvent{
		Type:         model.EventTypeAdded,
		FullPath:     path,
		ProjectID:    "test-project",
		DatabaseID:   "test-db",
		DocumentPath: "docs/doc1",
		Data:         map[string]interface{}{"key": "value"},
		Timestamp:    time.Now(),
	}
	err = rtu.PublishEvent(ctx, event4)
	require.NoError(t, err)
	select {
	case received := <-eventChan4:
		assert.Equal(t, event4.Data, received.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No event received for selectFields projection")
	}
}
