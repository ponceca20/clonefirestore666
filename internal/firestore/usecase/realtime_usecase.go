package usecase

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"

	"go.uber.org/zap"
)

// Constants aligned with Google Cloud Firestore specifications
const (
	MaxEventsPerPath              = 1000
	DefaultHeartbeatInterval      = 30 * time.Second
	DefaultStaleConnectionTimeout = 5 * time.Minute
	MaxSubscriptionsPerClient     = 100
	EventChannelBufferSize        = 1000
)

// RealtimeUsecase defines the primary port for real-time operations following hexagonal architecture
type RealtimeUsecase interface {
	Subscribe(ctx context.Context, req SubscribeRequest) (*SubscribeResponse, error)
	Unsubscribe(ctx context.Context, req UnsubscribeRequest) error
	UnsubscribeAll(ctx context.Context, subscriberID string) error
	PublishEvent(ctx context.Context, event model.RealtimeEvent) error
	GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error)
	SendHeartbeat(ctx context.Context) error
	UpdateLastHeartbeat(subscriberID string, subscriptionID model.SubscriptionID) error
	CleanupStaleConnections(ctx context.Context, timeout time.Duration) error
	GetSubscriberCount(firestorePath string) int
	GetActiveSubscriptions(subscriberID string) map[model.SubscriptionID]*Subscription
	ValidatePermissions(ctx context.Context, subscriberID string, permissionValidator PermissionValidator) error
	GetHealthStatus() HealthStatus
	GetMetrics() RealtimeMetrics
}

// SubscribeRequest encapsulates subscription parameters following clean code principles
type SubscribeRequest struct {
	SubscriberID   string                     `json:"subscriber_id" validate:"required"`
	SubscriptionID model.SubscriptionID       `json:"subscription_id" validate:"required"`
	FirestorePath  string                     `json:"firestore_path" validate:"required"`
	EventChannel   chan<- model.RealtimeEvent `json:"-"`
	ResumeToken    model.ResumeToken          `json:"resume_token,omitempty"`
	Query          *model.Query               `json:"query,omitempty"`
	Options        SubscriptionOptions        `json:"options,omitempty"`
}

// SubscribeResponse contains subscription operation results
type SubscribeResponse struct {
	SubscriptionID  model.SubscriptionID `json:"subscription_id"`
	InitialSnapshot bool                 `json:"initial_snapshot"`
	ResumeToken     model.ResumeToken    `json:"resume_token"`
	CreatedAt       time.Time            `json:"created_at"`
}

// UnsubscribeRequest encapsulates unsubscription parameters
type UnsubscribeRequest struct {
	SubscriberID   string               `json:"subscriber_id" validate:"required"`
	SubscriptionID model.SubscriptionID `json:"subscription_id" validate:"required"`
}

// SubscriptionOptions provides additional subscription configuration
type SubscriptionOptions struct {
	IncludeMetadata   bool          `json:"include_metadata"`
	IncludeOldData    bool          `json:"include_old_data"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

// PermissionValidator validates access permissions
type PermissionValidator func(subscriberID, firestorePath string) error

// HealthStatus represents service health state
type HealthStatus struct {
	IsHealthy           bool      `json:"is_healthy"`
	ActiveSubscriptions int       `json:"active_subscriptions"`
	ActiveConnections   int       `json:"active_connections"`
	LastHealthCheck     time.Time `json:"last_health_check"`
	EventStoreSize      int       `json:"event_store_size"`
}

// RealtimeMetrics provides operational metrics
type RealtimeMetrics struct {
	TotalSubscriptions int64         `json:"total_subscriptions"`
	TotalEvents        int64         `json:"total_events"`
	ActiveSubscribers  int           `json:"active_subscribers"`
	EventsPerSecond    float64       `json:"events_per_second"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastMetricsUpdate  time.Time     `json:"last_metrics_update"`
}

// Subscription represents an active subscription with Firestore features
type Subscription struct {
	SubscriberID   string                     `json:"subscriber_id"`
	SubscriptionID model.SubscriptionID       `json:"subscription_id"`
	FirestorePath  string                     `json:"firestore_path"`
	EventChannel   chan<- model.RealtimeEvent `json:"-"`
	CreatedAt      time.Time                  `json:"created_at"`
	LastHeartbeat  time.Time                  `json:"last_heartbeat"`
	ResumeToken    model.ResumeToken          `json:"resume_token"`
	Query          *model.Query               `json:"query,omitempty"`
	IsActive       bool                       `json:"is_active"`
	Options        SubscriptionOptions        `json:"options"`
}

