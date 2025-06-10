package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Hexagonal Architecture Interfaces ---

// ClientInterface abstracts MongoDB client operations for testing
type ClientInterface interface {
	StartSession(opts ...*options.SessionOptions) (mongo.Session, error)
}

// MongoClientAdapter wraps *mongo.Client to implement ClientInterface
type MongoClientAdapter struct {
	client *mongo.Client
}

func NewMongoClientAdapter(client *mongo.Client) ClientInterface {
	return &MongoClientAdapter{client: client}
}

func (m *MongoClientAdapter) StartSession(opts ...*options.SessionOptions) (mongo.Session, error) {
	return m.client.StartSession(opts...)
}

// TenantManagerInterface abstracts tenant operations for testing
type TenantManagerInterface interface {
	CreateOrganizationDatabase(ctx context.Context, organizationID string) error
	DeleteOrganizationDatabase(ctx context.Context, organizationID string) error
	GetDatabaseForOrganization(ctx context.Context, organizationID string) (*mongo.Database, error)
}

// TenantManagerAdapter wraps *database.TenantManager to implement TenantManagerInterface
type TenantManagerAdapter struct {
	manager *database.TenantManager
}

func NewTenantManagerAdapter(manager *database.TenantManager) TenantManagerInterface {
	return &TenantManagerAdapter{manager: manager}
}

func (t *TenantManagerAdapter) CreateOrganizationDatabase(ctx context.Context, organizationID string) error {
	return t.manager.CreateOrganizationDatabase(ctx, organizationID)
}

func (t *TenantManagerAdapter) DeleteOrganizationDatabase(ctx context.Context, organizationID string) error {
	return t.manager.DeleteOrganizationDatabase(ctx, organizationID)
}

func (t *TenantManagerAdapter) GetDatabaseForOrganization(ctx context.Context, organizationID string) (*mongo.Database, error) {
	return t.manager.GetDatabaseForOrganization(ctx, organizationID)
}

