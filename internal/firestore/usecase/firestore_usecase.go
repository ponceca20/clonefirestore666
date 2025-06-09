package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/firestore"
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

	// Batch operations
	RunBatchWrite(ctx context.Context, req BatchWriteRequest) (*model.BatchWriteResponse, error)

	// Collection operations
	CreateCollection(ctx context.Context, req CreateCollectionRequest) (*model.Collection, error)
	ListCollections(ctx context.Context, req ListCollectionsRequest) ([]*model.Collection, error)
	DeleteCollection(ctx context.Context, req DeleteCollectionRequest) error

	// Subcollection operations
	ListSubcollections(ctx context.Context, req ListSubcollectionsRequest) ([]model.Subcollection, error)

	// Index operations
	CreateIndex(ctx context.Context, req CreateIndexRequest) (*model.Index, error)
	DeleteIndex(ctx context.Context, req DeleteIndexRequest) error
	ListIndexes(ctx context.Context, req ListIndexesRequest) ([]model.Index, error)

	// Query operations
	RunQuery(ctx context.Context, req QueryRequest) ([]*model.Document, error)

	// Legacy operations (for HTTP API compatibility)
	CreateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
	GetDocumentLegacy(ctx context.Context, path string) (map[string]interface{}, error)
	UpdateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
	DeleteDocumentLegacy(ctx context.Context, path string) error
	ListDocumentsLegacy(ctx context.Context, path string, limit int, pageToken string) ([]*model.Document, string, error)
	RunQueryLegacy(ctx context.Context, projectID string, queryJSON string) ([]*model.Document, error)

	// Transaction operations
	BeginTransaction(ctx context.Context, projectID string) (string, error)
	CommitTransaction(ctx context.Context, projectID string, transactionID string) error
}

// Request/Response DTOs
type CreateDocumentRequest struct {
	ProjectID    string         `json:"projectId" validate:"required"`
	DatabaseID   string         `json:"databaseId" validate:"required"`
	CollectionID string         `json:"collectionId" validate:"required"`
	DocumentID   string         `json:"documentId,omitempty"`
	Data         map[string]any `json:"data" validate:"required"`
}

type GetDocumentRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
	DocumentID   string `json:"documentId" validate:"required"`
}

type UpdateDocumentRequest struct {
	ProjectID    string         `json:"projectId" validate:"required"`
	DatabaseID   string         `json:"databaseId" validate:"required"`
	CollectionID string         `json:"collectionId" validate:"required"`
	DocumentID   string         `json:"documentId" validate:"required"`
	Data         map[string]any `json:"data" validate:"required"`
	Mask         []string       `json:"mask,omitempty"`
}

type DeleteDocumentRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
	DocumentID   string `json:"documentId" validate:"required"`
}

type ListDocumentsRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
	PageSize     int32  `json:"pageSize,omitempty"`
	PageToken    string `json:"pageToken,omitempty"`
	OrderBy      string `json:"orderBy,omitempty"`
	ShowMissing  bool   `json:"showMissing,omitempty"`
}

type BatchWriteRequest struct {
	ProjectID  string                      `json:"projectId" validate:"required"`
	DatabaseID string                      `json:"databaseId" validate:"required"`
	Writes     []model.BatchWriteOperation `json:"writes" validate:"required"`
	Labels     map[string]string           `json:"labels,omitempty"`
}

type CreateCollectionRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
}

type ListCollectionsRequest struct {
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
	PageSize   int32  `json:"pageSize,omitempty"`
	PageToken  string `json:"pageToken,omitempty"`
}

type DeleteCollectionRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
}

type ListSubcollectionsRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
	DocumentID   string `json:"documentId" validate:"required"`
}

type CreateIndexRequest struct {
	ProjectID  string      `json:"projectId" validate:"required"`
	DatabaseID string      `json:"databaseId" validate:"required"`
	Index      model.Index `json:"index" validate:"required"`
}

type DeleteIndexRequest struct {
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
	IndexName  string `json:"indexName" validate:"required"`
}

type ListIndexesRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId,omitempty"`
}

type QueryRequest struct {
	ProjectID       string       `json:"projectId" validate:"required"`
	DatabaseID      string       `json:"databaseId" validate:"required"`
	StructuredQuery *model.Query `json:"structuredQuery,omitempty"`
	Parent          string       `json:"parent,omitempty"`
}

