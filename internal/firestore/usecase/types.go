package usecase

import (
	"firestore-clone/internal/firestore/domain/model"
)

// Request/Response DTOs - Centralized type definitions

// Document operations
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

// Collection operations
type CreateCollectionRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
}

type GetCollectionRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
}

type UpdateCollectionRequest struct {
	ProjectID    string            `json:"projectId" validate:"required"`
	DatabaseID   string            `json:"databaseId" validate:"required"`
	CollectionID string            `json:"collectionId" validate:"required"`
	Collection   *model.Collection `json:"collection" validate:"required"`
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

// Index operations
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

// Query operations
type QueryRequest struct {
	ProjectID       string       `json:"projectId" validate:"required"`
	DatabaseID      string       `json:"databaseId" validate:"required"`
	StructuredQuery *model.Query `json:"structuredQuery,omitempty"`
	Parent          string       `json:"parent,omitempty"`
}

// Batch operations
type BatchWriteRequest struct {
	ProjectID  string                      `json:"projectId" validate:"required"`
	DatabaseID string                      `json:"databaseId" validate:"required"`
	Writes     []model.BatchWriteOperation `json:"writes" validate:"required"`
	Labels     map[string]string           `json:"labels,omitempty"`
}

// Project operations
type CreateProjectRequest struct {
	Project *model.Project `json:"project" validate:"required"`
}

type UpdateProjectRequest struct {
	Project *model.Project `json:"project" validate:"required"`
}

type DeleteProjectRequest struct {
	ProjectID string `json:"projectId" validate:"required"`
}

type GetProjectRequest struct {
	ProjectID string `json:"projectId" validate:"required"`
}

type ListProjectsRequest struct {
	OwnerEmail string `json:"ownerEmail,omitempty"`
}

// Database operations
type CreateDatabaseRequest struct {
	ProjectID string          `json:"projectId" validate:"required"`
	Database  *model.Database `json:"database" validate:"required"`
}

type UpdateDatabaseRequest struct {
	ProjectID string          `json:"projectId" validate:"required"`
	Database  *model.Database `json:"database" validate:"required"`
}

type DeleteDatabaseRequest struct {
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
}

type GetDatabaseRequest struct {
	ProjectID  string `json:"projectId" validate:"required"`
	DatabaseID string `json:"databaseId" validate:"required"`
}

type ListDatabasesRequest struct {
	ProjectID string `json:"projectId" validate:"required"`
}

// Atomic operations
type AtomicIncrementRequest struct {
	ProjectID    string      `json:"projectId" validate:"required"`
	DatabaseID   string      `json:"databaseId" validate:"required"`
	CollectionID string      `json:"collectionId" validate:"required"`
	DocumentID   string      `json:"documentId" validate:"required"`
	Field        string      `json:"field" validate:"required"`
	IncrementBy  interface{} `json:"incrementBy" validate:"required"`
}

type AtomicIncrementResponse struct {
	NewValue interface{} `json:"newValue"`
}

type AtomicArrayUnionRequest struct {
	ProjectID    string        `json:"projectId" validate:"required"`
	DatabaseID   string        `json:"databaseId" validate:"required"`
	CollectionID string        `json:"collectionId" validate:"required"`
	DocumentID   string        `json:"documentId" validate:"required"`
	Field        string        `json:"field" validate:"required"`
	Elements     []interface{} `json:"elements" validate:"required"`
}

type AtomicArrayRemoveRequest struct {
	ProjectID    string        `json:"projectId" validate:"required"`
	DatabaseID   string        `json:"databaseId" validate:"required"`
	CollectionID string        `json:"collectionId" validate:"required"`
	DocumentID   string        `json:"documentId" validate:"required"`
	Field        string        `json:"field" validate:"required"`
	Elements     []interface{} `json:"elements" validate:"required"`
}

type AtomicServerTimestampRequest struct {
	ProjectID    string `json:"projectId" validate:"required"`
	DatabaseID   string `json:"databaseId" validate:"required"`
	CollectionID string `json:"collectionId" validate:"required"`
	DocumentID   string `json:"documentId" validate:"required"`
	Field        string `json:"field" validate:"required"`
}