// EventStore defines the secondary port for event persistence
type EventStore interface {
	StoreEvent(ctx context.Context, event model.RealtimeEvent) error
	GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error)
	CleanupOldEvents(ctx context.Context, retentionPeriod time.Duration) error
	GetEventCount(firestorePath string) int
}

// InMemoryEventStore implements EventStore with in-memory storage
type InMemoryEventStore struct {
	events       map[string][]model.RealtimeEvent
	eventMetrics map[string]*EventPathMetrics
	mu           sync.RWMutex
	logger       logger.Logger
}

// EventPathMetrics tracks metrics per path
type EventPathMetrics struct {
	TotalEvents   int64     `json:"total_events"`
	LastEventTime time.Time `json:"last_event_time"`
	StorageSize   int       `json:"storage_size"`
}

// NewInMemoryEventStore creates a new event store
func NewInMemoryEventStore(log logger.Logger) EventStore {
	return &InMemoryEventStore{
		events:       make(map[string][]model.RealtimeEvent),
		eventMetrics: make(map[string]*EventPathMetrics),
		logger:       log,
	}
}

// StoreEvent stores an event with memory management
func (es *InMemoryEventStore) StoreEvent(ctx context.Context, event model.RealtimeEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Initialize path storage if needed
	if es.events[event.FullPath] == nil {
		es.events[event.FullPath] = make([]model.RealtimeEvent, 0, MaxEventsPerPath)
		es.eventMetrics[event.FullPath] = &EventPathMetrics{
			TotalEvents:   0,
			LastEventTime: time.Now(),
			StorageSize:   0,
		}
	}

	// Add event
	es.events[event.FullPath] = append(es.events[event.FullPath], event)

	// Update metrics
	metrics := es.eventMetrics[event.FullPath]
	metrics.TotalEvents++
	metrics.LastEventTime = event.Timestamp
	metrics.StorageSize = len(es.events[event.FullPath])

	// Maintain sliding window to prevent memory leaks
	if len(es.events[event.FullPath]) > MaxEventsPerPath {
		// Remove oldest events
		excess := len(es.events[event.FullPath]) - MaxEventsPerPath
		copy(es.events[event.FullPath], es.events[event.FullPath][excess:])
		es.events[event.FullPath] = es.events[event.FullPath][:MaxEventsPerPath]
		metrics.StorageSize = MaxEventsPerPath
	}

	return nil
}

// GetEventsSince retrieves events after a resume token
func (es *InMemoryEventStore) GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, exists := es.events[firestorePath]
	if !exists {
		return []model.RealtimeEvent{}, nil
	}

	if resumeToken == "" {
		return append([]model.RealtimeEvent{}, events...), nil
	}

	// Find events after resume token
	var result []model.RealtimeEvent
	for _, event := range events {
		if event.ResumeToken > resumeToken {
			result = append(result, event)
		}
	}

	return result, nil
}

// CleanupOldEvents removes events older than retention period
func (es *InMemoryEventStore) CleanupOldEvents(ctx context.Context, retentionPeriod time.Duration) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	cleanedPaths := 0

	for path, events := range es.events {
		var filteredEvents []model.RealtimeEvent
		for _, event := range events {
			if event.Timestamp.After(cutoff) {
				filteredEvents = append(filteredEvents, event)
			}
		}

		if len(filteredEvents) != len(events) {
			es.events[path] = filteredEvents
			if metrics := es.eventMetrics[path]; metrics != nil {
				metrics.StorageSize = len(filteredEvents)
			}
			cleanedPaths++
		}
	}

	if cleanedPaths > 0 {
		es.logger.Info("Cleaned up old events", zap.Int("paths_cleaned", cleanedPaths))
	}

	return nil
}

// GetEventCount returns event count for a path
func (es *InMemoryEventStore) GetEventCount(firestorePath string) int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if events, exists := es.events[firestorePath]; exists {
		return len(events)
	}
	return 0
}

