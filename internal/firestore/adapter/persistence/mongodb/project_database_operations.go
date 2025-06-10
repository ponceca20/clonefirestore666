package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ProjectDatabaseOperations handles project and database operations
type ProjectDatabaseOperations struct {
	repo       *DocumentRepository
	projectCol *mongo.Collection
	dbCol      *mongo.Collection
}

// NewProjectDatabaseOperations creates a new ProjectDatabaseOperations instance
func NewProjectDatabaseOperations(repo *DocumentRepository) *ProjectDatabaseOperations {
	return &ProjectDatabaseOperations{
		repo:       repo,
		projectCol: repo.db.Collection("projects"),
		dbCol:      repo.db.Collection("databases"),
	}
}

// --- Project operations ---

// CreateProject creates a new project
func (p *ProjectDatabaseOperations) CreateProject(ctx context.Context, project *model.Project) error {
	now := time.Now()
	// Check if project already exists
	filter := bson.M{"project_id": project.ProjectID}
	count, err := p.projectCol.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check project existence: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("project already exists")
	}
	// Set creation metadata
	project.CreatedAt = now
	project.UpdatedAt = now
	if project.ID.IsZero() {
		project.ID = primitive.NewObjectID()
	}

	_, err = p.projectCol.InsertOne(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID
func (p *ProjectDatabaseOperations) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	filter := bson.M{"project_id": projectID}

	var project model.Project
	err := p.projectCol.FindOne(ctx, filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// UpdateProject updates a project
func (p *ProjectDatabaseOperations) UpdateProject(ctx context.Context, project *model.Project) error {
	filter := bson.M{"project_id": project.ProjectID}

	updateDoc := bson.M{
		"$set": bson.M{
			"display_name": project.DisplayName,
			"location_id":  project.LocationID,
			"updated_at":   time.Now(),
		},
	}

	result, err := p.projectCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// DeleteProject deletes a project by ID
func (p *ProjectDatabaseOperations) DeleteProject(ctx context.Context, projectID string) error {
	// Check if project has any databases
	dbFilter := bson.M{"project_id": projectID}
	dbCount, err := p.dbCol.CountDocuments(ctx, dbFilter)
	if err != nil {
		return fmt.Errorf("failed to check databases count: %w", err)
	}
	if dbCount > 0 {
		return fmt.Errorf("cannot delete project with existing databases")
	}

	filter := bson.M{"project_id": projectID}
	result, err := p.projectCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// ListProjects lists all projects for an owner
func (p *ProjectDatabaseOperations) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	filter := bson.M{"owner_email": ownerEmail}

	cursor, err := p.projectCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer cursor.Close(ctx)

	var projects []*model.Project
	for cursor.Next(ctx) {
		var project model.Project
		if err := cursor.Decode(&project); err != nil {
			return nil, fmt.Errorf("failed to decode project: %w", err)
		}
		projects = append(projects, &project)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return projects, nil
}

// --- Database operations ---

// CreateDatabase creates a new database
func (p *ProjectDatabaseOperations) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	now := time.Now()

	// Check if database already exists
	filter := bson.M{
		"project_id":  projectID,
		"database_id": database.DatabaseID,
	}

	count, err := p.dbCol.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("database already exists")
	}

	// Set metadata
	database.ID = primitive.NewObjectID()
	database.ProjectID = projectID
	database.CreatedAt = now
	database.UpdatedAt = now
	database.State = model.DatabaseStateActive

	_, err = p.dbCol.InsertOne(ctx, database)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// GetDatabase retrieves a database by ID
func (p *ProjectDatabaseOperations) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	filter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	var database model.Database
	err := p.dbCol.FindOne(ctx, filter).Decode(&database)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("database not found")
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &database, nil
}

// UpdateDatabase updates a database
func (p *ProjectDatabaseOperations) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	filter := bson.M{
		"project_id":  projectID,
		"database_id": database.DatabaseID,
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"display_name": database.DisplayName,
			"location_id":  database.LocationID,
			"updated_at":   time.Now(),
		},
	}

	result, err := p.dbCol.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("database not found")
	}

	return nil
}

// DeleteDatabase deletes a database by ID
func (p *ProjectDatabaseOperations) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	// Check if database has any collections
	collectionFilter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	collectionCount, err := p.repo.collectionsCol.CountDocuments(ctx, collectionFilter)
	if err != nil {
		return fmt.Errorf("failed to check collections count: %w", err)
	}
	if collectionCount > 0 {
		return fmt.Errorf("cannot delete database with existing collections")
	}

	filter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	result, err := p.dbCol.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("database not found")
	}

	return nil
}

// ListDatabases lists all databases in a project
func (p *ProjectDatabaseOperations) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	filter := bson.M{"project_id": projectID}

	cursor, err := p.dbCol.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer cursor.Close(ctx)

	var databases []*model.Database
	for cursor.Next(ctx) {
		var database model.Database
		if err := cursor.Decode(&database); err != nil {
			return nil, fmt.Errorf("failed to decode database: %w", err)
		}
		databases = append(databases, &database)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return databases, nil
}
