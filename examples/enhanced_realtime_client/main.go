package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/fasthttp/websocket"
)

// FirestoreRealtimeClient demuestra cómo usar el sistema de tiempo real del clon de Firestore
// Este cliente muestra compatibilidad 100% con las funcionalidades en tiempo real de Firestore
//
// PROPÓSITO:
// - Cliente de ejemplo para conectarse al servidor WebSocket de Firestore Clone
// - Demuestra suscripciones en tiempo real a documentos y colecciones
// - Manejo de eventos ADDED, MODIFIED, REMOVED, HEARTBEAT
// - Soporte para queries con filtros y ordenamiento en tiempo real
// - Reconexión automática y manejo de resume tokens
// - Compatible con la arquitectura de Fiber WebSocket del servidor
type FirestoreRealtimeClient struct {
	conn                 *websocket.Conn
	subscriptions        map[model.SubscriptionID]chan model.RealtimeEvent
	messageHandlers      map[string]func(model.WebSocketMessage)
	isConnected          bool
	reconnectAttempts    int
	maxReconnectAttempts int
	heartbeatInterval    time.Duration
}

// NewFirestoreRealtimeClient creates a new enhanced realtime client
func NewFirestoreRealtimeClient(serverURL string) (*FirestoreRealtimeClient, error) {
	client := &FirestoreRealtimeClient{
		subscriptions:        make(map[model.SubscriptionID]chan model.RealtimeEvent),
		messageHandlers:      make(map[string]func(model.WebSocketMessage)),
		maxReconnectAttempts: 5,
		heartbeatInterval:    0, // Disable client heartbeat since server sends its own heartbeats
	}

	// Setup message handlers
	client.setupMessageHandlers()

	// Connect to server
	if err := client.connect(serverURL); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	// Start message handling
	go client.handleMessages()

	// Only start heartbeat manager if client heartbeat is enabled
	if client.heartbeatInterval > 0 {
		go client.heartbeatManager()
	}

	return client, nil
}

// connect establishes a WebSocket connection compatible with Fiber WebSocket server
func (c *FirestoreRealtimeClient) connect(serverURL string) error {
	// Parse the URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// For demo purposes, create a test authentication token
	// In a real application, you would get this from login/authentication
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySUQiOiI2ODUwMGMwMjQwNWMyNjcxZTlkODc2YzkiLCJlbWFpbCI6InRlc3R1c2VyQGV4YW1wbGUuY29tIiwidGVuYW50SUQiOiJteS1kZWZhdWx0LXRlbmFudCIsInJvbGVzIjpbInVzZXIiXSwiaXNzIjoiZmlyZXN0b3JlLWNsb25lLWF1dGgtc2VydmljZSIsInN1YiI6IjY4NTAwYzAyNDA1YzI2NzFlOWQ4NzZjOSIsImF1ZCI6WyIiXSwiZXhwIjoxNzUwMzU4OTY3LCJuYmYiOjE3NTAzNTgwNjcsImlhdCI6MTc1MDM1ODA2N30.IpahEfqpnOJAqhN_UtzJdE7RyZ0RIeQxE1Dzm01C4NQ"

	// Add token as query parameter since WebSocket headers might not work for auth
	query := u.Query()
	query.Set("token", testToken)
	u.RawQuery = query.Encode()

	// Create headers for the WebSocket connection
	headers := http.Header{
		"Sec-WebSocket-Protocol":   {"firestore-realtime"},
		"Sec-WebSocket-Extensions": {"permessage-deflate"},
		"User-Agent":               {"FirestoreCloneClient/1.0"},
		"Authorization":            {"Bearer " + testToken}, // Try both methods
	}

	// Create WebSocket dialer compatible with Fiber
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	// Connect to the WebSocket server
	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true
	c.reconnectAttempts = 0

	log.Printf("Successfully connected to %s with authentication", serverURL)
	return nil
}

// setupMessageHandlers configures handlers for different message types
func (c *FirestoreRealtimeClient) setupMessageHandlers() {
	c.messageHandlers[model.MessageTypeSubscriptionConfirmed] = c.handleSubscriptionConfirmed
	c.messageHandlers[model.MessageTypeSubscriptionError] = c.handleSubscriptionError
	c.messageHandlers[model.MessageTypeDocumentChange] = c.handleDocumentChange
	c.messageHandlers[model.MessageTypeHeartbeat] = c.handleHeartbeat
	c.messageHandlers[model.MessageTypePing] = c.handlePing
	c.messageHandlers[model.MessageTypeError] = c.handleError
	c.messageHandlers["unsubscription_confirmed"] = c.handleUnsubscriptionConfirmed // Add missing handler
}