// realtimeUsecaseImpl implements RealtimeUsecase with enhanced Firestore compatibility
type realtimeUsecaseImpl struct {
	// Core subscription management
	subscriptions   map[string]map[model.SubscriptionID]*Subscription
	pathSubscribers map[string]map[string]map[model.SubscriptionID]bool

	// Event management
	eventStore      EventStore
	sequenceCounter int64

	// Synchronization and logging
	mu     sync.RWMutex
	logger logger.Logger

	// Metrics tracking
	startTime    time.Time
	eventCounter int64
}

// NewRealtimeUsecase creates a new realtime usecase following hexagonal architecture
func NewRealtimeUsecase(log logger.Logger) RealtimeUsecase {
	eventStore := NewInMemoryEventStore(log)

	return &realtimeUsecaseImpl{
		subscriptions:   make(map[string]map[model.SubscriptionID]*Subscription),
		pathSubscribers: make(map[string]map[string]map[model.SubscriptionID]bool),
		eventStore:      eventStore,
		logger:          log,
		startTime:       time.Now(),
	}
}

// NewRealtimeUsecaseWithEventStore creates a new realtime usecase with custom event store
// This constructor enables dependency injection for Redis or other persistent event stores
func NewRealtimeUsecaseWithEventStore(log logger.Logger, eventStore EventStore) RealtimeUsecase {
	return &realtimeUsecaseImpl{
		subscriptions:   make(map[string]map[model.SubscriptionID]*Subscription),
		pathSubscribers: make(map[string]map[string]map[model.SubscriptionID]bool),
		eventStore:      eventStore,
		logger:          log,
		startTime:       time.Now(),
	}
}

// Subscribe implements subscription with full Firestore compatibility
func (r *realtimeUsecaseImpl) Subscribe(ctx context.Context, req SubscribeRequest) (*SubscribeResponse, error) {
	if err := r.validateSubscribeRequest(req); err != nil {
		return nil, err
	}

	if err := r.checkSubscriptionLimits(req.SubscriberID); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize subscriber map if needed
	if r.subscriptions[req.SubscriberID] == nil {
		r.subscriptions[req.SubscriberID] = make(map[model.SubscriptionID]*Subscription)
	}

	// Check for duplicate subscription
	if _, exists := r.subscriptions[req.SubscriberID][req.SubscriptionID]; exists {
		return nil, errors.NewValidationError(fmt.Sprintf("subscription %s already exists", req.SubscriptionID))
	}

	// Create subscription
	subscription := &Subscription{
		SubscriberID:   req.SubscriberID,
		SubscriptionID: req.SubscriptionID,
		FirestorePath:  req.FirestorePath,
		EventChannel:   req.EventChannel,
		CreatedAt:      time.Now(),
		LastHeartbeat:  time.Now(),
		ResumeToken:    req.ResumeToken,
		Query:          req.Query,
		IsActive:       true,
		Options:        req.Options,
	}

	// Store subscription
	r.subscriptions[req.SubscriberID][req.SubscriptionID] = subscription
	r.addToPathSubscribers(req.FirestorePath, req.SubscriberID, req.SubscriptionID)

	response := &SubscribeResponse{
		SubscriptionID:  req.SubscriptionID,
		InitialSnapshot: req.ResumeToken == "",
		ResumeToken:     req.ResumeToken,
		CreatedAt:       subscription.CreatedAt,
	}

	r.logger.Info("Subscription created",
		zap.String("subscriberID", req.SubscriberID),
		zap.String("subscriptionID", string(req.SubscriptionID)),
		zap.String("path", req.FirestorePath))

	// Send historical events if resume token provided
	if req.ResumeToken != "" {
		go r.sendEventsFromResumeToken(ctx, subscription, req.ResumeToken)
	}

	return response, nil
}

