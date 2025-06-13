package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	sharedErrors "firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"
	"firestore-clone/internal/shared/utils"
)

// ErrDocumentNotFound is returned when a document is not found.
var (
	ErrDocumentNotFound   = errors.New("document not found")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrInvalidPath        = errors.New("invalid document path")
	ErrPreconditionFailed = errors.New("precondition failed")
)

// DocumentRepository implements the Firestore document repository using CollectionInterface.
type DocumentRepository struct {
	db             DatabaseProvider
	eventBus       *eventbus.EventBus
	logger         logger.Logger
	documentsCol   CollectionInterface
	indexesCol     CollectionInterface
	collectionsCol CollectionInterface

	// Specialized operation handlers
	documentOps   *DocumentOperations
	batchOps      *BatchOperations
	collectionOps *CollectionOperations
	atomicOps     *AtomicOperations
	projectDbOps  *ProjectDatabaseOperations
	indexOps      *IndexOperations
}

// DatabaseProvider abstracts the database for testability.
type DatabaseProvider interface {
	Collection(name string) CollectionInterface
	Client() interface{}
}

// NewDocumentRepository creates a new document repository.
func NewDocumentRepository(db DatabaseProvider, eventBus *eventbus.EventBus, logger logger.Logger) *DocumentRepository {
	docsCol := db.Collection("documents")
	collsCol := db.Collection("collections")
	idxCol := db.Collection("indexes")

	repo := &DocumentRepository{
		db:             db,
		eventBus:       eventBus,
		logger:         logger,
		documentsCol:   docsCol,
		collectionsCol: collsCol,
		indexesCol:     idxCol,
	}
	repo.documentOps = NewDocumentOperations(repo)
	repo.batchOps = NewBatchOperations(repo)
	repo.collectionOps = NewCollectionOperations(repo)
	repo.atomicOps = NewAtomicOperations(repo.db)
	repo.projectDbOps = NewProjectDatabaseOperations(repo)
	repo.indexOps = NewIndexOperations(
		NewIndexCollectionAdapter(repo.indexesCol),
		NewDocumentCollectionAdapter(repo.documentsCol),
		repo.logger,
	)
	return repo
}

// DocumentRepository implements ProjectRepository for project operations
var _ ProjectRepository = (*DocumentRepository)(nil)

// --- Batch Operations ---

// RunBatchWrite executes multiple write operations atomically
func (r *DocumentRepository) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return r.batchOps.RunBatchWrite(ctx, projectID, databaseID, writes)
}

// --- Document Operations ---

// GetDocument retrieves a document by ID
func (r *DocumentRepository) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return r.documentOps.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// CreateDocument creates a new document
func (r *DocumentRepository) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	return r.documentOps.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
}

// UpdateDocument updates a document
func (r *DocumentRepository) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return r.documentOps.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
}

// SetDocument sets (creates or updates) a document
func (r *DocumentRepository) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	return r.documentOps.SetDocument(ctx, projectID, databaseID, collectionID, documentID, data, merge)
}

// DeleteDocument deletes a document by ID
func (r *DocumentRepository) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return r.documentOps.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// GetDocumentByPath retrieves a document by path
func (r *DocumentRepository) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return r.documentOps.GetDocumentByPath(ctx, path)
}

// CreateDocumentByPath creates a document by path
func (r *DocumentRepository) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return r.documentOps.CreateDocumentByPath(ctx, path, data)
}

// UpdateDocumentByPath updates a document by path
func (r *DocumentRepository) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return r.documentOps.UpdateDocumentByPath(ctx, path, data, updateMask)
}

// DeleteDocumentByPath deletes a document by path
func (r *DocumentRepository) DeleteDocumentByPath(ctx context.Context, path string) error {
	return r.documentOps.DeleteDocumentByPath(ctx, path)
}

// ListDocuments lists documents in a collection
func (r *DocumentRepository) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return r.documentOps.ListDocuments(ctx, projectID, databaseID, collectionID, pageSize, pageToken, orderBy, showMissing)
}

// --- Collection Operations ---

// CreateCollection creates a new collection
func (r *DocumentRepository) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return r.collectionOps.CreateCollection(ctx, projectID, databaseID, collection)
}

// GetCollection retrieves a collection by ID
func (r *DocumentRepository) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return r.collectionOps.GetCollection(ctx, projectID, databaseID, collectionID)
}

// UpdateCollection updates a collection
func (r *DocumentRepository) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return r.collectionOps.UpdateCollection(ctx, projectID, databaseID, collection)
}

// DeleteCollection deletes a collection by ID
func (r *DocumentRepository) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return r.collectionOps.DeleteCollection(ctx, projectID, databaseID, collectionID)
}

// ListCollections lists all collections in a database
func (r *DocumentRepository) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return r.collectionOps.ListCollections(ctx, projectID, databaseID)
}