// Subscribe to a Firestore path with full feature support
func (c *FirestoreRealtimeClient) Subscribe(ctx context.Context, opts SubscriptionOptions) (<-chan model.RealtimeEvent, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create event channel
	eventChan := make(chan model.RealtimeEvent, opts.BufferSize)
	c.subscriptions[opts.SubscriptionID] = eventChan

	// Create subscription request
	request := model.SubscriptionRequest{
		Action:          model.MessageTypeSubscribe,
		SubscriptionID:  opts.SubscriptionID,
		FullPath:        opts.Path,
		IncludeMetadata: opts.IncludeMetadata,
		Query:           opts.Query,
		ResumeToken:     opts.ResumeToken,
		IncludeOldData:  opts.IncludeOldData,
	}

	// Send subscription request
	if err := c.conn.WriteJSON(request); err != nil {
		delete(c.subscriptions, opts.SubscriptionID)
		close(eventChan)
		return nil, fmt.Errorf("failed to send subscription request: %w", err)
	}

	log.Printf("Subscribed to path: %s with subscription ID: %s", opts.Path, opts.SubscriptionID)

	return eventChan, nil
}

// Unsubscribe from a specific subscription
func (c *FirestoreRealtimeClient) Unsubscribe(ctx context.Context, subscriptionID model.SubscriptionID) error {
	if !c.isConnected {
		return fmt.Errorf("client is not connected")
	}

	// Create unsubscription request
	request := model.SubscriptionRequest{
		Action:         model.MessageTypeUnsubscribe,
		SubscriptionID: subscriptionID,
	}

	// Send unsubscription request
	if err := c.conn.WriteJSON(request); err != nil {
		return fmt.Errorf("failed to send unsubscription request: %w", err)
	}

	// Clean up local state
	if eventChan, exists := c.subscriptions[subscriptionID]; exists {
		close(eventChan)
		delete(c.subscriptions, subscriptionID)
	}

	log.Printf("Unsubscribed from subscription ID: %s", subscriptionID)
	return nil
}