// Unsubscribe removes a specific subscription
func (r *realtimeUsecaseImpl) Unsubscribe(ctx context.Context, req UnsubscribeRequest) error {
	if err := r.validateUnsubscribeRequest(req); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	subscriberSubs, exists := r.subscriptions[req.SubscriberID]
	if !exists {
		// Gracefully handle non-existent subscriber - log and return nil
		r.logger.Debug("Subscriber not found during unsubscribe - gracefully handling",
			zap.String("subscriberID", req.SubscriberID))
		return nil
	}

	subscription, exists := subscriberSubs[req.SubscriptionID]
	if !exists {
		// Gracefully handle non-existent subscription - log and return nil
		r.logger.Debug("Subscription not found during unsubscribe - gracefully handling",
			zap.String("subscriberID", req.SubscriberID),
			zap.String("subscriptionID", string(req.SubscriptionID)))
		return nil
	}

	// Deactivate and clean up
	subscription.IsActive = false
	r.removeFromPathSubscribers(subscription.FirestorePath, req.SubscriberID, req.SubscriptionID)

	delete(subscriberSubs, req.SubscriptionID)
	if len(subscriberSubs) == 0 {
		delete(r.subscriptions, req.SubscriberID)
	}

	r.logger.Info("Subscription removed",
		zap.String("subscriberID", req.SubscriberID),
		zap.String("subscriptionID", string(req.SubscriptionID)))

	return nil
}

