package mongodb

import (
	context "context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"fmt"
	"log"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoQueryEngine implements repository.QueryEngine for MongoDB
// It translates Firestore queries to MongoDB queries in una forma minimalista y extensible.
type MongoQueryEngine struct {
	db                *mongo.Database
	fieldPathResolver repository.FieldPathResolver
	typeCache         map[string]model.FieldValueType // Cache for field type inference
}

// NewMongoQueryEngine creates a new MongoQueryEngine
func NewMongoQueryEngine(db *mongo.Database) *MongoQueryEngine {
	return &MongoQueryEngine{
		db:                db,
		fieldPathResolver: NewMongoFieldPathResolver(),
		typeCache:         make(map[string]model.FieldValueType),
	}
}

// ExecuteQuery ejecuta una consulta Firestore sobre una colección MongoDB
func (qe *MongoQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	log.Printf("[MongoQueryEngine] Ejecutando consulta: collectionPath=%s, query=%+v", collectionPath, query)

	// Check if this is a collection group query
	if query.AllDescendants {
		log.Printf("[MongoQueryEngine] Ejecutando Collection Group query para collectionId=%s", query.CollectionID)
		return qe.executeCollectionGroupQuery(ctx, query)
	}

	// Regular single collection query
	return qe.executeSingleCollectionQuery(ctx, collectionPath, query)
}

// executeCollectionGroupQuery ejecuta una consulta sobre múltiples colecciones que comparten el mismo collectionId
func (qe *MongoQueryEngine) executeCollectionGroupQuery(ctx context.Context, query model.Query) ([]*model.Document, error) {
	// Collection group queries search across all collections with the same name
	// In MongoDB, we need to find all collections that end with the collection ID

	collectionNames, err := qe.findCollectionsWithSuffix(ctx, query.CollectionID)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error finding collections for group query: %v", err)
		return nil, err
	}

	log.Printf("[MongoQueryEngine] Found %d collections for collection group '%s': %v",
		len(collectionNames), query.CollectionID, collectionNames)

	var allDocs []*model.Document

	// Execute query on each matching collection
	for _, collectionName := range collectionNames {
		docs, err := qe.executeSingleCollectionQuery(ctx, collectionName, query)
		if err != nil {
			log.Printf("[MongoQueryEngine] Error querying collection %s: %v", collectionName, err)
			continue // Continue with other collections
		}
		allDocs = append(allDocs, docs...)
	}

	// Apply global limit and ordering if needed
	if query.Limit > 0 && len(allDocs) > int(query.Limit) {
		// TODO: For proper ordering across collections, we should sort first then limit
		// For now, just limit the total results
		allDocs = allDocs[:query.Limit]
	}

	log.Printf("[MongoQueryEngine] Collection group query completed. Total documents found: %d", len(allDocs))
	return allDocs, nil
}

