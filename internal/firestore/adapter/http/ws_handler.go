package http

import (
	"context"
	"encoding/json"
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"
	"sync" // Added for managing client subscriptions within the handler

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid" // For generating unique subscriber IDs
	"go.uber.org/zap"
)

// WebSocketHandler manages WebSocket connections for real-time updates.
type WebSocketHandler struct {
	realtimeUC usecase.RealtimeUsecase
	authClient client.AuthClient // To validate tokens for WS connections
	log        logger.Logger
	// Store active subscriptions per client to manage them on disconnect
	// clientSubscriptions map[string]map[string]bool // subscriberID -> path -> true
	// mu sync.RWMutex // To protect clientSubscriptions
}

// NewWebSocketHandler creates a new WebSocketHandler.
func NewWebSocketHandler(
	rtuc usecase.RealtimeUsecase,
	ac client.AuthClient,
	log logger.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		realtimeUC: rtuc,
		authClient: ac,
		log:        log,
		// clientSubscriptions: make(map[string]map[string]bool),
	}
}

// RegisterRoutes registers the WebSocket endpoint.
func (h *WebSocketHandler) RegisterRoutes(app *fiber.App) {
	// Middleware to ensure it's a WebSocket upgrade request
	app.Use("/ws/v1/listen", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/v1/listen", websocket.New(h.handleWebSocketConnection))
}

// ClientSubscriptionMessage defines the structure for messages from the client.
type ClientSubscriptionMessage struct {
	Action string `json:"action"` // "subscribe" or "unsubscribe"
	Path   string `json:"path"`
	Token  string `json:"token,omitempty"` // Optional: token for initial auth if not in query params
}

// handleWebSocketConnection is called when a new WebSocket connection is established.
func (h *WebSocketHandler) handleWebSocketConnection(conn *websocket.Conn) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx() // Ensure context is cancelled when handler exits

	subscriberID := uuid.NewString()
	log := h.log
	log.Info(ctx, "New WebSocket connection established")

	// Simple token authentication (example: from query param during connection)
	// In a real app, this might come from a subprotocol header or an initial message.
	// token := conn.Query("token")
	// if token == "" {
	// log.Warn(ctx, "Missing token for WebSocket connection")
	// conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Missing authentication token"))
	// conn.Close()
	// return
	// }
	//
	// _, err := h.authClient.ValidateToken(ctx, token)
	// if err != nil {
	// log.Error(ctx, "Invalid token for WebSocket connection", zap.Error(err))
	// conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid authentication token"))
	// conn.Close()
	// return
	// }
	// log.Info(ctx, "WebSocket connection authenticated")

	// Each client needs its own set of channels for subscriptions.
	// When a client subscribes to a path, a new channel is made for that specific subscription.
	// We need to keep track of these channels to send data and to close them on unsubscribe/disconnect.
	// activeSubEventChannels: path -> chan model.RealtimeEvent
	activeSubEventChannels := make(map[string]chan model.RealtimeEvent)
	var activeSubMu sync.Mutex // Protects activeSubEventChannels for this specific client

	defer func() {
		log.Info(ctx, "WebSocket connection closing")
		activeSubMu.Lock()
		for path, ch := range activeSubEventChannels {
			log.Info(ctx, "Unsubscribing from path on disconnect", zap.String("path", path))
			// No need to pass context here if Unsubscribe doesn't use it for critical ops
			if err := h.realtimeUC.Unsubscribe(context.Background(), subscriberID, path); err != nil {
				log.Error(ctx, "Error unsubscribing on disconnect", zap.String("path", path), zap.Error(err))
			}
			close(ch) // Close the channel associated with this subscription
		}
		activeSubMu.Unlock()
	}()

	// Goroutine to read messages from the client
	go func() {
		defer func() {
			log.Info(ctx, "Stopping incoming message handler for client")
			cancelCtx()  // Signal outgoing handler to stop
			conn.Close() // Ensure connection is closed if read loop exits first
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				messageType, p, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
						log.Error(ctx, "Error reading message from WebSocket", zap.Error(err))
					} else {
						log.Info(ctx, "WebSocket gracefully closed by client or due to error", zap.Error(err))
					}
					return // Exit goroutine
				}

				if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
					var msg ClientSubscriptionMessage
					if err := json.Unmarshal(p, &msg); err != nil {
						log.Warn(ctx, "Error unmarshalling client message", zap.Error(err), zap.ByteString("payload", p))
						conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: "", Data: map[string]interface{}{"error": "invalid message format"}})
						continue
					}

					log.Info(ctx, "Received message from client", zap.String("action", msg.Action), zap.String("path", msg.Path))

					activeSubMu.Lock()
					currentChan, isSubscribed := activeSubEventChannels[msg.Path]
					activeSubMu.Unlock()

					switch msg.Action {
					case "subscribe":
						if isSubscribed {
							log.Warn(ctx, "Client already subscribed to path", zap.String("path", msg.Path))
							conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: msg.Path, Data: map[string]interface{}{"error": "already subscribed"}})
							continue
						}

						// Create a new channel for this specific subscription
						eventChan := make(chan model.RealtimeEvent, 10) // Buffered channel
						if err := h.realtimeUC.Subscribe(ctx, subscriberID, msg.Path, eventChan); err != nil {
							log.Error(ctx, "Error subscribing client", zap.String("path", msg.Path), zap.Error(err))
							conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: msg.Path, Data: map[string]interface{}{"error": "subscription failed"}})
							close(eventChan) // Important: close channel if subscribe failed
							continue
						}
						activeSubMu.Lock()
						activeSubEventChannels[msg.Path] = eventChan
						activeSubMu.Unlock()
						log.Info(ctx, "Client subscribed to path", zap.String("path", msg.Path))
						conn.WriteJSON(model.RealtimeEvent{Type: "system", Path: msg.Path, Data: map[string]interface{}{"status": "subscribed"}})

						// Start a goroutine to listen on this new channel and send to client
						go func(path string, ch <-chan model.RealtimeEvent) {
							// Use the existing log variable; logger.Logger does not have a With method
							log.Info(ctx, "Starting outgoing event handler for path", zap.String("path", path))
							defer log.Info(ctx, "Stopping outgoing event handler for path", zap.String("path", path))
							for {
								select {
								case <-ctx.Done(): // Connection context is cancelled
									return
								case event, ok := <-ch:
									if !ok { // Channel closed by Unsubscribe or disconnect
										log.Info(ctx, "Event channel closed for path")
										return
									}
									log.Debug(ctx, "Sending event to client", zap.Any("event", event))
									if err := conn.WriteJSON(event); err != nil {
										log.Error(ctx, "Error writing JSON to WebSocket", zap.Error(err))
										// If write fails, the connection might be dead.
										// The main read loop will likely detect this and trigger cleanup.
										// Or we can signal cancellation here.
										cancelCtx()
										return
									}
								}
							}
						}(msg.Path, eventChan)

					case "unsubscribe":
						if !isSubscribed {
							log.Warn(ctx, "Client not subscribed to path, cannot unsubscribe", zap.String("path", msg.Path))
							conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: msg.Path, Data: map[string]interface{}{"error": "not subscribed"}})
							continue
						}
						if err := h.realtimeUC.Unsubscribe(ctx, subscriberID, msg.Path); err != nil {
							log.Error(ctx, "Error unsubscribing client", zap.String("path", msg.Path), zap.Error(err))
							conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: msg.Path, Data: map[string]interface{}{"error": "unsubscription failed"}})
							continue
						}
						activeSubMu.Lock()
						delete(activeSubEventChannels, msg.Path)
						activeSubMu.Unlock()
						close(currentChan) // Close the channel after unsubscribing
						log.Info(ctx, "Client unsubscribed from path", zap.String("path", msg.Path))
						conn.WriteJSON(model.RealtimeEvent{Type: "system", Path: msg.Path, Data: map[string]interface{}{"status": "unsubscribed"}})

					default:
						log.Warn(ctx, "Unknown action from client", zap.String("action", msg.Action))
						conn.WriteJSON(model.RealtimeEvent{Type: "error", Path: "", Data: map[string]interface{}{"error": "unknown action"}})
					}
				} else {
					log.Info(ctx, "Received non-text/binary message from WebSocket", zap.Int("messageType", messageType))
				}
			}
		}
	}()
	log.Info(ctx, "WebSocket connection handler fully initialized and listening")
}