// UnsubscribeAll removes all subscriptions for a subscriber
func (r *realtimeUsecaseImpl) UnsubscribeAll(ctx context.Context, subscriberID string) error {
	if subscriberID == "" {
		return errors.NewValidationError("subscriberID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	subscriberSubs, exists := r.subscriptions[subscriberID]
	if !exists {
		return nil // Already clean
	}

	// Deactivate all subscriptions
	for _, subscription := range subscriberSubs {
		subscription.IsActive = false
		r.removeFromPathSubscribers(subscription.FirestorePath, subscriberID, subscription.SubscriptionID)
	}

	delete(r.subscriptions, subscriberID)

	r.logger.Info("All subscriptions removed",
		zap.String("subscriberID", subscriberID),
		zap.Int("count", len(subscriberSubs)))

	return nil
}

// PublishEvent broadcasts events following Firestore semantics
func (r *realtimeUsecaseImpl) PublishEvent(ctx context.Context, event model.RealtimeEvent) error {
	if event.FullPath == "" {
		return errors.NewValidationError("event path cannot be empty")
	}

	// Generate sequence and resume token
	sequence := atomic.AddInt64(&r.sequenceCounter, 1)
	event.SequenceNumber = sequence
	event.ResumeToken = event.GenerateResumeToken()

	// Store event
	if err := r.eventStore.StoreEvent(ctx, event); err != nil {
		r.logger.Error("Failed to store event", zap.Error(err))
	}

	atomic.AddInt64(&r.eventCounter, 1)

	// Find target subscriptions
	r.mu.RLock()
	var targetSubscriptions []*Subscription

	if pathSubscribers, exists := r.pathSubscribers[event.FullPath]; exists {
		for subscriberID, subscriptionIDs := range pathSubscribers {
			if subscriberSubs, exists := r.subscriptions[subscriberID]; exists {
				for subscriptionID := range subscriptionIDs {
					if subscription, exists := subscriberSubs[subscriptionID]; exists && subscription.IsActive {
						targetSubscriptions = append(targetSubscriptions, subscription)
					}
				}
			}
		}
	}
	r.mu.RUnlock()

	// Send events concurrently
	var wg sync.WaitGroup
	for _, subscription := range targetSubscriptions {
		wg.Add(1)
		go func(sub *Subscription) {
			defer wg.Done()
			r.sendEventToSubscription(ctx, event, sub)
		}(subscription)
	}
	wg.Wait()

	r.logger.Debug("Event published",
		zap.String("path", event.FullPath),
		zap.String("eventType", string(event.Type)),
		zap.Int("subscribers", len(targetSubscriptions)))

	return nil
}

// GetEventsSince returns events for replay
func (r *realtimeUsecaseImpl) GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error) {
	return r.eventStore.GetEventsSince(ctx, firestorePath, resumeToken)
}

// SendHeartbeat sends heartbeats to all subscribers
func (r *realtimeUsecaseImpl) SendHeartbeat(ctx context.Context) error {
	r.mu.RLock()
	var allSubscriptions []*Subscription
	for _, subscriberSubs := range r.subscriptions {
		for _, subscription := range subscriberSubs {
			if subscription.IsActive {
				allSubscriptions = append(allSubscriptions, subscription)
			}
		}
	}
	r.mu.RUnlock()

	heartbeatEvent := model.RealtimeEvent{
		Type:           model.EventTypeHeartbeat,
		FullPath:       "",
		Timestamp:      time.Now(),
		SequenceNumber: atomic.AddInt64(&r.sequenceCounter, 1),
	}

	for _, subscription := range allSubscriptions {
		select {
		case subscription.EventChannel <- heartbeatEvent:
			subscription.LastHeartbeat = time.Now()
		case <-ctx.Done():
			return ctx.Err()
		default:
			r.logger.Warn("Heartbeat channel full",
				zap.String("subscriberID", subscription.SubscriberID))
		}
	}

	return nil
}

// UpdateLastHeartbeat updates heartbeat timestamp
func (r *realtimeUsecaseImpl) UpdateLastHeartbeat(subscriberID string, subscriptionID model.SubscriptionID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subscriberSubs, exists := r.subscriptions[subscriberID]; exists {
		if subscription, exists := subscriberSubs[subscriptionID]; exists {
			subscription.LastHeartbeat = time.Now()
			return nil
		}
	}
	return errors.NewNotFoundError("subscription not found")
}

// CleanupStaleConnections removes inactive subscriptions
func (r *realtimeUsecaseImpl) CleanupStaleConnections(ctx context.Context, timeout time.Duration) error {
	r.mu.Lock()
	var staleSubscriptions []struct {
		subscriberID   string
		subscriptionID model.SubscriptionID
	}

	cutoff := time.Now().Add(-timeout)
	for subscriberID, subscriberSubs := range r.subscriptions {
		for subscriptionID, subscription := range subscriberSubs {
			if subscription.LastHeartbeat.Before(cutoff) {
				staleSubscriptions = append(staleSubscriptions, struct {
					subscriberID   string
					subscriptionID model.SubscriptionID
				}{subscriberID, subscriptionID})
			}
		}
	}
	r.mu.Unlock()

	// Remove stale subscriptions
	for _, stale := range staleSubscriptions {
		r.Unsubscribe(ctx, UnsubscribeRequest{
			SubscriberID:   stale.subscriberID,
			SubscriptionID: stale.subscriptionID,
		})
	}

	if len(staleSubscriptions) > 0 {
		r.logger.Info("Cleaned up stale connections", zap.Int("count", len(staleSubscriptions)))
	}

	return nil
}

// GetSubscriberCount returns subscriber count for a path
func (r *realtimeUsecaseImpl) GetSubscriberCount(firestorePath string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	if pathSubs, exists := r.pathSubscribers[firestorePath]; exists {
		for _, subscriptionIDs := range pathSubs {
			count += len(subscriptionIDs)
		}
	}
	return count
}

// GetActiveSubscriptions returns active subscriptions for a subscriber
func (r *realtimeUsecaseImpl) GetActiveSubscriptions(subscriberID string) map[model.SubscriptionID]*Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[model.SubscriptionID]*Subscription)
	if subscriberSubs, exists := r.subscriptions[subscriberID]; exists {
		for subscriptionID, subscription := range subscriberSubs {
			if subscription.IsActive {
				// Return a copy to prevent race conditions
				result[subscriptionID] = &Subscription{
					SubscriberID:   subscription.SubscriberID,
					SubscriptionID: subscription.SubscriptionID,
					FirestorePath:  subscription.FirestorePath,
					EventChannel:   subscription.EventChannel,
					CreatedAt:      subscription.CreatedAt,
					LastHeartbeat:  subscription.LastHeartbeat,
					ResumeToken:    subscription.ResumeToken,
					Query:          subscription.Query,
					IsActive:       subscription.IsActive,
					Options:        subscription.Options,
				}
			}
		}
	}
	return result
}

// ValidatePermissions revalidates permissions for active subscriptions
func (r *realtimeUsecaseImpl) ValidatePermissions(ctx context.Context, subscriberID string, permissionValidator PermissionValidator) error {
	r.mu.RLock()
	subscriberSubs, exists := r.subscriptions[subscriberID]
	if !exists {
		r.mu.RUnlock()
		return nil
	}

	var subscriptionsToValidate []*Subscription
	for _, subscription := range subscriberSubs {
		if subscription.IsActive {
			subscriptionsToValidate = append(subscriptionsToValidate, subscription)
		}
	}
	r.mu.RUnlock()

	for _, subscription := range subscriptionsToValidate {
		if err := permissionValidator(subscriberID, subscription.FirestorePath); err != nil {
			r.Unsubscribe(ctx, UnsubscribeRequest{
				SubscriberID:   subscriberID,
				SubscriptionID: subscription.SubscriptionID,
			})
			r.logger.Warn("Subscription removed due to permission failure",
				zap.String("subscriberID", subscriberID),
				zap.Error(err))
		}
	}

	return nil
}

