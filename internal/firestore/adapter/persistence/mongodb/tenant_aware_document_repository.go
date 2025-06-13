package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/logger"
	"firestore-clone/internal/shared/utils"

	"go.mongodb.org/mongo-driver/mongo"
)

// TenantAwareDocumentRepository implements FirestoreRepository with multi-tenant support
// Following Firestore's architecture: Organization → Project → Database → Documents
// This repository uses composition to leverage the existing DocumentRepository implementation
type TenantAwareDocumentRepository struct {
	client        *mongo.Client
	tenantManager *database.TenantManager
	eventBus      *eventbus.EventBus
	logger        logger.Logger

	// Cache of tenant-specific DocumentRepository instances
	// Key: organizationID, Value: *DocumentRepository
	tenantRepos map[string]*DocumentRepository
}

// NewTenantAwareDocumentRepository creates a new tenant-aware document repository
func NewTenantAwareDocumentRepository(
	client *mongo.Client,
	tenantManager *database.TenantManager,
	eventBus *eventbus.EventBus,
	logger logger.Logger,
) repository.FirestoreRepository {
	return &TenantAwareDocumentRepository{
		client:        client,
		tenantManager: tenantManager,
		eventBus:      eventBus,
		logger:        logger,
		tenantRepos:   make(map[string]*DocumentRepository),
	}
}

// getTenantRepository gets or creates a DocumentRepository for a specific tenant
func (r *TenantAwareDocumentRepository) getTenantRepository(ctx context.Context, organizationID string) (*DocumentRepository, error) {
	// Check cache first
	if repo, exists := r.tenantRepos[organizationID]; exists {
		return repo, nil
	}

	// Get tenant-specific database
	db, err := r.tenantManager.GetDatabaseForOrganization(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization database: %w", err)
	}

	// Create tenant-specific DocumentRepository
	dbProvider := NewMongoDatabaseAdapter(db)
	tenantRepo := NewDocumentRepository(dbProvider, r.eventBus, r.logger)

	// Cache it
	r.tenantRepos[organizationID] = tenantRepo

	return tenantRepo, nil
}

// extractOrganizationID extracts organization ID from context
func (r *TenantAwareDocumentRepository) extractOrganizationID(ctx context.Context) (string, error) {
	organizationID, err := utils.GetOrganizationIDFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("organization ID required: %w", err)
	}
	return organizationID, nil
}

// Project operations with tenant isolation

func (r *TenantAwareDocumentRepository) CreateProject(ctx context.Context, project *model.Project) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.CreateProject(ctx, project)
}

func (r *TenantAwareDocumentRepository) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.GetProject(ctx, projectID)
}

func (r *TenantAwareDocumentRepository) UpdateProject(ctx context.Context, project *model.Project) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.UpdateProject(ctx, project)
}

func (r *TenantAwareDocumentRepository) DeleteProject(ctx context.Context, projectID string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteProject(ctx, projectID)
}

func (r *TenantAwareDocumentRepository) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.ListProjects(ctx, ownerEmail)
}

// Database operations with tenant isolation

func (r *TenantAwareDocumentRepository) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.CreateDatabase(ctx, projectID, database)
}

func (r *TenantAwareDocumentRepository) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.GetDatabase(ctx, projectID, databaseID)
}

func (r *TenantAwareDocumentRepository) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.UpdateDatabase(ctx, projectID, database)
}

func (r *TenantAwareDocumentRepository) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteDatabase(ctx, projectID, databaseID)
}

func (r *TenantAwareDocumentRepository) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.ListDatabases(ctx, projectID)
}

// Collection operations with tenant isolation

func (r *TenantAwareDocumentRepository) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.CreateCollection(ctx, projectID, databaseID, collection)
}

func (r *TenantAwareDocumentRepository) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.GetCollection(ctx, projectID, databaseID, collectionID)
}

func (r *TenantAwareDocumentRepository) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.UpdateCollection(ctx, projectID, databaseID, collection)
}

func (r *TenantAwareDocumentRepository) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteCollection(ctx, projectID, databaseID, collectionID)
}

func (r *TenantAwareDocumentRepository) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.ListCollections(ctx, projectID, databaseID)
}

func (r *TenantAwareDocumentRepository) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.ListSubcollections(ctx, projectID, databaseID, collectionID, documentID)
}

// Document operations - Core Firestore CRUD with tenant isolation

func (r *TenantAwareDocumentRepository) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

func (r *TenantAwareDocumentRepository) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
}

func (r *TenantAwareDocumentRepository) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
}

func (r *TenantAwareDocumentRepository) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.SetDocument(ctx, projectID, databaseID, collectionID, documentID, data, merge)
}

