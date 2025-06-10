package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Hexagonal, shared interfaces for MongoDB collection abstraction ---
type CollectionInterface interface {
	CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)
	InsertOne(ctx context.Context, doc interface{}) (interface{}, error)
	FindOne(ctx context.Context, filter interface{}) SingleResultInterface
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error)
	DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error)
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error)
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error)
	ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error)
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface // Added for atomic update+get
}

type SingleResultInterface interface {
	Decode(v interface{}) error
}
type UpdateResultInterface interface{ Matched() int64 }
type DeleteResultInterface interface{ Deleted() int64 }
type CursorInterface interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Close(ctx context.Context) error
	Err() error
}

// Adapter to make *mongo.Collection compatible with CollectionInterface for production code

type MongoCollectionAdapter struct {
	col *mongo.Collection
}

func NewMongoCollectionAdapter(col *mongo.Collection) *MongoCollectionAdapter {
	return &MongoCollectionAdapter{col: col}
}

func (m *MongoCollectionAdapter) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return m.col.CountDocuments(ctx, filter, opts...)
}

func (m *MongoCollectionAdapter) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	res, err := m.col.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

func (m *MongoCollectionAdapter) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &MongoSingleResultAdapter{res: m.col.FindOne(ctx, filter)}
}

func (m *MongoCollectionAdapter) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	res, err := m.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	return &MongoUpdateResultAdapter{matched: res.MatchedCount}, nil
}

func (m *MongoCollectionAdapter) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	res, err := m.col.DeleteOne(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &MongoDeleteResultAdapter{deleted: res.DeletedCount}, nil
}

func (m *MongoCollectionAdapter) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	cur, err := m.col.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoCursorAdapter{cur: cur}, nil
}

func (m *MongoCollectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	cur, err := m.col.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoCursorAdapter{cur: cur}, nil
}

func (m *MongoCollectionAdapter) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	res, err := m.col.ReplaceOne(ctx, filter, replacement, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoUpdateResultAdapter{matched: res.MatchedCount}, nil
}

func (m *MongoCollectionAdapter) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &MongoSingleResultAdapter{res: m.col.FindOneAndUpdate(ctx, filter, update, opts...)}
}

// --- Adapters for result types ---
type MongoSingleResultAdapter struct {
	res *mongo.SingleResult
}

func (m *MongoSingleResultAdapter) Decode(v interface{}) error {
	return m.res.Decode(v)
}

// MongoUpdateResultAdapter wraps the matched count
type MongoUpdateResultAdapter struct {
	matched int64
}

func (m *MongoUpdateResultAdapter) Matched() int64 { return m.matched }

// MongoDeleteResultAdapter wraps the deleted count
type MongoDeleteResultAdapter struct {
	deleted int64
}

func (m *MongoDeleteResultAdapter) Deleted() int64 { return m.deleted }

type MongoCursorAdapter struct {
	cur *mongo.Cursor
}

func (m *MongoCursorAdapter) Next(ctx context.Context) bool   { return m.cur.Next(ctx) }
func (m *MongoCursorAdapter) Decode(val interface{}) error    { return m.cur.Decode(val) }
func (m *MongoCursorAdapter) Close(ctx context.Context) error { return m.cur.Close(ctx) }
func (m *MongoCursorAdapter) Err() error                      { return m.cur.Err() }

// Las interfaces hexagonales ya están aquí, no deben redeclararse en ningún otro archivo.
