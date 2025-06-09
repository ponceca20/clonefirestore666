package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrDocumentNotFound   = errors.New("document not found")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrInvalidPath        = errors.New("invalid document path")
	ErrPreconditionFailed = errors.New("precondition failed")
	ErrIndexAlreadyExists = errors.New("index already exists")
	ErrIndexNotFound      = errors.New("index not found")
)

// DocumentRepository implements the Firestore document repository using MongoDB
// This is the main repository that coordinates different operation types
type DocumentRepository struct {
	db             *mongo.Database
	eventBus       *eventbus.EventBus
	logger         logger.Logger
	documentsCol   *mongo.Collection
	indexesCol     *mongo.Collection
	collectionsCol *mongo.Collection

	// Specialized operation handlers
	documentOps   *DocumentOperations
	batchOps      *BatchOperations
	collectionOps *CollectionOperations
	atomicOps     *AtomicOperations
	projectDbOps  *ProjectDatabaseOperations
	indexOps      *IndexOperations
}

// NewDocumentRepository creates a new MongoDB-backed document repository
func NewDocumentRepository(db *mongo.Database, eventBus *eventbus.EventBus, logger logger.Logger) *DocumentRepository {
	repo := &DocumentRepository{
		db:             db,
		eventBus:       eventBus,
		logger:         logger,
		documentsCol:   db.Collection("documents"),
		indexesCol:     db.Collection("indexes"),
		collectionsCol: db.Collection("collections"),
	}

	// Initialize specialized operation handlers
	repo.documentOps = NewDocumentOperations(repo)
	repo.batchOps = NewBatchOperations(repo)
	repo.collectionOps = NewCollectionOperations(repo)
	repo.atomicOps = NewAtomicOperations(repo)
	repo.projectDbOps = NewProjectDatabaseOperations(repo)
	repo.indexOps = NewIndexOperations(repo)

	return repo
}

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

// CreateProject creates a new project
func (r *DocumentRepository) CreateProject(ctx context.Context, project *model.Project) error {
	return r.projectDbOps.CreateProject(ctx, project)
}

// GetProject retrieves a project by ID
func (r *DocumentRepository) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return r.projectDbOps.GetProject(ctx, projectID)
}

// UpdateProject updates a project
func (r *DocumentRepository) UpdateProject(ctx context.Context, project *model.Project) error {
	return r.projectDbOps.UpdateProject(ctx, project)
}

// DeleteProject deletes a project by ID
func (r *DocumentRepository) DeleteProject(ctx context.Context, projectID string) error {
	return r.projectDbOps.DeleteProject(ctx, projectID)
}

// ListProjects lists all projects for an owner
func (r *DocumentRepository) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return r.projectDbOps.ListProjects(ctx, ownerEmail)
}

// --- Database Operations ---

// CreateDatabase creates a new database
func (r *DocumentRepository) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return r.projectDbOps.CreateDatabase(ctx, projectID, database)
}

// GetDatabase retrieves a database by ID
func (r *DocumentRepository) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return r.projectDbOps.GetDatabase(ctx, projectID, databaseID)
}

// UpdateDatabase updates a database
func (r *DocumentRepository) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return r.projectDbOps.UpdateDatabase(ctx, projectID, database)
}

// DeleteDatabase deletes a database by ID
func (r *DocumentRepository) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return r.projectDbOps.DeleteDatabase(ctx, projectID, databaseID)
}

// ListDatabases lists all databases in a project
func (r *DocumentRepository) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return r.projectDbOps.ListDatabases(ctx, projectID)
}

// --- Index Operations ---

// CreateIndex creates a new index
func (r *DocumentRepository) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	return r.indexOps.CreateIndex(ctx, projectID, databaseID, collectionID, index)
}

// DeleteIndex deletes an index
func (r *DocumentRepository) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexID string) error {
	return r.indexOps.DeleteIndex(ctx, projectID, databaseID, collectionID, indexID)
}

