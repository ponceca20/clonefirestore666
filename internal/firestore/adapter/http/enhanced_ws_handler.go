package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// EnhancedWebSocketHandler manages WebSocket connections with full Firestore compatibility
type EnhancedWebSocketHandler struct {
	realtimeUC usecase.RealtimeUsecase
	securityUC usecase.SecurityUsecase
	authClient client.AuthClient
	log        logger.Logger

	// Connection management
	connections map[string]*ConnectionState
	connMutex   sync.RWMutex

	// Heartbeat configuration
	heartbeatInterval time.Duration
	connectionTimeout time.Duration
}

// ConnectionState tracks the state of a WebSocket connection
type ConnectionState struct {
	SubscriberID    string
	Connection      *websocket.Conn
	User            *authModel.User
	ActiveSubs      map[model.SubscriptionID]chan model.RealtimeEvent
	LastHeartbeat   time.Time
	Context         context.Context
	CancelFunc      context.CancelFunc
	MessageQueue    chan model.WebSocketMessage
	IsAuthenticated bool
	mutex           sync.RWMutex
}

// NewEnhancedWebSocketHandler creates a new enhanced WebSocket handler
func NewEnhancedWebSocketHandler(
	rtuc usecase.RealtimeUsecase,
	secUC usecase.SecurityUsecase,
	ac client.AuthClient,
	log logger.Logger,
) *EnhancedWebSocketHandler {
	handler := &EnhancedWebSocketHandler{
		realtimeUC:        rtuc,
		securityUC:        secUC,
		authClient:        ac,
		log:               log,
		connections:       make(map[string]*ConnectionState),
		heartbeatInterval: 30 * time.Second, // Send heartbeat every 30 seconds
		connectionTimeout: 90 * time.Second, // Timeout after 90 seconds without response
	}

	// Start background processes
	go handler.startHeartbeatManager()
	go handler.startConnectionCleanup()

	return handler
}

// RegisterRoutes registers the enhanced WebSocket endpoints
func (h *EnhancedWebSocketHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	// Create WebSocket group
	wsGroup := router.Group("/ws")

	// Apply authentication middleware first, then WebSocket upgrade middleware
	wsGroup.Use("/listen", authMiddleware, func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// Handle protocol version header if present
			if protocolVer := c.Get("X-Firestore-Protocol-Ver"); protocolVer != "" {
				c.Set("X-Firestore-Protocol-Ver", protocolVer)
			}
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Enhanced WebSocket endpoint
	wsGroup.Get("/listen", websocket.New(h.handleEnhancedWebSocketConnection))
}

// handleEnhancedWebSocketConnection handles new WebSocket connections with full Firestore features
func (h *EnhancedWebSocketHandler) handleEnhancedWebSocketConnection(conn *websocket.Conn) {
	subscriberID := uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())

	// Create connection state
	connState := &ConnectionState{
		SubscriberID:    subscriberID,
		Connection:      conn,
		ActiveSubs:      make(map[model.SubscriptionID]chan model.RealtimeEvent),
		LastHeartbeat:   time.Now(),
		Context:         ctx,
		CancelFunc:      cancel,
		MessageQueue:    make(chan model.WebSocketMessage, 100),
		IsAuthenticated: false,
	}

	// Register connection
	h.connMutex.Lock()
	h.connections[subscriberID] = connState
	h.connMutex.Unlock()

	h.log.Info("Enhanced WebSocket connection established",
		zap.String("subscriberID", subscriberID))

	// Setup cleanup on disconnect
	defer h.cleanupConnection(subscriberID)

	// Start connection handlers
	go h.handleIncomingMessages(connState)
	go h.handleOutgoingMessages(connState)
	go h.handleRealtimeEvents(connState)

	// Authentication flow
	if err := h.authenticateConnection(connState); err != nil {
		h.log.Error("Authentication failed",
			zap.String("subscriberID", subscriberID),
			zap.Error(err))
		h.sendError(connState, model.MessageTypeError, "Authentication required")
		return
	}

	// Keep connection alive and handle ping/pong
	h.keepConnectionAlive(connState)
}

