package mongodb

import (
	"context"
	"fmt"
	"log"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnhancedMongoQueryEngine implements repository.QueryEngine with full field path support
// This is the adapter layer that translates domain queries to MongoDB operations
type EnhancedMongoQueryEngine struct {
	db                *mongo.Database
	fieldPathResolver repository.FieldPathResolver
	capabilities      repository.QueryCapabilities
	typeCache         map[string]model.FieldValueType // Cache for field type inference
}

// NewEnhancedMongoQueryEngine creates a new enhanced MongoDB query engine
func NewEnhancedMongoQueryEngine(db *mongo.Database) repository.QueryEngine {
	resolver := NewMongoFieldPathResolver()

	return &EnhancedMongoQueryEngine{
		db:                db,
		fieldPathResolver: resolver,
		typeCache:         make(map[string]model.FieldValueType), // Initialize type inference cache
		capabilities: repository.QueryCapabilities{
			SupportsNestedFields:     true,
			SupportsArrayContains:    true,
			SupportsArrayContainsAny: true,
			SupportsCompositeFilters: true,
			SupportsOrderBy:          true,
			SupportsCursorPagination: true,
			SupportsOffsetPagination: true,
			SupportsProjection:       true,
			MaxFilterCount:           100,
			MaxOrderByCount:          32,
			MaxNestingDepth:          100,
		},
	}
}

// ExecuteQuery executes a query with enhanced field path support
func (e *EnhancedMongoQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	log.Printf("[EnhancedMongoQueryEngine] Executing query: collection=%s", collectionPath)

	// Validate query first
	if err := e.ValidateQuery(query); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Build MongoDB filter using enhanced field path resolution with type inference context
	filter, err := e.buildEnhancedMongoFilterWithContext(ctx, collectionPath, query.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter: %w", err)
	}

	log.Printf("[EnhancedMongoQueryEngine] Generated filter: %+v", filter)

	// Build cursor filter
	cursorFilter, err := e.buildCursorFilter(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build cursor filter: %w", err)
	}

	// Merge filters
	finalFilter := e.mergeFilters(filter, cursorFilter)

	// Build find options
	findOpts, err := e.buildEnhancedFindOptions(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build find options: %w", err)
	}

	log.Printf("[EnhancedMongoQueryEngine] Final filter: %+v", finalFilter)
	log.Printf("[EnhancedMongoQueryEngine] Find options: %+v", findOpts)

	// Execute query
	cur, err := e.db.Collection(collectionPath).Find(ctx, finalFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("MongoDB find failed: %w", err)
	}
	defer cur.Close(ctx)

	// Decode results
	var docs []*model.Document
	for cur.Next(ctx) {
		var mongoDoc MongoDocumentFlat
		if err := cur.Decode(&mongoDoc); err != nil {
			log.Printf("[EnhancedMongoQueryEngine] Error decoding document: %v", err)
			continue
		}
		docs = append(docs, mongoFlatToModelDocument(&mongoDoc))
	}

	// Handle limitToLast
	if query.LimitToLast && len(docs) > 0 {
		e.reverseDocs(docs)
		if query.Limit > 0 && len(docs) > int(query.Limit) {
			docs = docs[:query.Limit]
		}
	}
	log.Printf("[EnhancedMongoQueryEngine] Documents found: %d", len(docs))

	// Ensure we always return a non-nil slice
	if docs == nil {
		docs = []*model.Document{}
	}

	return docs, nil
}

// ExecuteQueryWithProjection executes a query with field projection
func (e *EnhancedMongoQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	// Create a copy of the query with projection
	queryWithProjection := query
	queryWithProjection.SelectFields = projection

	return e.ExecuteQuery(ctx, collectionPath, queryWithProjection)
}

// CountDocuments returns the count of documents matching the query
func (e *EnhancedMongoQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	// Build filter (without cursor operations for counting) with type inference context
	filter, err := e.buildEnhancedMongoFilterWithContext(ctx, collectionPath, query.Filters)
	if err != nil {
		return 0, fmt.Errorf("failed to build filter for count: %w", err)
	}

	return e.db.Collection(collectionPath).CountDocuments(ctx, filter)
}

// ValidateQuery validates if a query is supported by the engine
func (e *EnhancedMongoQueryEngine) ValidateQuery(query model.Query) error {
	// Validate basic query structure
	if err := query.ValidateQuery(); err != nil {
		return err
	}

	// Check filter count limits
	if len(query.Filters) > e.capabilities.MaxFilterCount {
		return fmt.Errorf("too many filters: %d exceeds maximum %d",
			len(query.Filters), e.capabilities.MaxFilterCount)
	}

	// Check order count limits
	if len(query.Orders) > e.capabilities.MaxOrderByCount {
		return fmt.Errorf("too many order clauses: %d exceeds maximum %d",
			len(query.Orders), e.capabilities.MaxOrderByCount)
	}

	// Validate each filter
	for i, filter := range query.Filters {
		if err := e.validateFilter(filter); err != nil {
			return fmt.Errorf("filter %d validation failed: %w", i, err)
		}
	}

	// Validate order clauses
	for i, order := range query.Orders {
		if err := e.validateOrder(order); err != nil {
			return fmt.Errorf("order %d validation failed: %w", i, err)
		}
	}

	return nil
}

// GetQueryCapabilities returns the capabilities of this query engine
func (e *EnhancedMongoQueryEngine) GetQueryCapabilities() repository.QueryCapabilities {
	return e.capabilities
}

// buildEnhancedMongoFilterWithContext builds MongoDB filters with enhanced field path support and type inference
func (e *EnhancedMongoQueryEngine) buildEnhancedMongoFilterWithContext(ctx context.Context, collectionPath string, filters []model.Filter) (bson.M, error) {
	if len(filters) == 0 {
		return bson.M{}, nil
	}

	var andFilters []bson.M

	for _, filter := range filters {
		if filter.IsComposite() {
			// Handle composite filters (AND/OR)
			compositeFilter, err := e.buildCompositeFilterWithContext(ctx, collectionPath, filter)
			if err != nil {
				return nil, err
			}
			if len(compositeFilter) > 0 {
				andFilters = append(andFilters, compositeFilter)
			}
		} else {
			// Handle single filters with type inference
			singleFilter, err := e.buildSingleFilterWithContext(ctx, collectionPath, filter)
			if err != nil {
				return nil, err
			}
			if len(singleFilter) > 0 {
				andFilters = append(andFilters, singleFilter)
			}
		}
	}

	// Return the appropriate filter structure
	if len(andFilters) == 0 {
		return bson.M{}, nil
	}
	if len(andFilters) == 1 {
		return andFilters[0], nil
	}
	return bson.M{"$and": andFilters}, nil
}

// buildCompositeFilterWithContext handles AND/OR composite filters with type inference
func (e *EnhancedMongoQueryEngine) buildCompositeFilterWithContext(ctx context.Context, collectionPath string, filter model.Filter) (bson.M, error) {
	if !filter.IsComposite() {
		return nil, fmt.Errorf("not a composite filter")
	}

	switch filter.Composite {
	case "and":
		return e.buildEnhancedMongoFilterWithContext(ctx, collectionPath, filter.SubFilters)
	case "or":
		var orFilters []bson.M
		for _, subFilter := range filter.SubFilters {
			subFilterBson, err := e.buildSingleFilterWithContext(ctx, collectionPath, subFilter)
			if err != nil {
				return nil, err
			}
			if len(subFilterBson) > 0 {
				orFilters = append(orFilters, subFilterBson)
			}
		}
		if len(orFilters) > 0 {
			return bson.M{"$or": orFilters}, nil
		}
		return bson.M{}, nil
	default:
		return nil, fmt.Errorf("unsupported composite operator: %s", filter.Composite)
	}
}

// buildSingleFilterWithContext builds a MongoDB filter for a single condition with type inference
func (e *EnhancedMongoQueryEngine) buildSingleFilterWithContext(ctx context.Context, collectionPath string, filter model.Filter) (bson.M, error) {
	// Extract primitive value (same as MongoQueryEngine)
	var primitiveValue interface{}
	if filter.Operator == model.OperatorArrayContains {
		// For array contains, check if the value is already an object and preserve it
		if _, ok := filter.Value.(map[string]interface{}); ok {
			primitiveValue = filter.Value
		} else {
			primitiveValue = e.extractPrimitiveValue(filter.Value)
		}
	} else {
		primitiveValue = e.extractPrimitiveValue(filter.Value)
	}

	// Create FieldPath from the filter field (same as MongoQueryEngine)
	fieldPath, err := filter.GetEffectiveFieldPath()
	if err != nil {
		log.Printf("[EnhancedMongoQueryEngine] Error getting field path from %s: %v. Using fallback.", filter.Field, err)
		// Fallback: use simple field path construction
		fallbackPath := fmt.Sprintf("fields.%s.stringValue", filter.Field)
		return e.buildFilterBSON(fallbackPath, filter.Operator, primitiveValue), nil
	}

	// Use the same type inference strategy as MongoQueryEngine
	var valueType model.FieldValueType
	if e.isArrayOperation(filter.Operator) {
		valueType = model.FieldTypeArray
	} else if filter.Operator == model.OperatorIn || filter.Operator == model.OperatorNotIn {
		// For IN/NOT_IN operators, determine type from the first element of the array
		if arrayValue, ok := primitiveValue.([]interface{}); ok && len(arrayValue) > 0 {
			valueType = model.DetermineValueType(arrayValue[0])
		} else {
			valueType = model.DetermineValueType(primitiveValue)
		}
	} else {
		// Use hybrid type inference strategy (same as MongoQueryEngine)
		if collectionPath != "" {
			valueType = e.inferFieldTypeForFiltering(ctx, collectionPath, filter.Field)
		} else {
			// Fallback: determine value type from the primitive value for non-array operations
			valueType = model.DetermineValueType(primitiveValue)
		}
	}

	// Resolve the MongoDB field path using type inference
	mongoFieldPath, err := e.fieldPathResolver.ResolveFieldPath(fieldPath, valueType)
	if err != nil {
		log.Printf("[EnhancedMongoQueryEngine] Error resolving field path %s: %v. Using fallback.", filter.Field, err)
		mongoFieldPath = fmt.Sprintf("fields.%s.stringValue", filter.Field)
	}

	log.Printf("[EnhancedMongoQueryEngine] Building filter with type inference: field=%s -> mongoPath=%s, operator=%s, value=%v, type=%s",
		filter.Field, mongoFieldPath, filter.Operator, primitiveValue, valueType)

	return e.buildFilterBSON(mongoFieldPath, filter.Operator, primitiveValue), nil
}

// buildArrayFilter builds MongoDB filters for array operations
func (e *EnhancedMongoQueryEngine) buildArrayFilter(filter model.Filter, fieldPath *model.FieldPath) (bson.M, error) {
	// For array operations, use arrayValue path
	resolver := e.fieldPathResolver.(*MongoFieldPathResolver)
	arrayFieldPath, err := resolver.ResolveArrayFieldPath(fieldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve array field path: %w", err)
	}

	switch filter.Operator {
	case model.OperatorArrayContains:
		// Check if the original value is an object before extracting primitive value
		if objValue, ok := filter.Value.(map[string]interface{}); ok {
			// Object in array - use $elemMatch with the original object
			return bson.M{arrayFieldPath: bson.M{"$elemMatch": objValue}}, nil
		}
		// Primitive in array - extract primitive value and use direct match
		primitiveValue := e.extractPrimitiveValue(filter.Value)
		return bson.M{arrayFieldPath: primitiveValue}, nil

	case model.OperatorArrayContainsAny:
		// Array contains any - use $in with primitive value
		primitiveValue := e.extractPrimitiveValue(filter.Value)
		return bson.M{arrayFieldPath: bson.M{"$in": primitiveValue}}, nil

	default:
		return nil, fmt.Errorf("unsupported array operator: %s", filter.Operator)
	}
}

// Helper methods

func (e *EnhancedMongoQueryEngine) extractPrimitiveValue(val interface{}) interface{} {
	if m, ok := val.(map[string]interface{}); ok {
		for _, v := range m {
			return v // Return the first value (e.g., booleanValue, stringValue, etc.)
		}
	}
	return val
}

func (e *EnhancedMongoQueryEngine) mergeFilters(filter1, filter2 bson.M) bson.M {
	if len(filter1) == 0 {
		return filter2
	}
	if len(filter2) == 0 {
		return filter1
	}
	return bson.M{"$and": []bson.M{filter1, filter2}}
}

func (e *EnhancedMongoQueryEngine) reverseDocs(docs []*model.Document) {
	for i, j := 0, len(docs)-1; i < j; i, j = i+1, j-1 {
		docs[i], docs[j] = docs[j], docs[i]
	}
}

// TODO: Implement remaining methods (buildCursorFilter, buildEnhancedFindOptions, validateFilter, validateOrder)
// These will be added in the next iteration to keep this implementation focused

// buildCursorFilter builds cursor-based pagination filters
func (e *EnhancedMongoQueryEngine) buildCursorFilter(query model.Query) (bson.M, error) {
	if len(query.Orders) == 0 {
		return bson.M{}, nil
	}

	var filters []bson.M

	for i, order := range query.Orders {
		fieldPath, err := model.NewFieldPath(order.Field)
		if err != nil {
			return nil, fmt.Errorf("invalid order field path %s: %w", order.Field, err)
		}

		// Determine value type for cursor field
		var valueType model.FieldValueType = model.FieldTypeString // Default
		var cursorValue interface{}

		// Get cursor value based on cursor type
		if len(query.StartAt) > i {
			cursorValue = query.StartAt[i]
			valueType = e.inferValueType(cursorValue)
		} else if len(query.StartAfter) > i {
			cursorValue = query.StartAfter[i]
			valueType = e.inferValueType(cursorValue)
		} else if len(query.EndAt) > i {
			cursorValue = query.EndAt[i]
			valueType = e.inferValueType(cursorValue)
		} else if len(query.EndBefore) > i {
			cursorValue = query.EndBefore[i]
			valueType = e.inferValueType(cursorValue)
		} else {
			continue // No cursor value for this field
		}

		// Resolve field path
		mongoFieldPath, err := e.fieldPathResolver.ResolveFieldPath(fieldPath, valueType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve cursor field path: %w", err)
		}

		// Determine comparison operator based on sort direction
		isDesc := order.Direction == model.DirectionDescending

		// Build cursor condition
		if len(query.StartAt) > i {
			op := "$gte"
			if isDesc {
				op = "$lte"
			}
			filters = append(filters, bson.M{mongoFieldPath: bson.M{op: cursorValue}})
		} else if len(query.StartAfter) > i {
			op := "$gt"
			if isDesc {
				op = "$lt"
			}
			filters = append(filters, bson.M{mongoFieldPath: bson.M{op: cursorValue}})
		} else if len(query.EndAt) > i {
			op := "$lte"
			if isDesc {
				op = "$gte"
			}
			filters = append(filters, bson.M{mongoFieldPath: bson.M{op: cursorValue}})
		} else if len(query.EndBefore) > i {
			op := "$lt"
			if isDesc {
				op = "$gt"
			}
			filters = append(filters, bson.M{mongoFieldPath: bson.M{op: cursorValue}})
		}
	}

	if len(filters) == 0 {
		return bson.M{}, nil
	}
	if len(filters) == 1 {
		return filters[0], nil
	}
	return bson.M{"$and": filters}, nil
}

// buildEnhancedFindOptions builds MongoDB find options with enhanced field path support
func (e *EnhancedMongoQueryEngine) buildEnhancedFindOptions(query model.Query) (*options.FindOptions, error) {
	opts := options.Find()

	// Set limit
	if query.Limit > 0 {
		opts.SetLimit(int64(query.Limit))
	}

	// Set offset
	if query.Offset > 0 {
		opts.SetSkip(int64(query.Offset))
	}

	// Build sort with enhanced field path resolution
	if len(query.Orders) > 0 {
		sort := bson.D{}
		for _, order := range query.Orders {
			fieldPath, err := model.NewFieldPath(order.Field)
			if err != nil {
				return nil, fmt.Errorf("invalid order field path %s: %w", order.Field, err)
			}

			// For ordering, default to stringValue if no type specified
			mongoFieldPath, err := e.fieldPathResolver.ResolveOrderFieldPath(fieldPath, "")
			if err != nil {
				return nil, fmt.Errorf("failed to resolve order field path: %w", err)
			}

			sortOrder := 1
			if order.Direction == model.DirectionDescending {
				sortOrder = -1
			}

			sort = append(sort, bson.E{Key: mongoFieldPath, Value: sortOrder})
		}
		opts.SetSort(sort)
	}
	// Build projection
	if len(query.SelectFields) > 0 {
		projection := bson.M{}
		for _, field := range query.SelectFields {
			fieldPath, err := model.NewFieldPath(field)
			if err != nil {
				return nil, fmt.Errorf("invalid select field path %s: %w", field, err)
			}

			// For projection, include the entire field structure
			mongoFieldPath := fmt.Sprintf("fields.%s", fieldPath.Root())
			projection[mongoFieldPath] = 1
		}
		// Always include metadata fields (using correct BSON field names)
		projection["projectID"] = 1
		projection["databaseID"] = 1
		projection["collectionID"] = 1
		projection["documentID"] = 1
		projection["path"] = 1
		projection["parentPath"] = 1
		projection["createTime"] = 1
		projection["updateTime"] = 1
		projection["version"] = 1
		projection["exists"] = 1

		opts.SetProjection(projection)
	}

	return opts, nil
}

// validateFilter validates a single filter
func (e *EnhancedMongoQueryEngine) validateFilter(filter model.Filter) error {
	// Validate field path
	fieldPath, err := filter.GetEffectiveFieldPath()
	if err != nil {
		return fmt.Errorf("invalid field path: %w", err)
	}

	// Check nesting depth
	if fieldPath.Depth() > e.capabilities.MaxNestingDepth {
		return fmt.Errorf("field path %s exceeds maximum nesting depth %d",
			fieldPath.Raw(), e.capabilities.MaxNestingDepth)
	}

	// Validate operator
	if !e.isValidOperator(filter.Operator) {
		return fmt.Errorf("unsupported operator: %s", filter.Operator)
	}

	// Validate array operations
	if filter.IsArrayOperation() && fieldPath.IsNested() {
		return fmt.Errorf("array operations not supported on nested fields: %s", fieldPath.Raw())
	}

	// Validate composite filters
	if filter.IsComposite() {
		for i, subFilter := range filter.SubFilters {
			if err := e.validateFilter(subFilter); err != nil {
				return fmt.Errorf("sub-filter %d validation failed: %w", i, err)
			}
		}
	}

	return nil
}

// validateOrder validates an order clause
func (e *EnhancedMongoQueryEngine) validateOrder(order model.Order) error {
	// Validate field path
	fieldPath, err := model.NewFieldPath(order.Field)
	if err != nil {
		return fmt.Errorf("invalid order field path %s: %w", order.Field, err)
	}

	// Check nesting depth
	if fieldPath.Depth() > e.capabilities.MaxNestingDepth {
		return fmt.Errorf("order field path %s exceeds maximum nesting depth %d",
			fieldPath.Raw(), e.capabilities.MaxNestingDepth)
	}

	// Validate direction
	if order.Direction != model.DirectionAscending && order.Direction != model.DirectionDescending {
		return fmt.Errorf("invalid sort direction: %s", order.Direction)
	}

	return nil
}

// inferValueType infers the field value type from a Go value
func (e *EnhancedMongoQueryEngine) inferValueType(value interface{}) model.FieldValueType {
	if value == nil {
		return model.FieldTypeNull
	}

	switch value.(type) {
	case bool:
		return model.FieldTypeBool
	case string:
		return model.FieldTypeString
	case int, int32, int64:
		return model.FieldTypeInt
	case float32, float64:
		return model.FieldTypeDouble
	case []interface{}:
		return model.FieldTypeArray
	case map[string]interface{}:
		return model.FieldTypeMap
	default:
		return model.FieldTypeString // Default fallback
	}
}

// isValidOperator checks if an operator is supported
func (e *EnhancedMongoQueryEngine) isValidOperator(op model.Operator) bool {
	validOps := []model.Operator{
		model.OperatorEqual, model.OperatorNotEqual,
		model.OperatorLessThan, model.OperatorLessThanOrEqual,
		model.OperatorGreaterThan, model.OperatorGreaterThanOrEqual,
		model.OperatorArrayContains, model.OperatorArrayContainsAny,
		model.OperatorIn, model.OperatorNotIn,
	}

	for _, validOp := range validOps {
		if op == validOp {
			return true
		}
	}
	return false
}

// isArrayOperation determines if an operator is an array operation
func (e *EnhancedMongoQueryEngine) isArrayOperation(operator model.Operator) bool {
	return operator == model.OperatorArrayContains ||
		operator == model.OperatorArrayContainsAny
}

// buildFilterBSON creates the BSON filter for a given operator and value
func (e *EnhancedMongoQueryEngine) buildFilterBSON(mongoPath string, operator model.Operator, primitiveValue interface{}) bson.M {
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
		// Check if the original value is an object for array contains
		if objValue, ok := primitiveValue.(map[string]interface{}); ok {
			// Object in array - use $elemMatch with the original object
			return bson.M{mongoPath: bson.M{"$elemMatch": objValue}}
		}
		// Primitive in array - use direct match
		return bson.M{mongoPath: primitiveValue}
	case model.OperatorArrayContainsAny:
		return bson.M{mongoPath: bson.M{"$in": primitiveValue}}
	default:
		// Fallback for unknown operators
		return bson.M{mongoPath: primitiveValue}
	}
}