// OrganizationRepository handles organization CRUD operations
// This manages the top-level tenant isolation following Firestore's architecture
type OrganizationRepository struct {
	client        ClientInterface        // Use interface for mocking
	masterDB      *mongo.Database        // Master database for organization metadata
	tenantManager TenantManagerInterface // Use interface for mocking
	collection    CollectionInterface    // Use interface for mocking
	logger        logger.Logger
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(
	client *mongo.Client,
	masterDB *mongo.Database,
	tenantManager *database.TenantManager,
	logger logger.Logger,
) *OrganizationRepository {
	collection := NewMongoCollectionAdapter(masterDB.Collection("organizations")) // use adapter

	return &OrganizationRepository{
		client:        NewMongoClientAdapter(client),
		masterDB:      masterDB,
		tenantManager: NewTenantManagerAdapter(tenantManager),
		collection:    collection,
		logger:        logger,
	}
}

// CreateOrganization creates a new organization and its dedicated database
func (r *OrganizationRepository) CreateOrganization(ctx context.Context, org *model.Organization) error {
	// Validate organization
	if err := model.ValidateOrganizationID(org.OrganizationID); err != nil {
		return fmt.Errorf("invalid organization: %w", err)
	}

	// Check if organization already exists
	exists, err := r.organizationExists(ctx, org.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to check organization existence: %w", err)
	}
	if exists {
		return model.ErrOrganizationExists
	}

	// Set creation metadata
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now
	org.State = model.OrganizationStateActive

	// Start transaction
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute in transaction
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// 1. Create organization record in master database
		_, err := r.collection.InsertOne(sc, org)
		if err != nil {
			return fmt.Errorf("failed to insert organization: %w", err)
		}

		// 2. Create dedicated database for organization
		err = r.tenantManager.CreateOrganizationDatabase(sc, org.OrganizationID)
		if err != nil {
			return fmt.Errorf("failed to create organization database: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"organization_id": org.OrganizationID,
		"display_name":    org.DisplayName,
	}).Info("Created organization successfully")

	return nil
}

// GetOrganization retrieves an organization by ID
func (r *OrganizationRepository) GetOrganization(ctx context.Context, organizationID string) (*model.Organization, error) {
	if err := model.ValidateOrganizationID(organizationID); err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	filter := bson.M{"organization_id": organizationID}

	var org model.Organization
	err := r.collection.FindOne(ctx, filter).Decode(&org)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// UpdateOrganization updates an existing organization
func (r *OrganizationRepository) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	if err := model.ValidateOrganizationID(org.OrganizationID); err != nil {
		return fmt.Errorf("invalid organization: %w", err)
	}

	// Set update timestamp
	org.UpdatedAt = time.Now()

	filter := bson.M{"organization_id": org.OrganizationID}
	update := bson.M{"$set": org}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	if result.Matched() == 0 {
		return model.ErrOrganizationNotFound
	}

	r.logger.WithFields(map[string]interface{}{
		"organization_id": org.OrganizationID,
	}).Info("Updated organization successfully")

	return nil
}

// DeleteOrganization deletes an organization and its database
func (r *OrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	if err := model.ValidateOrganizationID(organizationID); err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	// Start transaction
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute in transaction
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// 1. Delete organization record
		filter := bson.M{"organization_id": organizationID}
		result, err := r.collection.DeleteOne(sc, filter)
		if err != nil {
			return fmt.Errorf("failed to delete organization: %w", err)
		}

		if result.Deleted() == 0 {
			return model.ErrOrganizationNotFound
		}

		// 2. Delete organization database
		err = r.tenantManager.DeleteOrganizationDatabase(sc, organizationID)
		if err != nil {
			return fmt.Errorf("failed to delete organization database: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"organization_id": organizationID,
	}).Info("Deleted organization successfully")

	return nil
}

// ListOrganizations lists organizations with pagination
func (r *OrganizationRepository) ListOrganizations(ctx context.Context, limit int, offset int) ([]*model.Organization, error) {
	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer cursor.Close(ctx)

	var organizations []*model.Organization
	for cursor.Next(ctx) {
		var org model.Organization
		if err := cursor.Decode(&org); err != nil {
			return nil, fmt.Errorf("failed to decode organization: %w", err)
		}
		organizations = append(organizations, &org)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return organizations, nil
}

// ListOrganizationsByAdmin lists organizations where the user is an admin
func (r *OrganizationRepository) ListOrganizationsByAdmin(ctx context.Context, adminEmail string) ([]*model.Organization, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"billing_email": adminEmail},
			{"admin_emails": adminEmail},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations by admin: %w", err)
	}
	defer cursor.Close(ctx)

	var organizations []*model.Organization
	for cursor.Next(ctx) {
		var org model.Organization
		if err := cursor.Decode(&org); err != nil {
			return nil, fmt.Errorf("failed to decode organization: %w", err)
		}
		organizations = append(organizations, &org)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return organizations, nil
}

// GetOrganizationDatabase returns the dedicated database for an organization
func (r *OrganizationRepository) GetOrganizationDatabase(ctx context.Context, organizationID string) (*mongo.Database, error) {
	// First verify organization exists
	_, err := r.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// Get the organization's dedicated database
	return r.tenantManager.GetDatabaseForOrganization(ctx, organizationID)
}

// UpdateUsageStats updates organization usage statistics
func (r *OrganizationRepository) UpdateUsageStats(ctx context.Context, organizationID string, usage *model.OrganizationUsage) error {
	filter := bson.M{"organization_id": organizationID}
	update := bson.M{
		"$set": bson.M{
			"usage":      usage,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update usage stats: %w", err)
	}

	if result.Matched() == 0 {
		return model.ErrOrganizationNotFound
	}

	return nil
}

// CreateIndexes creates necessary indexes for the organization collection
func (r *OrganizationRepository) CreateIndexes(ctx context.Context) error {
	// Use the underlying *mongo.Collection for index creation
	adapter, ok := r.collection.(*MongoCollectionAdapter)
	if !ok {
		return fmt.Errorf("collection does not support index creation")
	}
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "organization_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "billing_email", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "admin_emails", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "state", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	_, err := adapter.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// Private methods

// organizationExists checks if an organization with the given ID already exists
func (r *OrganizationRepository) organizationExists(ctx context.Context, organizationID string) (bool, error) {
	filter := bson.M{"organization_id": organizationID}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