// GetHealthStatus returns current service health
func (r *realtimeUsecaseImpl) GetHealthStatus() HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalSubscriptions := 0
	for _, subscriberSubs := range r.subscriptions {
		totalSubscriptions += len(subscriberSubs)
	}

	return HealthStatus{
		IsHealthy:           true,
		ActiveSubscriptions: totalSubscriptions,
		ActiveConnections:   len(r.subscriptions),
		LastHealthCheck:     time.Now(),
		EventStoreSize:      r.eventStore.GetEventCount(""),
	}
}

// GetMetrics returns operational metrics
func (r *realtimeUsecaseImpl) GetMetrics() RealtimeMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalSubscriptions := 0
	for _, subscriberSubs := range r.subscriptions {
		totalSubscriptions += len(subscriberSubs)
	}

	// Calculate events per second
	uptime := time.Since(r.startTime).Seconds()
	eventsPerSecond := float64(atomic.LoadInt64(&r.eventCounter)) / uptime

	return RealtimeMetrics{
		TotalSubscriptions: int64(totalSubscriptions),
		TotalEvents:        atomic.LoadInt64(&r.eventCounter),
		ActiveSubscribers:  len(r.subscriptions),
		EventsPerSecond:    eventsPerSecond,
		AverageLatency:     time.Millisecond, // TODO: Implement actual latency tracking
		LastMetricsUpdate:  time.Now(),
	}
}

// Helper methods following clean code principles

func (r *realtimeUsecaseImpl) validateSubscribeRequest(req SubscribeRequest) error {
	if req.SubscriberID == "" {
		return errors.NewValidationError("subscriberID cannot be empty")
	}
	if req.SubscriptionID == "" {
		return errors.NewValidationError("subscriptionID cannot be empty")
	}
	if req.FirestorePath == "" {
		return errors.NewValidationError("firestorePath cannot be empty")
	}
	if req.EventChannel == nil {
		return errors.NewValidationError("eventChannel cannot be nil")
	}
	if !r.isValidFirestorePath(req.FirestorePath) {
		return errors.NewValidationError(fmt.Sprintf("invalid Firestore path: %s", req.FirestorePath))
	}
	return nil
}

func (r *realtimeUsecaseImpl) validateUnsubscribeRequest(req UnsubscribeRequest) error {
	if req.SubscriberID == "" {
		return errors.NewValidationError("subscriberID cannot be empty")
	}
	if req.SubscriptionID == "" {
		return errors.NewValidationError("subscriptionID cannot be empty")
	}
	return nil
}

func (r *realtimeUsecaseImpl) checkSubscriptionLimits(subscriberID string) error {
	if subscriberSubs, exists := r.subscriptions[subscriberID]; exists {
		if len(subscriberSubs) >= MaxSubscriptionsPerClient {
			return errors.NewValidationError("subscription limit exceeded")
		}
	}
	return nil
}

func (r *realtimeUsecaseImpl) isValidFirestorePath(path string) bool {
	_, err := firestore.ParseFirestorePath(path)
	return err == nil
}

func (r *realtimeUsecaseImpl) addToPathSubscribers(firestorePath, subscriberID string, subscriptionID model.SubscriptionID) {
	if r.pathSubscribers[firestorePath] == nil {
		r.pathSubscribers[firestorePath] = make(map[string]map[model.SubscriptionID]bool)
	}
	if r.pathSubscribers[firestorePath][subscriberID] == nil {
		r.pathSubscribers[firestorePath][subscriberID] = make(map[model.SubscriptionID]bool)
	}
	r.pathSubscribers[firestorePath][subscriberID][subscriptionID] = true
}