// inferFieldTypeForFiltering infers the field type for filtering operations using a hybrid approach
// Priority order: Cache → Sample document → Fallback
// Note: We don't use filter analysis here to avoid circular dependency
func (e *EnhancedMongoQueryEngine) inferFieldTypeForFiltering(ctx context.Context, collectionPath string, fieldName string) model.FieldValueType {
	cacheKey := collectionPath + "." + fieldName

	// 1. Cache hit - highest priority for performance
	if cachedType, exists := e.typeCache[cacheKey]; exists {
		log.Printf("[EnhancedMongoQueryEngine] Cache hit for field %s: %s", fieldName, cachedType)
		return cachedType
	}

	// 2. Sample document analysis - accurate type information from existing data
	if inferredType := e.inferTypeFromSampleDocument(ctx, collectionPath, fieldName); inferredType != "" {
		log.Printf("[EnhancedMongoQueryEngine] Inferred type from sample document for field %s: %s", fieldName, inferredType)
		e.typeCache[cacheKey] = inferredType
		return inferredType
	}

	// 3. Fallback to string value - Firestore-compatible default
	log.Printf("[EnhancedMongoQueryEngine] Using fallback stringValue for field %s", fieldName)
	defaultType := model.FieldTypeString
	e.typeCache[cacheKey] = defaultType

	return defaultType
}

