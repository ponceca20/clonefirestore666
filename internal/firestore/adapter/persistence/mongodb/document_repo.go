package mongodb

import (
	"context"
	"fmt"
	"time"

	"errors"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/config"
	firestoremodel "firestore-clone/internal/firestore/domain/model"
	firestorerepo "firestore-clone/internal/firestore/domain/repository" // Added for compile-time check
	"firestore-clone/internal/shared/utils"                            // For GetTenantIDFromContext
	"firestore-clone/pkg/logger"                                       // Assuming a shared logger package

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson/primitive" // Added for ObjectID and DateTime
)
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const firestoreDatabaseSuffix = "_fs"

// DocumentRepository holds the MongoDB client, logger and config
// It implements the repository.FirestoreRepository interface (implicitly for now, will add methods)
type DocumentRepository struct {
	client *mongo.Client
	logger logger.Logger
	config *config.FirestoreConfig
}

// Helper to get tenant-specific database and parse path
func (r *DocumentRepository) getTenantDBCollectionAndDocID(ctx context.Context, path string) (*mongo.Collection, string, error) {
	tenantID, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		r.logger.Error("Failed to get TenantID from context", "error", err)
		return nil, "", fmt.Errorf("failed to get TenantID from context: %w", err)
	}
	if tenantID == "" {
		r.logger.Error("TenantID from context is empty")
		return nil, "", errors.New("tenantID from context is empty")
	}

	// Using DefaultDatabaseName from config as a prefix for tenant DB for clarity, or just tenantID + suffix
	// For example, if DefaultDatabaseName is "main", tenant DB could be "main_tenant_<id>_fs"
	// Or simply "tenant_<id>_fs"
	// Let's use "tenant_<id>_fs" for simplicity and clear separation.
	tenantDBName := "tenant_" + tenantID + firestoreDatabaseSuffix
	db := r.client.Database(tenantDBName)

	parts := strings.SplitN(path, "/", 2)
	collectionName := parts[0]
	if collectionName == "" {
		return nil, "", errors.New("collection name cannot be empty in path")
	}

	var docID string
	if len(parts) > 1 {
		docID = parts[1]
		if docID == "" { // Path like "myCollection/" is invalid
			return nil, "", errors.New("document ID cannot be empty if specified in path")
		}
	}
	// If docID is empty here, it means the path was just "myCollection"

	return db.Collection(collectionName), docID, nil
}

// NewDocumentRepository creates a new MongoDB document repository
func NewDocumentRepository(cfg *config.FirestoreConfig, log logger.Logger) (*DocumentRepository, error) {
	if cfg.MongoDBURI == "" {
		log.Error("MongoDB URI is not configured", "config_field", "MongoDBURI")
		return nil, fmt.Errorf("MongoDB URI is required but not configured")
	}

	log.Info("Connecting to MongoDB...", "uri", cfg.MongoDBURI)

	// Set client options
	clientOptions := options.Client().ApplyURI(cfg.MongoDBURI)
	// Set a server selection timeout to prevent indefinite blocking if MongoDB is not reachable.
	// This is crucial for application startup.
	clientOptions.SetServerSelectionTimeout(15 * time.Second)


	// Connect to MongoDB
	// Use a context with a timeout for the connection attempt itself.
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 20*time.Second) // Slightly longer than server selection.
	defer connectCancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		log.Error("Failed to connect to MongoDB", "error", err, "uri", cfg.MongoDBURI)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the primary to verify the connection.
	// Use a separate context with a shorter timeout for the ping.
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		log.Error("Failed to ping MongoDB", "error", err, "uri", cfg.MongoDBURI)
		// Attempt to disconnect if ping fails after a successful connect call
		// Use a background context for disconnect as the pingCtx might be done.
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			log.Error("Failed to disconnect MongoDB client after ping failure", "error", disconnectErr)
		}
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Info("Successfully connected to MongoDB and pinged primary replica", "uri", cfg.MongoDBURI)

	return &DocumentRepository{
		client: client,
		logger: log,
		config: cfg,
	}, nil
}

// Disconnect allows graceful disconnection from MongoDB
func (r *DocumentRepository) Disconnect(ctx context.Context) error {
	if r.client == nil {
		r.logger.Info("MongoDB client is not connected, nothing to disconnect.")
		return nil
	}
	r.logger.Info("Disconnecting MongoDB client...")
	err := r.client.Disconnect(ctx)
	if err != nil {
		r.logger.Error("Failed to disconnect MongoDB client", "error", err)
		return err
	}
	r.logger.Info("MongoDB client disconnected successfully.")
	return nil
}

// GetClient returns the underlying mongo client.
// This might be useful for more complex operations or transaction management not covered by the repository.
func (r *DocumentRepository) GetClient() *mongo.Client {
	return r.client
}

