package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/eventbus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BatchOperations handles batch write operations
type BatchOperations struct {
	repo *DocumentRepository
}

// NewBatchOperations creates a new BatchOperations instance
func NewBatchOperations(repo *DocumentRepository) *BatchOperations {
	return &BatchOperations{repo: repo}
}

// RunBatchWrite executes multiple write operations atomically
func (b *BatchOperations) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	if len(writes) == 0 {
		return []*model.WriteResult{}, nil
	}

	// Start a MongoDB session for transaction
	session, err := b.repo.db.Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	var writeResults []*model.WriteResult

	// Execute operations in a transaction
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		writeResults = make([]*model.WriteResult, len(writes))

		for i, write := range writes {
			result, err := b.executeBatchOperation(sc, projectID, databaseID, write)
			if err != nil {
				// Abort transaction on error
				if abortErr := session.AbortTransaction(sc); abortErr != nil {
					b.repo.logger.Error("Failed to abort transaction: %v", abortErr)
				}
				return fmt.Errorf("operation %d failed: %w", i, err)
			}
			writeResults[i] = result
		}

		// Commit transaction if all operations succeeded
		if err := session.CommitTransaction(sc); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Emit events for successful operations
		for _, write := range writes {
			b.emitDocumentEvent(ctx, projectID, databaseID, write)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("batch write transaction failed: %w", err)
	}

	return writeResults, nil
}

// executeBatchOperation executes a single operation within a batch
func (b *BatchOperations) executeBatchOperation(ctx context.Context, projectID, databaseID string, write *model.WriteOperation) (*model.WriteResult, error) {
	now := time.Now()

	// Parse the document path to extract collection and document ID
	pathParts := strings.Split(strings.Trim(write.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", write.Path)
	}

	collectionID := pathParts[len(pathParts)-2]
	documentID := pathParts[len(pathParts)-1]

	// Create the document filter
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	switch write.Type {
	case model.WriteTypeCreate:
		return b.executeCreateOperation(ctx, filter, write, now)
	case model.WriteTypeUpdate:
		return b.executeUpdateOperation(ctx, filter, write, now)
	case model.WriteTypeSet:
		return b.executeSetOperation(ctx, filter, write, now)
	case model.WriteTypeDelete:
		return b.executeDeleteOperation(ctx, filter, write, now)
	default:
		return nil, fmt.Errorf("unsupported write operation type: %s", write.Type)
	}
}

// executeCreateOperation handles create operations for WriteOperation
func (b *BatchOperations) executeCreateOperation(ctx context.Context, filter bson.M, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	// Check if document already exists
	count, err := b.repo.documentsCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to check document existence: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("document already exists")
	}

	// Convert data to FieldValue format for Firestore compatibility
	fields := convertToFieldValues(write.Data)

	// Create the document
	doc := &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    filter["project_id"].(string),
		DatabaseID:   filter["database_id"].(string),
		CollectionID: filter["collection_id"].(string),
		DocumentID:   filter["document_id"].(string),
		Path:         write.Path,
		Fields:       fields,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}

	_, err = b.repo.documentsCol.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return &model.WriteResult{
		UpdateTime: now,
	}, nil
}

