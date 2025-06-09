package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
)

// AtomicOperations handles atomic document operations
type AtomicOperations struct {
	repo *DocumentRepository
}

// NewAtomicOperations creates a new AtomicOperations instance
func NewAtomicOperations(repo *DocumentRepository) *AtomicOperations {
	return &AtomicOperations{repo: repo}
}

// AtomicIncrement performs an atomic increment operation on a numeric field
func (a *AtomicOperations) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Build the update operation
	updateDoc := bson.M{
		"$inc": bson.M{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic increment
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic increment: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicArrayUnion performs an atomic array union operation
func (a *AtomicOperations) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Convert FieldValue elements to interface{} for MongoDB
	var values []interface{}
	for _, element := range elements {
		values = append(values, element.Value)
	}

	// Build the update operation
	updateDoc := bson.M{
		"$addToSet": bson.M{
			fmt.Sprintf("fields.%s.value", field): bson.M{
				"$each": values,
			},
		},
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic array union
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic array union: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicArrayRemove performs an atomic array remove operation
func (a *AtomicOperations) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Convert FieldValue elements to interface{} for MongoDB
	var values []interface{}
	for _, element := range elements {
		values = append(values, element.Value)
	}

	// Build the update operation
	updateDoc := bson.M{
		"$pullAll": bson.M{
			fmt.Sprintf("fields.%s.value", field): values,
		},
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic array remove
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic array remove: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicServerTimestamp sets a field to the current server timestamp
func (a *AtomicOperations) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	now := time.Now()

	// Build the update operation
	updateDoc := bson.M{
		"$set": bson.M{
			fmt.Sprintf("fields.%s", field): &model.FieldValue{
				ValueType: model.FieldTypeTimestamp,
				Value:     now,
			},
			"update_time": now,
		},
	}

	// Execute the atomic server timestamp
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic server timestamp: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicDelete performs an atomic field deletion
func (a *AtomicOperations) AtomicDelete(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields []string) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	// Build the unset operation for each field
	unsetFields := bson.M{}
	for _, field := range fields {
		unsetFields[fmt.Sprintf("fields.%s", field)] = ""
	}

	updateDoc := bson.M{
		"$unset": unsetFields,
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic field deletion
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic field deletion: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicSetIfEmpty sets a field only if it doesn't exist or is empty
func (a *AtomicOperations) AtomicSetIfEmpty(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value *model.FieldValue) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
		"$or": []bson.M{
			{fmt.Sprintf("fields.%s", field): bson.M{"$exists": false}},
			{fmt.Sprintf("fields.%s.value", field): nil},
			{fmt.Sprintf("fields.%s.value", field): ""},
		},
	}

	updateDoc := bson.M{
		"$set": bson.M{
			fmt.Sprintf("fields.%s", field): value,
			"update_time":                   time.Now(),
		},
	}

	// Execute the conditional set operation
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic set if empty: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found or field is not empty")
	}

	return nil
}

// AtomicMaximum sets a field to the maximum of its current value and the provided value
func (a *AtomicOperations) AtomicMaximum(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value interface{}) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	updateDoc := bson.M{
		"$max": bson.M{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic maximum operation
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic maximum: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// AtomicMinimum sets a field to the minimum of its current value and the provided value
func (a *AtomicOperations) AtomicMinimum(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value interface{}) error {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
		"document_id":   documentID,
	}

	updateDoc := bson.M{
		"$min": bson.M{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": bson.M{
			"update_time": time.Now(),
		},
	}

	// Execute the atomic minimum operation
	result, err := a.repo.documentsCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic minimum: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}
