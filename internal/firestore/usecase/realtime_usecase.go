package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"

	"go.uber.org/zap"
)

// RealtimeUsecase defines the interface for managing real-time subscriptions and event broadcasting.
type RealtimeUsecase interface {
	// Subscribe allows a client to subscribe to changes on a specific Firestore path.
	// subscriberID should be unique per client connection.
	// firestorePath is the complete Firestore path: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{DOCUMENT_PATH}
	// eventChannel is a channel owned by the subscriber to receive events.
	Subscribe(ctx context.Context, subscriberID string, firestorePath string, eventChannel chan<- model.RealtimeEvent) error

	// Unsubscribe removes a client's subscription.
	Unsubscribe(ctx context.Context, subscriberID string, firestorePath string) error

	// UnsubscribeAll removes all subscriptions for a subscriber (used when client disconnects)
	UnsubscribeAll(ctx context.Context, subscriberID string) error

	// PublishEvent broadcasts an event to all clients subscribed to the event's path.
	// This method would be called by FirestoreUsecase when data changes.
	PublishEvent(ctx context.Context, event model.RealtimeEvent) error

	// GetSubscriberCount returns the number of subscribers for a given path
	GetSubscriberCount(firestorePath string) int
}

// Subscription represents an active subscription
type Subscription struct {
	SubscriberID  string
	FirestorePath string
	EventChannel  chan<- model.RealtimeEvent
	CreatedAt     time.Time
}

type realtimeUsecaseImpl struct {
	// subscriptions maps a Firestore path to a map of subscriber IDs to their subscriptions
	subscriptions map[string]map[string]*Subscription

	// subscriberPaths maps subscriber IDs to all paths they're subscribed to
	subscriberPaths map[string]map[string]bool

	mu  sync.RWMutex
	log logger.Logger
}

// NewRealtimeUsecase creates a new instance of RealtimeUsecase.
func NewRealtimeUsecase(log logger.Logger) RealtimeUsecase {
	return &realtimeUsecaseImpl{
		subscriptions:   make(map[string]map[string]*Subscription),
		subscriberPaths: make(map[string]map[string]bool),
		log:             log,
	}
}

