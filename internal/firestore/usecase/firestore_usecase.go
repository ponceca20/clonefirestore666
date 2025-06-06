package usecase

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger" // Assuming logger is used

	"go.uber.org/zap" // For logging fields
)

// FirestoreUsecase defines the interface for Firestore core operations.
type FirestoreUsecase interface {
	CreateDocument(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
	GetDocument(ctx context.Context, path string) (map[string]interface{}, error)
	UpdateDocument(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
	DeleteDocument(ctx context.Context, path string) error
	ListDocuments(ctx context.Context, path string, limit int, pageToken string) ([]map[string]interface{}, string, error)
	RunQuery(ctx context.Context, projectID string, query string) ([]map[string]interface{}, error)
	BeginTransaction(ctx context.Context, projectID string) (string, error)
	CommitTransaction(ctx context.Context, projectID string, transactionID string) error
}

type firestoreUsecaseImpl struct {
	repo repository.FirestoreRepository
	log  logger.Logger
	rtu  RealtimeUsecase // Added: RealtimeUsecase dependency
	// queryEngine repository.QueryEngine // For query operations
	// securityUseCase SecurityUsecase   // For security rules
}

// NewFirestoreUsecase creates a new instance of FirestoreUsecase.
func NewFirestoreUsecase(
	repo repository.FirestoreRepository,
	log logger.Logger,
	rtu RealtimeUsecase, // Added
	// queryEngine repository.QueryEngine,
	// securityUseCase SecurityUsecase,
) FirestoreUsecase {
	return &firestoreUsecaseImpl{
		repo: repo,
		log:  log,
		rtu:  rtu, // Added
		// queryEngine: queryEngine,
		// securityUseCase: securityUseCase,
	}
}

// CreateDocument creates a new document at the given path.
// Path would be like "collections/{collectionId}/documents/{documentId}"
// or just "collections/{collectionId}" for auto-ID.
func (uc *firestoreUsecaseImpl) CreateDocument(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	// Basic path parsing to get collection and document ID (simplified)
	// collectionID, docID := parsePath(path) // This function would need to be robust

	// Actual document creation logic using uc.repo.CreateDocument(...)
	// For this example, assume it returns the created document data and its final path.
	// createdDoc, finalPath, err := uc.repo.CreateDocument(ctx, collectionID, docID, data)
	// if err != nil {
	// 	uc.log.Error(ctx, "Error creating document in repo", zap.Error(err), zap.String("path", path))
	// 	return nil, err
	// }

	// Placeholder for actual repo call and result
	createdDoc := data
	finalPath := path // Assume path is the final path for simplicity here
	uc.log.Info(ctx, "Document created successfully (placeholder)", zap.String("path", finalPath))

	// Publish real-time event
	if uc.rtu != nil { // Ensure rtu is initialized
		event := model.RealtimeEvent{
			Type: model.EventTypeCreated,
			Path: finalPath, // Use the actual path where the document was created
			Data: createdDoc,
		}
		if err := uc.rtu.PublishEvent(ctx, event); err != nil {
			uc.log.Error(ctx, "Failed to publish document creation event", zap.Error(err), zap.String("path", finalPath))
			// Not returning this error to client, as DB operation was successful
		} else {
			uc.log.Info(ctx, "Published document creation event", zap.String("path", finalPath))
		}
	}

	return createdDoc, nil
}

// UpdateDocument updates an existing document at the given path.
func (uc *firestoreUsecaseImpl) UpdateDocument(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	// collectionID, docID := parsePath(path)

	// updatedDoc, err := uc.repo.UpdateDocument(ctx, collectionID, docID, data)
	// if err != nil {
	// 	uc.log.Error(ctx, "Error updating document in repo", zap.Error(err), zap.String("path", path))
	// 	return nil, err
	// }

	// Placeholder
	updatedDoc := data
	uc.log.Info(ctx, "Document updated successfully (placeholder)", zap.String("path", path))

	if uc.rtu != nil {
		event := model.RealtimeEvent{
			Type: model.EventTypeUpdated,
			Path: path,
			Data: updatedDoc,
		}
		if err := uc.rtu.PublishEvent(ctx, event); err != nil {
			uc.log.Error(ctx, "Failed to publish document update event", zap.Error(err), zap.String("path", path))
		} else {
			uc.log.Info(ctx, "Published document update event", zap.String("path", path))
		}
	}
	return updatedDoc, nil
}

// DeleteDocument deletes a document at the given path.
func (uc *firestoreUsecaseImpl) DeleteDocument(ctx context.Context, path string) error {
	// collectionID, docID := parsePath(path)
	// err := uc.repo.DeleteDocument(ctx, collectionID, docID)
	// if err != nil {
	// 	uc.log.Error(ctx, "Error deleting document in repo", zap.Error(err), zap.String("path", path))
	// 	return err
	// }

	// Placeholder
	uc.log.Info(ctx, "Document deleted successfully (placeholder)", zap.String("path", path))

	if uc.rtu != nil {
		event := model.RealtimeEvent{
			Type: model.EventTypeDeleted,
			Path: path,
			Data: nil, // Or map[string]interface{}{"id": docID}
		}
		if err := uc.rtu.PublishEvent(ctx, event); err != nil {
			uc.log.Error(ctx, "Failed to publish document deletion event", zap.Error(err), zap.String("path", path))
		} else {
			uc.log.Info(ctx, "Published document deletion event", zap.String("path", path))
		}
	}
	return nil
}

// GetDocument is a read operation, so no event publishing.
func (uc *firestoreUsecaseImpl) GetDocument(ctx context.Context, path string) (map[string]interface{}, error) {
	// ...
	// uc.log.Info(ctx, "GetDocument called", zap.String("path", path))
	// return uc.repo.GetDocument(ctx, collectionID, docID)
	return map[string]interface{}{"message": "GetDocument placeholder"}, nil // Placeholder
}

// ListDocuments lists documents in a collection
func (uc *firestoreUsecaseImpl) ListDocuments(ctx context.Context, path string, limit int, pageToken string) ([]map[string]interface{}, string, error) {
	// TODO: Implement actual document listing
	uc.log.Info(ctx, "ListDocuments called (placeholder)", zap.String("path", path), zap.Int("limit", limit))

	// Placeholder implementation
	docs := []map[string]interface{}{
		{"id": "doc1", "data": map[string]interface{}{"field": "value1"}},
		{"id": "doc2", "data": map[string]interface{}{"field": "value2"}},
	}

	return docs, "", nil // Empty nextPageToken for now
}

// RunQuery executes a structured query
func (uc *firestoreUsecaseImpl) RunQuery(ctx context.Context, projectID string, query string) ([]map[string]interface{}, error) {
	// TODO: Implement actual query execution
	uc.log.Info(ctx, "RunQuery called (placeholder)", zap.String("projectID", projectID))

	// Placeholder implementation
	docs := []map[string]interface{}{
		{"id": "result1", "data": map[string]interface{}{"query": "result"}},
	}

	return docs, nil
}

// BeginTransaction starts a new transaction
func (uc *firestoreUsecaseImpl) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	// TODO: Implement actual transaction logic
	uc.log.Info(ctx, "BeginTransaction called (placeholder)", zap.String("projectID", projectID))

	// Return a placeholder transaction ID
	return "transaction-123", nil
}

// CommitTransaction commits a transaction
func (uc *firestoreUsecaseImpl) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	// TODO: Implement actual transaction commit logic
	uc.log.Info(ctx, "CommitTransaction called (placeholder)", zap.String("projectID", projectID), zap.String("transactionID", transactionID))

	return nil
}

// parsePath would be a utility function, e.g.:
// func parsePath(path string) (collectionID string, documentID string) { ... }
// This is a simplification. Real path parsing needs to handle subcollections etc.