// FirestoreUsecase implements Firestore business logic
type FirestoreUsecase struct {
	firestoreRepo repository.FirestoreRepository
	securityRepo  repository.SecurityRulesEngine
	queryEngine   repository.QueryEngine
	logger        logger.Logger
}

// NewFirestoreUsecase creates a new Firestore usecase
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

// RunBatchWrite executes multiple write operations atomically
func (uc *FirestoreUsecase) RunBatchWrite(ctx context.Context, req BatchWriteRequest) (*model.BatchWriteResponse, error) {
	// Validate request
	if len(req.Writes) == 0 {
		return &model.BatchWriteResponse{}, nil
	}

	if len(req.Writes) > 500 { // Firestore limit
		return nil, fmt.Errorf("batch write operations cannot exceed 500 operations")
	}

	// Validate each operation
	for i, write := range req.Writes {
		if err := uc.validateBatchOperation(write); err != nil {
			return nil, fmt.Errorf("invalid operation at index %d: %w", i, err)
		}
	}

	// Convert BatchWriteOperation to WriteOperation
	writeOps := make([]*model.WriteOperation, len(req.Writes))
	for i, write := range req.Writes {
		writeOps[i] = &model.WriteOperation{
			Type: model.WriteOperationType(write.Type),
			Path: write.Path,
			Data: write.Data,
		}
	}

	// Execute batch write using repository method
	writeResults, err := uc.firestoreRepo.RunBatchWrite(ctx, req.ProjectID, req.DatabaseID, writeOps)
	if err != nil {
		return nil, err
	}

	// Convert WriteResult to BatchWriteResponse
	batchResponse := &model.BatchWriteResponse{
		WriteResults: make([]model.WriteResult, len(writeResults)),
		Status:       make([]model.Status, len(writeResults)),
	}

	for i, result := range writeResults {
		batchResponse.WriteResults[i] = model.WriteResult{
			UpdateTime: result.UpdateTime,
		}
		batchResponse.Status[i] = model.Status{
			Code:    0,
			Message: "OK",
		}
	}

	return batchResponse, nil
}

// ListSubcollections lists all subcollections under a document
func (uc *FirestoreUsecase) ListSubcollections(ctx context.Context, req ListSubcollectionsRequest) ([]model.Subcollection, error) { // Validate the document exists
	_, err := uc.getDocumentInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("parent document not found: %w", err)
	}
	// List subcollections
	subcollectionNames, err := uc.firestoreRepo.ListSubcollections(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
	if err != nil {
		return nil, err
	}

	// Convert []string to []model.Subcollection
	subcollections := make([]model.Subcollection, len(subcollectionNames))
	for i, name := range subcollectionNames {
		subcollections[i] = model.Subcollection{
			ID:   name,
			Path: fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s/%s", req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, name),
		}
	}

	return subcollections, nil
}

// CreateIndex crea un nuevo índice
func (uc *FirestoreUsecase) CreateIndex(ctx context.Context, req CreateIndexRequest) (*model.Index, error) {
	if err := uc.validateIndex(&req.Index); err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}
	collIdx := model.CollectionIndex{
		Name:   req.Index.Name,
		Fields: req.Index.Fields,
		State:  req.Index.State,
	}
	err := uc.firestoreRepo.CreateIndex(ctx, req.ProjectID, req.DatabaseID, req.Index.Collection, &collIdx)
	if err != nil {
		return nil, err
	}
	return &req.Index, nil
}

// DeleteIndex elimina un índice existente
func (uc *FirestoreUsecase) DeleteIndex(ctx context.Context, req DeleteIndexRequest) error {
	// Se requiere collectionID, pero no está en DeleteIndexRequest, así que lo dejamos vacío si no se usa
	return uc.firestoreRepo.DeleteIndex(ctx, req.ProjectID, req.DatabaseID, "", req.IndexName)
}

// ListIndexes lista todos los índices para una colección o base de datos
func (uc *FirestoreUsecase) ListIndexes(ctx context.Context, req ListIndexesRequest) ([]model.Index, error) {
	collIndexes, err := uc.firestoreRepo.ListIndexes(ctx, req.ProjectID, req.DatabaseID, req.CollectionID)
	if err != nil {
		return nil, err
	}
	indexes := make([]model.Index, len(collIndexes))
	for i, idx := range collIndexes {
		indexes[i] = model.Index{
			Name:   idx.Name,
			Fields: idx.Fields,
			State:  idx.State,
		}
	}
	return indexes, nil
}