// Subscribe implements the RealtimeUsecase interface.
func (uc *realtimeUsecaseImpl) Subscribe(ctx context.Context, subscriberID string, firestorePath string, eventChannel chan<- model.RealtimeEvent) error {
	// Validate Firestore path
	pathInfo, err := firestore.ParseFirestorePath(firestorePath)
	if err != nil {
		return errors.NewValidationError(fmt.Sprintf("invalid firestore path: %v", err))
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Initialize path subscriptions if not exists
	if _, ok := uc.subscriptions[firestorePath]; !ok {
		uc.subscriptions[firestorePath] = make(map[string]*Subscription)
	}

	// Initialize subscriber paths if not exists
	if _, ok := uc.subscriberPaths[subscriberID]; !ok {
		uc.subscriberPaths[subscriberID] = make(map[string]bool)
	}

	// Check if already subscribed
	if _, ok := uc.subscriptions[firestorePath][subscriberID]; ok {
		uc.log.Warn("Subscriber already subscribed to path, overwriting subscription",
			zap.String("subscriberID", subscriberID),
			zap.String("firestorePath", firestorePath))
	}

	// Create subscription
	subscription := &Subscription{
		SubscriberID:  subscriberID,
		FirestorePath: firestorePath,
		EventChannel:  eventChannel,
		CreatedAt:     time.Now(),
	}

	uc.subscriptions[firestorePath][subscriberID] = subscription
	uc.subscriberPaths[subscriberID][firestorePath] = true

	uc.log.Info("Client subscribed",
		zap.String("subscriberID", subscriberID),
		zap.String("firestorePath", firestorePath),
		zap.String("projectID", pathInfo.ProjectID),
		zap.String("databaseID", pathInfo.DatabaseID))

	return nil
}

// Unsubscribe implements the RealtimeUsecase interface.
func (uc *realtimeUsecaseImpl) Unsubscribe(ctx context.Context, subscriberID string, firestorePath string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	return uc.unsubscribeInternal(subscriberID, firestorePath)
}

// UnsubscribeAll removes all subscriptions for a subscriber
func (uc *realtimeUsecaseImpl) UnsubscribeAll(ctx context.Context, subscriberID string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	subscriberPaths, exists := uc.subscriberPaths[subscriberID]
	if !exists {
		uc.log.Debug("No subscriptions found for subscriber", zap.String("subscriberID", subscriberID))
		return nil
	}

	// Unsubscribe from all paths
	for firestorePath := range subscriberPaths {
		uc.unsubscribeInternal(subscriberID, firestorePath)
	}

	// Clean up subscriber paths
	delete(uc.subscriberPaths, subscriberID)

	uc.log.Info("All subscriptions removed for subscriber", zap.String("subscriberID", subscriberID))
	return nil
}

// unsubscribeInternal handles unsubscription logic (must be called with lock held)
func (uc *realtimeUsecaseImpl) unsubscribeInternal(subscriberID string, firestorePath string) error {
	if subscribers, pathExists := uc.subscriptions[firestorePath]; pathExists {
		if _, subscriberExists := subscribers[subscriberID]; subscriberExists {
			delete(subscribers, subscriberID)

			// Clean up empty path subscriptions
			if len(subscribers) == 0 {
				delete(uc.subscriptions, firestorePath)
			}

			// Remove from subscriber paths
			if subscriberPaths, exists := uc.subscriberPaths[subscriberID]; exists {
				delete(subscriberPaths, firestorePath)

				// Clean up empty subscriber entries
				if len(subscriberPaths) == 0 {
					delete(uc.subscriberPaths, subscriberID)
				}
			}

			uc.log.Info("Client unsubscribed",
				zap.String("subscriberID", subscriberID),
				zap.String("firestorePath", firestorePath))
			return nil
		}
	}

	uc.log.Debug("Subscription not found during unsubscribe",
		zap.String("subscriberID", subscriberID),
		zap.String("firestorePath", firestorePath))
	return nil
}

// PublishEvent implements the RealtimeUsecase interface.
// Publishes events to subscribers based on Firestore path matching
func (uc *realtimeUsecaseImpl) PublishEvent(ctx context.Context, event model.RealtimeEvent) error {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	// Get direct subscribers for the exact path
	exactSubscribers := uc.getExactPathSubscribers(event.FullPath)

	// Get collection subscribers (if this is a document event)
	collectionSubscribers := uc.getCollectionSubscribers(event.FullPath)

	allSubscribers := make(map[string]*Subscription)

	// Merge subscribers (avoid duplicates)
	for id, sub := range exactSubscribers {
		allSubscribers[id] = sub
	}
	for id, sub := range collectionSubscribers {
		allSubscribers[id] = sub
	}

	if len(allSubscribers) == 0 {
		uc.log.Debug("No subscribers for event path", zap.String("fullPath", event.FullPath))
		return nil
	}

	uc.log.Info("Publishing event",
		zap.String("fullPath", event.FullPath),
		zap.String("eventType", string(event.Type)),
		zap.Int("subscriberCount", len(allSubscribers)))

	// Send event to all subscribers
	for subID, subscription := range allSubscribers {
		select {
		case subscription.EventChannel <- event:
			uc.log.Debug("Event sent to subscriber",
				zap.String("subscriberID", subID),
				zap.String("fullPath", event.FullPath))
		default:
			// Channel is full or closed
			uc.log.Warn("Failed to send event to subscriber (channel full or closed)",
				zap.String("subscriberID", subID),
				zap.String("fullPath", event.FullPath))
		}
	}

	return nil
}

// getExactPathSubscribers returns subscribers for the exact path
func (uc *realtimeUsecaseImpl) getExactPathSubscribers(firestorePath string) map[string]*Subscription {
	subscribers := make(map[string]*Subscription)

	if pathSubscribers, exists := uc.subscriptions[firestorePath]; exists {
		for id, sub := range pathSubscribers {
			subscribers[id] = sub
		}
	}

	return subscribers
}

// getCollectionSubscribers returns subscribers listening to the collection containing this document
func (uc *realtimeUsecaseImpl) getCollectionSubscribers(documentPath string) map[string]*Subscription {
	subscribers := make(map[string]*Subscription)

	// Parse the document path to find parent collection
	pathInfo, err := firestore.ParseFirestorePath(documentPath)
	if err != nil {
		return subscribers
	}

	// If this is not a document path, return empty
	if !pathInfo.IsDocument {
		return subscribers
	}

	// Build collection path by removing the last segment (document ID)
	segments := pathInfo.Segments
	if len(segments) < 2 {
		return subscribers
	}

	// Remove document ID to get collection path
	collectionSegments := segments[:len(segments)-1]
	collectionDocPath := strings.Join(collectionSegments, "/")
	collectionFullPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s",
		pathInfo.ProjectID, pathInfo.DatabaseID, collectionDocPath)

	// Find subscribers to the collection
	if pathSubscribers, exists := uc.subscriptions[collectionFullPath]; exists {
		for id, sub := range pathSubscribers {
			subscribers[id] = sub
		}
	}

	return subscribers
}

// GetSubscriberCount returns the number of subscribers for a given path
func (uc *realtimeUsecaseImpl) GetSubscriberCount(firestorePath string) int {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	if subscribers, exists := uc.subscriptions[firestorePath]; exists {
		return len(subscribers)
	}

	return 0
}
