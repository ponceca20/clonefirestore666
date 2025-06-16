package mongodb

import (
	context "context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"fmt"
	"log"
	"time"

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
	log.Printf("[MongoQueryEngine] Ejecutando consulta: collectionPath=%s, query=%+v", collectionPath, query)
	// Construir filtro principal y filtro de cursores Firestore
	filter := buildMongoFilter(query.Filters)
	log.Printf("[MongoQueryEngine] Filtro MongoDB generado: %+v", filter)
	cursorFilter := buildCursorFilter(query)
	if len(cursorFilter) > 0 {
		log.Printf("[MongoQueryEngine] CursorFilter generado: %+v", cursorFilter)
		// Merge: $and entre filtro principal y filtro de cursores
		filter = mergeFiltersWithAnd(filter, cursorFilter)
		log.Printf("[MongoQueryEngine] Filtro final después de merge: %+v", filter)
	}
	findOpts := buildMongoFindOptions(query)
	log.Printf("[MongoQueryEngine] FindOptions: %+v", findOpts)
	cur, err := qe.db.Collection(collectionPath).Find(ctx, filter, findOpts)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error en Find: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []*model.Document
	for cur.Next(ctx) {
		var mongoDoc MongoDocumentFlat
		if err := cur.Decode(&mongoDoc); err != nil {
			log.Printf("[MongoQueryEngine] Error decodificando documento: %v", err)
			continue
		}
		docs = append(docs, mongoFlatToModelDocument(&mongoDoc))
	}
	if query.LimitToLast && len(docs) > 0 {
		reverseDocs(docs)
		if query.Limit > 0 && len(docs) > int(query.Limit) {
			docs = docs[:query.Limit]
		}
	}
	log.Printf("[MongoQueryEngine] Documentos encontrados: %d", len(docs))
	return docs, nil
}

// mergeFiltersWithAnd safely merges two filters with $and, handling existing $and conditions
func mergeFiltersWithAnd(filter1, filter2 bson.M) bson.M {
	// If filter1 is empty, return filter2
	if len(filter1) == 0 {
		return filter2
	}
	// If filter2 is empty, return filter1
	if len(filter2) == 0 {
		return filter1
	}

	// Check if both filters have $and - merge all elements into a single $and
	if existingAnd1, exists1 := filter1["$and"]; exists1 {
		if existingAnd2, exists2 := filter2["$and"]; exists2 {
			if andArray1, ok1 := existingAnd1.([]bson.M); ok1 {
				if andArray2, ok2 := existingAnd2.([]bson.M); ok2 {
					// Both have $and, merge all elements
					return bson.M{"$and": append(andArray1, andArray2...)}
				}
			}
		}
	}

	// Check if filter1 already contains $and
	if existingAnd, exists := filter1["$and"]; exists {
		if andArray, ok := existingAnd.([]bson.M); ok {
			// filter1 already has $and, append filter2 to the array
			return bson.M{"$and": append(andArray, filter2)}
		}
	}

	// Check if filter2 contains $and
	if existingAnd, exists := filter2["$and"]; exists {
		if andArray, ok := existingAnd.([]bson.M); ok {
			// filter2 already has $and, prepend filter1 to the array
			return bson.M{"$and": append([]bson.M{filter1}, andArray...)}
		}
	}

	// Neither filter has $and, create new $and with both
	return bson.M{"$and": []bson.M{filter1, filter2}}
}

// buildMongoFilter soporta filtros compuestos y operadores avanzados
func buildMongoFilter(filters []model.Filter) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}

	var andFilters []bson.M
	for _, f := range filters {
		if f.Composite == "or" && len(f.SubFilters) > 0 {
			var orFilters []bson.M
			for _, sub := range f.SubFilters {
				subFilter := singleMongoFilter(sub)
				if len(subFilter) > 0 {
					orFilters = append(orFilters, subFilter)
				}
			}
			if len(orFilters) > 0 {
				andFilters = append(andFilters, bson.M{"$or": orFilters})
			}
		} else if f.Composite == "and" && len(f.SubFilters) > 0 {
			// Handle AND composite filters by recursively processing sub-filters
			subAndFilter := buildMongoFilter(f.SubFilters)
			if len(subAndFilter) > 0 {
				andFilters = append(andFilters, subAndFilter)
			}
		} else if f.Composite == "" {
			// Handle regular (non-composite) filters
			singleFilter := singleMongoFilter(f)
			if len(singleFilter) > 0 {
				andFilters = append(andFilters, singleFilter)
			}
		}
	}

	if len(andFilters) == 0 {
		return bson.M{}
	}
	if len(andFilters) == 1 {
		return andFilters[0]
	}
	return bson.M{"$and": andFilters}
}