func (r *realtimeUsecaseImpl) removeFromPathSubscribers(firestorePath, subscriberID string, subscriptionID model.SubscriptionID) {
	if pathSubs, exists := r.pathSubscribers[firestorePath]; exists {
		if subscriberPathSubs, exists := pathSubs[subscriberID]; exists {
			delete(subscriberPathSubs, subscriptionID)
			if len(subscriberPathSubs) == 0 {
				delete(pathSubs, subscriberID)
			}
		}
		if len(pathSubs) == 0 {
			delete(r.pathSubscribers, firestorePath)
		}
	}
}

func (r *realtimeUsecaseImpl) sendEventsFromResumeToken(ctx context.Context, subscription *Subscription, resumeToken model.ResumeToken) {
	events, err := r.eventStore.GetEventsSince(ctx, subscription.FirestorePath, resumeToken)
	if err != nil {
		r.logger.Error("Failed to get events from resume token",
			zap.String("resumeToken", string(resumeToken)),
			zap.Error(err))
		return
	}

	for _, event := range events {
		r.sendEventToSubscription(ctx, event, subscription)
	}
}

func (r *realtimeUsecaseImpl) sendEventToSubscription(ctx context.Context, event model.RealtimeEvent, subscription *Subscription) {
	if !subscription.IsActive {
		return
	}

	// Apply query filtering if present
	if subscription.Query != nil && !r.matchesQuery(event, subscription.Query) {
		return
	}

	// Set subscription ID in event
	event.SubscriptionID = string(subscription.SubscriptionID)

	select {
	case subscription.EventChannel <- event:
		r.logger.Debug("Event sent to subscription",
			zap.String("subscriberID", subscription.SubscriberID),
			zap.String("eventType", string(event.Type)))
	case <-ctx.Done():
		return
	default:
		r.logger.Warn("Event channel full, dropping event",
			zap.String("subscriberID", subscription.SubscriberID))
	}
}

func (r *realtimeUsecaseImpl) matchesQuery(event model.RealtimeEvent, query *model.Query) bool {
	if query == nil {
		return true
	}
	data := event.Data
	if data == nil {
		return false
	}
	// Filtros compuestos y simples
	if !matchFilters(data, query.Filters) {
		return false
	}
	// allDescendants: si está activo, permite coincidencia en subcolecciones
	if query.AllDescendants && !pathMatchesDescendants(event.FullPath, query.Path) {
		return false
	}
	// Proyección de campos: si selectFields está definido, solo esos campos deben estar presentes
	if len(query.SelectFields) > 0 && !fieldsMatchProjection(data, query.SelectFields) {
		return false
	}
	// Paginación y ordenamiento: para eventos individuales, solo se puede filtrar por startAt/startAfter/endAt/endBefore si están presentes
	if !matchCursors(data, query) {
		return false
	}
	return true
}

// matchFilters soporta filtros simples y compuestos (AND/OR)
func matchFilters(data map[string]interface{}, filters []model.Filter) bool {
	for _, filter := range filters {
		if len(filter.SubFilters) > 0 && filter.Composite != "" {
			if filter.Composite == "and" {
				if !matchFilters(data, filter.SubFilters) {
					return false
				}
			} else if filter.Composite == "or" {
				matched := false
				for _, sub := range filter.SubFilters {
					if matchFilters(data, []model.Filter{sub}) {
						matched = true
						break
					}
				}
				if !matched {
					return false
				}
			}
			continue
		}
		var fieldValue interface{}
		if filter.FieldPath != nil {
			fieldValue = getNestedField(data, filter.FieldPath.Segments())
		} else {
			fieldValue = data[filter.Field]
		}
		if !applyOperator(fieldValue, filter.Operator, filter.Value) {
			return false
		}
	}
	return true
}

// pathMatchesDescendants verifica coincidencia de ruta o subcolección
func pathMatchesDescendants(fullPath, queryPath string) bool {
	return len(fullPath) >= len(queryPath) && fullPath[:len(queryPath)] == queryPath
}

