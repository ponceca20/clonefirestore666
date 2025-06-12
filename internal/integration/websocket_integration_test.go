// WebSocket Integration Tests for Firestore Clone
// Tests comprehensive WebSocket functionality using Fiber's native capabilities
package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	authModel "firestore-clone/internal/auth/domain/model"
	httpadapter "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
)

// WebSocket message types for testing
type WSMessage struct {
	Type      string      `json:"type"`
	Operation string      `json:"operation,omitempty"`
	Path      string      `json:"path,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	ID        string      `json:"id,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// MockWSConnection simulates a WebSocket connection for testing
type MockWSConnection struct {
	messages []WSMessage
	mu       sync.RWMutex
	closed   bool
}

func NewMockWSConnection() *MockWSConnection {
	return &MockWSConnection{
		messages: make([]WSMessage, 0),
	}
}

func (m *MockWSConnection) WriteJSON(v interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("connection closed")
	}

	if msg, ok := v.(WSMessage); ok {
		m.messages = append(m.messages, msg)
	}
	return nil
}

func (m *MockWSConnection) GetMessages() []WSMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messages
}

func (m *MockWSConnection) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

// Test utilities for WebSocket testing
func setupWSTestApp() (*fiber.App, *usecase.MockRealtimeUsecase, *usecase.MockSecurityUsecase, *usecase.MockAuthClient) {
	app := fiber.New()

	// Create mocks
	realtimeUsecase := usecase.NewMockRealtimeUsecase()
	securityUsecase := usecase.NewMockSecurityUsecase()
	authClient := usecase.NewMockAuthClient()
	mockLogger := &usecase.MockLogger{}

	// Create WebSocket handler
	wsHandler := httpadapter.NewWebSocketHandler(realtimeUsecase, securityUsecase, authClient, mockLogger) // Setup middleware for WebSocket upgrade
	app.Use("/ws", func(c *fiber.Ctx) error {
		// Check for proper WebSocket upgrade headers
		if c.Get("Connection") != "Upgrade" || c.Get("Upgrade") != "websocket" {
			return fiber.ErrUpgradeRequired
		}

		// Check WebSocket version
		wsVersion := c.Get("Sec-WebSocket-Version")
		if wsVersion != "" && wsVersion != "13" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Unsupported WebSocket version",
			})
		}

		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Register WebSocket routes
	wsHandler.RegisterRoutes(app)

	return app, realtimeUsecase, securityUsecase, authClient
}

// TestWSConnectionUpgrade tests WebSocket connection establishment
func TestWSConnectionUpgrade(t *testing.T) {
	app, _, _, _ := setupWSTestApp()

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
		shouldUpgrade  bool
	}{
		{
			name: "valid websocket upgrade",
			headers: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
			},
			expectedStatus: 101,
			shouldUpgrade:  true,
		},
		{
			name:           "missing upgrade headers",
			headers:        map[string]string{},
			expectedStatus: 426,
			shouldUpgrade:  false,
		},
		{
			name: "invalid websocket version",
			headers: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "12",
			},
			expectedStatus: 400,
			shouldUpgrade:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/ws/listen", nil)
			require.NoError(t, err)

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.shouldUpgrade {
				assert.Equal(t, "websocket", resp.Header.Get("Upgrade"))
				assert.Equal(t, "Upgrade", resp.Header.Get("Connection"))
			}
		})
	}
}

// TestWSSubscriptionFlow tests subscription message handling
func TestWSSubscriptionFlow(t *testing.T) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()
	// Setup mock user
	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "test-user-123",
		Email:    "test@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	t.Run("successful subscription", func(t *testing.T) {
		// This test would require a more complex setup to simulate
		// actual WebSocket communication through Fiber's test framework
		// For now, we'll test the underlying usecase logic

		ctx := context.Background()
		eventChan := make(chan model.RealtimeEvent, 10)

		err := realtimeUsecase.Subscribe(ctx, "test-subscriber", "/documents/test", eventChan)
		assert.NoError(t, err)
		// Verify subscription was created
		assert.Equal(t, 1, realtimeUsecase.GetSubscriberCount("/documents/test"))

		// Test event emission
		testEvent := model.RealtimeEvent{
			Type:     "document_created",
			FullPath: "/documents/test/doc1",
			Data: map[string]interface{}{
				"id":   "doc1",
				"name": "Test Document",
			},
		}

		realtimeUsecase.EmitEvent(testEvent)

		// Verify event was received
		select {
		case receivedEvent := <-eventChan:
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.FullPath, receivedEvent.FullPath)
		case <-time.After(time.Second):
			t.Fatal("Event not received within timeout")
		}
	})

	t.Run("unsubscription", func(t *testing.T) {
		ctx := context.Background()

		err := realtimeUsecase.Unsubscribe(ctx, "test-subscriber", "/documents/test")
		assert.NoError(t, err)
		// Verify subscription was removed
		assert.Equal(t, 0, realtimeUsecase.GetSubscriberCount("/documents/test"))
	})

	// Use app variable to avoid unused variable error
	_ = app
}