func (r *TenantAwareDocumentRepository) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// Document path-based operations (for compatibility)

func (r *TenantAwareDocumentRepository) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.GetDocumentByPath(ctx, path)
}

func (r *TenantAwareDocumentRepository) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.CreateDocumentByPath(ctx, path, data)
}

func (r *TenantAwareDocumentRepository) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.UpdateDocumentByPath(ctx, path, data, updateMask)
}

func (r *TenantAwareDocumentRepository) DeleteDocumentByPath(ctx context.Context, path string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteDocumentByPath(ctx, path)
}

// Query operations

func (r *TenantAwareDocumentRepository) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.ExecuteQuery(ctx, projectID, databaseID, collectionID, query)
}

func (r *TenantAwareDocumentRepository) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// Collection group queries are just regular queries across multiple collections
	// For now, delegate to ExecuteQuery as the basic implementation
	return tenantRepo.ExecuteQuery(ctx, projectID, databaseID, collectionID, query)
}

// RunAggregationQuery performs aggregation operations on documents
// Note: This is a simplified implementation for the Firestore clone
func (r *TenantAwareDocumentRepository) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// For simplicity, execute regular query and wrap results in AggregationResult
	documents, err := tenantRepo.ExecuteQuery(ctx, projectID, databaseID, collectionID, query)
	if err != nil {
		return nil, err
	}
	// Create a simple aggregation result
	count := int64(len(documents))
	result := &model.AggregationResult{
		Count:    &count,
		ReadTime: time.Now(),
	}

	return result, nil
}

// List documents with pagination

func (r *TenantAwareDocumentRepository) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, "", err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, "", err
	}

	return tenantRepo.ListDocuments(ctx, projectID, databaseID, collectionID, pageSize, pageToken, orderBy, showMissing)
}

// Batch operations

func (r *TenantAwareDocumentRepository) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return tenantRepo.RunBatchWrite(ctx, projectID, databaseID, writes)
}

func (r *TenantAwareDocumentRepository) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.RunTransaction(ctx, fn)
}

// Atomic field transforms

func (r *TenantAwareDocumentRepository) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.AtomicIncrement(ctx, projectID, databaseID, collectionID, documentID, field, value)
}

func (r *TenantAwareDocumentRepository) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.AtomicArrayUnion(ctx, projectID, databaseID, collectionID, documentID, field, elements)
}

func (r *TenantAwareDocumentRepository) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.AtomicArrayRemove(ctx, projectID, databaseID, collectionID, documentID, field, elements)
}

func (r *TenantAwareDocumentRepository) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.AtomicServerTimestamp(ctx, projectID, databaseID, collectionID, documentID, field)
}

// Index operations

func (r *TenantAwareDocumentRepository) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	// Convert CollectionIndex to Index for the underlying repository
	modelIndex := &model.Index{
		Name:   index.Name,
		Fields: index.Fields,
		State:  index.State,
	}

	return tenantRepo.CreateIndex(ctx, projectID, databaseID, collectionID, modelIndex)
}

func (r *TenantAwareDocumentRepository) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return err
	}

	return tenantRepo.DeleteIndex(ctx, projectID, databaseID, collectionID, indexID)
}

func (r *TenantAwareDocumentRepository) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	organizationID, err := r.extractOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepo, err := r.getTenantRepository(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// Get indexes from underlying repository
	indexes, err := tenantRepo.ListIndexes(ctx, projectID, databaseID, collectionID)
	if err != nil {
		return nil, err
	}

	// Convert []*model.Index to []*model.CollectionIndex
	collectionIndexes := make([]*model.CollectionIndex, len(indexes))
	for i, idx := range indexes {
		collectionIndexes[i] = &model.CollectionIndex{
			Name:   idx.Name,
			Fields: idx.Fields,
			State:  idx.State,
		}
	}

	return collectionIndexes, nil
}

// MongoDatabaseAdapter adapts *mongo.Database to implement DatabaseProvider interface
// This follows hexagonal architecture by abstracting MongoDB-specific implementations
type MongoDatabaseAdapter struct {
	db *mongo.Database
}

// NewMongoDatabaseAdapter creates a new database adapter for hexagonal architecture
func NewMongoDatabaseAdapter(db *mongo.Database) DatabaseProvider {
	return &MongoDatabaseAdapter{db: db}
}

// Collection returns a CollectionInterface for the given collection name
func (m *MongoDatabaseAdapter) Collection(name string) CollectionInterface {
	col := m.db.Collection(name)
	return NewMongoCollectionAdapter(col)
}

// Client returns the underlying client interface
func (m *MongoDatabaseAdapter) Client() interface{} {
	return m.db.Client()
}