// ListSubcollections lists subcollection names under a document
func (r *DocumentRepository) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return r.collectionOps.ListSubcollections(ctx, projectID, databaseID, collectionID, documentID)
}

// --- Atomic Operations ---

// AtomicIncrement performs an atomic increment operation
func (r *DocumentRepository) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	return r.atomicOps.AtomicIncrement(ctx, projectID, databaseID, collectionID, documentID, field, value)
}

// AtomicArrayUnion performs an atomic array union operation
func (r *DocumentRepository) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return r.atomicOps.AtomicArrayUnion(ctx, projectID, databaseID, collectionID, documentID, field, elements)
}

// AtomicArrayRemove performs an atomic array remove operation
func (r *DocumentRepository) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return r.atomicOps.AtomicArrayRemove(ctx, projectID, databaseID, collectionID, documentID, field, elements)
}

// AtomicServerTimestamp sets a field to the current server timestamp
func (r *DocumentRepository) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	return r.atomicOps.AtomicServerTimestamp(ctx, projectID, databaseID, collectionID, documentID, field)
}

// --- Project Operations ---

// CreateProject creates a new project in the 'projects' collection with organization isolation
func (r *DocumentRepository) CreateProject(ctx context.Context, project *model.Project) error {
	// Validate required fields
	if project.ProjectID == "" {
		return sharedErrors.NewValidationError("Project ID is required")
	}
	if project.OrganizationID == "" {
		return sharedErrors.NewValidationError("Organization ID is required")
	}

	// Extract organization ID from context for additional validation
	contextOrgID, err := utils.GetOrganizationIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("organization context required for project creation: %w", err)
	}

	// Ensure the organization ID in the project matches the context
	if project.OrganizationID != contextOrgID {
		return sharedErrors.NewValidationError("Project organization ID must match the request context")
	}

	// Set timestamps
	project.CreatedAt = time.Now()
	project.UpdatedAt = project.CreatedAt

	// Insert the project
	_, err = r.db.Collection("projects").InsertOne(ctx, project)
	if err != nil {
		// Check for duplicate key error (project already exists)
		return sharedErrors.NewConflictError(fmt.Sprintf("Project '%s' already exists in organization '%s'", project.ProjectID, project.OrganizationID))
	}

	return nil
}

// GetProject retrieves a project by ProjectID within the organization context
func (r *DocumentRepository) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	// Extract organization ID from context for tenant isolation
	organizationID, err := utils.GetOrganizationIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("organization context required for project access: %w", err)
	}

	// Filter by both project_id and organization_id for proper tenant isolation
	filter := map[string]interface{}{
		"project_id":      projectID,
		"organization_id": organizationID,
	}

	var project model.Project
	err = r.db.Collection("projects").FindOne(ctx, filter).Decode(&project)
	if err != nil {
		// Check if document not found
		return nil, sharedErrors.NewNotFoundError(fmt.Sprintf("Project '%s' in organization '%s'", projectID, organizationID))
	}

	return &project, nil
}

// UpdateProject actualiza los datos de un proyecto
func (r *DocumentRepository) UpdateProject(ctx context.Context, project *model.Project) error {
	if project.ProjectID == "" || project.OrganizationID == "" {
		return errors.New("projectID y organizationID son requeridos")
	}
	project.UpdatedAt = time.Now()
	filter := map[string]interface{}{"project_id": project.ProjectID}
	update := map[string]interface{}{"$set": project}
	res, err := r.db.Collection("projects").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error al actualizar proyecto: %w", err)
	}
	if res.Matched() == 0 {
		return errors.New("proyecto no encontrado para actualizar")
	}
	return nil
}

// DeleteProject elimina un proyecto por su ProjectID
func (r *DocumentRepository) DeleteProject(ctx context.Context, projectID string) error {
	filter := map[string]interface{}{"project_id": projectID}
	res, err := r.db.Collection("projects").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error al eliminar proyecto: %w", err)
	}
	if res.Deleted() == 0 {
		return errors.New("proyecto no encontrado para eliminar")
	}
	return nil
}

// ListProjects lista todos los proyectos de una organización o de un owner
func (r *DocumentRepository) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	filter := map[string]interface{}{}
	if ownerEmail != "" {
		filter["owner_email"] = ownerEmail
	}
	cursor, err := r.db.Collection("projects").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error al listar proyectos: %w", err)
	}
	defer cursor.Close(ctx)
	var projects []*model.Project
	for cursor.Next(ctx) {
		var project model.Project
		if err := cursor.Decode(&project); err != nil {
			return nil, fmt.Errorf("error al decodificar proyecto: %w", err)
		}
		projects = append(projects, &project)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("error en el cursor de proyectos: %w", err)
	}
	return projects, nil
}

