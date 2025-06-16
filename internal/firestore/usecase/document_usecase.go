package usecase

import (
	"context"
	"fmt"
	"strings"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/firestore"
)

// Document operations implementation
func (uc *FirestoreUsecase) CreateDocument(ctx context.Context, req CreateDocumentRequest) (*model.Document, error) {
	uc.logger.Info("Creating document",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID)

	fields := make(map[string]*model.FieldValue)
	for key, value := range req.Data {
		fields[key] = model.NewFieldValue(value)
	}

	if err := uc.validateFirestoreHierarchy(ctx, req.ProjectID, req.DatabaseID, req.CollectionID); err != nil {
		return nil, err
	}
	document, err := uc.firestoreRepo.CreateDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, fields)
	if err != nil {
		uc.logger.Error("Failed to create document", "error", err)
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	uc.logger.Info("Document created successfully", "documentID", document.DocumentID)
	return document, nil
}

func (uc *FirestoreUsecase) GetDocument(ctx context.Context, req GetDocumentRequest) (*model.Document, error) {
	uc.logger.Debug("Getting document",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID)

	document, err := uc.firestoreRepo.GetDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		uc.logger.Error("Failed to get document", "error", err)
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return document, nil
}

func (uc *FirestoreUsecase) UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (*model.Document, error) {
	uc.logger.Info("Updating document",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID)

	fields := make(map[string]*model.FieldValue)
	for key, value := range req.Data {
		fields[key] = model.NewFieldValue(value)
	}

	document, err := uc.firestoreRepo.UpdateDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, fields, req.Mask)
	if err != nil {
		uc.logger.Error("Failed to update document", "error", err)
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	uc.logger.Info("Document updated successfully", "documentID", document.DocumentID)
	return document, nil
}

func (uc *FirestoreUsecase) DeleteDocument(ctx context.Context, req DeleteDocumentRequest) error {
	uc.logger.Info("Deleting document",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID)

	err := uc.firestoreRepo.DeleteDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		uc.logger.Error("Failed to delete document", "error", err)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	uc.logger.Info("Document deleted successfully", "documentID", req.DocumentID)
	return nil
}

func (uc *FirestoreUsecase) ListDocuments(ctx context.Context, req ListDocumentsRequest) ([]*model.Document, error) {
	uc.logger.Debug("Listing documents",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	docs, _, err := uc.firestoreRepo.ListDocuments(
		ctx,
		req.ProjectID,
		req.DatabaseID,
		req.CollectionID,
		req.PageSize,
		req.PageToken,
		req.OrderBy,
		req.ShowMissing,
	)
	if err != nil {
		uc.logger.Error("Failed to list documents", "error", err)
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return docs, nil
}

func (uc *FirestoreUsecase) QueryDocuments(ctx context.Context, req QueryRequest) ([]*model.Document, error) {
	return uc.RunQuery(ctx, req)
}

func (uc *FirestoreUsecase) RunQuery(ctx context.Context, req QueryRequest) ([]*model.Document, error) {
	uc.logger.Debug("Running query",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)

	if req.StructuredQuery == nil {
		return nil, fmt.Errorf("structured query is required")
	}

	// Extract collection ID from parent path or use structured query info
	var collectionID string
	if req.Parent != "" {
		// Parse parent path to extract collection
		pathInfo, err := firestore.ParseFirestorePath(req.Parent)
		if err != nil {
			return nil, fmt.Errorf("invalid parent path: %w", err)
		}
		segments := strings.Split(pathInfo.DocumentPath, "/")
		if len(segments) > 0 {
			collectionID = segments[0]
		}
	}

	// Use QueryEngine if available, otherwise fallback to repository
	if uc.queryEngine != nil {
		uc.logger.Debug("Using QueryEngine for query execution", "collectionID", collectionID)
		return uc.queryEngine.ExecuteQuery(ctx, collectionID, *req.StructuredQuery)
	}

	// Fallback to repository method
	uc.logger.Debug("Using Repository for query execution", "collectionID", collectionID)
	docs, err := uc.firestoreRepo.RunQuery(ctx, req.ProjectID, req.DatabaseID, collectionID, req.StructuredQuery)
	if err != nil {
		uc.logger.Error("Failed to run query", "error", err)
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	return docs, nil
}

// Document path-based operations (for compatibility)
func (uc *FirestoreUsecase) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	req := GetDocumentRequest{
		ProjectID:    pathInfo.ProjectID,
		DatabaseID:   pathInfo.DatabaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}

	return uc.GetDocument(ctx, req)
}

func (uc *FirestoreUsecase) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	// Convert FieldValue map to interface map
	dataMap := make(map[string]any)
	for key, fieldValue := range data {
		dataMap[key] = fieldValue.ToInterface()
	}

	req := CreateDocumentRequest{
		ProjectID:    pathInfo.ProjectID,
		DatabaseID:   pathInfo.DatabaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Data:         dataMap,
	}

	return uc.CreateDocument(ctx, req)
}

func (uc *FirestoreUsecase) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	// Convert FieldValue map to interface map
	dataMap := make(map[string]any)
	for key, fieldValue := range data {
		dataMap[key] = fieldValue.ToInterface()
	}

	req := UpdateDocumentRequest{
		ProjectID:    pathInfo.ProjectID,
		DatabaseID:   pathInfo.DatabaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Data:         dataMap,
		Mask:         updateMask,
	}

	return uc.UpdateDocument(ctx, req)
}

func (uc *FirestoreUsecase) DeleteDocumentByPath(ctx context.Context, path string) error {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	req := DeleteDocumentRequest{
		ProjectID:    pathInfo.ProjectID,
		DatabaseID:   pathInfo.DatabaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
	}

	return uc.DeleteDocument(ctx, req)
}
