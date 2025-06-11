package usecase

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
)

// Atomic operations implementation
func (uc *FirestoreUsecase) AtomicIncrement(ctx context.Context, req AtomicIncrementRequest) (*AtomicIncrementResponse, error) {
	uc.logger.Info("Performing atomic increment",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID,
		"field", req.Field)

	// Validate and convert increment value
	inc, ok := req.IncrementBy.(int64)
	if !ok {
		// Try to convert from int or float64
		switch v := req.IncrementBy.(type) {
		case int:
			inc = int64(v)
		case float64:
			inc = int64(v)
		default:
			return nil, fmt.Errorf("incrementBy must be int64, int, or float64")
		}
	}

	// Use repository atomic increment for true atomicity
	err := uc.firestoreRepo.AtomicIncrement(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, req.Field, inc)
	if err != nil {
		uc.logger.Error("Atomic increment failed", "error", err)
		return nil, fmt.Errorf("atomic increment failed: %w", err)
	}

	// Fetch the updated document and return the new value
	doc, err := uc.firestoreRepo.GetDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		uc.logger.Error("Failed to fetch updated document", "error", err)
		return nil, fmt.Errorf("failed to fetch updated document: %w", err)
	}

	fieldVal, ok := doc.Fields[req.Field]
	if !ok {
		return nil, fmt.Errorf("field not found after increment: %s", req.Field)
	}

	uc.logger.Info("Atomic increment completed successfully",
		"field", req.Field,
		"newValue", fieldVal.ToInterface())

	return &AtomicIncrementResponse{NewValue: fieldVal.ToInterface()}, nil
}

func (uc *FirestoreUsecase) AtomicArrayUnion(ctx context.Context, req AtomicArrayUnionRequest) error {
	uc.logger.Info("Performing atomic array union",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID,
		"field", req.Field)

	// Convert elements to FieldValues
	fieldValues := make([]*model.FieldValue, len(req.Elements))
	for i, elem := range req.Elements {
		fieldValues[i] = model.NewFieldValue(elem)
	}

	err := uc.firestoreRepo.AtomicArrayUnion(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, req.Field, fieldValues)
	if err != nil {
		uc.logger.Error("Atomic array union failed", "error", err)
		return fmt.Errorf("atomic array union failed: %w", err)
	}

	uc.logger.Info("Atomic array union completed successfully", "field", req.Field)
	return nil
}

func (uc *FirestoreUsecase) AtomicArrayRemove(ctx context.Context, req AtomicArrayRemoveRequest) error {
	uc.logger.Info("Performing atomic array remove",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID,
		"field", req.Field)

	// Convert elements to FieldValues
	fieldValues := make([]*model.FieldValue, len(req.Elements))
	for i, elem := range req.Elements {
		fieldValues[i] = model.NewFieldValue(elem)
	}

	err := uc.firestoreRepo.AtomicArrayRemove(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, req.Field, fieldValues)
	if err != nil {
		uc.logger.Error("Atomic array remove failed", "error", err)
		return fmt.Errorf("atomic array remove failed: %w", err)
	}

	uc.logger.Info("Atomic array remove completed successfully", "field", req.Field)
	return nil
}

func (uc *FirestoreUsecase) AtomicServerTimestamp(ctx context.Context, req AtomicServerTimestampRequest) error {
	uc.logger.Info("Performing atomic server timestamp",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID,
		"field", req.Field)

	// Create server timestamp field value
	timestampValue := model.NewFieldValue(time.Now())

	// Update the document with server timestamp
	data := map[string]*model.FieldValue{
		req.Field: timestampValue,
	}

	_, err := uc.firestoreRepo.UpdateDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, data, []string{req.Field})
	if err != nil {
		uc.logger.Error("Atomic server timestamp failed", "error", err)
		return fmt.Errorf("atomic server timestamp failed: %w", err)
	}

	uc.logger.Info("Atomic server timestamp completed successfully", "field", req.Field)
	return nil
}