// --- Database Operations ---

// CreateDatabase creates a new database
func (r *DocumentRepository) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	if database == nil {
		return sharedErrors.NewValidationError("Database cannot be nil")
	}

	// Validate required fields
	if projectID == "" {
		return sharedErrors.NewValidationError("Project ID is required")
	}
	if database.DatabaseID == "" {
		return sharedErrors.NewValidationError("Database ID is required")
	}

	// Set timestamps
	database.CreatedAt = time.Now()
	database.UpdatedAt = database.CreatedAt

	// Create the database document in MongoDB
	_, err := r.db.Collection("databases").InsertOne(ctx, database)
	if err != nil {
		// Check for duplicate key error (database already exists)
		return sharedErrors.NewConflictError(fmt.Sprintf("Database '%s' already exists in project '%s'", database.DatabaseID, projectID))
	}

	r.logger.Info(ctx, "Database created successfully", map[string]interface{}{
		"projectID":  projectID,
		"databaseID": database.DatabaseID,
	})

	return nil
}

// GetDatabase retrieves a database by ID
func (r *DocumentRepository) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	if projectID == "" || databaseID == "" {
		return nil, sharedErrors.NewValidationError("Project ID and Database ID are required")
	}

	filter := map[string]interface{}{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	var database model.Database
	err := r.db.Collection("databases").FindOne(ctx, filter).Decode(&database)
	if err != nil {
		return nil, sharedErrors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", databaseID, projectID))
	}

	return &database, nil
}

// UpdateDatabase updates a database
func (r *DocumentRepository) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	if database == nil {
		return sharedErrors.NewValidationError("Database cannot be nil")
	}
	if projectID == "" || database.DatabaseID == "" {
		return sharedErrors.NewValidationError("Project ID and Database ID are required")
	}

	filter := map[string]interface{}{
		"project_id":  projectID,
		"database_id": database.DatabaseID,
	}

	// Update timestamp
	database.UpdatedAt = time.Now()

	update := map[string]interface{}{"$set": database}

	result, err := r.db.Collection("databases").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update database '%s': %w", database.DatabaseID, err)
	}

	if result.Matched() == 0 {
		return sharedErrors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", database.DatabaseID, projectID))
	}

	r.logger.Info(ctx, "Database updated successfully", map[string]interface{}{
		"projectID":  projectID,
		"databaseID": database.DatabaseID,
	})

	return nil
}

// DeleteDatabase deletes a database by ID
func (r *DocumentRepository) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	if projectID == "" || databaseID == "" {
		return sharedErrors.NewValidationError("Project ID and Database ID are required")
	}

	filter := map[string]interface{}{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	result, err := r.db.Collection("databases").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete database '%s': %w", databaseID, err)
	}

	if result.Deleted() == 0 {
		return sharedErrors.NewNotFoundError(fmt.Sprintf("Database '%s' not found in project '%s'", databaseID, projectID))
	}

	r.logger.Info(ctx, "Database deleted successfully", map[string]interface{}{
		"projectID":  projectID,
		"databaseID": databaseID,
	})

	return nil
}

// ListDatabases lists all databases in a project
func (r *DocumentRepository) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	if projectID == "" {
		return nil, sharedErrors.NewValidationError("Project ID is required")
	}

	filter := map[string]interface{}{"project_id": projectID}
	cursor, err := r.db.Collection("databases").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases for project '%s': %w", projectID, err)
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
		return nil, fmt.Errorf("cursor error while listing databases: %w", err)
	}

	return databases, nil
}

// --- Index Operations ---

// CreateIndex creates a new index
func (r *DocumentRepository) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.Index) error {
	// Convert model.Index to model.CollectionIndex
	collectionIndex := &model.CollectionIndex{
		Name:   index.Name,
		Fields: index.Fields,
		State:  index.State,
	}
	return r.indexOps.CreateIndex(ctx, projectID, databaseID, collectionID, collectionIndex)
}

// DeleteIndex deletes an index
func (r *DocumentRepository) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	return r.indexOps.DeleteIndex(ctx, projectID, databaseID, collectionID, indexID)
}

// ListIndexes lists all indexes for a collection
func (r *DocumentRepository) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.Index, error) {
	collectionIndexes, err := r.indexOps.ListIndexes(ctx, projectID, databaseID, collectionID)
	if err != nil {
		return nil, err
	}

	// Convert []*model.CollectionIndex to []*model.Index
	indexes := make([]*model.Index, len(collectionIndexes))
	for i, ci := range collectionIndexes {
		indexes[i] = &model.Index{
			Name:   ci.Name,
			Fields: ci.Fields,
			State:  ci.State,
		}
	}
	return indexes, nil
}

