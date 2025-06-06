package repository

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
)

// QueryEngine defines the interface for executing queries against a collection.
type QueryEngine interface {
	ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error)
	// This might eventually need to stream results for large datasets
}
