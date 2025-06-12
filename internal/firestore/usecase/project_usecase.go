package usecase

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/model"
)

// Project operations implementation
func (uc *FirestoreUsecase) CreateProject(ctx context.Context, req CreateProjectRequest) (*model.Project, error) {
	uc.logger.Info("Creating new project", "projectID", req.Project.ProjectID)

	err := uc.firestoreRepo.CreateProject(ctx, req.Project)
	if err != nil {
		uc.logger.Error("Failed to create project", "error", err, "projectID", req.Project.ProjectID)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	uc.logger.Info("Project created successfully", "projectID", req.Project.ProjectID)
	return req.Project, nil
}

func (uc *FirestoreUsecase) GetProject(ctx context.Context, req GetProjectRequest) (*model.Project, error) {
	uc.logger.Debug("Getting project", "projectID", req.ProjectID)

	project, err := uc.firestoreRepo.GetProject(ctx, req.ProjectID)
	if err != nil {
		uc.logger.Error("Failed to get project", "error", err, "projectID", req.ProjectID)
		return nil, fmt.Errorf("failed to get project: %w", err)
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