// ListIndexes lists all indexes for a collection
func (r *DocumentRepository) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return r.indexOps.ListIndexes(ctx, projectID, databaseID, collectionID)
}

// --- Query Operations ---

// RunQuery ejecuta una consulta sobre una colección usando MongoDB
func (r *DocumentRepository) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}
	// Aplica filtros adicionales del modelo Query
	for _, f := range query.Filters {
		switch f.Operator {
		case model.OperatorEqual:
			filter["fields."+f.Field+".value"] = f.Value
			// Agrega más operadores según sea necesario
		}
	}

	findOpts := options.Find()
	if query.Limit > 0 {
		findOpts.SetLimit(int64(query.Limit))
	}
	if query.Offset > 0 {
		findOpts.SetSkip(int64(query.Offset))
	}
	if len(query.Orders) > 0 {
		orderBy := bson.D{}
		for _, o := range query.Orders {
			dir := 1
			if o.Direction == model.DirectionDescending {
				dir = -1
			}
			orderBy = append(orderBy, bson.E{Key: "fields." + o.Field + ".value", Value: dir})
		}
		findOpts.SetSort(orderBy)
	}

	cursor, err := r.documentsCol.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []*model.Document
	for cursor.Next(ctx) {
		var doc model.Document
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return docs, nil
}

