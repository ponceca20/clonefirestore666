package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// FieldPath represents a Firestore field path that can be nested
// This follows Firestore's dot notation for nested fields like "customer.ruc"
type FieldPath struct {
	segments []string
	raw      string
}

// NewFieldPath creates a new field path from a dot-separated string
// Examples:
// - "status" -> simple field
// - "customer.ruc" -> nested field level 1
// - "customer.address.city" -> nested field level 2
func NewFieldPath(path string) (*FieldPath, error) {
	if path == "" {
		return nil, ErrEmptyFieldPath
	}

	// Validate field path format
	if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return nil, ErrInvalidFieldPathFormat
	}

	if strings.Contains(path, "..") {
		return nil, ErrInvalidFieldPathFormat
	}
	segments := strings.Split(path, ".")

	// Validate depth
	if len(segments) > MaxFieldPathDepth {
		return nil, ErrFieldPathTooDeep
	}

	// Validate each segment
	for _, segment := range segments {
		if segment == "" {
			return nil, ErrInvalidFieldPathFormat
		}
		if !isValidFieldName(segment) {
			return nil, fmt.Errorf("%w: invalid segment '%s'", ErrInvalidFieldName, segment)
		}
	}

	return &FieldPath{
		segments: segments,
		raw:      path,
	}, nil
}

// MustNewFieldPath creates a field path or panics if invalid
// Use only with compile-time known valid paths
func MustNewFieldPath(path string) *FieldPath {
	fp, err := NewFieldPath(path)
	if err != nil {
		panic(fmt.Sprintf("invalid field path '%s': %v", path, err))
	}
	return fp
}

// Raw returns the original dot-separated string
func (fp *FieldPath) Raw() string {
	return fp.raw
}

// Segments returns the individual path segments
func (fp *FieldPath) Segments() []string {
	return append([]string{}, fp.segments...) // Return copy to prevent mutation
}

// IsNested returns true if the field path has multiple segments
func (fp *FieldPath) IsNested() bool {
	return len(fp.segments) > 1
}

// Root returns the root field name (first segment)
func (fp *FieldPath) Root() string {
	if len(fp.segments) == 0 {
		return ""
	}
	return fp.segments[0]
}

// NestedPath returns the nested path after the root
// For "customer.address.city" returns "address.city"
func (fp *FieldPath) NestedPath() string {
	if len(fp.segments) <= 1 {
		return ""
	}
	return strings.Join(fp.segments[1:], ".")
}

// Depth returns the nesting depth (number of segments)
func (fp *FieldPath) Depth() int {
	return len(fp.segments)
}

// Parent returns the parent field path, or nil if this is a root field
// For "customer.address.city" returns "customer.address"
func (fp *FieldPath) Parent() *FieldPath {
	if len(fp.segments) <= 1 {
		return nil
	}

	parentSegments := fp.segments[:len(fp.segments)-1]
	return &FieldPath{
		segments: parentSegments,
		raw:      strings.Join(parentSegments, "."),
	}
}

// Child creates a new field path by appending a segment
// For "customer" + "address" returns "customer.address"
func (fp *FieldPath) Child(segment string) (*FieldPath, error) {
	if !isValidFieldName(segment) {
		return nil, fmt.Errorf("%w: invalid segment '%s'", ErrInvalidFieldName, segment)
	}

	newPath := fp.raw + "." + segment
	return &FieldPath{
		segments: append(fp.segments, segment),
		raw:      newPath,
	}, nil
}

// String implements Stringer interface
func (fp *FieldPath) String() string {
	return fp.raw
}

// Equal checks if two field paths are equal
func (fp *FieldPath) Equal(other *FieldPath) bool {
	if other == nil {
		return false
	}
	return fp.raw == other.raw
}

// Validate checks if the field path is valid according to Firestore rules
func (fp *FieldPath) Validate() error {
	if len(fp.segments) == 0 {
		return ErrEmptyFieldPath
	}

	// Check maximum nesting depth (Firestore allows up to 100 levels)
	if len(fp.segments) > MaxFieldPathDepth {
		return fmt.Errorf("%w: depth %d exceeds maximum %d",
			ErrFieldPathTooDeep, len(fp.segments), MaxFieldPathDepth)
	}

	// Validate each segment
	for _, segment := range fp.segments {
		if !isValidFieldName(segment) {
			return fmt.Errorf("%w: invalid segment '%s'", ErrInvalidFieldName, segment)
		}
	}

	return nil
}

// isValidFieldName checks if a field name follows Firestore naming rules
func isValidFieldName(name string) bool {
	if name == "" {
		return false
	}

	// Firestore field names must:
	// - Not exceed 1500 bytes
	// - Not contain certain characters
	// - Not start with certain prefixes

	if len(name) > MaxFieldNameLength {
		return false
	}

	// Check for invalid characters
	invalidChars := []string{"/", "[", "]", "*", "`"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return false
		}
	}

	// Check for reserved prefixes
	reservedPrefixes := []string{"__"}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}

	return true
}

// DetermineValueType determines the FieldValueType based on the Go type of a value
// This is used by query engines to properly resolve field paths with type information
func DetermineValueType(value interface{}) FieldValueType {
	if value == nil {
		return FieldTypeNull
	}

	switch v := value.(type) {
	case bool:
		return FieldTypeBool
	case int, int8, int16, int32, int64:
		return FieldTypeInt
	case uint, uint8, uint16, uint32, uint64:
		return FieldTypeInt
	case float32, float64:
		return FieldTypeDouble
	case string:
		// Check if this looks like a timestamp string
		parser := NewTimestampParser()
		if parser.IsTimestampString(v) {
			return FieldTypeTimestamp
		}
		return FieldTypeString
	case time.Time:
		return FieldTypeTimestamp
	case []byte:
		return FieldTypeBytes
	case []interface{}:
		return FieldTypeArray
	case map[string]interface{}:
		return FieldTypeMap
	default:
		// For arrays, slices and other complex types, default to stringValue as fallback
		return FieldTypeString
	}
}

// Constants for validation
const (
	MaxFieldPathDepth  = 100  // Firestore maximum nesting depth
	MaxFieldNameLength = 1500 // Firestore maximum field name length in bytes
)

// Field path errors
var (
	ErrEmptyFieldPath         = errors.New("field path cannot be empty")
	ErrInvalidFieldPathFormat = errors.New("invalid field path format")
	ErrInvalidFieldName       = errors.New("invalid field name")
	ErrFieldPathTooDeep       = errors.New("field path exceeds maximum depth")
)
