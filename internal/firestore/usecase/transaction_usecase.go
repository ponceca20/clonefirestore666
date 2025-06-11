package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"
)

// Transaction and batch operations implementation
func (uc *FirestoreUsecase) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	uc.logger.Info("Beginning transaction", "projectID", projectID)

	// Generate a unique transaction ID
	transactionID := fmt.Sprintf("tx_%d_%s", time.Now().UnixNano(), projectID)

	// For now, we'll return the transaction ID
	// In a full implementation, this would interact with the repository to create a transaction context
	uc.logger.Info("Transaction started", "transactionID", transactionID)

	return transactionID, nil
}

func (uc *FirestoreUsecase) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	uc.logger.Info("Committing transaction", "projectID", projectID, "transactionID", transactionID)

	// Validate transaction ID format
	if !strings.HasPrefix(transactionID, "tx_") {
		return fmt.Errorf("invalid transaction ID format: %s", transactionID)
	}

	// For now, we'll just log the commit
	// In a full implementation, this would interact with the repository to commit the transaction
	uc.logger.Info("Transaction committed", "transactionID", transactionID)

	return nil
}

func (uc *FirestoreUsecase) RunBatchWrite(ctx context.Context, req BatchWriteRequest) (*model.BatchWriteResponse, error) {
	uc.logger.Info("Running batch write",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"operationsCount", len(req.Writes))

	// Validate request
	if len(req.Writes) == 0 {
		return &model.BatchWriteResponse{}, nil
	}

	if len(req.Writes) > 500 { // Firestore limit
		return nil, fmt.Errorf("batch write operations cannot exceed 500 operations")
	}

	// Validate each operation
	for i, write := range req.Writes {
		if err := uc.validateBatchOperation(write); err != nil {
			return nil, fmt.Errorf("invalid operation at index %d: %w", i, err)
		}
	}

	// Convert BatchWriteOperation to WriteOperation
	writeOps := make([]*model.WriteOperation, len(req.Writes))
	for i, write := range req.Writes {
		writeOps[i] = &model.WriteOperation{
			Type: model.WriteOperationType(write.Type),
			Path: write.Path,
			Data: write.Data,
		}
	}

	// Execute batch write using repository method
	writeResults, err := uc.firestoreRepo.RunBatchWrite(ctx, req.ProjectID, req.DatabaseID, writeOps)
	if err != nil {
		uc.logger.Error("Batch write failed", "error", err)
		return nil, fmt.Errorf("batch write failed: %w", err)
	}

	// Convert WriteResult to BatchWriteResponse
	batchResponse := &model.BatchWriteResponse{
		WriteResults: make([]model.WriteResult, len(writeResults)),
		Status:       make([]model.Status, len(writeResults)),
	}

	for i, result := range writeResults {
		batchResponse.WriteResults[i] = model.WriteResult{
			UpdateTime: result.UpdateTime,
		}
		batchResponse.Status[i] = model.Status{
			Code:    0,
			Message: "OK",
		}
	}

	uc.logger.Info("Batch write completed successfully",
		"operationsCount", len(writeResults))

	return batchResponse, nil
}

// validateBatchOperation validates a single batch operation
func (uc *FirestoreUsecase) validateBatchOperation(op model.BatchWriteOperation) error {
	if op.DocumentID == "" {
		return errors.New("document ID is required")
	}

	if op.Path == "" {
		return errors.New("document path is required")
	}

	switch op.Type {
	case model.BatchOperationTypeCreate, model.BatchOperationTypeSet:
		if op.Data == nil {
			return errors.New("data is required for create/set operations")
		}
	case model.BatchOperationTypeUpdate:
		if op.Data == nil {
			return errors.New("data is required for update operations")
		}
	case model.BatchOperationTypeDelete:
		// No additional validation needed
	default:
		return fmt.Errorf("unsupported operation type: %s", op.Type)
	}

	return nil
}
