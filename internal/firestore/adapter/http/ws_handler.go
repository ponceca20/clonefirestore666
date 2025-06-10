package http

import (
	"context"
	"sync"
	"time"

	authModel "firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebSocketHandler manages WebSocket connections for real-time updates.
type WebSocketHandler struct {
	realtimeUC usecase.RealtimeUsecase
	securityUC usecase.SecurityUsecase
	authClient client.AuthClient
	log        logger.Logger
}

// NewWebSocketHandler creates a new WebSocketHandler.
func NewWebSocketHandler(
	rtuc usecase.RealtimeUsecase,
	secUC usecase.SecurityUsecase,
	ac client.AuthClient,
	log logger.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		realtimeUC: rtuc,
		securityUC: secUC,
		authClient: ac,
		log:        log,
	}
}

// RegisterRoutes registers the WebSocket endpoint.
func (h *WebSocketHandler) RegisterRoutes(router fiber.Router) {
	// Create WebSocket group
	wsGroup := router.Group("/ws")

	// Middleware to ensure it's a WebSocket upgrade request
	wsGroup.Use("/listen", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	wsGroup.Get("/listen", websocket.New(h.handleWebSocketConnection))
}

// WebSocketMessage represents messages sent/received via WebSocket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// handleWebSocketConnection is called when a new WebSocket connection is established.
func (h *WebSocketHandler) handleWebSocketConnection(conn *websocket.Conn) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	subscriberID := uuid.NewString()

	h.log.Info("New WebSocket connection established",
		zap.String("subscriberID", subscriberID))

	// Track active subscriptions for this client
	activeSubscriptions := make(map[string]chan model.RealtimeEvent)
	var subscriptionMu sync.Mutex

	// Cleanup on disconnect
	defer func() {
		h.log.Info("WebSocket connection closing",
			zap.String("subscriberID", subscriberID))

		// Unsubscribe from all paths
		if err := h.realtimeUC.UnsubscribeAll(ctx, subscriberID); err != nil {
			h.log.Error("Error unsubscribing all paths",
				zap.String("subscriberID", subscriberID),
				zap.Error(err))
		}

		// Close all event channels
		subscriptionMu.Lock()
		for path, ch := range activeSubscriptions {
			close(ch)
			delete(activeSubscriptions, path)
		}
		subscriptionMu.Unlock()
	}()

	// Start message handler goroutine
	go h.handleIncomingMessages(ctx, conn, subscriberID, activeSubscriptions, &subscriptionMu)

	// Start event forwarding goroutine
	go h.handleEventForwarding(ctx, conn, subscriberID, activeSubscriptions, &subscriptionMu)

	// Keep connection alive
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Keep reading to detect disconnection
			if _, _, err := conn.ReadMessage(); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.log.Error("WebSocket error",
						zap.String("subscriberID", subscriberID),
						zap.Error(err))
				}
				return
			}
		}
	}
}

// handleIncomingMessages processes messages from the client
func (h *WebSocketHandler) handleIncomingMessages(
	ctx context.Context,
	conn *websocket.Conn,
	subscriberID string,
	activeSubscriptions map[string]chan model.RealtimeEvent,
	subscriptionMu *sync.Mutex,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg model.SubscriptionRequest
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.log.Error("Error reading WebSocket message",
						zap.String("subscriberID", subscriberID),
						zap.Error(err))
				}
				return
			}

			switch msg.Action {
			case "subscribe":
				h.handleSubscribe(ctx, conn, subscriberID, msg, activeSubscriptions, subscriptionMu)
			case "unsubscribe":
				h.handleUnsubscribe(ctx, conn, subscriberID, msg, activeSubscriptions, subscriptionMu)
			default:
				h.sendError(conn, "invalid_action", "Unknown action: "+msg.Action)
			}
		}
	}
}