// authenticateConnection handles the authentication flow
func (h *EnhancedWebSocketHandler) authenticateConnection(connState *ConnectionState) error {
	// In a real implementation, you'd extract auth tokens from headers or initial message
	// For now, we'll use a simplified approach where the user context is set by middleware

	// Wait for authentication message or timeout
	select {
	case <-time.After(10 * time.Second):
		return fiber.NewError(fiber.StatusUnauthorized, "Authentication timeout")
	case <-connState.Context.Done():
		return fiber.NewError(fiber.StatusRequestTimeout, "Connection closed during auth")
	default:
		// For demo purposes, mark as authenticated
		// In production, implement proper token validation
		connState.IsAuthenticated = true
		return nil
	}
}

// handleIncomingMessages processes incoming WebSocket messages
func (h *EnhancedWebSocketHandler) handleIncomingMessages(connState *ConnectionState) {
	defer connState.CancelFunc()

	for {
		select {
		case <-connState.Context.Done():
			return
		default:
			// Set read deadline
			connState.Connection.SetReadDeadline(time.Now().Add(h.connectionTimeout))

			var msg model.SubscriptionRequest
			if err := connState.Connection.ReadJSON(&msg); err != nil {
				var syntaxErr *json.SyntaxError
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.log.Error("WebSocket read error",
						zap.String("subscriberID", connState.SubscriberID),
						zap.Error(err))
					return
				} else if errors.As(err, &syntaxErr) || strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "cannot unmarshal") {
					// Error de JSON inválido: enviar error y continuar
					h.log.Warn("Invalid JSON received from WebSocket client",
						zap.String("subscriberID", connState.SubscriberID),
						zap.Error(err))
					h.sendError(connState, model.MessageTypeError, "Invalid JSON format")
					continue
				} else {
					// Otro error de bajo nivel: cerrar la conexión
					h.log.Error("WebSocket low-level error",
						zap.String("subscriberID", connState.SubscriberID),
						zap.Error(err))
					return
				}
			}

			// Update heartbeat
			connState.mutex.Lock()
			connState.LastHeartbeat = time.Now()
			connState.mutex.Unlock()

			// Process message
			h.processSubscriptionMessage(connState, msg)
		}
	}
}

// processSubscriptionMessage processes subscription/unsubscription messages
func (h *EnhancedWebSocketHandler) processSubscriptionMessage(connState *ConnectionState, msg model.SubscriptionRequest) {
	if !connState.IsAuthenticated {
		h.sendError(connState, model.MessageTypeError, "Authentication required")
		return
	}

	switch msg.Action {
	case model.MessageTypeSubscribe:
		h.handleEnhancedSubscribe(connState, msg)
	case model.MessageTypeUnsubscribe:
		h.handleEnhancedUnsubscribe(connState, msg)
	default:
		h.sendError(connState, model.MessageTypeError, "Unknown action: "+msg.Action)
	}
}