// findCollectionsWithSuffix encuentra todas las colecciones que terminan con el sufijo dado
func (qe *MongoQueryEngine) findCollectionsWithSuffix(ctx context.Context, suffix string) ([]string, error) {
	if qe.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// List all collections in the database
	collections, err := qe.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	log.Printf("[MongoQueryEngine] All collections in database: %v", collections)
	log.Printf("[MongoQueryEngine] Looking for collections with suffix: %s", suffix)

	var matchingCollections []string

	// Find collections that end with the suffix or are exactly the suffix
	for _, collectionName := range collections {
		log.Printf("[MongoQueryEngine] Checking collection: %s", collectionName)

		// Helper function to check if a collection matches the suffix
		matchesExact := func(collName, suf string) bool {
			if collName == suf {
				return true
			}
			// Also check URL encoded/decoded versions
			if decodedCollName, err := url.QueryUnescape(collName); err == nil && decodedCollName == suf {
				return true
			}
			if encodedSuf := url.QueryEscape(suf); encodedSuf == collName {
				return true
			}
			return false
		}

		// Helper function to check if collection ends with suffix
		matchesEnding := func(collName, suf string) bool {
			if len(collName) <= len(suf) {
				return false
			}

			// Check normal ending
			if collName[len(collName)-len(suf):] == suf && collName[len(collName)-len(suf)-1] == '/' {
				return true
			}

			// Check URL encoded/decoded endings
			if decodedCollName, err := url.QueryUnescape(collName); err == nil {
				if len(decodedCollName) > len(suf) &&
					decodedCollName[len(decodedCollName)-len(suf):] == suf &&
					decodedCollName[len(decodedCollName)-len(suf)-1] == '/' {
					return true
				}
			}

			encodedSuf := url.QueryEscape(suf)
			if len(collName) > len(encodedSuf) &&
				collName[len(collName)-len(encodedSuf):] == encodedSuf &&
				collName[len(collName)-len(encodedSuf)-1] == '/' {
				return true
			}

			return false
		}

		// Match exact collection name or collections ending with the pattern /{suffix}
		if matchesExact(collectionName, suffix) || matchesEnding(collectionName, suffix) {
			log.Printf("[MongoQueryEngine] Collection %s matches suffix %s", collectionName, suffix)
			matchingCollections = append(matchingCollections, collectionName)
		} else {
			log.Printf("[MongoQueryEngine] Collection %s does NOT match suffix %s", collectionName, suffix)
		}
	}

	// If no subcollections found, at least try the base collection
	if len(matchingCollections) == 0 {
		matchingCollections = append(matchingCollections, suffix)
	}

	return matchingCollections, nil
}

// executeSingleCollectionQuery ejecuta una consulta en una sola colección
func (qe *MongoQueryEngine) executeSingleCollectionQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	// Construir filtro principal y filtro de cursores Firestore con contexto para inferencia de tipos
	filter := qe.buildMongoFilterWithContext(ctx, collectionPath, query.Filters)
	log.Printf("[MongoQueryEngine] Filtro MongoDB generado: %+v", filter)
	cursorFilter := qe.buildCursorFilter(query)
	if len(cursorFilter) > 0 {
		log.Printf("[MongoQueryEngine] CursorFilter generado: %+v", cursorFilter)
		// Merge: $and entre filtro principal y filtro de cursores
		filter = mergeFiltersWithAnd(filter, cursorFilter)
		log.Printf("[MongoQueryEngine] Filtro final después de merge: %+v", filter)
	}
	findOpts := qe.buildMongoFindOptions(ctx, collectionPath, query)
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
	log.Printf("[MongoQueryEngine] Documentos encontrados en %s: %d", collectionPath, len(docs))

	// Ensure we always return a non-nil slice
	if docs == nil {
		docs = []*model.Document{}
	}

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
func (qe *MongoQueryEngine) buildMongoFilter(filters []model.Filter) bson.M {
	return qe.buildMongoFilterWithContext(context.Background(), "", filters)
}

