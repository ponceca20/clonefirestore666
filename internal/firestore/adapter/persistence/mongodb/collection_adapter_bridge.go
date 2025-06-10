package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Adapter: CollectionInterface -> CollectionUpdater (for AtomicOperations) ---
type CollectionUpdaterAdapter struct {
	col CollectionInterface
}

func NewCollectionUpdaterAdapter(col CollectionInterface) *CollectionUpdaterAdapter {
	return &CollectionUpdaterAdapter{col: col}
}

func (a *CollectionUpdaterAdapter) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	res, err := a.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	return &mongo.UpdateResult{MatchedCount: res.Matched()}, nil
}

// --- Adapter: CollectionInterface -> IndexCollection (for IndexOperations) ---
type IndexCollectionAdapter struct {
	col CollectionInterface
}

func NewIndexCollectionAdapter(col CollectionInterface) *IndexCollectionAdapter {
	return &IndexCollectionAdapter{col: col}
}

func (a *IndexCollectionAdapter) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return a.col.CountDocuments(ctx, filter)
}
func (a *IndexCollectionAdapter) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return a.col.InsertOne(ctx, doc)
}
func (a *IndexCollectionAdapter) DeleteOne(ctx context.Context, filter interface{}) (DeleteResult, error) {
	res, err := a.col.DeleteOne(ctx, filter)
	if err != nil {
		return DeleteResult{}, err
	}
	return DeleteResult{DeletedCount: res.Deleted()}, nil
}
func (a *IndexCollectionAdapter) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResult, error) {
	_, err := a.col.UpdateOne(ctx, filter, update)
	return UpdateResult{}, err
}
func (a *IndexCollectionAdapter) Find(ctx context.Context, filter interface{}) (Cursor, error) {
	cur, err := a.col.Find(ctx, filter)
	return cur, err
}
func (a *IndexCollectionAdapter) FindOne(ctx context.Context, filter interface{}) SingleResult {
	return a.col.FindOne(ctx, filter)
}

// --- Adapter: CollectionInterface -> DocumentCollection (for IndexOperations) ---
type DocumentCollectionAdapter struct {
	col CollectionInterface
}

func NewDocumentCollectionAdapter(col CollectionInterface) *DocumentCollectionAdapter {
	return &DocumentCollectionAdapter{col: col}
}

func (a *DocumentCollectionAdapter) Indexes() IndexManager {
	// Not implemented: you may need to adapt this if you use index management
	return nil
}
func (a *DocumentCollectionAdapter) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return a.col.CountDocuments(ctx, filter)
}