// RunCollectionGroupQuery ejecuta una consulta sobre todas las colecciones con el mismo ID
func (r *DocumentRepository) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	r.logger.Info("Running collection group query",
		"projectID", projectID,
		"databaseID", databaseID,
		"collectionID", collectionID)

	// Build filter for collection group query
	filter := bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}
	// Add query filters if provided
	if query != nil && len(query.Filters) > 0 {
		queryFilter := r.buildQueryFilter(query.Filters)
		filter = bson.M{"$and": []bson.M{filter, queryFilter}}
	}

	// Set up find options
	opts := options.Find()
	if query != nil {
		if query.Limit > 0 {
			opts.SetLimit(int64(query.Limit))
		}
		if len(query.Orders) > 0 {
			sort := bson.D{}
			for _, order := range query.Orders {
				direction := 1
				if order.Direction == model.DirectionDescending {
					direction = -1
				}
				sort = append(sort, bson.E{Key: "fields." + order.Field, Value: direction})
			}
			opts.SetSort(sort)
		}
	}
	// Execute query across all collections with the same name
	cursor, err := r.db.Collection("documents").Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("Failed to execute collection group query", "error", err)
		return nil, fmt.Errorf("failed to execute collection group query: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*model.Document
	for cursor.Next(ctx) {
		var doc model.Document
		if err := cursor.Decode(&doc); err != nil {
			r.logger.Error("Failed to decode document", "error", err)
			continue
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("Cursor error during collection group query", "error", err)
		return nil, fmt.Errorf("cursor error during collection group query: %w", err)
	}

	r.logger.Info("Collection group query completed", "resultCount", len(documents))
	return documents, nil
}

// RunAggregationQuery ejecuta una consulta de agregación tipo Firestore
func (r *DocumentRepository) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	r.logger.Info("Running aggregation query",
		"projectID", projectID,
		"databaseID", databaseID,
		"collectionID", collectionID)
	// Build base match stage
	matchStage := bson.D{{Key: "$match", Value: bson.M{
		"project_id":    projectID,
		"database_id":   databaseID,
		"collection_id": collectionID,
	}}}

	// Build aggregation pipeline
	pipeline := mongo.Pipeline{matchStage}
	// Add query filters if provided
	if query != nil && len(query.Filters) > 0 {
		for _, filter := range query.Filters {
			if filter.Field != "" && filter.Value != nil {
				fieldPath := "fields." + filter.Field + ".value"
				switch filter.Operator {
				case model.OperatorEqual:
					pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{fieldPath: filter.Value}}})
				case model.OperatorGreaterThan:
					pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{fieldPath: bson.M{"$gt": filter.Value}}}})
				case model.OperatorLessThan:
					pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{fieldPath: bson.M{"$lt": filter.Value}}}})
				}
			}
		}
	}

	// Add count aggregation
	pipeline = append(pipeline, bson.D{{Key: "$count", Value: "total"}})

	// Execute aggregation
	cursor, err := r.documentsCol.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("Failed to execute aggregation query", "error", err)
		return nil, fmt.Errorf("failed to execute aggregation query: %w", err)
	}
	defer cursor.Close(ctx)

	// Get result
	var result struct {
		Total int64 `bson:"total"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			r.logger.Error("Failed to decode aggregation result", "error", err)
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("Cursor error during aggregation query", "error", err)
		return nil, fmt.Errorf("cursor error during aggregation query: %w", err)
	}

	aggregationResult := &model.AggregationResult{
		Count:    &result.Total,
		ReadTime: time.Now(),
	}

	r.logger.Info("Aggregation query completed", "count", result.Total)
	return aggregationResult, nil
}

// RunTransaction ejecuta una función dentro de una transacción MongoDB
func (r *DocumentRepository) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	r.logger.Info("Starting MongoDB transaction")

	// Start a MongoDB session
	session, err := r.db.Client().StartSession()
	if err != nil {
		r.logger.Error("Failed to start MongoDB session", "error", err)
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute the transaction
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Start transaction
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Create a transaction wrapper
		tx := &mongoTransaction{
			session: session,
			repo:    r,
			ctx:     sc,
		}

		// Execute the user function
		if err := fn(tx); err != nil {
			// Abort transaction on error
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				r.logger.Error("Failed to abort transaction", "error", abortErr)
			}
			return err
		}

		// Commit transaction
		if err := session.CommitTransaction(sc); err != nil {
			r.logger.Error("Failed to commit transaction", "error", err)
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	})

	if err != nil {
		r.logger.Error("Transaction failed", "error", err)
		return fmt.Errorf("transaction failed: %w", err)
	}

	r.logger.Info("Transaction completed successfully")
	return nil
}

// mongoTransaction implements repository.Transaction for MongoDB
type mongoTransaction struct {
	session mongo.Session
	repo    *DocumentRepository
	ctx     mongo.SessionContext
}

// GetDocument retrieves a document within the transaction
func (tx *mongoTransaction) GetDocument(projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	// Use the transaction context
	return tx.repo.documentOps.GetDocument(tx.ctx, projectID, databaseID, collectionID, documentID)
}

// SetDocument sets a document within the transaction
func (tx *mongoTransaction) SetDocument(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.documentOps.SetDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, false)
	return err
}

// UpdateDocument updates a document within the transaction
func (tx *mongoTransaction) UpdateDocument(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.documentOps.UpdateDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, nil)
	return err
}

// DeleteDocument deletes a document within the transaction
func (tx *mongoTransaction) DeleteDocument(projectID, databaseID, collectionID, documentID string) error {
	return tx.repo.documentOps.DeleteDocument(tx.ctx, projectID, databaseID, collectionID, documentID)
}

// Get retrieves a document within the transaction (alias for GetDocument)
func (tx *mongoTransaction) Get(projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return tx.GetDocument(projectID, databaseID, collectionID, documentID)
}

// GetByPath retrieves a document by path within the transaction
func (tx *mongoTransaction) GetByPath(path string) (*model.Document, error) {
	return tx.repo.documentOps.GetDocumentByPath(tx.ctx, path)
}

// Create creates a new document within the transaction
func (tx *mongoTransaction) Create(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.documentOps.CreateDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data)
	return err
}

// CreateByPath creates a document by path within the transaction
func (tx *mongoTransaction) CreateByPath(path string, data map[string]*model.FieldValue) error {
	_, err := tx.repo.documentOps.CreateDocumentByPath(tx.ctx, path, data)
	return err
}

// Update updates a document within the transaction
func (tx *mongoTransaction) Update(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) error {
	_, err := tx.repo.documentOps.UpdateDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
	return err
}

// UpdateByPath updates a document by path within the transaction
func (tx *mongoTransaction) UpdateByPath(path string, data map[string]*model.FieldValue, updateMask []string) error {
	_, err := tx.repo.documentOps.UpdateDocumentByPath(tx.ctx, path, data, updateMask)
	return err
}

// Set sets (creates or updates) a document within the transaction
func (tx *mongoTransaction) Set(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) error {
	_, err := tx.repo.documentOps.SetDocument(tx.ctx, projectID, databaseID, collectionID, documentID, data, merge)
	return err
}

// SetByPath sets a document by path within the transaction
func (tx *mongoTransaction) SetByPath(path string, data map[string]*model.FieldValue, merge bool) error {
	// Parse path to extract components
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	_, err = tx.repo.documentOps.SetDocument(tx.ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, data, merge)
	return err
}

// Delete deletes a document within the transaction (alias for DeleteDocument)
func (tx *mongoTransaction) Delete(projectID, databaseID, collectionID, documentID string) error {
	return tx.DeleteDocument(projectID, databaseID, collectionID, documentID)
}

// DeleteByPath deletes a document by path within the transaction
func (tx *mongoTransaction) DeleteByPath(path string) error {
	return tx.repo.documentOps.DeleteDocumentByPath(tx.ctx, path)
}

// Query executes a query within the transaction (read-only)
func (tx *mongoTransaction) Query(projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return tx.repo.RunQuery(tx.ctx, projectID, databaseID, collectionID, query)
}

// GetTransactionID returns the transaction ID
func (tx *mongoTransaction) GetTransactionID() string {
	// MongoDB doesn't expose transaction IDs directly, return a placeholder
	return fmt.Sprintf("mongo-tx-%p", tx.session)
}

// GetStartTime returns the transaction start time
func (tx *mongoTransaction) GetStartTime() time.Time {
	// MongoDB doesn't track start time directly, return current time as approximation
	return time.Now()
}

// IsReadOnly returns whether this is a read-only transaction
func (tx *mongoTransaction) IsReadOnly() bool {
	// For simplicity, we'll return false since MongoDB transactions can be read-write
	return false
}

// buildQueryFilter converts Firestore query filters to MongoDB filters
func (r *DocumentRepository) buildQueryFilter(filters []model.Filter) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}

	result := bson.M{}
	for _, filter := range filters {
		if filter.Field == "" {
			continue
		}

		fieldPath := "fields." + filter.Field + ".value"

		switch filter.Operator {
		case model.OperatorEqual:
			result[fieldPath] = filter.Value
		case model.OperatorNotEqual:
			result[fieldPath] = bson.M{"$ne": filter.Value}
		case model.OperatorLessThan:
			result[fieldPath] = bson.M{"$lt": filter.Value}
		case model.OperatorLessThanOrEqual:
			result[fieldPath] = bson.M{"$lte": filter.Value}
		case model.OperatorGreaterThan:
			result[fieldPath] = bson.M{"$gt": filter.Value}
		case model.OperatorGreaterThanOrEqual:
			result[fieldPath] = bson.M{"$gte": filter.Value}
		case model.OperatorIn:
			result[fieldPath] = bson.M{"$in": filter.Value}
		case model.OperatorNotIn:
			result[fieldPath] = bson.M{"$nin": filter.Value}
		case model.OperatorArrayContains:
			result[fieldPath] = bson.M{"$elemMatch": bson.M{"$eq": filter.Value}}
		case model.OperatorArrayContainsAny:
			if values, ok := filter.Value.([]interface{}); ok {
				result[fieldPath] = bson.M{"$elemMatch": bson.M{"$in": values}}
			}
		}
	}

	return result
}
