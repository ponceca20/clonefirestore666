package repository

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
)

// FirestoreRepository defines the interface for Firestore data persistence.
type FirestoreRepository interface {
	// Document methods
	GetDocument(ctx context.Context, path string) (*model.Document, error)
	CreateDocument(ctx context.Context, path string, data map[string]interface{}) (*model.Document, error)
	UpdateDocument(ctx context.Context, path string, data map[string]interface{}) (*model.Document, error)
	DeleteDocument(ctx context.Context, path string) error
	ListDocuments(ctx context.Context, parentPath string, query model.Query) ([]*model.Document, error) // Simplified for now

	// Batch operations
	RunTransaction(ctx context.Context, fn func(tx Transaction) error) error
	RunBatch(ctx context.Context, operations []model.WriteOperation) error // WriteOperation to be defined in model

	// Atomic operations (Field Transforms)
	Increment(ctx context.Context, path string, field string, value int64) error
	ArrayUnion(ctx context.Context, path string, field string, elements []interface{}) error
	ArrayRemove(ctx context.Context, path string, field string, elements []interface{}) error
}

// Transaction defines the interface for operations within a transaction.
// This allows for atomic reads and writes.
type Transaction interface {
	Get(path string) (*model.Document, error)
	Create(path string, data map[string]interface{}) error
	Update(path string, data map[string]interface{}) error
	Delete(path string) error
	// TODO: Consider adding Set (Create or Update)
}
