package mongodb

import (
	context "context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoQueryEngine implements repository.QueryEngine for MongoDB
// It translates Firestore queries to MongoDB queries in una forma minimalista y extensible.
type MongoQueryEngine struct {
	db *mongo.Database
}

// NewMongoQueryEngine creates a new MongoQueryEngine
func NewMongoQueryEngine(db *mongo.Database) *MongoQueryEngine {
	return &MongoQueryEngine{db: db}
}

// ExecuteQuery ejecuta una consulta Firestore sobre una colección MongoDB
func (qe *MongoQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	// Construir filtro principal y filtro de cursores Firestore
	filter := buildMongoFilter(query.Filters)
	cursorFilter := buildCursorFilter(query)
	if len(cursorFilter) > 0 {
		// Merge: $and entre filtro principal y filtro de cursores
		filter = bson.M{"$and": []bson.M{filter, cursorFilter}}
	}
	findOpts := buildMongoFindOptions(query)
	cur, err := qe.db.Collection(collectionPath).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []*model.Document
	for cur.Next(ctx) {
		var doc model.Document
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		docs = append(docs, &doc)
	}
	if query.LimitToLast && len(docs) > 0 {
		reverseDocs(docs)
		if query.Limit > 0 && len(docs) > int(query.Limit) {
			docs = docs[:query.Limit]
		}
	}
	return docs, nil
}

// buildMongoFilter soporta filtros compuestos y operadores avanzados
func buildMongoFilter(filters []model.Filter) bson.M {
	var andFilters []bson.M
	for _, f := range filters {
		if f.Composite == "or" && len(f.SubFilters) > 0 {
			var orFilters []bson.M
			for _, sub := range f.SubFilters {
				orFilters = append(orFilters, buildMongoFilter([]model.Filter{sub}))
			}
			andFilters = append(andFilters, bson.M{"$or": orFilters})
			continue
		}
		andFilters = append(andFilters, singleMongoFilter(f))
	}
	if len(andFilters) == 1 {
		return andFilters[0]
	}
	return bson.M{"$and": andFilters}
}

// singleMongoFilter traduce un filtro simple
func singleMongoFilter(f model.Filter) bson.M {
	switch f.Operator {
	case "==":
		return bson.M{f.Field: f.Value}
	case "!=":
		return bson.M{f.Field: bson.M{"$ne": f.Value}}
	case ">":
		return bson.M{f.Field: bson.M{"$gt": f.Value}}
	case ">=":
		return bson.M{f.Field: bson.M{"$gte": f.Value}}
	case "<":
		return bson.M{f.Field: bson.M{"$lt": f.Value}}
	case "<=":
		return bson.M{f.Field: bson.M{"$lte": f.Value}}
	case "in":
		return bson.M{f.Field: bson.M{"$in": f.Value}}
	case "not-in":
		return bson.M{f.Field: bson.M{"$nin": f.Value}}
	case "array-contains":
		return bson.M{f.Field: bson.M{"$elemMatch": bson.M{"$eq": f.Value}}}
	case "array-contains-any":
		return bson.M{f.Field: bson.M{"$in": f.Value}}
	default:
		return bson.M{f.Field: f.Value}
	}
}

// buildMongoFindOptions soporta proyecciones y ordenamientos
func buildMongoFindOptions(query model.Query) *options.FindOptions {
	opts := options.Find()
	if query.Limit > 0 {
		opts.SetLimit(int64(query.Limit))
	}
	if query.Offset > 0 {
		opts.SetSkip(int64(query.Offset))
	}
	if len(query.Orders) > 0 {
		sort := bson.D{}
		for _, o := range query.Orders {
			order := 1
			if o.Direction == "desc" {
				order = -1
			}
			sort = append(sort, bson.E{Key: o.Field, Value: order})
		}
		opts.SetSort(sort)
	}
	if len(query.SelectFields) > 0 {
		proj := bson.M{}
		for _, field := range query.SelectFields {
			proj[field] = 1
		}
		opts.SetProjection(proj)
	}
	return opts
}

// buildCursorFilter construye el filtro de cursores Firestore (multi-campo)
func buildCursorFilter(query model.Query) bson.M {
	if len(query.Orders) == 0 {
		return nil
	}
	var filters []bson.M
	fields := query.Orders
	// Soporte multi-campo como Firestore
	for i, order := range fields {
		field := order.Field
		orderDir := 1
		if order.Direction == "desc" {
			orderDir = -1
		}
		// startAt/startAfter
		if len(query.StartAt) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{field: bson.M{"$gte": query.StartAt[i]}})
			} else {
				filters = append(filters, bson.M{field: bson.M{"$lte": query.StartAt[i]}})
			}
		}
		if len(query.StartAfter) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{field: bson.M{"$gt": query.StartAfter[i]}})
			} else {
				filters = append(filters, bson.M{field: bson.M{"$lt": query.StartAfter[i]}})
			}
		}
		// endAt/endBefore
		if len(query.EndAt) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{field: bson.M{"$lte": query.EndAt[i]}})
			} else {
				filters = append(filters, bson.M{field: bson.M{"$gte": query.EndAt[i]}})
			}
		}
		if len(query.EndBefore) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{field: bson.M{"$lt": query.EndBefore[i]}})
			} else {
				filters = append(filters, bson.M{field: bson.M{"$gt": query.EndBefore[i]}})
			}
		}
	}
	if len(filters) == 0 {
		return nil
	}
	return bson.M{"$and": filters}
}

// reverseDocs invierte el slice de documentos (para LimitToLast)
func reverseDocs(docs []*model.Document) {
	n := len(docs)
	for i := 0; i < n/2; i++ {
		docs[i], docs[n-1-i] = docs[n-1-i], docs[i]
	}
}

// Asegúrate de que cumple la interfaz
var _ repository.QueryEngine = (*MongoQueryEngine)(nil)