// inferTypeFromSampleDocument gets a sample document to infer field type
// This is more expensive but provides accurate type information
func (e *EnhancedMongoQueryEngine) inferTypeFromSampleDocument(ctx context.Context, collectionPath string, fieldName string) model.FieldValueType {
	// Safety check: if db is nil (e.g., in unit tests), return empty
	if e.db == nil {
		log.Printf("[EnhancedMongoQueryEngine] Database is nil, cannot sample document for field %s", fieldName)
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
	err := e.db.Collection(collectionPath).FindOne(ctx, filter, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[EnhancedMongoQueryEngine] No documents found with field %s in collection %s", fieldName, collectionPath)
		} else {
			log.Printf("[EnhancedMongoQueryEngine] Error sampling document for field %s: %v", fieldName, err)
		}
		return ""
	}

	// Extract the field value and determine its type
	if fields, ok := result["fields"].(bson.M); ok {
		if fieldValue, exists := fields[fieldName]; exists {
			if fieldMap, ok := fieldValue.(bson.M); ok {
				// Determine the value type from the Firestore field structure
				return e.determineFirestoreFieldType(fieldMap)
			}
		}
	}

	return ""
}

// determineFirestoreFieldType determines the Firestore field type from a MongoDB field structure
func (e *EnhancedMongoQueryEngine) determineFirestoreFieldType(fieldMap bson.M) model.FieldValueType {
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

// BuildMongoFilter builds a MongoDB filter from Firestore filters (implementa repository.QueryEngine)
func (e *EnhancedMongoQueryEngine) BuildMongoFilter(filters []model.Filter) (interface{}, error) {
	if len(filters) == 0 {
		return bson.M{}, nil
	}

	// Use the enhanced filter building method without context (for aggregation pipeline building)
	mongoFilter, err := e.buildEnhancedMongoFilterWithContext(context.Background(), "", filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build enhanced mongo filter: %w", err)
	}

	return mongoFilter, nil
}

// ExecuteAggregationPipeline executes a MongoDB aggregation pipeline (implementa repository.QueryEngine)
func (e *EnhancedMongoQueryEngine) ExecuteAggregationPipeline(ctx context.Context, projectID, databaseID, collectionPath string, pipeline []interface{}) ([]map[string]interface{}, error) {
	log.Printf("[EnhancedMongoQueryEngine] Executing aggregation pipeline on collection: %s", collectionPath)
	log.Printf("[EnhancedMongoQueryEngine] Pipeline: %+v", pipeline)

	collection := e.db.Collection(collectionPath)

	// Ejecutar el pipeline de agregación
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation pipeline: %w", err)
	}
	defer cursor.Close(ctx)

	// Recopilar todos los resultados
	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("[EnhancedMongoQueryEngine] Warning: failed to decode aggregation result: %v", err)
			continue
		}
		results = append(results, result)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("aggregation cursor error: %w", err)
	}

	log.Printf("[EnhancedMongoQueryEngine] Aggregation completed. Results count: %d", len(results))
	return results, nil
}