// handleSubscribe processes subscription requests
func (h *WebSocketHandler) handleSubscribe(
	ctx context.Context,
	conn *websocket.Conn,
	subscriberID string,
	req model.SubscriptionRequest,
	activeSubscriptions map[string]chan model.RealtimeEvent,
	subscriptionMu *sync.Mutex,
) {
	// Validate Firestore path
	pathInfo, err := firestore.ParseFirestorePath(req.FullPath)
	if err != nil {
		h.log.Warn("Invalid Firestore path in subscription",
			zap.String("subscriberID", subscriberID),
			zap.String("path", req.FullPath),
			zap.Error(err))
		h.sendError(conn, "invalid_path", "Invalid Firestore path")
		return
	}

	// Validate security permissions
	if h.securityUC != nil {
		// Extract user information from context (set by auth middleware)
		userInterface := ctx.Value("user")
		if userInterface == nil {
			h.log.Warn("No user in context for WebSocket subscription",
				zap.String("subscriberID", subscriberID),
				zap.String("path", req.FullPath))
			h.sendError(conn, "unauthorized", "Authentication required")
			return
		}
		// Type assert to get the user
		user, ok := userInterface.(*authModel.User)
		if !ok {
			h.log.Error("Invalid user type in context",
				zap.String("subscriberID", subscriberID),
				zap.String("path", req.FullPath))
			h.sendError(conn, "unauthorized", "Invalid user context")
			return
		}

		// Validate read access to the path
		if err := h.securityUC.ValidateRead(ctx, user, req.FullPath); err != nil {
			h.log.Warn("Security validation failed for WebSocket subscription",
				zap.String("subscriberID", subscriberID),
				zap.String("path", req.FullPath),
				zap.String("userID", user.ID.Hex()),
				zap.Error(err))
			h.sendError(conn, "forbidden", "Access denied to this path")
			return
		}
	}

	// Create event channel for this subscription
	eventChan := make(chan model.RealtimeEvent, 100) // Buffered channel

	// Register subscription
	if err := h.realtimeUC.Subscribe(ctx, subscriberID, req.FullPath, eventChan); err != nil {
		h.log.Error("Error subscribing to path",
			zap.String("subscriberID", subscriberID),
			zap.String("path", req.FullPath),
			zap.Error(err))
		h.sendError(conn, "subscription_failed", "Failed to subscribe to path")
		close(eventChan)
		return
	}

	// Track subscription
	subscriptionMu.Lock()
	activeSubscriptions[req.FullPath] = eventChan
	subscriptionMu.Unlock()

	h.log.Info("Client subscribed to path",
		zap.String("subscriberID", subscriberID),
		zap.String("path", req.FullPath),
		zap.String("projectID", pathInfo.ProjectID),
		zap.String("databaseID", pathInfo.DatabaseID))

	// Send confirmation
	response := WebSocketMessage{
		Type: "subscription_confirmed",
		Data: map[string]interface{}{
			"fullPath":   req.FullPath,
			"projectId":  pathInfo.ProjectID,
			"databaseId": pathInfo.DatabaseID,
		},
	}
	conn.WriteJSON(response)
}

// handleUnsubscribe processes unsubscription requests
func (h *WebSocketHandler) handleUnsubscribe(
	ctx context.Context,
	conn *websocket.Conn,
	subscriberID string,
	req model.SubscriptionRequest,
	activeSubscriptions map[string]chan model.RealtimeEvent,
	subscriptionMu *sync.Mutex,
) {
	// Unregister from realtime service
	if err := h.realtimeUC.Unsubscribe(ctx, subscriberID, req.FullPath); err != nil {
		h.log.Error("Error unsubscribing from path",
			zap.String("subscriberID", subscriberID),
			zap.String("path", req.FullPath),
			zap.Error(err))
	}

	// Close and remove event channel
	subscriptionMu.Lock()
	if eventChan, exists := activeSubscriptions[req.FullPath]; exists {
		close(eventChan)
		delete(activeSubscriptions, req.FullPath)
	}
	subscriptionMu.Unlock()

	h.log.Info("Client unsubscribed from path",
		zap.String("subscriberID", subscriberID),
		zap.String("path", req.FullPath))

	// Send confirmation
	response := WebSocketMessage{
		Type: "unsubscription_confirmed",
		Data: map[string]interface{}{
			"fullPath": req.FullPath,
		},
	}
	conn.WriteJSON(response)
}

// handleEventForwarding forwards real-time events to the client
func (h *WebSocketHandler) handleEventForwarding(
	ctx context.Context,
	conn *websocket.Conn,
	subscriberID string,
	activeSubscriptions map[string]chan model.RealtimeEvent,
	subscriptionMu *sync.Mutex,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check all active subscriptions for events
			subscriptionMu.Lock()
			for path, eventChan := range activeSubscriptions {
				select {
				case event, ok := <-eventChan:
					if !ok {
						// Channel was closed, remove from active subscriptions
						delete(activeSubscriptions, path)
						continue
					}

					// Forward event to client
					response := WebSocketMessage{
						Type: "document_change",
						Data: event,
					}

					if err := conn.WriteJSON(response); err != nil {
						h.log.Error("Error sending event to client",
							zap.String("subscriberID", subscriberID),
							zap.String("path", path),
							zap.Error(err))
						subscriptionMu.Unlock()
						return
					}

					h.log.Debug("Event forwarded to client",
						zap.String("subscriberID", subscriberID),
						zap.String("path", path),
						zap.String("eventType", string(event.Type)))
				default:
					// No event available, continue
				}
			}
			subscriptionMu.Unlock()

			// Small delay to prevent busy waiting
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorType, message string) {
	response := WebSocketMessage{
		Type: "error",
		Data: ErrorResponse{
			Error:   errorType,
			Message: message,
		},
	}
	conn.WriteJSON(response)
}