// Helper to extract primitive value from Firestore-typed value
func extractPrimitiveValue(val interface{}) interface{} {
	if m, ok := val.(map[string]interface{}); ok {
		for _, v := range m {
			return v // Return the first value (e.g., booleanValue, stringValue, etc.)
		}
	}
	return val
}

// determineFieldPath determines the correct MongoDB field path based on the value type and operator
func determineFieldPath(fieldName string, value interface{}, operator string) string {
	// For array operations, we know the field is an array
	if operator == "array-contains" || operator == "array-contains-any" {
		return fmt.Sprintf("fields.%s.arrayValue", fieldName)
	}

	// Determine the Firestore value type based on the Go type
	switch v := value.(type) {
	case string:
		// Check if this looks like a timestamp
		parser := model.NewTimestampParser()
		if parser.IsTimestampString(v) {
			return fmt.Sprintf("fields.%s.timestampValue", fieldName)
		}
		return fmt.Sprintf("fields.%s.stringValue", fieldName)
	case bool:
		return fmt.Sprintf("fields.%s.booleanValue", fieldName)
	case int, int32, int64:
		return fmt.Sprintf("fields.%s.integerValue", fieldName)
	case float32, float64:
		return fmt.Sprintf("fields.%s.doubleValue", fieldName)
	case time.Time:
		return fmt.Sprintf("fields.%s.timestampValue", fieldName)
	case nil:
		return fmt.Sprintf("fields.%s.nullValue", fieldName)
	default:
		// For arrays, maps, and other complex types, we might need more sophisticated handling
		// For now, default to checking for stringValue as fallback
		return fmt.Sprintf("fields.%s.stringValue", fieldName)
	}
}