// buildMongoFilterWithContext soporta filtros compuestos y operadores avanzados con contexto para inferencia de tipos
func (qe *MongoQueryEngine) buildMongoFilterWithContext(ctx context.Context, collectionPath string, filters []model.Filter) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}

	var andFilters []bson.M
	for _, f := range filters {
		if f.Composite == "or" && len(f.SubFilters) > 0 {
			var orFilters []bson.M
			for _, sub := range f.SubFilters {
				subFilter := qe.singleMongoFilterWithContext(ctx, collectionPath, sub)
				if len(subFilter) > 0 {
					orFilters = append(orFilters, subFilter)
				}
			}
			if len(orFilters) > 0 {
				andFilters = append(andFilters, bson.M{"$or": orFilters})
			}
		} else if f.Composite == "and" && len(f.SubFilters) > 0 {
			// Handle AND composite filters by recursively processing sub-filters
			subAndFilter := qe.buildMongoFilterWithContext(ctx, collectionPath, f.SubFilters)
			if len(subAndFilter) > 0 {
				andFilters = append(andFilters, subAndFilter)
			}
		} else if f.Composite == "" {
			// Handle regular (non-composite) filters
			singleFilter := qe.singleMongoFilterWithContext(ctx, collectionPath, f)
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

// singleMongoFilter traduce un filtro simple
func (qe *MongoQueryEngine) singleMongoFilter(f model.Filter) bson.M {
	return qe.singleMongoFilterWithContext(context.Background(), "", f)
}

// singleMongoFilterWithContext traduce un filtro simple con contexto para inferencia de tipos
func (qe *MongoQueryEngine) singleMongoFilterWithContext(ctx context.Context, collectionPath string, f model.Filter) bson.M {
	// For array contains with objects, preserve the original value structure
	var primitiveValue interface{}
	if f.Operator == model.OperatorArrayContains {
		// For array contains, check if the value is already an object and preserve it
		if _, ok := f.Value.(map[string]interface{}); ok {
			primitiveValue = f.Value
		} else {
			primitiveValue = extractPrimitiveValue(f.Value)
		}
	} else {
		primitiveValue = extractPrimitiveValue(f.Value)
	}

	// Create FieldPath from the field string
	fieldPath, err := model.NewFieldPath(f.Field)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error creating FieldPath from %s: %v. Using fallback.", f.Field, err)
		mongoPath := fmt.Sprintf("fields.%s.stringValue", f.Field)
		return qe.buildFilterBSON(mongoPath, f.Operator, primitiveValue)
	}

	// For array operations, always use arrayValue regardless of the actual value type
	var valueType model.FieldValueType
	if qe.isArrayOperation(f.Operator) {
		valueType = model.FieldTypeArray
	} else if f.Operator == model.OperatorIn || f.Operator == model.OperatorNotIn {
		// For IN/NOT_IN operators, determine type from the first element of the array
		if arrayValue, ok := primitiveValue.([]interface{}); ok && len(arrayValue) > 0 {
			valueType = model.DetermineValueType(arrayValue[0])
		} else {
			valueType = model.DetermineValueType(primitiveValue)
		}
	} else {
		// First try to determine type from the current filter value
		valueType = model.DetermineValueType(primitiveValue)
		log.Printf("[MongoQueryEngine] Determined type from filter value for field %s: %s", f.Field, valueType)

		// If we have a collectionPath, also try hybrid inference for caching
		if collectionPath != "" {
			// Try to infer and cache the type for future use
			inferredType := qe.inferFieldTypeForFiltering(ctx, collectionPath, f.Field)
			// Use the inferred type only if we couldn't determine from value or they match
			if valueType == model.FieldTypeString && inferredType != model.FieldTypeString {
				valueType = inferredType
				log.Printf("[MongoQueryEngine] Used inferred type from collection analysis: %s", inferredType)
			}
		}
	}

	// Use the FieldPathResolver to correctly translate Firestore field paths to MongoDB paths
	mongoPath, err := qe.fieldPathResolver.ResolveFieldPath(fieldPath, valueType)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error resolving field path %s: %v. Using fallback.", f.Field, err)
		mongoPath = fmt.Sprintf("fields.%s.stringValue", f.Field)
	}

	log.Printf("[MongoQueryEngine] Traduciendo filtro: field=%s -> mongoPath=%s, operator=%s, value=%v",
		f.Field, mongoPath, f.Operator, primitiveValue)

	return qe.buildFilterBSON(mongoPath, f.Operator, primitiveValue)
}

// isArrayOperation determines if an operator is an array operation
func (qe *MongoQueryEngine) isArrayOperation(operator model.Operator) bool {
	return operator == model.OperatorArrayContains ||
		operator == model.OperatorArrayContainsAny
}