// Project operations
func (uc *FirestoreUsecase) CreateProject(ctx context.Context, project *model.Project) (*model.Project, error) {
	uc.logger.Info("Creating new project", "projectID", project.ProjectID)

	err := uc.firestoreRepo.CreateProject(ctx, project)
	if err != nil {
		uc.logger.Error("Failed to create project", "error", err, "projectID", project.ProjectID)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	uc.logger.Info("Project created successfully", "projectID", project.ProjectID)
	return project, nil
}

func (uc *FirestoreUsecase) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	project, err := uc.firestoreRepo.GetProject(ctx, projectID)
	if err != nil {
		uc.logger.Error("Failed to get project", "error", err, "projectID", projectID)
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

func (uc *FirestoreUsecase) UpdateProject(ctx context.Context, project *model.Project) (*model.Project, error) {
	uc.logger.Info("Updating project", "projectID", project.ProjectID)

	err := uc.firestoreRepo.UpdateProject(ctx, project)
	if err != nil {
		uc.logger.Error("Failed to update project", "error", err, "projectID", project.ProjectID)
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

func (uc *FirestoreUsecase) DeleteProject(ctx context.Context, projectID string) error {
	uc.logger.Info("Deleting project", "projectID", projectID)

	err := uc.firestoreRepo.DeleteProject(ctx, projectID)
	if err != nil {
		uc.logger.Error("Failed to delete project", "error", err, "projectID", projectID)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

func (uc *FirestoreUsecase) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	projects, err := uc.firestoreRepo.ListProjects(ctx, ownerEmail)
	if err != nil {
		uc.logger.Error("Failed to list projects", "error", err, "ownerEmail", ownerEmail)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// Database operations
func (uc *FirestoreUsecase) CreateDatabase(ctx context.Context, projectID string, database *model.Database) (*model.Database, error) {
	uc.logger.Info("Creating new database", "projectID", projectID, "databaseID", database.DatabaseID)

	// Validate project exists
	_, err := uc.firestoreRepo.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project validation failed: %w", err)
	}

	err = uc.firestoreRepo.CreateDatabase(ctx, projectID, database)
	if err != nil {
		uc.logger.Error("Failed to create database", "error", err, "projectID", projectID, "databaseID", database.DatabaseID)
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return database, nil
}

func (uc *FirestoreUsecase) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	database, err := uc.firestoreRepo.GetDatabase(ctx, projectID, databaseID)
	if err != nil {
		uc.logger.Error("Failed to get database", "error", err, "projectID", projectID, "databaseID", databaseID)
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return database, nil
}

func (uc *FirestoreUsecase) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) (*model.Database, error) {
	uc.logger.Info("Updating database", "projectID", projectID, "databaseID", database.DatabaseID)

	err := uc.firestoreRepo.UpdateDatabase(ctx, projectID, database)
	if err != nil {
		uc.logger.Error("Failed to update database", "error", err, "projectID", projectID, "databaseID", database.DatabaseID)
		return nil, fmt.Errorf("failed to update database: %w", err)
	}

	return database, nil
}

func (uc *FirestoreUsecase) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	uc.logger.Info("Deleting database", "projectID", projectID, "databaseID", databaseID)

	err := uc.firestoreRepo.DeleteDatabase(ctx, projectID, databaseID)
	if err != nil {
		uc.logger.Error("Failed to delete database", "error", err, "projectID", projectID, "databaseID", databaseID)
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

func (uc *FirestoreUsecase) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	databases, err := uc.firestoreRepo.ListDatabases(ctx, projectID)
	if err != nil {
		uc.logger.Error("Failed to list databases", "error", err, "projectID", projectID)
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	return databases, nil
}

// Collection operations - Internal methods with parameter-based signatures
func (uc *FirestoreUsecase) createCollectionInternal(ctx context.Context, projectID, databaseID string, collection *model.Collection) (*model.Collection, error) {
	uc.logger.Info("Creating new collection", "projectID", projectID, "databaseID", databaseID, "collectionID", collection.CollectionID)

	// Validate hierarchy
	if err := uc.validateFirestoreHierarchy(ctx, projectID, databaseID, ""); err != nil {
		return nil, err
	}

	err := uc.firestoreRepo.CreateCollection(ctx, projectID, databaseID, collection)
	if err != nil {
		uc.logger.Error("Failed to create collection", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collection.CollectionID)
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return collection, nil
}

func (uc *FirestoreUsecase) getCollectionInternal(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	collection, err := uc.firestoreRepo.GetCollection(ctx, projectID, databaseID, collectionID)
	if err != nil {
		uc.logger.Error("Failed to get collection", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID)
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	return collection, nil
}

func (uc *FirestoreUsecase) updateCollectionInternal(ctx context.Context, projectID, databaseID string, collection *model.Collection) (*model.Collection, error) {
	uc.logger.Info("Updating collection", "projectID", projectID, "databaseID", databaseID, "collectionID", collection.CollectionID)

	err := uc.firestoreRepo.UpdateCollection(ctx, projectID, databaseID, collection)
	if err != nil {
		uc.logger.Error("Failed to update collection", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collection.CollectionID)
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	return collection, nil
}

func (uc *FirestoreUsecase) deleteCollectionInternal(ctx context.Context, projectID, databaseID, collectionID string) error {
	uc.logger.Info("Deleting collection", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID)

	err := uc.firestoreRepo.DeleteCollection(ctx, projectID, databaseID, collectionID)
	if err != nil {
		uc.logger.Error("Failed to delete collection", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID)
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

func (uc *FirestoreUsecase) listCollectionsInternal(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	collections, err := uc.firestoreRepo.ListCollections(ctx, projectID, databaseID)
	if err != nil {
		uc.logger.Error("Failed to list collections", "error", err, "projectID", projectID, "databaseID", databaseID)
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return collections, nil
}

// Document operations - Internal methods with parameter-based signatures
func (uc *FirestoreUsecase) getDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	document, err := uc.firestoreRepo.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		uc.logger.Error("Failed to get document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return document, nil
}

func (uc *FirestoreUsecase) createDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	uc.logger.Info("Creating document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	// Validate hierarchy
	if err := uc.validateFirestoreHierarchy(ctx, projectID, databaseID, collectionID); err != nil {
		return nil, err
	}

	document, err := uc.firestoreRepo.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
	if err != nil {
		uc.logger.Error("Failed to create document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	uc.logger.Info("Document created successfully", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
	return document, nil
}

func (uc *FirestoreUsecase) updateDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	uc.logger.Info("Updating document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	document, err := uc.firestoreRepo.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
	if err != nil {
		uc.logger.Error("Failed to update document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return document, nil
}

func (uc *FirestoreUsecase) setDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	uc.logger.Info("Setting document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID, "merge", merge)

	// Validate hierarchy
	if err := uc.validateFirestoreHierarchy(ctx, projectID, databaseID, collectionID); err != nil {
		return nil, err
	}

	document, err := uc.firestoreRepo.SetDocument(ctx, projectID, databaseID, collectionID, documentID, data, merge)
	if err != nil {
		uc.logger.Error("Failed to set document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to set document: %w", err)
	}

	return document, nil
}

func (uc *FirestoreUsecase) deleteDocumentInternal(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	uc.logger.Info("Deleting document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	err := uc.firestoreRepo.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		uc.logger.Error("Failed to delete document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Document path-based operations (for compatibility)
func (uc *FirestoreUsecase) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	return uc.getDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID)
}

func (uc *FirestoreUsecase) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	return uc.createDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, data)
}

func (uc *FirestoreUsecase) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	// Extract document components from path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}

	collectionID := segments[0]
	documentID := segments[1]

	return uc.updateDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, data, updateMask)
}

func (uc *FirestoreUsecase) DeleteDocumentByPath(ctx context.Context, path string) error {
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

	return uc.deleteDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID)
}

// ListDocumentsLegacy lists documents in a collection using legacy path format
func (uc *FirestoreUsecase) ListDocumentsLegacy(ctx context.Context, path string, limit int, pageToken string) ([]*model.Document, string, error) {
	uc.logger.Info("Listing documents via legacy API", "path", path, "limit", limit)

	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, "", fmt.Errorf("invalid path format: %w", err)
	}

	// Extract collection ID from document path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 1 {
		return nil, "", fmt.Errorf("invalid collection path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]

	// Use the repository's ListDocuments method
	docs, nextPageToken, err := uc.firestoreRepo.ListDocuments(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, int32(limit), pageToken, "", false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list documents: %w", err)
	}

	return docs, nextPageToken, nil
}

// RunQueryLegacy executes a structured query using legacy JSON format
func (uc *FirestoreUsecase) RunQueryLegacy(ctx context.Context, projectID string, queryJSON string) ([]*model.Document, error) {
	uc.logger.Info("Running query via legacy API", "projectID", projectID)

	// Parse the JSON query
	var queryReq map[string]interface{}
	if err := json.Unmarshal([]byte(queryJSON), &queryReq); err != nil {
		return nil, fmt.Errorf("invalid query JSON: %w", err)
	}

	// Extract parent path to determine collection
	parent, ok := queryReq["parent"].(string)
	if !ok {
		return nil, fmt.Errorf("parent field is required in query")
	}

	// Parse parent path to extract collection info
	pathInfo, err := firestore.ParseFirestorePath(parent)
	if err != nil {
		return nil, fmt.Errorf("invalid parent path format: %w", err)
	}

	// Extract collection ID from document path
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 1 {
		return nil, fmt.Errorf("invalid parent path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]
	// Create a basic query structure
	query := &model.Query{
		CollectionID: collectionID,
		Path:         parent,
	}

	// Execute the query using the repository
	docs, err := uc.firestoreRepo.RunQuery(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	return docs, nil
}

// BeginTransaction starts a new transaction
func (uc *FirestoreUsecase) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	uc.logger.Info("Beginning transaction", "projectID", projectID)

	// Generate a unique transaction ID
	transactionID := fmt.Sprintf("tx_%d_%s", time.Now().UnixNano(), projectID)

	// For now, we'll return the transaction ID
	// In a full implementation, this would interact with the repository to create a transaction context
	uc.logger.Info("Transaction started", "transactionID", transactionID)

	return transactionID, nil
}

// CommitTransaction commits an existing transaction
func (uc *FirestoreUsecase) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	uc.logger.Info("Committing transaction", "projectID", projectID, "transactionID", transactionID)

	// Validate transaction ID format
	if !strings.HasPrefix(transactionID, "tx_") {
		return fmt.Errorf("invalid transaction ID format: %s", transactionID)
	}

	// For now, we'll just log the commit
	// In a full implementation, this would interact with the repository to commit the transaction
	uc.logger.Info("Transaction committed", "transactionID", transactionID)

	return nil
}

// Legacy methods for backward compatibility
func (uc *FirestoreUsecase) CreateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	uc.logger.Info("Creating document via legacy API", "path", path)

	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]
	documentID := segments[1]

	fieldData := make(map[string]*model.FieldValue)
	for key, value := range data {
		fieldData[key] = model.NewFieldValue(value)
	}

	doc, err := uc.createDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, fieldData)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["name"] = doc.Path
	result["createTime"] = doc.CreateTime.Format(time.RFC3339)
	result["updateTime"] = doc.UpdateTime.Format(time.RFC3339)

	fields := make(map[string]interface{})
	for key, fieldValue := range doc.Fields {
		fields[key] = fieldValue.Value
	}
	result["fields"] = fields

	return result, nil
}

func (uc *FirestoreUsecase) GetDocumentLegacy(ctx context.Context, path string) (map[string]interface{}, error) {
	uc.logger.Info("Getting document via legacy API", "path", path)

	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]
	documentID := segments[1]

	doc, err := uc.getDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["name"] = doc.Path
	result["createTime"] = doc.CreateTime.Format(time.RFC3339)
	result["updateTime"] = doc.UpdateTime.Format(time.RFC3339)

	fields := make(map[string]interface{})
	for key, fieldValue := range doc.Fields {
		fields[key] = fieldValue.Value
	}
	result["fields"] = fields

	return result, nil
}

func (uc *FirestoreUsecase) UpdateDocumentLegacy(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	uc.logger.Info("Updating document via legacy API", "path", path)

	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]
	documentID := segments[1]

	fieldData := make(map[string]*model.FieldValue)
	for key, value := range data {
		fieldData[key] = model.NewFieldValue(value)
	}
	doc, err := uc.updateDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID, fieldData, []string{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["name"] = doc.Path
	result["createTime"] = doc.CreateTime.Format(time.RFC3339)
	result["updateTime"] = doc.UpdateTime.Format(time.RFC3339)

	fields := make(map[string]interface{})
	for key, fieldValue := range doc.Fields {
		fields[key] = fieldValue.Value
	}
	result["fields"] = fields

	return result, nil
}

func (uc *FirestoreUsecase) DeleteDocumentLegacy(ctx context.Context, path string) error {
	uc.logger.Info("Deleting document via legacy API", "path", path)

	pathInfo, err := firestore.ParseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid path format: %w", err)
	}
	segments := strings.Split(pathInfo.DocumentPath, "/")
	if len(segments) < 2 {
		return fmt.Errorf("invalid document path: %s", pathInfo.DocumentPath)
	}
	collectionID := segments[0]
	documentID := segments[1]

	return uc.deleteDocumentInternal(ctx, pathInfo.ProjectID, pathInfo.DatabaseID, collectionID, documentID)
}

// validateBatchOperation validates a single batch operation
func (uc *FirestoreUsecase) validateBatchOperation(op model.BatchWriteOperation) error {
	if op.DocumentID == "" {
		return errors.New("document ID is required")
	}

	if op.Path == "" {
		return errors.New("document path is required")
	}

	switch op.Type {
	case model.BatchOperationTypeCreate, model.BatchOperationTypeSet:
		if op.Data == nil {
			return errors.New("data is required for create/set operations")
		}
	case model.BatchOperationTypeUpdate:
		if op.Data == nil {
			return errors.New("data is required for update operations")
		}
	case model.BatchOperationTypeDelete:
		// No additional validation needed
	default:
		return fmt.Errorf("unsupported operation type: %s", op.Type)
	}

	return nil
}

// validateIndex validates index configuration
func (uc *FirestoreUsecase) validateIndex(index *model.Index) error {
	if index.Name == "" {
		return errors.New("index name is required")
	}

	if index.Collection == "" {
		return errors.New("collection is required")
	}

	if len(index.Fields) == 0 {
		return errors.New("at least one field is required")
	}

	if len(index.Fields) > 100 { // Firestore limit
		return errors.New("index cannot have more than 100 fields")
	}

	for i, field := range index.Fields {
		if field.Path == "" {
			return fmt.Errorf("field path is required at index %d", i)
		}

		if field.Order != model.IndexFieldOrderAscending && field.Order != model.IndexFieldOrderDescending {
			return fmt.Errorf("invalid field order at index %d", i)
		}
	}

	return nil
}

// validateFirestoreHierarchy validates that the project, database, and optionally collection exist
func (uc *FirestoreUsecase) validateFirestoreHierarchy(ctx context.Context, projectID, databaseID, collectionID string) error {
	// Validate project exists
	if _, err := uc.firestoreRepo.GetProject(ctx, projectID); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Validate database exists
	if _, err := uc.firestoreRepo.GetDatabase(ctx, projectID, databaseID); err != nil {
		return fmt.Errorf("database not found: %w", err)
	}

	// Validate collection exists if provided
	if collectionID != "" {
		if _, err := uc.firestoreRepo.GetCollection(ctx, projectID, databaseID, collectionID); err != nil {
			return fmt.Errorf("collection not found: %w", err)
		}
	}

	return nil
}

// Interface-compliant methods using request DTOs

// CreateDocument implements the interface method using CreateDocumentRequest
func (uc *FirestoreUsecase) CreateDocument(ctx context.Context, req CreateDocumentRequest) (*model.Document, error) {
	// Convert request fields to FieldValue map
	fields := make(map[string]*model.FieldValue)
	for key, value := range req.Data {
		fields[key] = model.NewFieldValue(value)
	}

	return uc.createDocumentInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, fields)
}

// GetDocument implements the interface method using GetDocumentRequest
func (uc *FirestoreUsecase) GetDocument(ctx context.Context, req GetDocumentRequest) (*model.Document, error) {
	return uc.getDocumentInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
}

// UpdateDocument implements the interface method using UpdateDocumentRequest
func (uc *FirestoreUsecase) UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (*model.Document, error) {
	// Convert request fields to FieldValue map
	fields := make(map[string]*model.FieldValue)
	for key, value := range req.Data {
		fields[key] = model.NewFieldValue(value)
	}

	return uc.updateDocumentInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID, fields, req.Mask)
}

// DeleteDocument implements the interface method using DeleteDocumentRequest
func (uc *FirestoreUsecase) DeleteDocument(ctx context.Context, req DeleteDocumentRequest) error {
	return uc.deleteDocumentInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.DocumentID)
}

// ListDocuments implements the interface method using ListDocumentsRequest
func (uc *FirestoreUsecase) ListDocuments(ctx context.Context, req ListDocumentsRequest) ([]*model.Document, error) {
	docs, _, err := uc.firestoreRepo.ListDocuments(ctx, req.ProjectID, req.DatabaseID, req.CollectionID, req.PageSize, req.PageToken, req.OrderBy, req.ShowMissing)
	return docs, err
}

// CreateCollection implements the interface method using CreateCollectionRequest
func (uc *FirestoreUsecase) CreateCollection(ctx context.Context, req CreateCollectionRequest) (*model.Collection, error) {
	collection := model.NewCollection(req.ProjectID, req.DatabaseID, req.CollectionID)
	return uc.createCollectionInternal(ctx, req.ProjectID, req.DatabaseID, collection)
}

// ListCollections implements the interface method using ListCollectionsRequest
func (uc *FirestoreUsecase) ListCollections(ctx context.Context, req ListCollectionsRequest) ([]*model.Collection, error) {
	return uc.listCollectionsInternal(ctx, req.ProjectID, req.DatabaseID)
}

// DeleteCollection implements the interface method using DeleteCollectionRequest
func (uc *FirestoreUsecase) DeleteCollection(ctx context.Context, req DeleteCollectionRequest) error {
	return uc.deleteCollectionInternal(ctx, req.ProjectID, req.DatabaseID, req.CollectionID)
}

// RunQuery implements the interface method using QueryRequest
func (uc *FirestoreUsecase) RunQuery(ctx context.Context, req QueryRequest) ([]*model.Document, error) {
	if req.StructuredQuery == nil {
		return nil, fmt.Errorf("structured query is required")
	}

	// Extract collection ID from parent path or use structured query info
	var collectionID string
	if req.Parent != "" {
		// Parse parent path to extract collection
		pathInfo, err := firestore.ParseFirestorePath(req.Parent)
		if err != nil {
			return nil, fmt.Errorf("invalid parent path: %w", err)
		}
		segments := strings.Split(pathInfo.DocumentPath, "/")
		if len(segments) > 0 {
			collectionID = segments[0]
		}
	}

	return uc.firestoreRepo.RunQuery(ctx, req.ProjectID, req.DatabaseID, collectionID, req.StructuredQuery)
}

// Rename existing methods to avoid conflicts

// createDocument is the internal method for creating documents
func (uc *FirestoreUsecase) createDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	uc.logger.Info("Creating document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	// Validate hierarchy
	if err := uc.validateFirestoreHierarchy(ctx, projectID, databaseID, collectionID); err != nil {
		return nil, err
	}

	document, err := uc.firestoreRepo.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
	if err != nil {
		uc.logger.Error("Failed to create document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	uc.logger.Info("Document created successfully", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
	return document, nil
}

// getDocument is the internal method for getting documents
func (uc *FirestoreUsecase) getDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	document, err := uc.firestoreRepo.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		uc.logger.Error("Failed to get document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return document, nil
}

// updateDocument is the internal method for updating documents
func (uc *FirestoreUsecase) updateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	uc.logger.Info("Updating document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	document, err := uc.firestoreRepo.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
	if err != nil {
		uc.logger.Error("Failed to update document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return document, nil
}

// deleteDocument is the internal method for deleting documents
func (uc *FirestoreUsecase) deleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	uc.logger.Info("Deleting document", "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)

	err := uc.firestoreRepo.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
	if err != nil {
		uc.logger.Error("Failed to delete document", "error", err, "projectID", projectID, "databaseID", databaseID, "collectionID", collectionID, "documentID", documentID)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}