// GetDefaultDatabaseName returns the default database name from the config.
func (r *DocumentRepository) GetDefaultDatabaseName() string {
	return r.config.DefaultDatabaseName
}

// GetCollection returns a handle to a MongoDB collection using the default database name.
// This specific helper might be less used if tenant-specific databases are always used for document operations.
// It could be used for admin/shared collections not tied to a tenant.
func (r *DocumentRepository) GetSharedCollection(collectionName string) *mongo.Collection {
	return r.client.Database(r.config.DefaultDatabaseName).Collection(collectionName)
}

// --- Implementing repository.FirestoreRepository ---

// GetDocument retrieves a document from Firestore.
func (r *DocumentRepository) GetDocument(ctx context.Context, path string) (*firestoremodel.Document, error) {
	coll, docID, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return nil, err
	}
	if docID == "" {
		return nil, errors.New("GetDocument requires a full document path (collection/docID)")
	}

	var result bson.M // Using bson.M to get raw data
	err = coll.FindOne(ctx, bson.M{"_id": docID}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Firestore returns nil, nil if document not found
		}
		r.logger.Error("Failed to get document from MongoDB", "path", path, "error", err)
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	// Assuming 'createdAt' and 'updatedAt' are stored as top-level fields in BSON
	// and '_id' is the document ID.
	// The data map for firestoremodel.Document should exclude these.
	docData := make(map[string]interface{})
	var createdAt, updatedAt time.Time

	for key, value := range result {
		switch key {
		case "_id":
			// ID is already docID
		case "createdAt":
			if t, ok := value.(primitive.DateTime); ok {
				createdAt = t.Time()
			} else if t, ok := value.(time.Time); ok {
				createdAt = t
			}
		case "updatedAt":
			if t, ok := value.(primitive.DateTime); ok {
				updatedAt = t.Time()
			} else if t, ok := value.(time.Time); ok {
				updatedAt = t
			}
		default:
			docData[key] = value
		}
	}
	// If createdAt/updatedAt are not found or not times, they will be zero, which is standard.

	return &firestoremodel.Document{
		ID:        docID,
		Data:      docData,
		Path:      path, // Full path used for retrieval
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// CreateDocument creates a new document in Firestore.
// Path can be "myCollection" (docID auto-generated) or "myCollection/myDocID".
func (r *DocumentRepository) CreateDocument(ctx context.Context, path string, data map[string]interface{}) (*firestoremodel.Document, error) {
	coll, docID, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	docToInsert := bson.M{}
	for k, v := range data {
		docToInsert[k] = v
	}
	docToInsert["createdAt"] = now
	docToInsert["updatedAt"] = now

	var finalDocID string
	if docID == "" { // Auto-generate ID
		finalDocID = primitive.NewObjectID().Hex() // MongoDB auto-generated ID
		docToInsert["_id"] = finalDocID
		_, err = coll.InsertOne(ctx, docToInsert)
	} else { // Use provided ID, check for existence first (Firestore behavior)
		finalDocID = docID
		docToInsert["_id"] = finalDocID
		// To strictly mimic Firestore's CreateDocument (which fails if doc exists),
		// we'd ideally use a transaction or a find then insert.
		// For simplicity here, InsertOne will fail if _id already exists due to unique index.
		// However, a more robust check would be:
		//  var existingDoc bson.M
		//  errCheck := coll.FindOne(ctx, bson.M{"_id": finalDocID}).Decode(&existingDoc)
		//  if errCheck == nil { // Document found
		//    return nil, errors.New("document already exists at path: " + path + "/" + finalDocID)
		//  }
		//  if !errors.Is(errCheck, mongo.ErrNoDocuments) { // Some other error during check
		//     return nil, fmt.Errorf("failed to check for existing document: %w", errCheck)
		//  }
		// If we reach here, doc does not exist, proceed with insert.
		_, err = coll.InsertOne(ctx, docToInsert)
	}

	if err != nil {
		r.logger.Error("Failed to create document in MongoDB", "path", path, "docID", finalDocID, "error", err)
		// Could check for duplicate key error specifically: mongo.IsDuplicateKeyError(err)
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	fullPath := path
	if docID == "" { // If ID was auto-generated, path was just collection name
		fullPath = path + "/" + finalDocID
	}


	return &firestoremodel.Document{
		ID:        finalDocID,
		Data:      data, // Return original data, not internal bson.M
		Path:      fullPath,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateDocument updates an existing document.
// For simplicity, this will be an upsert operation if docID doesn't exist,
// or a full document replace if it does. Firestore's Update is field-level.
// A true Firestore Update would use bson.M{"$set": data}.
func (r *DocumentRepository) UpdateDocument(ctx context.Context, path string, data map[string]interface{}) (*firestoremodel.Document, error) {
	coll, docID, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return nil, err
	}
	if docID == "" {
		return nil, errors.New("UpdateDocument requires a full document path (collection/docID)")
	}

	now := time.Now()
	updateData := bson.M{}
	for k, v := range data {
		updateData[k] = v
	}
	updateData["updatedAt"] = now
	// To ensure createdAt is preserved or set on creation (if upserting)
	// updatePayload := bson.M{"$set": updateData, "$setOnInsert": bson.M{"createdAt": now}}
	// For a simple replace, ensure all fields are set.
	// For field-level update as per Firestore:
	setPayload := bson.M{"$set": updateData}
	// If the document might not exist and we want to create it with 'createdAt':
	// setPayload["$setOnInsert"] = bson.M{"createdAt": now}


	// Using UpdateOne with $set for field-level updates. Upsert can be true or false.
	// Firestore's Update fails if the document does not exist. So, Upsert should be false.
	opts := options.Update().SetUpsert(false)
	res, err := coll.UpdateOne(ctx, bson.M{"_id": docID}, setPayload, opts)
	if err != nil {
		r.logger.Error("Failed to update document in MongoDB", "path", path, "error", err)
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	if res.MatchedCount == 0 {
		return nil, errors.New("document not found for update at path: " + path) // Firestore behavior
	}

	// To return the updated document, we would typically re-fetch it.
	// For now, returning based on input, assuming success.
	// A proper implementation would fetch the document to get its actual state (including createdAt).

	// Fetch the document to get its current state including createdAt
    updatedDoc, getErr := r.GetDocument(ctx, path)
    if getErr != nil {
        r.logger.Error("Failed to fetch document after update", "path", path, "error", getErr)
        // Fallback or error. For now, construct from available info.
        // This means CreatedAt might be zero if it wasn't part of this update.
        return &firestoremodel.Document{
            ID:        docID,
            Data:      data, // This is just the updated fields
            Path:      path,
            // CreatedAt: // Unknown without fetch or if not part of updateData logic
            UpdatedAt: now,
        }, nil
    }
    return updatedDoc, nil
}

// DeleteDocument deletes a document from Firestore.
func (r *DocumentRepository) DeleteDocument(ctx context.Context, path string) error {
	coll, docID, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return err
	}
	if docID == "" {
		return errors.New("DeleteDocument requires a full document path (collection/docID)")
	}

	res, err := coll.DeleteOne(ctx, bson.M{"_id": docID})
	if err != nil {
		r.logger.Error("Failed to delete document from MongoDB", "path", path, "error", err)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if res.DeletedCount == 0 {
		// In Firestore, deleting a non-existent document is not an error.
		r.logger.Info("Attempted to delete non-existent document", "path", path)
		return nil // Or return a specific error/status if required by interface contract
	}

	return nil
}

// ListDocuments is not implemented yet.
func (r *DocumentRepository) ListDocuments(ctx context.Context, parentPath string, query firestoremodel.Query) ([]*firestoremodel.Document, error) {
	_, _, err := r.getTenantDBCollectionAndDocID(ctx, parentPath)
	if err != nil {
		return nil, err // Basic validation of tenant and path
	}
	return nil, errors.New("ListDocuments not implemented yet")
}

// RunTransaction is not implemented yet.
func (r *DocumentRepository) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	// Getting tenant ID here would be important for logging or pre-checks,
	// but actual transaction would need to be on tenantDB.
	_, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RunTransaction: failed to get TenantID from context: %w", err)
	}
	return errors.New("RunTransaction not implemented yet")
}

// RunBatch is not implemented yet.
func (r *DocumentRepository) RunBatch(ctx context.Context, operations []firestoremodel.WriteOperation) error {
	_, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("RunBatch: failed to get TenantID from context: %w", err)
	}
	return errors.New("RunBatch not implemented yet")
}

// Increment is not implemented yet.
func (r *DocumentRepository) Increment(ctx context.Context, path string, field string, value int64) error {
	_, _, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return err // Basic validation
	}
	return errors.New("Increment not implemented yet")
}

// ArrayUnion is not implemented yet.
func (r *DocumentRepository) ArrayUnion(ctx context.Context, path string, field string, elements []interface{}) error {
	_, _, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return err // Basic validation
	}
	return errors.New("ArrayUnion not implemented yet")
}

// ArrayRemove is not implemented yet.
func (r *DocumentRepository) ArrayRemove(ctx context.Context, path string, field string, elements []interface{}) error {
	_, _, err := r.getTenantDBCollectionAndDocID(ctx, path)
	if err != nil {
		return err // Basic validation
	}
	return errors.New("ArrayRemove not implemented yet")
}

// Ensure DocumentRepository implements FirestoreRepository (compile-time check)
var _ firestorerepo.FirestoreRepository = (*DocumentRepository)(nil)