// handleEnhancedSubscribe processes enhanced subscription requests with full Firestore features
func (h *EnhancedWebSocketHandler) handleEnhancedSubscribe(connState *ConnectionState, req model.SubscriptionRequest) {
	// Validate Firestore path
	pathInfo, err := firestore.ParseFirestorePath(req.FullPath)
	if err != nil {
		h.log.Warn("Invalid Firestore path in subscription",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("path", req.FullPath),
			zap.Error(err))
		h.sendSubscriptionError(connState, req.SubscriptionID, "invalid_path", "Invalid Firestore path")
		return
	}

	// Validate security permissions
	if h.securityUC != nil && connState.User != nil {
		if err := h.securityUC.ValidateRead(connState.Context, connState.User, req.FullPath); err != nil {
			h.log.Warn("Security validation failed for WebSocket subscription",
				zap.String("subscriberID", connState.SubscriberID),
				zap.String("path", req.FullPath),
				zap.String("userID", connState.User.ID.Hex()),
				zap.Error(err))
			h.sendSubscriptionError(connState, req.SubscriptionID, "forbidden", "Access denied to this path")
			return
		}
	}

	// Create event channel for this subscription
	eventChan := make(chan model.RealtimeEvent, 200) // Larger buffer for better performance
	// Register subscription with enhanced features
	subscribeReq := usecase.SubscribeRequest{
		SubscriberID:   connState.SubscriberID,
		SubscriptionID: req.SubscriptionID,
		FirestorePath:  req.FullPath,
		EventChannel:   eventChan,
		ResumeToken:    req.ResumeToken,
		Query:          req.Query,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata:   true,
			IncludeOldData:    true,
			HeartbeatInterval: h.heartbeatInterval,
		},
	}

	resp, err := h.realtimeUC.Subscribe(connState.Context, subscribeReq)
	if err != nil {
		h.log.Error("Error subscribing to path",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("subscriptionID", string(req.SubscriptionID)),
			zap.String("path", req.FullPath),
			zap.Error(err))
		h.sendSubscriptionError(connState, req.SubscriptionID, "subscription_failed", "Failed to subscribe to path")
		close(eventChan)
		return
	}

	// Track subscription in connection state
	connState.mutex.Lock()
	connState.ActiveSubs[req.SubscriptionID] = eventChan
	connState.mutex.Unlock()

	h.log.Info("Client subscribed to path",
		zap.String("subscriberID", connState.SubscriberID),
		zap.String("subscriptionID", string(req.SubscriptionID)),
		zap.String("path", req.FullPath),
		zap.String("projectID", pathInfo.ProjectID),
		zap.String("databaseID", pathInfo.DatabaseID),
		zap.String("resumeToken", string(req.ResumeToken)))
	// Send subscription confirmation with enhanced response data
	response := model.SubscriptionResponse{
		Type:           model.MessageTypeSubscriptionConfirmed,
		SubscriptionID: req.SubscriptionID,
		Status:         "confirmed",
		Data: map[string]interface{}{
			"fullPath":        req.FullPath,
			"projectId":       pathInfo.ProjectID,
			"databaseId":      pathInfo.DatabaseID,
			"initialSnapshot": resp.InitialSnapshot,
			"resumeToken":     string(resp.ResumeToken),
			"createdAt":       resp.CreatedAt,
		},
	}

	h.sendSubscriptionResponse(connState, response)
}

// handleEnhancedUnsubscribe processes unsubscription requests
func (h *EnhancedWebSocketHandler) handleEnhancedUnsubscribe(connState *ConnectionState, req model.SubscriptionRequest) {
	// Unregister from realtime service
	unsubscribeReq := usecase.UnsubscribeRequest{
		SubscriberID:   connState.SubscriberID,
		SubscriptionID: req.SubscriptionID,
	}

	if err := h.realtimeUC.Unsubscribe(connState.Context, unsubscribeReq); err != nil {
		h.log.Error("Error unsubscribing from path",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("subscriptionID", string(req.SubscriptionID)),
			zap.Error(err))
	}

	// Close and remove event channel
	connState.mutex.Lock()
	if eventChan, exists := connState.ActiveSubs[req.SubscriptionID]; exists {
		close(eventChan)
		delete(connState.ActiveSubs, req.SubscriptionID)
	}
	connState.mutex.Unlock()

	h.log.Info("Client unsubscribed from path",
		zap.String("subscriberID", connState.SubscriberID),
		zap.String("subscriptionID", string(req.SubscriptionID)))

	// Send unsubscription confirmation
	response := model.SubscriptionResponse{
		Type:           "unsubscription_confirmed",
		SubscriptionID: req.SubscriptionID,
		Status:         "confirmed",
	}

	h.sendSubscriptionResponse(connState, response)
}