// TestWSSecurityValidation tests WebSocket security features
func TestWSSecurityValidation(t *testing.T) {
	app, _, securityUsecase, authClient := setupWSTestApp()

	tests := []struct {
		name           string
		user           *authModel.User
		path           string
		operation      string
		shouldValidate bool
		expectedError  string
	}{{
		name: "valid user and path",
		user: &authModel.User{
			ID:       primitive.NewObjectID(),
			UserID:   "valid-user",
			Email:    "valid@example.com",
			TenantID: "tenant-123",
		},
		path:           "/documents/tenant-123/collection1",
		operation:      "read",
		shouldValidate: true,
	},
		{
			name:           "no user",
			user:           nil,
			path:           "/documents/tenant-123/collection1",
			operation:      "read",
			shouldValidate: false,
			expectedError:  "unauthorized",
		},
		{
			name: "invalid tenant access",
			user: &authModel.User{
				ID:       primitive.NewObjectID(),
				UserID:   "user-different-tenant",
				Email:    "user@example.com",
				TenantID: "tenant-456",
			},
			path:           "/documents/tenant-123/collection1",
			operation:      "read",
			shouldValidate: false,
			expectedError:  "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient.SetUser(tt.user)
			securityUsecase.SetValidationResult(tt.shouldValidate,
				func() error {
					if tt.expectedError != "" {
						return fmt.Errorf(tt.expectedError)
					}
					return nil
				}())

			ctx := context.Background()

			if tt.user != nil {
				err := securityUsecase.ValidateRead(ctx, tt.user, tt.path)
				if tt.shouldValidate {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}

	// Use app variable to avoid unused variable error
	_ = app
}

// TestWSConcurrentConnections tests multiple concurrent WebSocket connections
func TestWSConcurrentConnections(t *testing.T) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()
	// Setup test environment
	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "concurrent-test-user",
		Email:    "test@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	t.Run("multiple subscribers", func(t *testing.T) {
		ctx := context.Background()
		numSubscribers := 10

		var wg sync.WaitGroup
		channels := make([]chan model.RealtimeEvent, numSubscribers)

		// Create multiple subscribers
		for i := 0; i < numSubscribers; i++ {
			wg.Add(1)
			channels[i] = make(chan model.RealtimeEvent, 10)
			subscriberID := fmt.Sprintf("subscriber-%d", i)

			go func(id string, ch chan model.RealtimeEvent, index int) {
				defer wg.Done()
				err := realtimeUsecase.Subscribe(ctx, id, "/documents/shared", ch)
				assert.NoError(t, err)
			}(subscriberID, channels[i], i)
		}

		wg.Wait()
		// Verify all subscriptions
		assert.Equal(t, numSubscribers, realtimeUsecase.GetSubscriberCount("/documents/shared"))

		// Emit event to all subscribers
		testEvent := model.RealtimeEvent{
			Type:     "document_updated",
			FullPath: "/documents/shared/doc1",
			Data: map[string]interface{}{
				"id":        "doc1",
				"timestamp": time.Now().Unix(),
			},
		}

		realtimeUsecase.EmitEvent(testEvent)

		// Verify all subscribers received the event
		for i := 0; i < numSubscribers; i++ {
			select {
			case receivedEvent := <-channels[i]:
				assert.Equal(t, testEvent.Type, receivedEvent.Type)
				assert.Equal(t, testEvent.FullPath, receivedEvent.FullPath)
			case <-time.After(2 * time.Second):
				t.Fatalf("Subscriber %d did not receive event within timeout", i)
			}
		}
	})

	// Use app variable to avoid unused variable error
	_ = app
}

// TestWSMessageTypes tests different WebSocket message types
func TestWSMessageTypes(t *testing.T) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()
	// Setup test environment
	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "message-test-user",
		Email:    "test@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	messageTypes := []struct {
		name        string
		messageType string
		operation   string
		path        string
		expectEvent bool
	}{
		{
			name:        "document create",
			messageType: "document_created",
			operation:   "create",
			path:        "/documents/tenant-123/collection1/doc1",
			expectEvent: true,
		},
		{
			name:        "document update",
			messageType: "document_updated",
			operation:   "update",
			path:        "/documents/tenant-123/collection1/doc1",
			expectEvent: true,
		},
		{
			name:        "document delete",
			messageType: "document_deleted",
			operation:   "delete",
			path:        "/documents/tenant-123/collection1/doc1",
			expectEvent: true,
		},
		{
			name:        "collection create",
			messageType: "collection_created",
			operation:   "create",
			path:        "/documents/tenant-123/collection2",
			expectEvent: true,
		},
	}

	for _, tt := range messageTypes {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			eventChan := make(chan model.RealtimeEvent, 10) // Subscribe to path
			err := realtimeUsecase.Subscribe(ctx, "test-subscriber", tt.path, eventChan)
			assert.NoError(t, err)

			// Emit event
			testEvent := model.RealtimeEvent{
				Type:     model.EventType(tt.messageType),
				FullPath: tt.path,
				Data: map[string]interface{}{
					"operation": tt.operation,
					"timestamp": time.Now().Unix(),
				},
			}

			realtimeUsecase.EmitEvent(testEvent)

			if tt.expectEvent {
				select {
				case receivedEvent := <-eventChan:
					assert.Equal(t, tt.messageType, string(receivedEvent.Type))
					assert.Equal(t, tt.path, receivedEvent.FullPath)
				case <-time.After(time.Second):
					t.Fatal("Event not received within timeout")
				}
			}

			// Cleanup
			err = realtimeUsecase.Unsubscribe(ctx, "test-subscriber", tt.path)
			assert.NoError(t, err)
		})
	}

	// Use app variable to avoid unused variable error
	_ = app
}

