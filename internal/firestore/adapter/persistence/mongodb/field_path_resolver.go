package mongodb

import (
	"errors"
	"fmt"
	"strings"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
)

// MongoFieldPathResolver implements repository.FieldPathResolver for MongoDB
// It translates Firestore field paths to MongoDB document structure
type MongoFieldPathResolver struct {
	capabilities repository.QueryCapabilities
}

// NewMongoFieldPathResolver creates a new MongoDB field path resolver
func NewMongoFieldPathResolver() repository.FieldPathResolver {
	return &MongoFieldPathResolver{
		capabilities: repository.QueryCapabilities{
			SupportsNestedFields:     true,
			SupportsArrayContains:    true,
			SupportsArrayContainsAny: true,
			SupportsCompositeFilters: true,
			SupportsOrderBy:          true,
			SupportsCursorPagination: true,
			SupportsOffsetPagination: true,
			SupportsProjection:       true,
			MaxFilterCount:           100, // MongoDB limit
			MaxOrderByCount:          32,  // MongoDB sort limit
			MaxNestingDepth:          100, // Firestore/MongoDB support deep nesting
		},
	}
}

// ResolveFieldPath converts a Firestore field path to MongoDB field path
// Examples:
// - "status" -> "fields.status.stringValue" (if valueType is string)
// - "customer.ruc" -> "fields.customer.value.ruc"
// - "items" -> "fields.items.arrayValue" (if valueType is array)
func (r *MongoFieldPathResolver) ResolveFieldPath(fieldPath *model.FieldPath, valueType model.FieldValueType) (string, error) {
	if fieldPath == nil {
		return "", ErrNilFieldPath
	}

	if err := fieldPath.Validate(); err != nil {
		return "", fmt.Errorf("invalid field path: %w", err)
	}

	// Check nesting depth
	if fieldPath.Depth() > r.capabilities.MaxNestingDepth {
		return "", fmt.Errorf("%w: depth %d exceeds maximum %d",
			ErrFieldPathTooDeep, fieldPath.Depth(), r.capabilities.MaxNestingDepth)
	}

	return r.buildMongoFieldPath(fieldPath, valueType), nil
}

// ResolveOrderFieldPath resolves field paths for ordering operations
// For ordering, we need to determine the value type more carefully
func (r *MongoFieldPathResolver) ResolveOrderFieldPath(fieldPath *model.FieldPath, valueType model.FieldValueType) (string, error) {
	if fieldPath == nil {
		return "", ErrNilFieldPath
	}

	// For ordering, if no type is specified, default to stringValue
	// This matches Firestore's behavior where ordering typically works on comparable types
	effectiveValueType := valueType
	if effectiveValueType == "" {
		effectiveValueType = model.FieldTypeString
	}

	return r.ResolveFieldPath(fieldPath, effectiveValueType)
}

// buildMongoFieldPath builds the actual MongoDB field path
func (r *MongoFieldPathResolver) buildMongoFieldPath(fieldPath *model.FieldPath, valueType model.FieldValueType) string {
	segments := fieldPath.Segments()

	if len(segments) == 1 {
		// Simple field: "status" -> "fields.status.stringValue"
		return r.buildSimpleFieldPath(segments[0], valueType)
	}

	// Nested field: "customer.ruc" -> "fields.customer.value.ruc"
	return r.buildNestedFieldPath(segments, valueType)
}

// buildSimpleFieldPath builds path for a simple (non-nested) field
func (r *MongoFieldPathResolver) buildSimpleFieldPath(fieldName string, valueType model.FieldValueType) string {
	if valueType == "" {
		// Default to stringValue if no type specified
		valueType = model.FieldTypeString
	}
	// Special handling for array types - need to access the values array
	if valueType == model.FieldTypeArray {
		return fmt.Sprintf("fields.%s.arrayValue.values", fieldName)
	}

	// For simple fields, use the specific value type
	// "fields.{fieldName}.{valueType}"
	return fmt.Sprintf("fields.%s.%s", fieldName, string(valueType))
}

// buildNestedFieldPath builds path for nested fields
// For deeply nested fields, each level needs ".value" to follow Firestore document structure
func (r *MongoFieldPathResolver) buildNestedFieldPath(segments []string, valueType model.FieldValueType) string {
	// Build the nested path with .value at each level for map traversal
	// "customer.ruc" -> "fields.customer.value.ruc"
	// "customer.address.city" -> "fields.customer.value.address.value.city"
	// "user.profile.settings.theme" -> "fields.user.value.profile.value.settings.value.theme"

	path := "fields." + segments[0]

	// Add .value for each segment except the last one
	for i := 1; i < len(segments); i++ {
		path += ".value." + segments[i]
	}

	return path
}

// SupportsNestedQueries returns true since MongoDB supports nested field queries
func (r *MongoFieldPathResolver) SupportsNestedQueries() bool {
	return r.capabilities.SupportsNestedFields
}

// SupportsArrayQueries returns true since MongoDB supports array operations
func (r *MongoFieldPathResolver) SupportsArrayQueries() bool {
	return r.capabilities.SupportsArrayContains
}

// GetMaxNestingDepth returns the maximum supported nesting depth
func (r *MongoFieldPathResolver) GetMaxNestingDepth() int {
	return r.capabilities.MaxNestingDepth
}

// GetCapabilities returns the full capabilities of this resolver
func (r *MongoFieldPathResolver) GetCapabilities() repository.QueryCapabilities {
	return r.capabilities
}

// Helper methods for specific field types

// ResolveArrayFieldPath resolves field paths for array operations
func (r *MongoFieldPathResolver) ResolveArrayFieldPath(fieldPath *model.FieldPath) (string, error) {
	if fieldPath == nil {
		return "", ErrNilFieldPath
	}

	// For array operations, always use "arrayValue.values" to access the actual array elements
	if fieldPath.IsNested() {
		return "", fmt.Errorf("%w: array operations not supported on nested fields", ErrUnsupportedOperation)
	}
	resolvedPath := fmt.Sprintf("fields.%s.arrayValue.values", fieldPath.Root())
	return resolvedPath, nil
}

// ResolveMapFieldPath resolves field paths for map value access
func (r *MongoFieldPathResolver) ResolveMapFieldPath(fieldPath *model.FieldPath) (string, error) {
	if fieldPath == nil {
		return "", ErrNilFieldPath
	}

	if !fieldPath.IsNested() {
		// Simple map field: "metadata" -> "fields.metadata.mapValue"
		return fmt.Sprintf("fields.%s.mapValue", fieldPath.Root()), nil
	}

	// Nested access into map: "metadata.version" -> "fields.metadata.mapValue.fields.version"
	// This is more complex and depends on how the map is stored
	segments := fieldPath.Segments()
	rootField := segments[0]
	nestedPath := strings.Join(segments[1:], ".fields.")

	return fmt.Sprintf("fields.%s.mapValue.fields.%s", rootField, nestedPath), nil
}

// InferValueTypeFromPath attempts to infer the value type from the field path
// This is useful when the type is not explicitly provided
func (r *MongoFieldPathResolver) InferValueTypeFromPath(fieldPath *model.FieldPath, value interface{}) model.FieldValueType {
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
		// Default to string for unknown types
		return model.FieldTypeString
	}
}

// Errors
var (
	ErrNilFieldPath         = errors.New("field path cannot be nil")
	ErrFieldPathTooDeep     = errors.New("field path exceeds maximum depth")
	ErrUnsupportedOperation = errors.New("operation not supported")
	ErrInvalidValueType     = errors.New("invalid value type")
)