// singleMongoFilter traduce un filtro simple
func singleMongoFilter(f model.Filter) bson.M {
	primitiveValue := extractPrimitiveValue(f.Value)

	// Para campos de documento Firestore, necesitamos determinar el tipo correcto del valor
	// y apuntar al campo específico: fields.{campo}.{tipoValue}
	fieldPath := determineFieldPath(f.Field, primitiveValue, string(f.Operator))

	log.Printf("[MongoQueryEngine] Traduciendo filtro: field=%s -> fieldPath=%s, operator=%s, value=%v",
		f.Field, fieldPath, f.Operator, primitiveValue)

	switch f.Operator {
	case model.OperatorEqual:
		return bson.M{fieldPath: primitiveValue}
	case model.OperatorNotEqual:
		return bson.M{fieldPath: bson.M{"$ne": primitiveValue}}
	case model.OperatorGreaterThan:
		return bson.M{fieldPath: bson.M{"$gt": primitiveValue}}
	case model.OperatorGreaterThanOrEqual:
		return bson.M{fieldPath: bson.M{"$gte": primitiveValue}}
	case model.OperatorLessThan:
		return bson.M{fieldPath: bson.M{"$lt": primitiveValue}}
	case model.OperatorLessThanOrEqual:
		return bson.M{fieldPath: bson.M{"$lte": primitiveValue}}
	case model.OperatorIn:
		return bson.M{fieldPath: bson.M{"$in": primitiveValue}}
	case model.OperatorNotIn:
		return bson.M{fieldPath: bson.M{"$nin": primitiveValue}}
	case model.OperatorArrayContains:
		return bson.M{fieldPath: bson.M{"$elemMatch": bson.M{"$eq": primitiveValue}}}
	case model.OperatorArrayContainsAny:
		return bson.M{fieldPath: bson.M{"$in": primitiveValue}}
	default:
		return bson.M{fieldPath: primitiveValue}
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
			// Para ordenamientos, usar por defecto stringValue si no hay más información
			// TODO: Mejorar esto para determinar el tipo correcto basado en metadatos del esquema
			fieldPath := fmt.Sprintf("fields.%s.stringValue", o.Field)
			sort = append(sort, bson.E{Key: fieldPath, Value: order})
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
		// Para cursores, necesitamos determinar el tipo basado en el valor
		var fieldPath string
		if len(query.StartAt) > i {
			fieldPath = determineFieldPath(order.Field, query.StartAt[i], ">=")
		} else if len(query.StartAfter) > i {
			fieldPath = determineFieldPath(order.Field, query.StartAfter[i], ">")
		} else if len(query.EndAt) > i {
			fieldPath = determineFieldPath(order.Field, query.EndAt[i], "<=")
		} else if len(query.EndBefore) > i {
			fieldPath = determineFieldPath(order.Field, query.EndBefore[i], "<")
		} else {
			// Default a stringValue si no hay información de tipo
			fieldPath = fmt.Sprintf("fields.%s.stringValue", order.Field)
		}

		orderDir := 1
		if order.Direction == "desc" {
			orderDir = -1
		}
		// startAt/startAfter
		if len(query.StartAt) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{fieldPath: bson.M{"$gte": query.StartAt[i]}})
			} else {
				filters = append(filters, bson.M{fieldPath: bson.M{"$lte": query.StartAt[i]}})
			}
		}
		if len(query.StartAfter) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{fieldPath: bson.M{"$gt": query.StartAfter[i]}})
			} else {
				filters = append(filters, bson.M{fieldPath: bson.M{"$lt": query.StartAfter[i]}})
			}
		}
		// endAt/endBefore
		if len(query.EndAt) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{fieldPath: bson.M{"$lte": query.EndAt[i]}})
			} else {
				filters = append(filters, bson.M{fieldPath: bson.M{"$gte": query.EndAt[i]}})
			}
		}
		if len(query.EndBefore) > i {
			if orderDir == 1 {
				filters = append(filters, bson.M{fieldPath: bson.M{"$lt": query.EndBefore[i]}})
			} else {
				filters = append(filters, bson.M{fieldPath: bson.M{"$gt": query.EndBefore[i]}})
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

// buildFieldFilter creates a MongoDB filter for a single field (used for testing)
func buildFieldFilter(field string, operator model.Operator, value interface{}) bson.M {
	filter := model.Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
	return singleMongoFilter(filter)
}

// buildSimpleFieldFilter creates a MongoDB filter without Firestore field paths (used for testing)
func buildSimpleFieldFilter(field string, operator model.Operator, value interface{}) bson.M {
	primitiveValue := extractPrimitiveValue(value)

	switch operator {
	case "==":
		return bson.M{field: primitiveValue}
	case "!=":
		return bson.M{field: bson.M{"$ne": primitiveValue}}
	case ">":
		return bson.M{field: bson.M{"$gt": primitiveValue}}
	case ">=":
		return bson.M{field: bson.M{"$gte": primitiveValue}}
	case "<":
		return bson.M{field: bson.M{"$lt": primitiveValue}}
	case "<=":
		return bson.M{field: bson.M{"$lte": primitiveValue}}
	case "in":
		return bson.M{field: bson.M{"$in": primitiveValue}}
	case "not-in":
		return bson.M{field: bson.M{"$nin": primitiveValue}}
	case "array-contains":
		return bson.M{field: primitiveValue}
	case "array-contains-any":
		return bson.M{field: bson.M{"$in": primitiveValue}}
	default:
		return bson.M{field: primitiveValue}
	}
}

// buildSimpleMongoFilter creates MongoDB filters without Firestore field paths (used for testing)
func buildSimpleMongoFilter(filters []model.Filter) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}

	var andFilters []bson.M
	for _, f := range filters {
		if f.Composite == "or" && len(f.SubFilters) > 0 {
			var orFilters []bson.M
			for _, sub := range f.SubFilters {
				subFilter := buildSimpleFieldFilter(sub.Field, sub.Operator, sub.Value)
				if len(subFilter) > 0 {
					orFilters = append(orFilters, subFilter)
				}
			}
			if len(orFilters) > 0 {
				andFilters = append(andFilters, bson.M{"$or": orFilters})
			}
			continue
		}
		if f.Composite == "and" && len(f.SubFilters) > 0 {
			// Handle AND composite filters
			for _, sub := range f.SubFilters {
				if sub.Composite == "or" && len(sub.SubFilters) > 0 {
					// Handle nested OR filters within the AND
					var orFilters []bson.M
					for _, orSub := range sub.SubFilters {
						subFilter := buildSimpleFieldFilter(orSub.Field, orSub.Operator, orSub.Value)
						if len(subFilter) > 0 {
							orFilters = append(orFilters, subFilter)
						}
					}
					if len(orFilters) > 0 {
						andFilters = append(andFilters, bson.M{"$or": orFilters})
					}
				} else if sub.Composite == "" {
					// Handle regular field filter within the AND
					subFilter := buildSimpleFieldFilter(sub.Field, sub.Operator, sub.Value)
					if len(subFilter) > 0 {
						andFilters = append(andFilters, subFilter)
					}
				}
			}
			continue
		}
		// Handle regular filters
		singleFilter := buildSimpleFieldFilter(f.Field, f.Operator, f.Value)
		if len(singleFilter) > 0 {
			andFilters = append(andFilters, singleFilter)
		}
	}

	if len(andFilters) == 0 {
		return bson.M{}
	}
	if len(andFilters) == 1 {
		return andFilters[0]
	}
	return bson.M{"$and": andFilters}
}
