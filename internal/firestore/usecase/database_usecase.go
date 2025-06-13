package usecase

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/errors"
)

// Database operations implementation
func (uc *FirestoreUsecase) CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*model.Database, error) {
	// Validar que la configuración de la base de datos no sea nil
	if req.Database == nil {
		return nil, errors.NewValidationError("Database configuration is required")
	}
	uc.logger.Info("Creating new database",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)

	// Validate input parameters
	if err := uc.validateCreateDatabaseRequest(req); err != nil {
		uc.logger.Error("Invalid database creation request", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)
		return nil, err
	}

	// Validate project exists with proper error handling aligned to Firestore API
	project, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Project validation failed", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)

		// Return Firestore-compatible error
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Project '%s' not found", req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to validate project").WithCause(err)
	}

	uc.logger.Debug("Project validation successful",
		"projectID", project.ProjectID,
		"organizationID", project.OrganizationID)
	// Validate that the database doesn't already exist
	existingDB, err := uc.firestoreRepo.GetDatabase(ctx, req.ProjectID, req.Database.DatabaseID)
	if err == nil && existingDB != nil {
		return nil, errors.NewConflictError(fmt.Sprintf("Database '%s' already exists in project '%s'", req.Database.DatabaseID, req.ProjectID))
	}
	if err != nil && !errors.IsNotFound(err) {
		uc.logger.Error("Failed to check existing database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)
		return nil, errors.NewInternalError("Failed to validate database uniqueness").WithCause(err)
	}
	// Set database metadata
	req.Database.ProjectID = req.ProjectID
	req.Database.CreatedAt = time.Now()
	req.Database.UpdatedAt = req.Database.CreatedAt
	req.Database.State = model.DatabaseStateCreating

	err = uc.firestoreRepo.CreateDatabase(ctx, req.ProjectID, req.Database)
	if err != nil {
		uc.logger.Error("Failed to create database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)
		return nil, errors.NewInternalError("Failed to create database").WithCause(err)
	}
	uc.logger.Info("Database created successfully",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID,
		"resourceName", req.Database.GetResourceName())

	return req.Database, nil
}

// validateCreateDatabaseRequest validates the database creation request
func (uc *FirestoreUsecase) validateCreateDatabaseRequest(req CreateDatabaseRequest) error {
	if req.ProjectID == "" {
		return errors.NewValidationError("Project ID is required")
	}

	if req.Database == nil {
		return errors.NewValidationError("Database configuration is required")
	}

	if req.Database.DatabaseID == "" {
		return errors.NewValidationError("Database ID is required")
	}

	// Validate database ID format (alphanumeric, hyphens, underscores)
	if !isValidFirestoreID(req.Database.DatabaseID) {
		return errors.NewValidationError("Database ID must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate location if provided
	if req.Database.LocationID != "" && !isValidLocationID(req.Database.LocationID) {
		return errors.NewValidationError("Invalid location ID format")
	}
	// Set default values if not provided
	if req.Database.Type == "" {
		req.Database.Type = model.DatabaseTypeFirestoreNative
	}

	if req.Database.ConcurrencyMode == "" {
		req.Database.ConcurrencyMode = model.ConcurrencyModeOptimistic
	}

	// Validate enum values
	if !isValidDatabaseType(string(req.Database.Type)) {
		return errors.NewValidationError("Invalid database type. Must be FIRESTORE_NATIVE or DATASTORE_MODE")
	}

	if !isValidConcurrencyMode(string(req.Database.ConcurrencyMode)) {
		return errors.NewValidationError("Invalid concurrency mode. Must be OPTIMISTIC or PESSIMISTIC")
	}

	return nil
}

// isValidFirestoreID validates Firestore ID format
func isValidFirestoreID(id string) bool {
	if len(id) == 0 || len(id) > 1500 {
		return false
	}

	for _, char := range id {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// isValidLocationID validates location ID format
func isValidLocationID(locationID string) bool {
	validLocations := map[string]bool{
		"us-central1":          true,
		"us-east1":             true,
		"us-east4":             true,
		"us-west1":             true,
		"us-west2":             true,
		"us-west3":             true,
		"us-west4":             true,
		"europe-west1":         true,
		"europe-west2":         true,
		"europe-west3":         true,
		"asia-east1":           true,
		"asia-east2":           true,
		"asia-northeast1":      true,
		"asia-southeast1":      true,
		"australia-southeast1": true,
	}
	return validLocations[locationID]
}

// isValidDatabaseType validates database type
func isValidDatabaseType(dbType string) bool {
	return dbType == "FIRESTORE_NATIVE" || dbType == "DATASTORE_MODE"
}

// isValidConcurrencyMode validates concurrency mode
func isValidConcurrencyMode(mode string) bool {
	return mode == "OPTIMISTIC" || mode == "PESSIMISTIC"
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

		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", req.DatabaseID, req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to get database").WithCause(err)
	}

	return database, nil
}

func (uc *FirestoreUsecase) UpdateDatabase(ctx context.Context, req UpdateDatabaseRequest) (*model.Database, error) {
	uc.logger.Info("Updating database",
		"projectID", req.ProjectID,
		"databaseID", req.Database.DatabaseID)

	// Validate project exists
	_, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Project validation failed", "error", err,
			"projectID", req.ProjectID)

		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Project '%s' not found", req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to validate project").WithCause(err)
	}

	err = uc.firestoreRepo.UpdateDatabase(ctx, req.ProjectID, req.Database)
	if err != nil {
		uc.logger.Error("Failed to update database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.Database.DatabaseID)

		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", req.Database.DatabaseID, req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to update database").WithCause(err)
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

	// Validate project exists
	_, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Project validation failed", "error", err,
			"projectID", req.ProjectID)

		if errors.IsNotFound(err) {
			return errors.NewNotFoundError(fmt.Sprintf("Project '%s' not found", req.ProjectID))
		}
		return errors.NewInternalError("Failed to validate project").WithCause(err)
	}

	err = uc.firestoreRepo.DeleteDatabase(ctx, req.ProjectID, req.DatabaseID)
	if err != nil {
		uc.logger.Error("Failed to delete database", "error", err,
			"projectID", req.ProjectID,
			"databaseID", req.DatabaseID)

		if errors.IsNotFound(err) {
			return errors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", req.DatabaseID, req.ProjectID))
		}
		return errors.NewInternalError("Failed to delete database").WithCause(err)
	}

	uc.logger.Info("Database deleted successfully",
		"projectID", req.ProjectID,
		"databaseID", req.DatabaseID)
	return nil
}

func (uc *FirestoreUsecase) ListDatabases(ctx context.Context, req ListDatabasesRequest) ([]*model.Database, error) {
	uc.logger.Debug("Listing databases", "projectID", req.ProjectID)

	// Validate project exists
	_, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Project validation failed", "error", err,
			"projectID", req.ProjectID)
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Project '%s' not found", req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to validate project").WithCause(err)
	}

	databases, err := uc.firestoreRepo.ListDatabases(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Failed to list databases", "error", err,
			"projectID", req.ProjectID)
		return nil, errors.NewInternalError("Failed to list databases").WithCause(err)
	}

	uc.logger.Debug("Listed databases successfully",
		"projectID", req.ProjectID,
		"count", len(databases))
	return databases, nil
}
