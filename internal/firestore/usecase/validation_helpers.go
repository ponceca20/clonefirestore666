package usecase

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/model"
)

// validateFirestoreHierarchy validates that the project, database, and optionally collection exist
// and creates them automatically if they don't exist (Firestore-like behavior)
func (uc *FirestoreUsecase) validateFirestoreHierarchy(ctx context.Context, projectID, databaseID, collectionID string) error {
	// Check and create project if it doesn't exist
	_, err := uc.firestoreRepo.GetProject(ctx, projectID)
	if err != nil {
		uc.logger.Info("Project not found, creating automatically", "projectID", projectID)
		// Create project automatically
		project := &model.Project{
			ProjectID:   projectID,
			DisplayName: projectID,
			LocationID:  "us-central1",         // Default location
			OwnerEmail:  "system@auto-created", // Auto-created project
			State:       model.ProjectStateActive,
		}
		if err := uc.firestoreRepo.CreateProject(ctx, project); err != nil {
			return fmt.Errorf("failed to auto-create project: %w", err)
		}
		uc.logger.Info("Project created automatically", "projectID", projectID)
	}

	// Check and create database if it doesn't exist
	_, err = uc.firestoreRepo.GetDatabase(ctx, projectID, databaseID)
	if err != nil {
		uc.logger.Info("Database not found, creating automatically", "projectID", projectID, "databaseID", databaseID)
		// Create database automatically
		database := model.NewDefaultDatabase(projectID)
		database.DatabaseID = databaseID
		if databaseID == "" || databaseID == "(default)" {
			database.DatabaseID = "(default)"
		}
		if err := uc.firestoreRepo.CreateDatabase(ctx, projectID, database); err != nil {
			return fmt.Errorf("failed to auto-create database: %w", err)
		}
		uc.logger.Info("Database created automatically", "projectID", projectID, "databaseID", database.DatabaseID)
	}

	// Check and create collection if provided and doesn't exist
	if collectionID != "" {
		_, err := uc.firestoreRepo.GetCollection(ctx, projectID, databaseID, collectionID)
		if err != nil {
			uc.logger.Info("Collection not found, creating automatically", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID)
			// Create collection automatically
			collection := &model.Collection{
				CollectionID: collectionID,
				ProjectID:    projectID,
				DatabaseID:   databaseID,
				Path:         fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID),
			}
			if err := uc.firestoreRepo.CreateCollection(ctx, projectID, databaseID, collection); err != nil {
				return fmt.Errorf("failed to auto-create collection: %w", err)
			}
			uc.logger.Info("Collection created automatically", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID)
		}
	}

	return nil
}

// Helper methods for backward compatibility and internal operations

// GetProjectInternal retrieves a project by ID (internal use)
func (uc *FirestoreUsecase) GetProjectInternal(ctx context.Context, projectID string) (*model.Project, error) {
	return uc.firestoreRepo.GetProject(ctx, projectID)
}

// GetDatabaseInternal retrieves a database by project and database ID (internal use)
func (uc *FirestoreUsecase) GetDatabaseInternal(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return uc.firestoreRepo.GetDatabase(ctx, projectID, databaseID)
}

// GetCollectionInternal retrieves a collection by project, database, and collection ID (internal use)
func (uc *FirestoreUsecase) GetCollectionInternal(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return uc.firestoreRepo.GetCollection(ctx, projectID, databaseID, collectionID)
}

// GetDocumentInternal retrieves a document by full path (internal use)
func (uc *FirestoreUsecase) GetDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return uc.firestoreRepo.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}