// handleRealtimeEvents forwards real-time events from subscriptions to the client
func (h *EnhancedWebSocketHandler) handleRealtimeEvents(connState *ConnectionState) {
	defer connState.CancelFunc()

	for {
		select {
		case <-connState.Context.Done():
			return
		default:
			// Check all active subscriptions for events
			connState.mutex.RLock()
			subscriptions := make(map[model.SubscriptionID]chan model.RealtimeEvent)
			for subID, ch := range connState.ActiveSubs {
				subscriptions[subID] = ch
			}
			connState.mutex.RUnlock()

			// Process events from all subscriptions
			for subID, eventChan := range subscriptions {
				select {
				case event, ok := <-eventChan:
					if !ok {
						// Channel was closed, remove from active subscriptions
						connState.mutex.Lock()
						delete(connState.ActiveSubs, subID)
						connState.mutex.Unlock()
						continue
					}

					// Forward event to client
					msg := model.WebSocketMessage{
						Type:           model.MessageTypeDocumentChange,
						SubscriptionID: subID,
						Data: map[string]interface{}{
							"event": event,
						},
						Timestamp: time.Now(),
					}

					select {
					case connState.MessageQueue <- msg:
						h.log.Debug("Event queued for client",
							zap.String("subscriberID", connState.SubscriberID),
							zap.String("subscriptionID", string(subID)),
							zap.String("eventType", string(event.Type)))
					default:
						h.log.Warn("Message queue full, dropping event",
							zap.String("subscriberID", connState.SubscriberID),
							zap.String("subscriptionID", string(subID)))
					}
				default:
					// No event available
				}
			}

			// Small delay to prevent busy waiting
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// handleOutgoingMessages processes the message queue and sends messages to the client
func (h *EnhancedWebSocketHandler) handleOutgoingMessages(connState *ConnectionState) {
	defer connState.CancelFunc()

	for {
		select {
		case <-connState.Context.Done():
			return
		case msg := <-connState.MessageQueue:
			connState.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := connState.Connection.WriteJSON(msg); err != nil {
				h.log.Error("Error sending message to client",
					zap.String("subscriberID", connState.SubscriberID),
					zap.Error(err))
				return
			}
		}
	}
}

// keepConnectionAlive manages the connection lifecycle and ping/pong
func (h *EnhancedWebSocketHandler) keepConnectionAlive(connState *ConnectionState) {
	pingTicker := time.NewTicker(h.heartbeatInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-connState.Context.Done():
			return
		case <-pingTicker.C:
			// Send ping
			if err := h.sendPing(connState); err != nil {
				h.log.Error("Failed to send ping",
					zap.String("subscriberID", connState.SubscriberID),
					zap.Error(err))
				return
			}

			// Update heartbeat record in realtime service
			for subID := range connState.ActiveSubs {
				h.realtimeUC.UpdateLastHeartbeat(connState.SubscriberID, subID)
			}
		}
	}
}

// startHeartbeatManager sends periodic heartbeats to all connections
func (h *EnhancedWebSocketHandler) startHeartbeatManager() {
	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := h.realtimeUC.SendHeartbeat(context.Background()); err != nil {
			h.log.Error("Failed to send global heartbeat", zap.Error(err))
		}
	}
}

// startConnectionCleanup periodically cleans up stale connections
func (h *EnhancedWebSocketHandler) startConnectionCleanup() {
	ticker := time.NewTicker(60 * time.Second) // Cleanup every minute
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		// Cleanup stale connections in realtime service
		if err := h.realtimeUC.CleanupStaleConnections(ctx, h.connectionTimeout); err != nil {
			h.log.Error("Failed to cleanup stale connections", zap.Error(err))
		}

		// Cleanup local connection state
		h.connMutex.Lock()
		now := time.Now()
		staleConnections := make([]string, 0)

		for subscriberID, connState := range h.connections {
			connState.mutex.RLock()
			timeSinceLastHeartbeat := now.Sub(connState.LastHeartbeat)
			connState.mutex.RUnlock()

			if timeSinceLastHeartbeat > h.connectionTimeout {
				staleConnections = append(staleConnections, subscriberID)
			}
		}

		for _, subscriberID := range staleConnections {
			h.log.Info("Cleaning up stale connection",
				zap.String("subscriberID", subscriberID))
			delete(h.connections, subscriberID)
		}
		h.connMutex.Unlock()
	}
}