// buildFilterBSON creates the BSON filter for a given operator and value
func (qe *MongoQueryEngine) buildFilterBSON(mongoPath string, operator model.Operator, primitiveValue interface{}) bson.M {
	switch operator {
	case model.OperatorEqual:
		return bson.M{mongoPath: primitiveValue}
	case model.OperatorNotEqual:
		return bson.M{mongoPath: bson.M{"$ne": primitiveValue}}
	case model.OperatorGreaterThan:
		return bson.M{mongoPath: bson.M{"$gt": primitiveValue}}
	case model.OperatorGreaterThanOrEqual:
		return bson.M{mongoPath: bson.M{"$gte": primitiveValue}}
	case model.OperatorLessThan:
		return bson.M{mongoPath: bson.M{"$lt": primitiveValue}}
	case model.OperatorLessThanOrEqual:
		return bson.M{mongoPath: bson.M{"$lte": primitiveValue}}
	case model.OperatorIn:
		return bson.M{mongoPath: bson.M{"$in": primitiveValue}}
	case model.OperatorNotIn:
		return bson.M{mongoPath: bson.M{"$nin": primitiveValue}}
	case model.OperatorArrayContains:
		// For array contains in Firestore format, we need to match elements with typed values
		// The mongoPath should be "fields.tags.arrayValue" and we need to find elements like {"stringValue": "gaming"}
		if objectValue, ok := primitiveValue.(map[string]interface{}); ok {
			// If the value is already a Firestore object (e.g., {"stringValue": "gaming"}), use it directly
			return bson.M{mongoPath: bson.M{"$elemMatch": objectValue}}
		} else {
			// For primitive values, wrap them in the appropriate Firestore format
			// Determine the type and create the appropriate wrapper
			firestoreValue := qe.wrapValueForFirestore(primitiveValue)
			return bson.M{mongoPath: bson.M{"$elemMatch": firestoreValue}}
		}
	case model.OperatorArrayContainsAny:
		// For array contains any, we need to find arrays that contain any of the specified values
		// Convert the array of primitive values to Firestore format
		if arrayValues, ok := primitiveValue.([]interface{}); ok {
			// Initialize with empty slice to handle empty arrays properly
			firestoreValues := make([]bson.M, 0, len(arrayValues))
			for _, val := range arrayValues {
				firestoreValues = append(firestoreValues, qe.wrapValueForFirestore(val))
			}
			// Use $elemMatch with $in to find any matching Firestore-formatted value
			return bson.M{mongoPath: bson.M{"$elemMatch": bson.M{"$in": firestoreValues}}}
		}
		// Fallback for single value (shouldn't happen with array-contains-any)
		firestoreValue := qe.wrapValueForFirestore(primitiveValue)
		return bson.M{mongoPath: bson.M{"$elemMatch": firestoreValue}}
	default:
		return bson.M{mongoPath: primitiveValue}
	}
}

