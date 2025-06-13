package usecase

import (
	"context"
	"errors"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"
)

var (
	ErrInvalidProjectPath = errors.New("invalid project path")
	ErrUnauthorized       = errors.New("unauthorized access")
)

// FirestoreUsecaseInterface defines the contract for Firestore operations
type FirestoreUsecaseInterface interface {
	// Document operations
	CreateDocument(ctx context.Context, req CreateDocumentRequest) (*model.Document, error)
	GetDocument(ctx context.Context, req GetDocumentRequest) (*model.Document, error)
	UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (*model.Document, error)
	DeleteDocument(ctx context.Context, req DeleteDocumentRequest) error
	ListDocuments(ctx context.Context, req ListDocumentsRequest) ([]*model.Document, error)

	// Collection operations
	CreateCollection(ctx context.Context, req CreateCollectionRequest) (*model.Collection, error)
	GetCollection(ctx context.Context, req GetCollectionRequest) (*model.Collection, error)
	UpdateCollection(ctx context.Context, req UpdateCollectionRequest) error
	ListCollections(ctx context.Context, req ListCollectionsRequest) ([]*model.Collection, error)
	DeleteCollection(ctx context.Context, req DeleteCollectionRequest) error
	ListSubcollections(ctx context.Context, req ListSubcollectionsRequest) ([]model.Subcollection, error)

	// Index operations
	CreateIndex(ctx context.Context, req CreateIndexRequest) (*model.Index, error)
	DeleteIndex(ctx context.Context, req DeleteIndexRequest) error
	ListIndexes(ctx context.Context, req ListIndexesRequest) ([]model.Index, error)

	// Query operations
	QueryDocuments(ctx context.Context, req QueryRequest) ([]*model.Document, error)
	RunQuery(ctx context.Context, req QueryRequest) ([]*model.Document, error)

	// Batch operations
	RunBatchWrite(ctx context.Context, req BatchWriteRequest) (*model.BatchWriteResponse, error)

	// Transaction operations
	BeginTransaction(ctx context.Context, projectID string) (string, error)
	CommitTransaction(ctx context.Context, projectID string, transactionID string) error

	// Project operations
	CreateProject(ctx context.Context, req CreateProjectRequest) (*model.Project, error)
	GetProject(ctx context.Context, req GetProjectRequest) (*model.Project, error)
	UpdateProject(ctx context.Context, req UpdateProjectRequest) (*model.Project, error)
	DeleteProject(ctx context.Context, req DeleteProjectRequest) error
	ListProjects(ctx context.Context, req ListProjectsRequest) ([]*model.Project, error)

	// Database operations
	CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*model.Database, error)
	GetDatabase(ctx context.Context, req GetDatabaseRequest) (*model.Database, error)
	UpdateDatabase(ctx context.Context, req UpdateDatabaseRequest) (*model.Database, error)
	DeleteDatabase(ctx context.Context, req DeleteDatabaseRequest) error
	ListDatabases(ctx context.Context, req ListDatabasesRequest) ([]*model.Database, error)

	// Atomic operations
	AtomicIncrement(ctx context.Context, req AtomicIncrementRequest) (*AtomicIncrementResponse, error)
	AtomicArrayUnion(ctx context.Context, req AtomicArrayUnionRequest) error
	AtomicArrayRemove(ctx context.Context, req AtomicArrayRemoveRequest) error
	AtomicServerTimestamp(ctx context.Context, req AtomicServerTimestampRequest) error
}

// FirestoreUsecase implements Firestore business logic con arquitectura optimizada (colecciones dinámicas)
type FirestoreUsecase struct {
	firestoreRepo repository.FirestoreRepository // Debe ser el tenant-aware repo optimizado
	securityRepo  repository.SecurityRulesEngine
	queryEngine   repository.QueryEngine
	logger        logger.Logger
}

// NewFirestoreUsecase crea un nuevo FirestoreUsecase SOLO con arquitectura optimizada
func NewFirestoreUsecase(
	firestoreRepo repository.FirestoreRepository,
	securityRepo repository.SecurityRulesEngine,
	queryEngine repository.QueryEngine,
	logger logger.Logger,
) FirestoreUsecaseInterface {
	return &FirestoreUsecase{
		firestoreRepo: firestoreRepo,
		securityRepo:  securityRepo,
		queryEngine:   queryEngine,
		logger:        logger,
	}
}