// executeUpdateOperation handles update operations for WriteOperation
func (b *BatchOperations) executeUpdateOperation(ctx context.Context, filter bson.M, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	// Check preconditions if present
	if write.Precondition != nil && !b.checkPrecondition(ctx, filter, write.Precondition) {
		return nil, fmt.Errorf("precondition failed")
	}

	// Convert data to FieldValue format
	fields := convertToFieldValues(write.Data)

	// Build update document
	updateDoc := bson.M{
		"$set": bson.M{
			"fields":      fields,
			"update_time": now,
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	result, err := b.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}
	if result.Matched() == 0 {
		return nil, fmt.Errorf("document not found")
	}

	return &model.WriteResult{
		UpdateTime: now,
	}, nil
}

// executeSetOperation handles set operations (upsert)
func (b *BatchOperations) executeSetOperation(ctx context.Context, filter bson.M, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	// Convert data to FieldValue format
	fields := convertToFieldValues(write.Data)

	// Prepare upsert document
	doc := &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    filter["project_id"].(string),
		DatabaseID:   filter["database_id"].(string),
		CollectionID: filter["collection_id"].(string),
		DocumentID:   filter["document_id"].(string),
		Path:         write.Path,
		Fields:       fields,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := b.repo.documentsCol.ReplaceOne(ctx, filter, doc, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to set document: %w", err)
	}

	return &model.WriteResult{
		UpdateTime: now,
	}, nil
}

// executeDeleteOperation handles delete operations for WriteOperation
func (b *BatchOperations) executeDeleteOperation(ctx context.Context, filter bson.M, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	// Check preconditions if present
	if write.Precondition != nil && !b.checkPrecondition(ctx, filter, write.Precondition) {
		return nil, fmt.Errorf("precondition failed")
	}

	result, err := b.repo.documentsCol.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	if result.Deleted() == 0 {
		return nil, fmt.Errorf("document not found")
	}

	return &model.WriteResult{
		UpdateTime: now,
	}, nil
}

// checkPrecondition validates operation preconditions
func (b *BatchOperations) checkPrecondition(ctx context.Context, filter bson.M, precondition *model.Precondition) bool {
	var doc model.Document
	err := b.repo.documentsCol.FindOne(ctx, filter).Decode(&doc)

	if precondition.Exists != nil {
		if *precondition.Exists && err == mongo.ErrNoDocuments {
			return false // Document should exist but doesn't
		}
		if !*precondition.Exists && err != mongo.ErrNoDocuments {
			return false // Document shouldn't exist but does
		}
	}

	if precondition.UpdateTime != nil && err == nil {
		if !doc.UpdateTime.Equal(*precondition.UpdateTime) {
			return false // Update time doesn't match
		}
	}

	return true
}

// emitDocumentEvent emits events for document changes
func (b *BatchOperations) emitDocumentEvent(ctx context.Context, projectID, databaseID string, op *model.WriteOperation) {
	if b.repo.eventBus == nil {
		return
	}

	eventType := ""
	switch op.Type {
	case model.WriteTypeCreate:
		eventType = "document.created"
	case model.WriteTypeUpdate:
		eventType = "document.updated"
	case model.WriteTypeDelete:
		eventType = "document.deleted"
	case model.WriteTypeSet:
		eventType = "document.set"
	}

	if eventType != "" {
		eventData := map[string]interface{}{
			"type":       eventType,
			"projectId":  projectID,
			"databaseId": databaseID,
			"path":       op.Path,
			"data":       op.Data,
			"timestamp":  time.Now(),
		}

		event := eventbus.NewBasicEventWithSource(eventType, eventData, "document_repository")
		if err := b.repo.eventBus.Publish(ctx, event); err != nil {
			b.repo.logger.Error("Failed to emit document event: %v", err)
		}
	}
}

// Utility functions

// inferFieldValueType infers the Firestore field value type from a Go value
func inferFieldValueType(value interface{}) model.FieldValueType {
	switch value.(type) {
	case string:
		return model.FieldTypeString
	case int, int32, int64:
		return model.FieldTypeInt
	case float32, float64:
		return model.FieldTypeDouble
	case bool:
		return model.FieldTypeBool
	case time.Time:
		return model.FieldTypeTimestamp
	case []interface{}:
		return model.FieldTypeArray
	case map[string]interface{}:
		return model.FieldTypeMap
	default:
		return model.FieldTypeString // default fallback
	}
}

// convertToFieldValues converts raw data to Firestore FieldValue format
func convertToFieldValues(data map[string]interface{}) map[string]*model.FieldValue {
	fields := make(map[string]*model.FieldValue)
	for key, value := range data {
		fields[key] = &model.FieldValue{
			ValueType: inferFieldValueType(value),
			Value:     value,
		}
	}
	return fields
}
