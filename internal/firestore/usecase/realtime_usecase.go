package usecase

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"
	"sync"

	"go.uber.org/zap" // Assuming zap logger is used via shared logger
)

// RealtimeUsecase defines the interface for managing real-time subscriptions and event broadcasting.
type RealtimeUsecase interface {
	// Subscribe allows a client to subscribe to changes on a specific path.
	// subscriberID should be unique per client connection.
	// path is the document or collection path.
	// eventChannel is a channel owned by the subscriber to receive events.
	Subscribe(ctx context.Context, subscriberID string, path string, eventChannel chan<- model.RealtimeEvent) error

	// Unsubscribe removes a client's subscription.
	Unsubscribe(ctx context.Context, subscriberID string, path string) error

	// PublishEvent broadcasts an event to all clients subscribed to the event's path.
	// This method would be called by FirestoreUsecase or an event bus when data changes.
	PublishEvent(ctx context.Context, event model.RealtimeEvent) error
}

type realtimeUsecaseImpl struct {
	// subscriptions maps a path to a map of subscriber IDs to their event channels.
	// Example: "documents/doc1" -> {"client1": client1Chan, "client2": client2Chan}
	subscriptions map[string]map[string]chan<- model.RealtimeEvent
	mu            sync.RWMutex
	log           logger.Logger
}

// NewRealtimeUsecase creates a new instance of RealtimeUsecase.
func NewRealtimeUsecase(log logger.Logger) RealtimeUsecase {
	return &realtimeUsecaseImpl{
		subscriptions: make(map[string]map[string]chan<- model.RealtimeEvent),
		log:           log,
	}
}

// Subscribe implements the RealtimeUsecase interface.
func (uc *realtimeUsecaseImpl) Subscribe(ctx context.Context, subscriberID string, path string, eventChannel chan<- model.RealtimeEvent) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if _, ok := uc.subscriptions[path]; !ok {
		uc.subscriptions[path] = make(map[string]chan<- model.RealtimeEvent)
	}

	if _, ok := uc.subscriptions[path][subscriberID]; ok {
		// Already subscribed, perhaps close old channel or return error?
		// For now, let's log and overwrite. Or return an error.
		uc.log.Warn(ctx, "Subscriber already subscribed to path, overwriting subscription", zap.String("subscriberID", subscriberID), zap.String("path", path))
	}

	uc.subscriptions[path][subscriberID] = eventChannel
	uc.log.Info(ctx, "Client subscribed", zap.String("subscriberID", subscriberID), zap.String("path", path))
	return nil
}

// Unsubscribe implements the RealtimeUsecase interface.
func (uc *realtimeUsecaseImpl) Unsubscribe(ctx context.Context, subscriberID string, path string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if subscribers, pathExists := uc.subscriptions[path]; pathExists {
		if ch, subscriberExists := subscribers[subscriberID]; subscriberExists {
			// It's the responsibility of the subscriber (e.g., ws_handler) to close the channel
			// when the client disconnects or explicitly unsubscribes from the client-side.
			// Here, we just remove it from our tracking.
			delete(subscribers, subscriberID)
			uc.log.Info(ctx, "Client channel removed from subscription", zap.String("subscriberID", subscriberID), zap.String("path", path), zap.Any("channel", ch))


			if len(subscribers) == 0 {
				delete(uc.subscriptions, path)
				uc.log.Info(ctx, "No more subscribers for path, removing path from subscriptions", zap.String("path", path))
			}
			uc.log.Info(ctx, "Client unsubscribed", zap.String("subscriberID", subscriberID), zap.String("path", path))
			return nil
		}
		uc.log.Warn(ctx, "Subscriber not found for path during unsubscribe", zap.String("subscriberID", subscriberID), zap.String("path", path))
		return nil // Or return an error like ErrNotFound
	}
	uc.log.Warn(ctx, "Path not found during unsubscribe", zap.String("subscriberID", subscriberID), zap.String("path", path))
	return nil // Or return an error
}

// PublishEvent implements the RealtimeUsecase interface.
// This is a basic implementation. It sends to exact path matches.
// Firestore also supports listening to collections (all documents under a path)
// and potentially ancestor listeners, which would require more complex matching.
func (uc *realtimeUsecaseImpl) PublishEvent(ctx context.Context, event model.RealtimeEvent) error {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	pathSubscribers, pathExists := uc.subscriptions[event.Path]
	if !pathExists {
		uc.log.Debug(ctx, "No subscribers for path on event publish", zap.String("path", event.Path))
		return nil // No one is listening to this specific document path
	}

	uc.log.Info(ctx, "Publishing event", zap.String("path", event.Path), zap.Any("eventType", event.Type), zap.Int("subscriberCount", len(pathSubscribers)))

	for subID, ch := range pathSubscribers {
		// Non-blocking send to prevent a slow client from blocking event distribution.
		// The channel should be buffered by the subscriber if needed.
		// If the channel is full, the event is dropped for that subscriber.
		select {
		case ch <- event:
			uc.log.Debug(ctx, "Event sent to subscriber", zap.String("subscriberID", subID), zap.String("path", event.Path))
		default:
			// This means the subscriber's channel is full or closed.
			// Consider logging this, or perhaps a mechanism to remove consistently slow/stuck subscribers.
			uc.log.Warn(ctx, "Failed to send event to subscriber (channel full or closed)", zap.String("subscriberID", subID), zap.String("path", event.Path))
			// Potentially, we could try to remove this subscriber if this happens often,
			// but that requires more state and logic.
		}
	}
	return nil
}
