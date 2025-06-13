package usecase

import (
	"context"
	"errors"
	"fmt"

	"firestore-clone/internal/firestore/domain/model"
)

// Index operations implementation
func (uc *FirestoreUsecase) CreateIndex(ctx context.Context, req CreateIndexRequest) (*model.Index, error) {
	uc.logger.Info("Creating index",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"indexName", req.Index.Name)

	if err := uc.validateIndex(&req.Index); err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	collIdx := model.CollectionIndex{
		Name:   req.Index.Name,
		Fields: req.Index.Fields,
		State:  req.Index.State,
	}

	err := uc.firestoreRepo.CreateIndex(ctx, req.ProjectID, req.DatabaseID, req.Index.Collection, &collIdx)
	if err != nil {
		uc.logger.Error("Failed to create index", "error", err)
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	uc.logger.Info("Index created successfully", "indexName", req.Index.Name)
	return &req.Index, nil
}

func (uc *FirestoreUsecase) DeleteIndex(ctx context.Context, req DeleteIndexRequest) error {
	uc.logger.Info("Deleting index",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"indexName", req.IndexName)

	err := uc.firestoreRepo.DeleteIndex(ctx, req.ProjectID, req.DatabaseID, "", req.IndexName)
	if err != nil {
		uc.logger.Error("Failed to delete index", "error", err)
		return fmt.Errorf("failed to delete index: %w", err)
	}

	uc.logger.Info("Index deleted successfully", "indexName", req.IndexName)
	return nil
}

func (uc *FirestoreUsecase) ListIndexes(ctx context.Context, req ListIndexesRequest) ([]model.Index, error) {
	uc.logger.Debug("Listing indexes",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	collIndexes, err := uc.firestoreRepo.ListIndexes(ctx, req.ProjectID, req.DatabaseID, req.CollectionID)
	if err != nil {
		uc.logger.Error("Failed to list indexes", "error", err)
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	indexes := make([]model.Index, len(collIndexes))
	for i, idx := range collIndexes {
		indexes[i] = model.Index{
			Name:       idx.Name,
			Collection: req.CollectionID, // Use the collection ID from request
			Fields:     idx.Fields,
			State:      idx.State,
		}
	}

	uc.logger.Debug("Listed indexes successfully", "count", len(indexes))
	return indexes, nil
}

// validateIndex validates index configuration
func (uc *FirestoreUsecase) validateIndex(index *model.Index) error {
	if index.Name == "" {
		return errors.New("index name is required")
	}

	if index.Collection == "" {
		return errors.New("collection is required")
	}

	if len(index.Fields) == 0 {
		return errors.New("at least one field is required")
	}

	if len(index.Fields) > 100 { // Firestore limit
		return errors.New("index cannot have more than 100 fields")
	}

	for i, field := range index.Fields {
		if field.Path == "" {
			return fmt.Errorf("field path is required at index %d", i)
		}

		if field.Order != model.IndexFieldOrderAscending && field.Order != model.IndexFieldOrderDescending {
			return fmt.Errorf("invalid field order at index %d", i)
		}
	}

	return nil
}