// cleanupConnection cleans up resources when a connection closes
func (h *EnhancedWebSocketHandler) cleanupConnection(subscriberID string) {
	h.log.Info("Cleaning up WebSocket connection",
		zap.String("subscriberID", subscriberID))

	// Remove from connections map
	h.connMutex.Lock()
	connState, exists := h.connections[subscriberID]
	delete(h.connections, subscriberID)
	h.connMutex.Unlock()

	if !exists {
		return
	}

	// Cancel context
	connState.CancelFunc()

	// Unsubscribe from all paths
	if err := h.realtimeUC.UnsubscribeAll(context.Background(), subscriberID); err != nil {
		h.log.Error("Error unsubscribing all paths",
			zap.String("subscriberID", subscriberID),
			zap.Error(err))
	}

	// Close all event channels
	connState.mutex.Lock()
	for subID, ch := range connState.ActiveSubs {
		close(ch)
		delete(connState.ActiveSubs, subID)
	}
	close(connState.MessageQueue)
	connState.mutex.Unlock()
}

// Helper methods for sending messages

// sendPing sends a ping message to the client
func (h *EnhancedWebSocketHandler) sendPing(connState *ConnectionState) error {
	pingMsg := model.WebSocketMessage{
		Type:      model.MessageTypePing,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "ping",
		},
	}

	select {
	case connState.MessageQueue <- pingMsg:
		return nil
	case <-time.After(5 * time.Second):
		return fiber.NewError(fiber.StatusRequestTimeout, "ping send timeout")
	}
}

// sendError sends an error message to the client
func (h *EnhancedWebSocketHandler) sendError(connState *ConnectionState, messageType, message string) {
	errorMsg := model.WebSocketMessage{
		Type:      messageType,
		Error:     message,
		Timestamp: time.Now(),
	}

	select {
	case connState.MessageQueue <- errorMsg:
	case <-time.After(5 * time.Second):
		h.log.Warn("Error message send timeout",
			zap.String("subscriberID", connState.SubscriberID))
	}
}

// sendSubscriptionError sends a subscription-specific error
func (h *EnhancedWebSocketHandler) sendSubscriptionError(connState *ConnectionState, subscriptionID model.SubscriptionID, errorType, message string) {
	errorResponse := model.SubscriptionResponse{
		Type:           model.MessageTypeSubscriptionError,
		SubscriptionID: subscriptionID,
		Status:         "error",
		Error:          fmt.Sprintf("%s: %s", errorType, message),
	}

	h.sendSubscriptionResponse(connState, errorResponse)
}

// sendSubscriptionResponse sends a subscription response
func (h *EnhancedWebSocketHandler) sendSubscriptionResponse(connState *ConnectionState, response model.SubscriptionResponse) {
	msg := model.WebSocketMessage{
		Type:           response.Type,
		SubscriptionID: response.SubscriptionID,
		Data: map[string]interface{}{
			"status": response.Status,
			"error":  response.Error,
			"data":   response.Data,
		},
		Timestamp: time.Now(),
	}

	select {
	case connState.MessageQueue <- msg:
	case <-time.After(5 * time.Second):
		h.log.Warn("Subscription response send timeout",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("subscriptionID", string(response.SubscriptionID)))
	}
}

// sendProtocolError sends a protocol-level error following Google Cloud Firestore error format
// This maintains compatibility with Firestore client SDKs
func (h *EnhancedWebSocketHandler) sendProtocolError(connState *ConnectionState, code string, message string) {
	// Create error response following Google Cloud Firestore protocol
	errorResponse := map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"status":  code,
		},
		"timestamp": time.Now().UTC(),
	}

	// Send error message through the message queue
	msg := model.WebSocketMessage{
		Type:      model.MessageTypeError,
		Data:      errorResponse,
		Timestamp: time.Now(),
	}

	select {
	case connState.MessageQueue <- msg:
		h.log.Debug("Protocol error sent to client",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("errorCode", code),
			zap.String("errorMessage", message))
	case <-time.After(5 * time.Second):
		h.log.Warn("Protocol error send timeout",
			zap.String("subscriberID", connState.SubscriberID),
			zap.String("errorCode", code))
	}
}