// buildMongoFindOptions soporta proyecciones y ordenamientos
func (qe *MongoQueryEngine) buildMongoFindOptions(ctx context.Context, collectionPath string, query model.Query) *options.FindOptions {
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

			// Inferir el tipo de campo para ordenamiento correcto
			fieldType := qe.inferFieldTypeForOrdering(ctx, collectionPath, o.Field, query)
			fieldPath := qe.buildOrderFieldPath(o.Field, fieldType)

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
func (qe *MongoQueryEngine) buildCursorFilter(query model.Query) bson.M {
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
			fieldPath = qe.resolveFieldPathForValue(order.Field, query.StartAt[i])
		} else if len(query.StartAfter) > i {
			fieldPath = qe.resolveFieldPathForValue(order.Field, query.StartAfter[i])
		} else if len(query.EndAt) > i {
			fieldPath = qe.resolveFieldPathForValue(order.Field, query.EndAt[i])
		} else if len(query.EndBefore) > i {
			fieldPath = qe.resolveFieldPathForValue(order.Field, query.EndBefore[i])
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

// ExecuteQueryWithProjection executes a query with field projection support
// This method extends the basic ExecuteQuery with field selection capabilities
func (qe *MongoQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	log.Printf("[MongoQueryEngine] Ejecutando consulta con proyección: collectionPath=%s, projection=%v", collectionPath, projection)

	// Build the main filter with context for type inference
	filter := qe.buildMongoFilterWithContext(ctx, collectionPath, query.Filters)

	// Build cursor filter if needed
	cursorFilter := qe.buildCursorFilter(query)
	if len(cursorFilter) > 0 {
		filter = mergeFiltersWithAnd(filter, cursorFilter)
	}

	// Build find options with projection
	findOpts := qe.buildMongoFindOptions(ctx, collectionPath, query)

	// Add projection if specified
	if len(projection) > 0 {
		projectionDoc := bson.M{}
		for _, field := range projection {
			// Convert Firestore field paths to MongoDB field paths
			mongoField := fmt.Sprintf("fields.%s", field)
			projectionDoc[mongoField] = 1
		}
		// Always include metadata fields
		projectionDoc["project_id"] = 1
		projectionDoc["database_id"] = 1
		projectionDoc["collection_id"] = 1
		projectionDoc["document_id"] = 1
		projectionDoc["create_time"] = 1
		projectionDoc["update_time"] = 1

		findOpts.SetProjection(projectionDoc)
	}

	cur, err := qe.db.Collection(collectionPath).Find(ctx, filter, findOpts)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error en Find con proyección: %v", err)
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

	log.Printf("[MongoQueryEngine] Documentos encontrados con proyección: %d", len(docs))
	return docs, nil
}

// CountDocuments returns the count of documents matching the query without fetching the actual documents
// This is more efficient than ExecuteQuery when you only need the count
func (qe *MongoQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	log.Printf("[MongoQueryEngine] Contando documentos: collectionPath=%s", collectionPath)

	// Build the filter (same as ExecuteQuery but without cursor filters for count) with context for type inference
	filter := qe.buildMongoFilterWithContext(ctx, collectionPath, query.Filters)
	log.Printf("[MongoQueryEngine] Filtro para conteo: %+v", filter)

	// Count documents matching the filter
	count, err := qe.db.Collection(collectionPath).CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("[MongoQueryEngine] Error contando documentos: %v", err)
		return 0, err
	}

	log.Printf("[MongoQueryEngine] Total documentos encontrados: %d", count)
	return count, nil
}

// ValidateQuery validates if a query is supported by this MongoDB engine
// Checks for Firestore compatibility and MongoDB limitations
func (qe *MongoQueryEngine) ValidateQuery(query model.Query) error {
	log.Printf("[MongoQueryEngine] Validando consulta: %+v", query)

	// Validate basic query structure
	if err := query.ValidateQuery(); err != nil {
		return fmt.Errorf("invalid query structure: %w", err)
	}

	// Check MongoDB-specific limitations
	if len(query.Filters) > 100 {
		return fmt.Errorf("too many filters: %d exceeds MongoDB limit of 100", len(query.Filters))
	}

	if len(query.Orders) > 32 {
		return fmt.Errorf("too many sort fields: %d exceeds MongoDB limit of 32", len(query.Orders))
	}

	// Validate each filter
	for i, filter := range query.Filters {
		if err := qe.validateFilter(filter); err != nil {
			return fmt.Errorf("invalid filter at index %d: %w", i, err)
		}
	}

	// Validate ordering fields
	for i, order := range query.Orders {
		if err := qe.validateOrder(order); err != nil {
			return fmt.Errorf("invalid order at index %d: %w", i, err)
		}
	}

	log.Printf("[MongoQueryEngine] Consulta validada exitosamente")
	return nil
}

// validateFilter validates a single filter for MongoDB compatibility
func (qe *MongoQueryEngine) validateFilter(filter model.Filter) error {
	// Check field path
	if filter.Field == "" && filter.FieldPath == nil {
		return fmt.Errorf("filter must have either Field or FieldPath specified")
	}

	// Validate field path if present
	if filter.FieldPath != nil {
		if err := filter.FieldPath.Validate(); err != nil {
			return fmt.Errorf("invalid field path: %w", err)
		}

		// Check nesting depth (MongoDB supports deep nesting but performance degrades)
		if filter.FieldPath.Depth() > 20 {
			log.Printf("[MongoQueryEngine] Warning: deep nesting detected (%d levels) - may impact performance", filter.FieldPath.Depth())
		}
	}

	// Validate operator
	validOps := []model.Operator{
		model.OperatorEqual, model.OperatorNotEqual, model.OperatorLessThan, model.OperatorLessThanOrEqual,
		model.OperatorGreaterThan, model.OperatorGreaterThanOrEqual, model.OperatorArrayContains,
		model.OperatorArrayContainsAny, model.OperatorIn, model.OperatorNotIn,
	}

	isValid := false
	for _, validOp := range validOps {
		if filter.Operator == validOp {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("unsupported operator: %s", filter.Operator)
	}

	// Validate composite filters
	if filter.IsComposite() {
		if filter.Composite != "and" && filter.Composite != "or" {
			return fmt.Errorf("unsupported composite operator: %s", filter.Composite)
		}

		// Validate sub-filters recursively
		for i, subFilter := range filter.SubFilters {
			if err := qe.validateFilter(subFilter); err != nil {
				return fmt.Errorf("invalid sub-filter at index %d: %w", i, err)
			}
		}
	}

	// Validate array operations
	if filter.IsArrayOperation() {
		// Array operations should not be on nested fields in MongoDB
		if filter.FieldPath != nil && filter.FieldPath.IsNested() {
			return fmt.Errorf("array operations not supported on nested fields: %s", filter.FieldPath.Raw())
		}
	}

	return nil
}

// validateOrder validates a single order specification
func (qe *MongoQueryEngine) validateOrder(order model.Order) error {
	if order.Field == "" {
		return fmt.Errorf("order field cannot be empty")
	}

	if order.Direction != model.DirectionAscending && order.Direction != model.DirectionDescending {
		return fmt.Errorf("invalid order direction: %s", order.Direction)
	}

	return nil
}

// GetQueryCapabilities returns the capabilities of this MongoDB query engine
// This follows the Firestore specification with MongoDB-specific limitations
func (qe *MongoQueryEngine) GetQueryCapabilities() repository.QueryCapabilities {
	return repository.QueryCapabilities{
		SupportsNestedFields:     true, // MongoDB supports nested field queries
		SupportsArrayContains:    true, // MongoDB supports $elemMatch
		SupportsArrayContainsAny: true, // MongoDB supports $in on arrays
		SupportsCompositeFilters: true, // MongoDB supports $and/$or
		SupportsOrderBy:          true, // MongoDB supports sorting
		SupportsCursorPagination: true, // Implemented with startAt/endAt
		SupportsOffsetPagination: true, // MongoDB supports skip/limit
		SupportsProjection:       true, // MongoDB supports field projection
		MaxFilterCount:           100,  // Reasonable limit for complex queries
		MaxOrderByCount:          32,   // MongoDB sort compound index limit
		MaxNestingDepth:          100,  // MongoDB supports deep nesting
	}
}

// Compile-time interface compliance check
var _ repository.QueryEngine = (*MongoQueryEngine)(nil)

// buildFieldFilter creates a MongoDB filter for a single field (used for testing)
func (qe *MongoQueryEngine) buildFieldFilter(field string, operator model.Operator, value interface{}) bson.M {
	filter := model.Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
	return qe.singleMongoFilter(filter)
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

// resolveFieldPathForValue is a helper function to resolve field paths for cursor values
func (qe *MongoQueryEngine) resolveFieldPathForValue(fieldName string, value interface{}) string {
	// Create FieldPath from the field string
	fieldPath, err := model.NewFieldPath(fieldName)
	if err != nil {
		return fmt.Sprintf("fields.%s.stringValue", fieldName)
	}

	// Determine value type from the primitive value
	valueType := model.DetermineValueType(value)

	// Use the FieldPathResolver to correctly translate Firestore field paths to MongoDB paths
	mongoPath, err := qe.fieldPathResolver.ResolveFieldPath(fieldPath, valueType)
	if err != nil {
		return fmt.Sprintf("fields.%s.stringValue", fieldName)
	}

	return mongoPath
}

// inferFieldTypeForOrdering infers the field type for ordering operations using a hybrid approach
// Priority order: Cache → Filter analysis → Sample document → Fallback
func (qe *MongoQueryEngine) inferFieldTypeForOrdering(ctx context.Context, collectionPath string, fieldName string, query model.Query) model.FieldValueType {
	cacheKey := collectionPath + "." + fieldName

	// 1. Cache hit - highest priority for performance
	if cachedType, exists := qe.typeCache[cacheKey]; exists {
		log.Printf("[MongoQueryEngine] Cache hit for field %s: %s", fieldName, cachedType)
		return cachedType
	}

	// 2. Analyze existing filters in the same query - very efficient
	if inferredType := qe.inferTypeFromQueryFilters(fieldName, query.Filters); inferredType != "" {
		log.Printf("[MongoQueryEngine] Inferred type from filters for field %s: %s", fieldName, inferredType)
		qe.typeCache[cacheKey] = inferredType
		return inferredType
	}

	// 3. Sample document analysis - more expensive but accurate
	if inferredType := qe.inferTypeFromSampleDocument(ctx, collectionPath, fieldName); inferredType != "" {
		log.Printf("[MongoQueryEngine] Inferred type from sample document for field %s: %s", fieldName, inferredType)
		qe.typeCache[cacheKey] = inferredType
		return inferredType
	}

	// 4. Fallback to string value - Firestore-compatible default
	log.Printf("[MongoQueryEngine] Using fallback stringValue for field %s", fieldName)
	defaultType := model.FieldTypeString
	qe.typeCache[cacheKey] = defaultType

	return defaultType
}

// inferFieldTypeForFiltering infers the field type for filtering operations using a hybrid approach
// Priority order: Cache → Sample document → Fallback
// Note: We don't use filter analysis here to avoid circular dependency
func (qe *MongoQueryEngine) inferFieldTypeForFiltering(ctx context.Context, collectionPath string, fieldName string) model.FieldValueType {
	cacheKey := collectionPath + "." + fieldName

	// 1. Cache hit - highest priority for performance
	if cachedType, exists := qe.typeCache[cacheKey]; exists {
		log.Printf("[MongoQueryEngine] Cache hit for field %s: %s", fieldName, cachedType)
		return cachedType
	}

	// 2. Sample document analysis - accurate type information from existing data
	if inferredType := qe.inferTypeFromSampleDocument(ctx, collectionPath, fieldName); inferredType != "" {
		log.Printf("[MongoQueryEngine] Inferred type from sample document for field %s: %s", fieldName, inferredType)
		qe.typeCache[cacheKey] = inferredType
		return inferredType
	}

	// 3. Fallback to string value - Firestore-compatible default
	log.Printf("[MongoQueryEngine] Using fallback stringValue for field %s", fieldName)
	defaultType := model.FieldTypeString
	qe.typeCache[cacheKey] = defaultType

	return defaultType
}

// inferTypeFromQueryFilters analyzes filters in the current query to infer field types
// This is very efficient as it uses data already available in the query
func (qe *MongoQueryEngine) inferTypeFromQueryFilters(fieldName string, filters []model.Filter) model.FieldValueType {
	for _, filter := range filters {
		if filter.Field == fieldName && filter.Value != nil {
			// Determine type from the filter value
			inferredType := model.DetermineValueType(filter.Value)
			log.Printf("[MongoQueryEngine] Found field %s in filters with type %s", fieldName, inferredType)
			return inferredType
		}

		// Check composite filters recursively
		if len(filter.SubFilters) > 0 {
			if inferredType := qe.inferTypeFromQueryFilters(fieldName, filter.SubFilters); inferredType != "" {
				return inferredType
			}
		}
	}

	return "" // Not found in filters
}

// inferTypeFromSampleDocument gets a sample document to infer field type
// This is more expensive but provides accurate type information
func (qe *MongoQueryEngine) inferTypeFromSampleDocument(ctx context.Context, collectionPath string, fieldName string) model.FieldValueType {
	// Safety check: if db is nil (e.g., in unit tests), return empty
	if qe.db == nil {
		log.Printf("[MongoQueryEngine] Database is nil, cannot sample document for field %s", fieldName)
		return ""
	}

	// Create a minimal filter to find any document with this field
	// Use $exists to ensure the field is present
	filter := bson.M{
		fmt.Sprintf("fields.%s", fieldName): bson.M{"$exists": true},
	}

	// Find options: limit to 1 document for efficiency
	opts := options.FindOne()

	var result bson.M
	err := qe.db.Collection(collectionPath).FindOne(ctx, filter, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[MongoQueryEngine] No documents found with field %s in collection %s", fieldName, collectionPath)
		} else {
			log.Printf("[MongoQueryEngine] Error sampling document for field %s: %v", fieldName, err)
		}
		return ""
	}

	// Extract the field value and determine its type
	if fields, ok := result["fields"].(bson.M); ok {
		if fieldValue, exists := fields[fieldName]; exists {
			if fieldMap, ok := fieldValue.(bson.M); ok {
				// Determine the value type from the Firestore field structure
				return qe.determineFirestoreFieldType(fieldMap)
			}
		}
	}

	return ""
}

// determineFirestoreFieldType determines the Firestore field type from a MongoDB field structure
func (qe *MongoQueryEngine) determineFirestoreFieldType(fieldMap bson.M) model.FieldValueType {
	// Check for different Firestore value types
	if _, exists := fieldMap["stringValue"]; exists {
		return model.FieldTypeString
	}
	if _, exists := fieldMap["doubleValue"]; exists {
		return model.FieldTypeDouble
	}
	if _, exists := fieldMap["integerValue"]; exists {
		return model.FieldTypeInt
	}
	if _, exists := fieldMap["booleanValue"]; exists {
		return model.FieldTypeBool
	}
	if _, exists := fieldMap["timestampValue"]; exists {
		return model.FieldTypeTimestamp
	}
	if _, exists := fieldMap["arrayValue"]; exists {
		return model.FieldTypeArray
	}
	if _, exists := fieldMap["mapValue"]; exists {
		return model.FieldTypeMap
	}
	if _, exists := fieldMap["nullValue"]; exists {
		return model.FieldTypeNull
	}
	if _, exists := fieldMap["bytesValue"]; exists {
		return model.FieldTypeBytes
	}
	if _, exists := fieldMap["referenceValue"]; exists {
		return model.FieldTypeReference
	}
	if _, exists := fieldMap["geoPointValue"]; exists {
		return model.FieldTypeGeoPoint
	}

	// Default fallback
	return model.FieldTypeString
}

// buildOrderFieldPath builds the MongoDB field path for ordering based on the field type
func (qe *MongoQueryEngine) buildOrderFieldPath(fieldName string, fieldType model.FieldValueType) string {
	return fmt.Sprintf("fields.%s.%s", fieldName, string(fieldType))
}

// wrapValueForFirestore wraps a primitive value in the appropriate Firestore format
func (qe *MongoQueryEngine) wrapValueForFirestore(value interface{}) bson.M {
	if value == nil {
		return bson.M{"nullValue": nil}
	}

	switch v := value.(type) {
	case string:
		return bson.M{"stringValue": v}
	case int:
		return bson.M{"integerValue": v}
	case int32:
		return bson.M{"integerValue": int(v)}
	case int64:
		return bson.M{"integerValue": int(v)}
	case float32:
		return bson.M{"doubleValue": float64(v)}
	case float64:
		return bson.M{"doubleValue": v}
	case bool:
		return bson.M{"booleanValue": v}
	default:
		// Default to stringValue for unknown types
		return bson.M{"stringValue": fmt.Sprintf("%v", v)}
	}
}
