package usecase

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/model"
)

// Database operations implementation
func (uc *FirestoreUsecase) CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*model.Database, error) {
	uc.logger.Info("Creating new database",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)

	// Validate project exists
	_, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project validation failed: %w", err)
	}

	err = uc.firestoreRepo.CreateDatabase(ctx, req.ProjectID, req.Database)
	if err != nil {
		uc.logger.Error("Failed to create database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	uc.logger.Info("Database created successfully",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)
	return req.Database, nil
}

func (uc *FirestoreUsecase) GetDatabase(ctx context.Context, req GetDatabaseRequest) (*model.Database, error) {
	uc.logger.Debug("Getting database",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)

	database, err := uc.firestoreRepo.GetDatabase(ctx, req.ProjectID, req.DatabaseID)
	if err != nil {
		uc.logger.Error("Failed to get database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.DatabaseID)
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return database, nil
}

func (uc *FirestoreUsecase) UpdateDatabase(ctx context.Context, req UpdateDatabaseRequest) (*model.Database, error) {
	uc.logger.Info("Updating database",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)

	err := uc.firestoreRepo.UpdateDatabase(ctx, req.ProjectID, req.Database)
	if err != nil {
		uc.logger.Error("Failed to update database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)
		return nil, fmt.Errorf("failed to update database: %w", err)
	}

	uc.logger.Info("Database updated successfully",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)
	return req.Database, nil
}

func (uc *FirestoreUsecase) DeleteDatabase(ctx context.Context, req DeleteDatabaseRequest) error {
	uc.logger.Info("Deleting database",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)

	err := uc.firestoreRepo.DeleteDatabase(ctx, req.ProjectID, req.DatabaseID)
	if err != nil {
		uc.logger.Error("Failed to delete database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.DatabaseID)
		return fmt.Errorf("failed to delete database: %w", err)
	}

	uc.logger.Info("Database deleted successfully",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)
	return nil
}

func (uc *FirestoreUsecase) ListDatabases(ctx context.Context, req ListDatabasesRequest) ([]*model.Database, error) {
	uc.logger.Debug("Listing databases", "projectID", req.ProjectID)

	databases, err := uc.firestoreRepo.ListDatabases(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Failed to list databases", "error", err, "projectID", req.ProjectID)
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	uc.logger.Debug("Listed databases successfully", "count", len(databases))
	return databases, nil
}
