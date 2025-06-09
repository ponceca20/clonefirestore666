package repository

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"time"
)

// FirestoreRepository defines the interface for Firestore data persistence
// following the exact Firestore hierarchy: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
type FirestoreRepository interface {
	// Project operations
	CreateProject(ctx context.Context, project *model.Project) error
	GetProject(ctx context.Context, projectID string) (*model.Project, error)
	UpdateProject(ctx context.Context, project *model.Project) error
	DeleteProject(ctx context.Context, projectID string) error
	ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error)

	// Database operations
	CreateDatabase(ctx context.Context, projectID string, database *model.Database) error
	GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error)
	UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error
	DeleteDatabase(ctx context.Context, projectID, databaseID string) error
	ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error)

	// Collection operations
	CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error
	GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error)
	UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error
	DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error
	ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error)

	// Document operations - Core Firestore CRUD
	GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error)
	CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error)
	UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error)
	SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error)
	DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error

	// Document path-based operations (for compatibility)
	GetDocumentByPath(ctx context.Context, path string) (*model.Document, error)
	CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error)
	UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error)
	DeleteDocumentByPath(ctx context.Context, path string) error

	// Query operations
	RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error)
	RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error)
	RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error)

	// List documents with pagination
	ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error)
	// Batch operations
	RunTransaction(ctx context.Context, fn func(tx Transaction) error) error
	RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error)

	// Atomic field transforms
	AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error
	AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error
	AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error
	AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error

	// Index operations
	CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error
	DeleteIndex(ctx context.Context, projectID, databaseID, collectionID string, indexID string) error
	ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error)

	// Subcollection operations
	ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error)
}

// Transaction defines the interface for operations within a Firestore transaction
// Provides atomic reads and writes with ACID guarantees
type Transaction interface {
	// Document operations within transaction
	Get(projectID, databaseID, collectionID, documentID string) (*model.Document, error)
	GetByPath(path string) (*model.Document, error)
	Create(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error
	CreateByPath(path string, data map[string]*model.FieldValue) error
	Update(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) error
	UpdateByPath(path string, data map[string]*model.FieldValue, updateMask []string) error
	Set(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) error
	SetByPath(path string, data map[string]*model.FieldValue, merge bool) error
	Delete(projectID, databaseID, collectionID, documentID string) error
	DeleteByPath(path string) error

	// Query within transaction (read-only)
	Query(projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error)

	// Transaction metadata	GetTransactionID() string
	GetStartTime() time.Time
	IsReadOnly() bool
}
