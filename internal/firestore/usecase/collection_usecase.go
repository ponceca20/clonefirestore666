package usecase

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/model"
)

// Collection operations implementation
func (uc *FirestoreUsecase) CreateCollection(ctx context.Context, req CreateCollectionRequest) (*model.Collection, error) {
	uc.logger.Info("Creating collection",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	if err := uc.validateFirestoreHierarchy(ctx, req.ProjectID, req.DatabaseID, ""); err != nil {
		return nil, err
	}

	collection := model.NewCollection(req.ProjectID, req.DatabaseID, req.CollectionID)
	err := uc.firestoreRepo.CreateCollection(ctx, req.ProjectID, req.DatabaseID, collection)
	if err != nil {
		uc.logger.Error("Failed to create collection", "error", err)
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	uc.logger.Info("Collection created successfully", "collectionID", req.CollectionID)
	return collection, nil
}

func (uc *FirestoreUsecase) GetCollection(ctx context.Context, req GetCollectionRequest) (*model.Collection, error) {
	uc.logger.Debug("Getting collection",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	collection, err := uc.firestoreRepo.GetCollection(ctx, req.ProjectID, req.DatabaseID, req.CollectionID)
	if err != nil {
		uc.logger.Error("Failed to get collection", "error", err)
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	return collection, nil
}

func (uc *FirestoreUsecase) UpdateCollection(ctx context.Context, req UpdateCollectionRequest) error {
	uc.logger.Info("Updating collection",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	err := uc.firestoreRepo.UpdateCollection(ctx, req.ProjectID, req.DatabaseID, req.Collection)
	if err != nil {
		uc.logger.Error("Failed to update collection", "error", err)
		return fmt.Errorf("failed to update collection: %w", err)
	}

	uc.logger.Info("Collection updated successfully", "collectionID", req.CollectionID)
	return nil
}

func (uc *FirestoreUsecase) ListCollections(ctx context.Context, req ListCollectionsRequest) ([]*model.Collection, error) {
	uc.logger.Debug("Listing collections",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)

	collections, err := uc.firestoreRepo.ListCollections(ctx, req.ProjectID, req.DatabaseID)
	if err != nil {
		uc.logger.Error("Failed to list collections", "error", err)
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return collections, nil
}

func (uc *FirestoreUsecase) DeleteCollection(ctx context.Context, req DeleteCollectionRequest) error {
	uc.logger.Info("Deleting collection",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID)

	err := uc.firestoreRepo.DeleteCollection(ctx, req.ProjectID, req.DatabaseID, req.CollectionID)
	if err != nil {
		uc.logger.Error("Failed to delete collection", "error", err)
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	uc.logger.Info("Collection deleted successfully", "collectionID", req.CollectionID)
	return nil
}

func (uc *FirestoreUsecase) ListSubcollections(ctx context.Context, req ListSubcollectionsRequest) ([]model.Subcollection, error) {
	uc.logger.Debug("Listing subcollections",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID,
		"collectionID", req.CollectionID,
		"documentID", req.DocumentID)

	// Validate the parent document exists
	_, err := uc.firestoreRepo.GetDocument(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("parent document not found: %w", err)
	}

	// List subcollections
	subcollectionNames, err := uc.firestoreRepo.ListSubcollections(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		uc.logger.Error("Failed to list subcollections", "error", err)
		return nil, fmt.Errorf("failed to list subcollections: %w", err)
	}

	// Convert []string to []model.Subcollection
	subcollections := make([]model.Subcollection, len(subcollectionNames))
	for i, name := range subcollectionNames {
		subcollections[i] = model.Subcollection{
			ID: name,
			Path: fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s/%s",
				req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, name),
		}
	}

	uc.logger.Debug("Listed subcollections successfully", "count", len(subcollections))
	return subcollections, nil
}
