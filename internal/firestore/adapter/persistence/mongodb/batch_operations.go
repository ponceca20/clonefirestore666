package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/eventbus"
)

// BatchOperations handles batch write operations
// Ahora usa solo interfaces hexagonales y tipos estándar

type BatchOperations struct {
	repo *DocumentRepository
}

func NewBatchOperations(repo *DocumentRepository) *BatchOperations {
	return &BatchOperations{repo: repo}
}

func (b *BatchOperations) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	if len(writes) == 0 {
		return []*model.WriteResult{}, nil
	}

	writeResults := make([]*model.WriteResult, len(writes))

	for i, write := range writes {
		result, err := b.executeBatchOperation(ctx, projectID, databaseID, write)
		if err != nil {
			return nil, fmt.Errorf("operation %d failed: %w", i, err)
		}
		writeResults[i] = result
		b.emitDocumentEvent(ctx, projectID, databaseID, write)
	}

	return writeResults, nil
}

func (b *BatchOperations) executeBatchOperation(ctx context.Context, projectID, databaseID string, write *model.WriteOperation) (*model.WriteResult, error) {
	now := time.Now()
	pathParts := strings.Split(strings.Trim(write.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", write.Path)
	}
	collectionID := pathParts[len(pathParts)-2]
	documentID := pathParts[len(pathParts)-1]
	filter := map[string]interface{}{
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

func (b *BatchOperations) executeCreateOperation(ctx context.Context, filter map[string]interface{}, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	collectionID, ok := filter["collection_id"].(string)
	if !ok {
		return nil, fmt.Errorf("collection_id not found in filter")
	}
	targetCollection := b.repo.db.Collection(collectionID)
	count, err := targetCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to check document existence: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("document already exists")
	}
	fields := convertToFieldValues(write.Data)
	doc := &model.Document{
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
	_, err = targetCollection.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}
	return &model.WriteResult{UpdateTime: now}, nil
}

func (b *BatchOperations) executeUpdateOperation(ctx context.Context, filter map[string]interface{}, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	collectionID, ok := filter["collection_id"].(string)
	if !ok {
		return nil, fmt.Errorf("collection_id not found in filter")
	}
	if write.Precondition != nil && !b.checkPrecondition(ctx, filter, write.Precondition) {
		return nil, fmt.Errorf("precondition failed")
	}
	fields := convertToFieldValues(write.Data)
	updateDoc := map[string]interface{}{
		"$set": map[string]interface{}{
			"fields":      fields,
			"update_time": now,
		},
		"$inc": map[string]interface{}{
			"version": 1,
		},
	}
	targetCollection := b.repo.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}
	if result.Matched() == 0 {
		return nil, fmt.Errorf("document not found")
	}
	return &model.WriteResult{UpdateTime: now}, nil
}

func (b *BatchOperations) executeSetOperation(ctx context.Context, filter map[string]interface{}, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	collectionID, ok := filter["collection_id"].(string)
	if !ok {
		return nil, fmt.Errorf("collection_id not found in filter")
	}
	fields := convertToFieldValues(write.Data)
	doc := &model.Document{
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
	targetCollection := b.repo.db.Collection(collectionID)
	_, err := targetCollection.ReplaceOne(ctx, filter, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to set document: %w", err)
	}
	return &model.WriteResult{UpdateTime: now}, nil
}

func (b *BatchOperations) executeDeleteOperation(ctx context.Context, filter map[string]interface{}, write *model.WriteOperation, now time.Time) (*model.WriteResult, error) {
	collectionID, ok := filter["collection_id"].(string)
	if !ok {
		return nil, fmt.Errorf("collection_id not found in filter")
	}
	if write.Precondition != nil && !b.checkPrecondition(ctx, filter, write.Precondition) {
		return nil, fmt.Errorf("precondition failed")
	}
	targetCollection := b.repo.db.Collection(collectionID)
	result, err := targetCollection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}
	if result.Deleted() == 0 {
		return nil, fmt.Errorf("document not found")
	}
	return &model.WriteResult{UpdateTime: now}, nil
}

func (b *BatchOperations) checkPrecondition(ctx context.Context, filter map[string]interface{}, precondition *model.Precondition) bool {
	collectionID, ok := filter["collection_id"].(string)
	if !ok {
		return false
	}

	targetCollection := b.repo.db.Collection(collectionID)
	var doc model.Document
	err := targetCollection.FindOne(ctx, filter).Decode(&doc)
	if precondition.Exists != nil {
		if *precondition.Exists && err != nil {
			return false
		}
		if !*precondition.Exists && err == nil {
			return false
		}
	}
	if precondition.UpdateTime != nil && err == nil {
		if !doc.UpdateTime.Equal(*precondition.UpdateTime) {
			return false
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

// convertToFieldValues converts raw data to Firestore FieldValue format with smart timestamp detection
func convertToFieldValues(data map[string]interface{}) map[string]*model.FieldValue {
	fields := make(map[string]*model.FieldValue)
	for key, value := range data {
		fields[key] = model.NewFieldValue(value)
	}
	return fields
}