// fieldsMatchProjection verifica que solo los campos proyectados estén presentes
func fieldsMatchProjection(data map[string]interface{}, selectFields []string) bool {
	for k := range data {
		found := false
		for _, f := range selectFields {
			if k == f {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// matchCursors soporta startAt, startAfter, endAt, endBefore, offset, limit, limitToLast
func matchCursors(data map[string]interface{}, query *model.Query) bool {
	// Para eventos individuales, solo se puede filtrar por startAt/startAfter/endAt/endBefore si están presentes
	// (En la práctica, esto se aplica mejor a lotes, pero aquí se filtra por el primer campo de ordenamiento)
	if len(query.Orders) > 0 {
		orderField := query.Orders[0].Field
		fieldValue := data[orderField]
		if len(query.StartAt) > 0 && compare(fieldValue, query.StartAt[0]) < 0 {
			return false
		}
		if len(query.StartAfter) > 0 && compare(fieldValue, query.StartAfter[0]) <= 0 {
			return false
		}
		if len(query.EndAt) > 0 && compare(fieldValue, query.EndAt[0]) > 0 {
			return false
		}
		if len(query.EndBefore) > 0 && compare(fieldValue, query.EndBefore[0]) >= 0 {
			return false
		}
	}
	// offset, limit, limitToLast: no aplican a eventos individuales, solo a lotes
	return true
}

// getNestedField permite acceder a campos anidados usando FieldPath
func getNestedField(data map[string]interface{}, segments []string) interface{} {
	current := data
	for i, seg := range segments {
		if i == len(segments)-1 {
			return current[seg]
		}
		if next, ok := current[seg].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	return nil
}

// applyOperator aplica el operador de Firestore entre el valor del documento y el valor del filtro
func applyOperator(fieldValue interface{}, operator model.Operator, filterValue interface{}) bool {
	switch operator {
	case model.OperatorEqual:
		return compare(fieldValue, filterValue) == 0
	case model.OperatorNotEqual:
		return compare(fieldValue, filterValue) != 0
	case model.OperatorLessThan:
		return compare(fieldValue, filterValue) < 0
	case model.OperatorLessThanOrEqual:
		return compare(fieldValue, filterValue) <= 0
	case model.OperatorGreaterThan:
		return compare(fieldValue, filterValue) > 0
	case model.OperatorGreaterThanOrEqual:
		return compare(fieldValue, filterValue) >= 0
	case model.OperatorIn:
		return valueInList(fieldValue, filterValue)
	case model.OperatorNotIn:
		return !valueInList(fieldValue, filterValue)
	case model.OperatorArrayContains:
		return arrayContains(fieldValue, filterValue)
	case model.OperatorArrayContainsAny:
		return arrayContainsAny(fieldValue, filterValue)
	default:
		return false
	}
}

// compare compara dos valores básicos (int, float, string, bool)
func compare(a, b interface{}) int {
	switch va := a.(type) {
	case int:
		vb, ok := b.(int)
		if !ok {
			return -2
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case int64:
		vb, ok := b.(int64)
		if !ok {
			return -2
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case float64:
		vb, ok := b.(float64)
		if !ok {
			return -2
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case string:
		vb, ok := b.(string)
		if !ok {
			return -2
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case bool:
		vb, ok := b.(bool)
		if !ok {
			return -2
		}
		if va == vb {
			return 0
		}
		if !va && vb {
			return -1
		}
		return 1
	default:
		return -2 // No comparable
	}
}

// valueInList verifica si fieldValue está en filterValue ([]interface{})
func valueInList(fieldValue interface{}, filterValue interface{}) bool {
	list, ok := filterValue.([]interface{})
	if !ok {
		return false
	}
	for _, v := range list {
		if compare(fieldValue, v) == 0 {
			return true
		}
	}
	return false
}

// arrayContains verifica si un array contiene el valor
func arrayContains(fieldValue interface{}, filterValue interface{}) bool {
	arr, ok := fieldValue.([]interface{})
	if !ok {
		return false
	}
	for _, v := range arr {
		if compare(v, filterValue) == 0 {
			return true
		}
	}
	return false
}

// arrayContainsAny verifica si un array contiene algún valor de una lista
func arrayContainsAny(fieldValue interface{}, filterValue interface{}) bool {
	arr, ok := fieldValue.([]interface{})
	if !ok {
		return false
	}
	list, ok := filterValue.([]interface{})
	if !ok {
		return false
	}
	for _, v := range arr {
		for _, f := range list {
			if compare(v, f) == 0 {
				return true
			}
		}
	}
	return false
}
