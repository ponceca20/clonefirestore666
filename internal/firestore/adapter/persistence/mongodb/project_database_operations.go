package mongodb

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
)

// ProjectRepository defines the minimal interface for project operations (for testability)
type ProjectRepository interface {
	CreateProject(ctx context.Context, project *model.Project) error
	GetProject(ctx context.Context, projectID string) (*model.Project, error)
	UpdateProject(ctx context.Context, project *model.Project) error
	DeleteProject(ctx context.Context, projectID string) error
	ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error)
}

// ProjectDatabaseOperations handles project and database operations
// Now depends on ProjectRepository interface for testability
type ProjectDatabaseOperations struct {
	repo ProjectRepository
}

// NewProjectDatabaseOperations creates a new ProjectDatabaseOperations instance
func NewProjectDatabaseOperations(repo ProjectRepository) *ProjectDatabaseOperations {
	return &ProjectDatabaseOperations{
		repo: repo,
	}
}

// CreateProject creates a new project document in the metadata collection
func (ops *ProjectDatabaseOperations) CreateProject(ctx context.Context, project *model.Project) error {
	return ops.repo.CreateProject(ctx, project)
}

// GetProject retrieves a project document by ID
func (ops *ProjectDatabaseOperations) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return ops.repo.GetProject(ctx, projectID)
}

// UpdateProject updates an existing project document
func (ops *ProjectDatabaseOperations) UpdateProject(ctx context.Context, project *model.Project) error {
	return ops.repo.UpdateProject(ctx, project)
}

// DeleteProject deletes a project document by ID
func (ops *ProjectDatabaseOperations) DeleteProject(ctx context.Context, projectID string) error {
	return ops.repo.DeleteProject(ctx, projectID)
}

// ListProjects lists all projects in the metadata collection, optionally filtered by ownerEmail
func (ops *ProjectDatabaseOperations) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return ops.repo.ListProjects(ctx, ownerEmail)
}

// DatabaseRepository defines the minimal interface for database operations (for testability)
type DatabaseRepository interface {
	CreateDatabase(ctx context.Context, projectID string, database *model.Database) error
	GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error)
	UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error
	DeleteDatabase(ctx context.Context, projectID, databaseID string) error
	ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error)
}

// DatabaseOperations handles database operations, depends on DatabaseRepository for testability
type DatabaseOperations struct {
	repo DatabaseRepository
}

// NewDatabaseOperations creates a new DatabaseOperations instance
func NewDatabaseOperations(repo DatabaseRepository) *DatabaseOperations {
	return &DatabaseOperations{repo: repo}
}

// CreateDatabase creates a new database document
func (ops *DatabaseOperations) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return ops.repo.CreateDatabase(ctx, projectID, database)
}

// GetDatabase retrieves a database document by ID
func (ops *DatabaseOperations) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return ops.repo.GetDatabase(ctx, projectID, databaseID)
}

// UpdateDatabase updates an existing database document
func (ops *DatabaseOperations) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return ops.repo.UpdateDatabase(ctx, projectID, database)
}

// DeleteDatabase deletes a database document by ID
func (ops *DatabaseOperations) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return ops.repo.DeleteDatabase(ctx, projectID, databaseID)
}

// ListDatabases lists all databases for a given project
func (ops *DatabaseOperations) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return ops.repo.ListDatabases(ctx, projectID)
}