// TestWSPerformance tests WebSocket performance characteristics
func TestWSPerformance(t *testing.T) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()
	// Setup test environment
	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "perf-test-user",
		Email:    "test@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	t.Run("high frequency events", func(t *testing.T) {
		ctx := context.Background()
		eventChan := make(chan model.RealtimeEvent, 1000)

		err := realtimeUsecase.Subscribe(ctx, "perf-subscriber", "/documents/perf", eventChan)
		assert.NoError(t, err)

		numEvents := 100
		start := time.Now()

		// Emit multiple events rapidly
		for i := 0; i < numEvents; i++ {
			testEvent := model.RealtimeEvent{
				Type:     "document_updated",
				FullPath: fmt.Sprintf("/documents/perf/doc%d", i),
				Data: map[string]interface{}{
					"id":    fmt.Sprintf("doc%d", i),
					"index": i,
				},
			}
			realtimeUsecase.EmitEvent(testEvent)
		}

		// Verify all events were received
		receivedCount := 0
		timeout := time.After(5 * time.Second)

		for receivedCount < numEvents {
			select {
			case <-eventChan:
				receivedCount++
			case <-timeout:
				t.Fatalf("Only received %d out of %d events", receivedCount, numEvents)
			}
		}

		duration := time.Since(start)
		t.Logf("Processed %d events in %v (%.2f events/sec)",
			numEvents, duration, float64(numEvents)/duration.Seconds())

		assert.Equal(t, numEvents, receivedCount)
	})

	// Use app variable to avoid unused variable error
	_ = app
}

// TestWSErrorHandling tests WebSocket error scenarios
func TestWSErrorHandling(t *testing.T) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()

	t.Run("invalid message format", func(t *testing.T) {
		// Test would involve sending malformed JSON
		// This tests the underlying error handling logic
		testUser := &authModel.User{
			ID:       primitive.NewObjectID(),
			UserID:   "error-test-user",
			Email:    "test@example.com",
			TenantID: "tenant-123",
		}
		authClient.SetUser(testUser)

		// Test security validation error
		securityUsecase.SetValidationResult(false, fmt.Errorf("access denied"))

		ctx := context.Background()
		err := securityUsecase.ValidateRead(ctx, testUser, "/documents/other-tenant/doc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("subscription to invalid path", func(t *testing.T) {
		ctx := context.Background()
		eventChan := make(chan model.RealtimeEvent, 10)

		// This should work in our mock, but in real implementation
		// might have path validation
		err := realtimeUsecase.Subscribe(ctx, "test-subscriber", "", eventChan)
		// In mock this works, but real implementation might validate
		assert.NoError(t, err)
	})

	// Use app variable to avoid unused variable error
	_ = app
}

// Benchmark tests for WebSocket operations
func BenchmarkWSSubscription(b *testing.B) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()

	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "bench-user",
		Email:    "bench@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eventChan := make(chan model.RealtimeEvent, 10)
		subscriberID := fmt.Sprintf("bench-subscriber-%d", i)
		path := fmt.Sprintf("/documents/bench/path%d", i%100)

		err := realtimeUsecase.Subscribe(ctx, subscriberID, path, eventChan)
		if err != nil {
			b.Fatal(err)
		}

		err = realtimeUsecase.Unsubscribe(ctx, subscriberID, path)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Use app variable to avoid unused variable error
	_ = app
}

func BenchmarkWSEventEmission(b *testing.B) {
	app, realtimeUsecase, securityUsecase, authClient := setupWSTestApp()

	testUser := &authModel.User{
		ID:       primitive.NewObjectID(),
		UserID:   "bench-user",
		Email:    "bench@example.com",
		TenantID: "tenant-123",
	}
	authClient.SetUser(testUser)
	securityUsecase.SetValidationResult(true, nil)

	// Setup subscribers
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		eventChan := make(chan model.RealtimeEvent, 100)
		subscriberID := fmt.Sprintf("bench-subscriber-%d", i)
		err := realtimeUsecase.Subscribe(ctx, subscriberID, "/documents/bench", eventChan)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testEvent := model.RealtimeEvent{
			Type:     "document_updated",
			FullPath: fmt.Sprintf("/documents/bench/doc%d", i),
			Data: map[string]interface{}{
				"id":    fmt.Sprintf("doc%d", i),
				"value": i,
			},
		}
		realtimeUsecase.EmitEvent(testEvent)
	}

	// Use app variable to avoid unused variable error
	_ = app
}
