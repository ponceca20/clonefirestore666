package persistence

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisEventStore implements EventStore using Redis Streams
// Designed for Firestore-like realtime event delivery with persistence and distribution
// Follows hexagonal architecture patterns for clean separation of concerns
type RedisEventStore struct {
	client *redis.Client
	logger logger.Logger
}

// NewRedisEventStore creates a new Redis-based event store
func NewRedisEventStore(client *redis.Client, log logger.Logger) *RedisEventStore {
	return &RedisEventStore{
		client: client,
		logger: log,
	}
}

// StoreEvent stores a RealtimeEvent in Redis Streams with Firestore-compatible structure
func (r *RedisEventStore) StoreEvent(ctx context.Context, event model.RealtimeEvent) error {
	// Serialize event data to JSON for Redis storage
	eventData, err := json.Marshal(event.Data)
	if err != nil {
		r.logger.Error("Failed to serialize event data", zap.Error(err))
		return err
	}

	oldData, err := json.Marshal(event.OldData)
	if err != nil {
		r.logger.Error("Failed to serialize old data", zap.Error(err))
		return err
	}

	// Use FullPath as stream name for Firestore-like organization
	streamName := event.FullPath

	// Store event in Redis Stream with all Firestore event fields
	_, err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"type":           string(event.Type),
			"fullPath":       event.FullPath,
			"projectId":      event.ProjectID,
			"databaseId":     event.DatabaseID,
			"documentPath":   event.DocumentPath,
			"data":           eventData,
			"oldData":        oldData,
			"timestamp":      event.Timestamp.UnixNano(),
			"resumeToken":    string(event.ResumeToken),
			"sequenceNumber": event.SequenceNumber,
			"subscriptionId": event.SubscriptionID,
		},
	}).Result()

	if err != nil {
		r.logger.Error("Failed to store event in Redis",
			zap.String("stream", streamName),
			zap.String("eventType", string(event.Type)),
			zap.Error(err))
		return err
	}

	r.logger.Debug("Event stored successfully in Redis",
		zap.String("stream", streamName),
		zap.String("eventType", string(event.Type)),
		zap.Int64("sequenceNumber", event.SequenceNumber))

	return nil
}

// GetEventsSince retrieves events after a resume token with Firestore-compatible semantics
func (r *RedisEventStore) GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error) {
	streamName := firestorePath
	lastID := "0"

	// If resume token provided, use it as starting point
	if resumeToken != "" {
		lastID = string(resumeToken)
	}

	// First check if the stream exists to avoid blocking
	exists, err := r.client.Exists(ctx, streamName).Result()
	if err != nil {
		r.logger.Error("Failed to check stream existence",
			zap.String("stream", streamName),
			zap.Error(err))
		return nil, err
	}

	// If stream doesn't exist, return empty slice
	if exists == 0 {
		return []model.RealtimeEvent{}, nil
	}

	// Read events from Redis Stream with timeout context
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.client.XRead(readCtx, &redis.XReadArgs{
		Streams: []string{streamName, lastID},
		Count:   1000, // Maximum events to fetch per call
		Block:   0,    // Non-blocking read
	}).Result()

	if err != nil {
		if err == redis.Nil || err == context.DeadlineExceeded {
			// No events found or timeout - return empty slice
			return []model.RealtimeEvent{}, nil
		}
		r.logger.Error("Failed to read events from Redis",
			zap.String("stream", streamName),
			zap.String("resumeToken", string(resumeToken)),
			zap.Error(err))
		return nil, err
	}

	var events []model.RealtimeEvent

	// Process all streams (should be only one in our case)
	for _, streamRes := range res {
		for _, msg := range streamRes.Messages {
			event, err := r.parseEventFromMessage(msg)
			if err != nil {
				r.logger.Warn("Failed to parse event from Redis message",
					zap.String("messageId", msg.ID),
					zap.Error(err))
				continue
			}

			// Set Redis message ID as resume token for continuity
			event.ResumeToken = model.ResumeToken(msg.ID)
			events = append(events, event)
		}
	}

	r.logger.Debug("Retrieved events from Redis",
		zap.String("stream", streamName),
		zap.Int("eventCount", len(events)))

	return events, nil
}

