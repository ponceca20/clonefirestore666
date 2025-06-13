package usecase

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/errors"
)

// Project operations implementation
func (uc *FirestoreUsecase) CreateProject(ctx context.Context, req CreateProjectRequest) (*model.Project, error) {
	uc.logger.Info("Creating new project",
		"projectID", req.Project.ProjectID,
		"organizationID", req.Project.OrganizationID)

	// Validate the create project request
	if err := uc.validateCreateProjectRequest(req); err != nil {
		uc.logger.Error("Invalid project creation request", "error", err,
			"projectID", req.Project.ProjectID,
			"organizationID", req.Project.OrganizationID)
		return nil, err
	}

	// Check if project already exists
	existingProject, err := uc.firestoreRepo.GetProject(ctx, req.Project.ProjectID)
	if err == nil && existingProject != nil {
		return nil, errors.NewConflictError(fmt.Sprintf("Project '%s' already exists in organization '%s'", req.Project.ProjectID, req.Project.OrganizationID))
	}
	if err != nil && !errors.IsNotFound(err) {
		uc.logger.Error("Failed to check existing project", "error", err,
			"projectID", req.Project.ProjectID)
		return nil, errors.NewInternalError("Failed to validate project uniqueness").WithCause(err)
	}

	// Set project metadata
	req.Project.CreatedAt = time.Now()
	req.Project.UpdatedAt = req.Project.CreatedAt
	req.Project.State = model.ProjectStateActive

	err = uc.firestoreRepo.CreateProject(ctx, req.Project)
	if err != nil {
		uc.logger.Error("Failed to create project", "error", err,
			"projectID", req.Project.ProjectID,
			"organizationID", req.Project.OrganizationID)
		return nil, errors.NewInternalError("Failed to create project").WithCause(err)
	}

	uc.logger.Info("Project created successfully",
		"projectID", req.Project.ProjectID,
		"organizationID", req.Project.OrganizationID,
		"resourceName", req.Project.GetResourceName())

	return req.Project, nil
}

// validateCreateProjectRequest validates the project creation request
func (uc *FirestoreUsecase) validateCreateProjectRequest(req CreateProjectRequest) error {
	if req.Project == nil {
		return errors.NewValidationError("Project configuration is required")
	}

	if req.Project.ProjectID == "" {
		return errors.NewValidationError("Project ID is required")
	}

	if req.Project.OrganizationID == "" {
		return errors.NewValidationError("Organization ID is required")
	}

	// Validate project ID format (alphanumeric, hyphens, underscores)
	if !isValidFirestoreID(req.Project.ProjectID) {
		return errors.NewValidationError("Project ID must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Set default values if not provided
	if req.Project.DisplayName == "" {
		req.Project.DisplayName = req.Project.ProjectID
	}

	if req.Project.LocationID == "" {
		req.Project.LocationID = "us-central1" // Default location
	}

	// Validate location if provided
	if !isValidLocationID(req.Project.LocationID) {
		return errors.NewValidationError("Invalid location ID format")
	}

	return nil
}

func (uc *FirestoreUsecase) GetProject(ctx context.Context, req GetProjectRequest) (*model.Project, error) {
	uc.logger.Debug("Getting project", "projectID", req.ProjectID)

	project, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Failed to get project", "error", err, "projectID", req.ProjectID)

		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError(fmt.Sprintf("Project '%s'", req.ProjectID))
		}
		return nil, errors.NewInternalError("Failed to get project").WithCause(err)
	}

	return project, nil
}

func (uc *FirestoreUsecase) UpdateProject(ctx context.Context, req UpdateProjectRequest) (*model.Project, error) {
	uc.logger.Info("Updating project", "projectID", req.Project.ProjectID)

	err := uc.firestoreRepo.UpdateProject(ctx, req.Project)
	if err != nil {
		uc.logger.Error("Failed to update project", "error", err, "projectID", req.Project.ProjectID)
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	uc.logger.Info("Project updated successfully", "projectID", req.Project.ProjectID)
	return req.Project, nil
}

func (uc *FirestoreUsecase) DeleteProject(ctx context.Context, req DeleteProjectRequest) error {
	uc.logger.Info("Deleting project", "projectID", req.ProjectID)

	err := uc.firestoreRepo.DeleteProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Failed to delete project", "error", err, "projectID", req.ProjectID)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	uc.logger.Info("Project deleted successfully", "projectID", req.ProjectID)
	return nil
}

func (uc *FirestoreUsecase) ListProjects(ctx context.Context, req ListProjectsRequest) ([]*model.Project, error) {
	uc.logger.Debug("Listing projects", "organizationID", req.OrganizationID, "ownerEmail", req.OwnerEmail)

	// The tenant-aware repository automatically filters by organization from context
	// OrganizationID is extracted from context by TenantAwareDocumentRepository
	projects, err := uc.firestoreRepo.ListProjects(ctx, req.OwnerEmail)
	if err != nil {
		uc.logger.Error("Failed to list projects", "error", err, "organizationID", req.OrganizationID, "ownerEmail", req.OwnerEmail)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	uc.logger.Debug("Listed projects successfully", "count", len(projects), "organizationID", req.OrganizationID)
	return projects, nil
}