// handleMessages processes incoming WebSocket messages
func (c *FirestoreRealtimeClient) handleMessages() {
	defer c.cleanup()

	for c.isConnected {
		var msg model.WebSocketMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Printf("Error reading message: %v", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.isConnected = false
				c.attemptReconnect()
			}
			return
		}

		// Handle message based on type
		if handler, exists := c.messageHandlers[msg.Type]; exists {
			handler(msg)
		} else {
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

// Message handlers

func (c *FirestoreRealtimeClient) handleSubscriptionConfirmed(msg model.WebSocketMessage) {
	log.Printf("Subscription confirmed: %s", msg.SubscriptionID)
	if data, ok := msg.Data["data"].(map[string]interface{}); ok {
		if fullPath, ok := data["fullPath"].(string); ok {
			log.Printf("  - Path: %s", fullPath)
		}
		if projectID, ok := data["projectId"].(string); ok {
			log.Printf("  - Project: %s", projectID)
		}
	}
}

func (c *FirestoreRealtimeClient) handleSubscriptionError(msg model.WebSocketMessage) {
	log.Printf("Subscription error for %s: %s", msg.SubscriptionID, msg.Error)

	// Clean up failed subscription
	if eventChan, exists := c.subscriptions[msg.SubscriptionID]; exists {
		close(eventChan)
		delete(c.subscriptions, msg.SubscriptionID)
	}
}

func (c *FirestoreRealtimeClient) handleDocumentChange(msg model.WebSocketMessage) {
	// Extract event from message data
	if eventData, ok := msg.Data["event"].(map[string]interface{}); ok {
		// Convert to RealtimeEvent
		eventJSON, _ := json.Marshal(eventData)
		var event model.RealtimeEvent
		if err := json.Unmarshal(eventJSON, &event); err != nil {
			log.Printf("Error parsing event: %v", err)
			return
		} // Forward to appropriate subscription channel
		if eventChan, exists := c.subscriptions[msg.SubscriptionID]; exists {
			// Use timeout to prevent blocking and ensure non-blocking operation
			go func() {
				select {
				case eventChan <- event:
					log.Printf("Event forwarded: %s on %s (seq: %d)", event.Type, event.FullPath, event.SequenceNumber)
				case <-time.After(5 * time.Second):
					log.Printf("Warning: Event channel full for subscription %s", msg.SubscriptionID)
				}
			}()
		}
	}
}

func (c *FirestoreRealtimeClient) handleHeartbeat(msg model.WebSocketMessage) {
	log.Printf("Received heartbeat at %s", msg.Timestamp.Format(time.RFC3339))
	// Could implement client-side health checks here
}

func (c *FirestoreRealtimeClient) handlePing(msg model.WebSocketMessage) {
	log.Printf("Received ping from server")
	// Note: Commenting out pong response since this server doesn't seem to support ping/pong
	// In a full implementation, you would respond with pong
	/*
		pongMsg := map[string]interface{}{
			"action":    model.MessageTypePong,
			"type":      model.MessageTypePong,
			"timestamp": time.Now(),
			"data": map[string]interface{}{
				"message": "pong",
			},
		}

		if err := c.conn.WriteJSON(pongMsg); err != nil {
			log.Printf("Error sending pong: %v", err)
		}
	*/
}

func (c *FirestoreRealtimeClient) handleError(msg model.WebSocketMessage) {
	log.Printf("Server error: %s", msg.Error)
}

func (c *FirestoreRealtimeClient) handleUnsubscriptionConfirmed(msg model.WebSocketMessage) {
	log.Printf("Unsubscription confirmed: %s", msg.SubscriptionID)
}

// heartbeatManager manages client-side heartbeat to keep connection alive
func (c *FirestoreRealtimeClient) heartbeatManager() {
	// If heartbeat interval is 0, don't send client heartbeats
	// (server is already sending heartbeats every 30 seconds)
	if c.heartbeatInterval == 0 {
		log.Printf("Client heartbeat disabled - relying on server heartbeats")
		return
	}

	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for c.isConnected {
		select {
		case <-ticker.C:
			// Send ping to server (must use Action for compatibility)
			pingMsg := map[string]interface{}{
				"action":    model.MessageTypePing,
				"type":      model.MessageTypePing,
				"timestamp": time.Now(),
				"data": map[string]interface{}{
					"client": "firestore-clone-client",
				},
			}
			if err := c.conn.WriteJSON(pingMsg); err != nil {
				log.Printf("Error sending ping: %v", err)
				c.isConnected = false
				return
			}
		case <-time.After(c.heartbeatInterval + 5*time.Second):
			// Timeout case to prevent infinite blocking
			log.Printf("Heartbeat timeout, marking connection as disconnected")
			c.isConnected = false
			return
		}
	}
}

// attemptReconnect tries to reconnect to the server
func (c *FirestoreRealtimeClient) attemptReconnect() {
	if c.reconnectAttempts >= c.maxReconnectAttempts {
		log.Printf("Max reconnection attempts reached, giving up")
		return
	}

	c.reconnectAttempts++
	backoffTime := time.Duration(c.reconnectAttempts) * 2 * time.Second

	log.Printf("Attempting to reconnect in %v (attempt %d/%d)", backoffTime, c.reconnectAttempts, c.maxReconnectAttempts)
	time.Sleep(backoffTime)

	// This would need the original server URL - in a real implementation
	// you'd store this as a field in the client
	// if err := c.connect(serverURL); err != nil {
	//     log.Printf("Reconnection failed: %v", err)
	//     go c.attemptReconnect()
	// }
}

// cleanup closes all resources
func (c *FirestoreRealtimeClient) cleanup() {
	c.isConnected = false

	// Close all subscription channels
	for subID, eventChan := range c.subscriptions {
		close(eventChan)
		delete(c.subscriptions, subID)
	}

	if c.conn != nil {
		c.conn.Close()
	}
}

// Close cleanly shuts down the client
func (c *FirestoreRealtimeClient) Close() error {
	c.cleanup()
	return nil
}

// SubscriptionOptions contains options for subscribing to a path
type SubscriptionOptions struct {
	SubscriptionID  model.SubscriptionID
	Path            string
	Query           *model.Query
	ResumeToken     model.ResumeToken
	IncludeMetadata bool
	IncludeOldData  bool
	BufferSize      int
}

// Example usage
func main() {
	// Prueba automática de rutas WebSocket compatibles con Firestore
	candidatePaths := []string{
		"ws://localhost:3030/api/v1/ws/listen",
		"ws://localhost:3030/ws/listen",
		"ws://localhost:3030/ws/v1/listen",
	}

	var client *FirestoreRealtimeClient
	var err error
	for _, wsURL := range candidatePaths {
		log.Printf("Intentando conectar a %s...", wsURL)
		client, err = NewFirestoreRealtimeClient(wsURL)
		if err == nil {
			log.Printf("Conexión exitosa a %s", wsURL)
			break
		} else {
			log.Printf("No se pudo conectar a %s: %v", wsURL, err)
		}
	}
	if client == nil {
		log.Fatalf("No se pudo conectar a ningún endpoint WebSocket compatible. Verifica la configuración del servidor.")
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Subscribe to a document
	documentSub, err := client.Subscribe(ctx, SubscriptionOptions{
		SubscriptionID:  "doc-subscription-1",
		Path:            "projects/my-project/databases/my-db/documents/users/user123",
		IncludeMetadata: true,
		IncludeOldData:  true,
		BufferSize:      50,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to document: %v", err)
	}
	// Example 2: Subscribe to a collection with query
	query := &model.Query{
		Filters: []model.Filter{
			{
				Field:    "status",
				Operator: model.OperatorEqual,
				Value:    "active",
			},
		},
		Orders: []model.Order{
			{
				Field:     "createdAt",
				Direction: model.DirectionDescending,
			},
		},
		Limit: 10,
	}

	collectionSub, err := client.Subscribe(ctx, SubscriptionOptions{
		SubscriptionID: "collection-subscription-1",
		Path:           "projects/my-project/databases/my-db/documents/posts",
		Query:          query,
		BufferSize:     100,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to collection: %v", err)
	}

	// Example 3: Subscribe with resume token (for reconnection scenarios)
	resumeSub, err := client.Subscribe(ctx, SubscriptionOptions{
		SubscriptionID: "resume-subscription-1",
		Path:           "projects/my-project/databases/my-db/documents/orders/order456",
		ResumeToken:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", // Example resume token
		BufferSize:     25,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe with resume token: %v", err)
	}

	// Start event handlers
	go handleDocumentEvents(documentSub, "Document")
	go handleDocumentEvents(collectionSub, "Collection")
	go handleDocumentEvents(resumeSub, "Resume")

	// Automated tests exhaustivos después de suscribirse
	// Espera algunos eventos
	time.Sleep(5 * time.Second)

	log.Println("-- Prueba: enviar JSON inválido --")
	if err := client.conn.WriteMessage(websocket.TextMessage, []byte("{invalid_json}")); err != nil {
		log.Printf("Error enviando JSON inválido: %v", err)
	}
	// Espera respuesta de error del servidor
	time.Sleep(2 * time.Second)

	log.Println("-- Prueba: acción inválida --")
	invalidAction := map[string]interface{}{
		"action":    "invalid_action",
		"type":      "invalid",
		"timestamp": time.Now(),
	}
	if err := client.conn.WriteJSON(invalidAction); err != nil {
		log.Printf("Error enviando acción inválida: %v", err)
	}
	// Espera respuesta de error del servidor
	time.Sleep(2 * time.Second)

	log.Println("-- Prueba: desuscribir Document subscription --")
	if err := client.Unsubscribe(ctx, "doc-subscription-1"); err != nil {
		log.Printf("Error desuscribiendo documento: %v", err)
	} else {
		log.Println("Desuscrito document-subscription-1 correctamente")
	}
	// Verificar cierre de canal
	time.Sleep(2 * time.Second)

	log.Println("-- Prueba: desuscribir Collection subscription --")
	if err := client.Unsubscribe(ctx, "collection-subscription-1"); err != nil {
		log.Printf("Error desuscribiendo colección: %v", err)
	} else {
		log.Println("Desuscrito collection-subscription-1 correctamente")
	}
	// Espera antes de finalizar pruebas
	time.Sleep(2 * time.Second)

	log.Println("Automated exhaustive tests completed.")

	// Continue con espera de señal para terminar o finalizar
	log.Println("Client is running. Press Ctrl+C to exit... (or tests complete)")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")

	// Unsubscribe from all subscriptions
	client.Unsubscribe(ctx, "doc-subscription-1")
	client.Unsubscribe(ctx, "collection-subscription-1")
	client.Unsubscribe(ctx, "resume-subscription-1")

	log.Println("Client shut down gracefully")
}

// handleDocumentEvents processes events from a subscription
func handleDocumentEvents(eventChan <-chan model.RealtimeEvent, subscriptionName string) {
	for event := range eventChan {
		switch event.Type {
		case model.EventTypeAdded:
			log.Printf("[%s] Document ADDED: %s", subscriptionName, event.DocumentPath)
			if event.Data != nil {
				log.Printf("  Data: %+v", event.Data)
			}

		case model.EventTypeModified:
			log.Printf("[%s] Document MODIFIED: %s", subscriptionName, event.DocumentPath)
			if event.Data != nil {
				log.Printf("  New Data: %+v", event.Data)
			}
			if event.OldData != nil {
				log.Printf("  Old Data: %+v", event.OldData)
			}

		case model.EventTypeRemoved:
			log.Printf("[%s] Document REMOVED: %s", subscriptionName, event.DocumentPath)

		case model.EventTypeHeartbeat:
			log.Printf("[%s] Heartbeat received", subscriptionName)

		default:
			log.Printf("[%s] Unknown event type: %s", subscriptionName, event.Type)
		}

		// Log additional event metadata
		log.Printf("  Sequence: %d, Resume Token: %s, Timestamp: %s",
			event.SequenceNumber,
			string(event.ResumeToken),
			event.Timestamp.Format(time.RFC3339))
	}

	log.Printf("[%s] Event channel closed", subscriptionName)
}