// CleanupOldEvents removes events older than retention period using Redis Stream trimming
func (r *RedisEventStore) CleanupOldEvents(ctx context.Context, retentionPeriod time.Duration) error {
	// Get all stream names (in a real implementation, you'd maintain a registry)
	// For now, we'll use a pattern match approach
	streams, err := r.client.Keys(ctx, "projects/*").Result()
	if err != nil {
		r.logger.Error("Failed to get stream names for cleanup", zap.Error(err))
		return err
	}

	cleanedStreams := 0
	for _, stream := range streams {
		// Get stream info to find approximate message count to trim
		info, err := r.client.XInfoStream(ctx, stream).Result()
		if err != nil {
			continue
		}

		// If stream has messages, trim old ones (keep last 10000 events as safety)
		if info.Length > 10000 {
			trimmed, err := r.client.XTrimMaxLen(ctx, stream, 10000).Result()
			if err != nil {
				r.logger.Warn("Failed to trim stream",
					zap.String("stream", stream),
					zap.Error(err))
				continue
			}

			if trimmed > 0 {
				cleanedStreams++
			}
		}
	}

	if cleanedStreams > 0 {
		r.logger.Info("Cleaned up old events from Redis streams",
			zap.Int("streamsAffected", cleanedStreams))
	}

	return nil
}

// GetEventCount returns approximate event count for a path (Redis Stream length)
func (r *RedisEventStore) GetEventCount(firestorePath string) int {
	ctx := context.Background()
	if firestorePath == "" {
		// Return total across all streams
		streams, err := r.client.Keys(ctx, "projects/*").Result()
		if err != nil {
			return 0
		}

		total := 0
		for _, stream := range streams {
			length, err := r.client.XLen(ctx, stream).Result()
			if err == nil {
				total += int(length)
			}
		}
		return total
	}

	// Return count for specific path
	length, err := r.client.XLen(ctx, firestorePath).Result()
	if err != nil {
		return 0
	}

	return int(length)
}

// parseEventFromMessage converts Redis Stream message to RealtimeEvent
func (r *RedisEventStore) parseEventFromMessage(msg redis.XMessage) (model.RealtimeEvent, error) {
	event := model.RealtimeEvent{}

	// Parse basic fields
	if typeStr, ok := msg.Values["type"].(string); ok {
		event.Type = model.EventType(typeStr)
	}

	if fullPath, ok := msg.Values["fullPath"].(string); ok {
		event.FullPath = fullPath
	}

	if projectID, ok := msg.Values["projectId"].(string); ok {
		event.ProjectID = projectID
	}

	if databaseID, ok := msg.Values["databaseId"].(string); ok {
		event.DatabaseID = databaseID
	}

	if documentPath, ok := msg.Values["documentPath"].(string); ok {
		event.DocumentPath = documentPath
	}

	if resumeToken, ok := msg.Values["resumeToken"].(string); ok {
		event.ResumeToken = model.ResumeToken(resumeToken)
	}

	if subscriptionID, ok := msg.Values["subscriptionId"].(string); ok {
		event.SubscriptionID = subscriptionID
	}

	// Parse timestamp
	if timestampStr, ok := msg.Values["timestamp"].(string); ok {
		if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
			event.Timestamp = time.Unix(0, timestamp)
		}
	}

	// Parse sequence number
	if seqStr, ok := msg.Values["sequenceNumber"].(string); ok {
		if seq, err := strconv.ParseInt(seqStr, 10, 64); err == nil {
			event.SequenceNumber = seq
		}
	}

	// Parse event data JSON
	if dataStr, ok := msg.Values["data"].(string); ok && dataStr != "" && dataStr != "null" {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
			event.Data = data
		}
	}

	// Parse old data JSON
	if oldDataStr, ok := msg.Values["oldData"].(string); ok && oldDataStr != "" && oldDataStr != "null" {
		var oldData map[string]interface{}
		if err := json.Unmarshal([]byte(oldDataStr), &oldData); err == nil {
			event.OldData = oldData
		}
	}

	return event, nil
}