// GetIndex retrieves an index by ID
func (r *DocumentRepository) GetIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) (*model.Index, error) {
	collectionIndex, err := r.indexOps.GetIndex(ctx, projectID, databaseID, collectionID, indexID)
	if err != nil {
		return nil, err
	}

	// Convert *model.CollectionIndex to *model.Index
	index := &model.Index{
		Name:   collectionIndex.Name,
		Fields: collectionIndex.Fields,
		State:  collectionIndex.State,
	}
	return index, nil
}

// --- Query Operations ---

// ExecuteQuery ejecuta una consulta sobre una colección usando la sintaxis simplificada para arquitectura hexagonal
func (r *DocumentRepository) ExecuteQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	filter := map[string]interface{}{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}
	cursor, err := r.db.Collection(collectionID).Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*model.Document
	for cursor.Next(ctx) {
		var doc model.Document
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, &doc)
	}
	return documents, nil
}

// --- Transaction operations (simplified for hexagonal architecture) ---

// RunTransaction ejecuta una función dentro de una transacción (simplificado para arquitectura hexagonal)
func (r *DocumentRepository) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	r.logger.Info("Starting transaction")
	tx := &simpleTransaction{
		repo:      r,
		ctx:       ctx,
		startTime: time.Now(),
	}
	err := fn(tx)
	if err != nil {
		r.logger.Error("Transaction failed", "error", err)
		return fmt.Errorf("transaction failed: %w", err)
	}
	r.logger.Info("Transaction completed successfully")
	return nil
}

// simpleTransaction implements repository.Transaction para arquitectura hexagonal
type simpleTransaction struct {
	repo      *DocumentRepository
	ctx       context.Context
	startTime time.Time
}

// GetDocument retrieves a document within the transaction
func (tx *simpleTransaction) Get(projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return tx.repo.GetDocument(tx.ctx, projectID, databaseID, collectionID, documentID)
}

// GetByPath retrieves a document by path within the transaction
func (tx *simpleTransaction) GetByPath(path string) (*model.Document, error) {
	return tx.repo.GetDocumentByPath(tx.ctx, path)
}

// Create creates a document within the transaction
func (tx *simpleTransaction) Create(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.CreateDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data)
	return err
}

// CreateByPath creates a document by path within the transaction
func (tx *simpleTransaction) CreateByPath(path string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.CreateDocumentByPath(tx.ctx, path, data)
	return err
}

// Update updates a document within the transaction
func (tx *simpleTransaction) Update(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) error {
	_, err := tx.repo.UpdateDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
	return err
}

// UpdateByPath updates a document by path within the transaction
func (tx *simpleTransaction) UpdateByPath(path string, data map[string]*model.FieldValue, updateMask []string) error {
	_, err := tx.repo.UpdateDocumentByPath(tx.ctx, path, data, updateMask)
	return err
}

// Set sets a document within the transaction
func (tx *simpleTransaction) Set(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) error {
	_, err := tx.repo.SetDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, merge)
	return err
}

// SetByPath sets a document by path within the transaction
func (tx *simpleTransaction) SetByPath(path string, data map[string]*model.FieldValue, merge bool) error {
	// Parse the path to extract project, database, collection, and document IDs
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid document path: %w", err)
	}

	// Extract document path segments (collection/document pairs)
	segments := firestore.ParseDocumentPath(pathInfo.DocumentPath)
	if len(segments) < 2 {
		return fmt.Errorf("invalid document path: must include collection and document")
	}

	// For simplicity, assume the last two segments are collection and document
	collectionID := segments[len(segments)-2]
	documentID := segments[len(segments)-1]

	_, err = tx.repo.SetDocument(tx.ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, data, merge)
	return err
}

// Delete deletes a document within the transaction
func (tx *simpleTransaction) Delete(projectID, databaseID, collectionID, documentID string) error {
	return tx.repo.DeleteDocument(tx.ctx, projectID, databaseID, collectionID, documentID)
}

// DeleteByPath deletes a document by path within the transaction
func (tx *simpleTransaction) DeleteByPath(path string) error {
	return tx.repo.DeleteDocumentByPath(tx.ctx, path)
}

// Query executes a query within the transaction
func (tx *simpleTransaction) Query(projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return tx.repo.ExecuteQuery(tx.ctx, projectID, databaseID, collectionID, query)
}

// GetTransactionID returns the transaction ID
func (tx *simpleTransaction) GetTransactionID() string {
	return "simple-tx-" + fmt.Sprintf("%d", tx.startTime.UnixNano())
}

// GetStartTime returns the transaction start time
func (tx *simpleTransaction) GetStartTime() time.Time {
	return tx.startTime
}

// IsReadOnly returns whether this is a read-only transaction
func (tx *simpleTransaction) IsReadOnly() bool {
	return false // This simple implementation supports read-write transactions
}
