package mongodb

import (
	"strings"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoFieldPathResolver_ResolveFieldPath(t *testing.T) {
	resolver := NewMongoFieldPathResolver()

	testCases := []struct {
		name        string
		fieldPath   string
		valueType   model.FieldValueType
		expected    string
		shouldError bool
	}{
		// Simple fields
		{
			name:        "Simple string field",
			fieldPath:   "status",
			valueType:   model.FieldTypeString,
			expected:    "fields.status.stringValue",
			shouldError: false,
		},
		{
			name:        "Simple boolean field",
			fieldPath:   "active",
			valueType:   model.FieldTypeBool,
			expected:    "fields.active.booleanValue",
			shouldError: false,
		},
		{
			name:        "Simple integer field",
			fieldPath:   "count",
			valueType:   model.FieldTypeInt,
			expected:    "fields.count.integerValue",
			shouldError: false,
		}, {
			name:        "Simple array field",
			fieldPath:   "tags",
			valueType:   model.FieldTypeArray,
			expected:    "fields.tags.arrayValue.values",
			shouldError: false,
		},

		// Nested fields level 1
		{
			name:        "Nested field level 1 - customer.ruc",
			fieldPath:   "customer.ruc",
			valueType:   model.FieldTypeString,
			expected:    "fields.customer.value.ruc",
			shouldError: false,
		},
		{
			name:        "Nested field level 1 - user.age",
			fieldPath:   "user.age",
			valueType:   model.FieldTypeInt,
			expected:    "fields.user.value.age",
			shouldError: false,
		},
		// Nested fields level 2
		{
			name:        "Nested field level 2 - customer.address.city",
			fieldPath:   "customer.address.city",
			valueType:   model.FieldTypeString,
			expected:    "fields.customer.value.address.value.city",
			shouldError: false,
		},
		{
			name:        "Nested field level 2 - user.profile.settings",
			fieldPath:   "user.profile.settings",
			valueType:   model.FieldTypeMap,
			expected:    "fields.user.value.profile.value.settings",
			shouldError: false,
		},

		// Deep nested fields
		{
			name:        "Deep nested field",
			fieldPath:   "user.profile.preferences.theme.colors.primary",
			valueType:   model.FieldTypeString,
			expected:    "fields.user.value.profile.value.preferences.value.theme.value.colors.value.primary",
			shouldError: false,
		},

		// Default value type (empty)
		{
			name:        "Simple field with no type defaults to string",
			fieldPath:   "description",
			valueType:   "",
			expected:    "fields.description.stringValue",
			shouldError: false,
		},

		// Error cases
		{
			name:        "Invalid field path",
			fieldPath:   "invalid..path",
			valueType:   model.FieldTypeString,
			expected:    "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fieldPath, err := model.NewFieldPath(tc.fieldPath)
			if tc.shouldError && err != nil {
				// Expected error during field path creation
				return
			}
			require.NoError(t, err, "Failed to create field path")

			result, err := resolver.ResolveFieldPath(fieldPath, tc.valueType)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMongoFieldPathResolver_ResolveOrderFieldPath(t *testing.T) {
	resolver := NewMongoFieldPathResolver()

	testCases := []struct {
		name      string
		fieldPath string
		valueType model.FieldValueType
		expected  string
	}{
		{
			name:      "Order by simple field with type",
			fieldPath: "timestamp",
			valueType: model.FieldTypeTimestamp,
			expected:  "fields.timestamp.timestampValue",
		},
		{
			name:      "Order by simple field without type defaults to string",
			fieldPath: "name",
			valueType: "",
			expected:  "fields.name.stringValue",
		}, {
			name:      "Order by nested field",
			fieldPath: "customer.name",
			valueType: model.FieldTypeString,
			expected:  "fields.customer.value.name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fieldPath := model.MustNewFieldPath(tc.fieldPath)

			result, err := resolver.ResolveOrderFieldPath(fieldPath, tc.valueType)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMongoFieldPathResolver_ResolveArrayFieldPath(t *testing.T) {
	resolver := NewMongoFieldPathResolver().(*MongoFieldPathResolver)

	testCases := []struct {
		name        string
		fieldPath   string
		expected    string
		shouldError bool
	}{{
		name:        "Simple array field",
		fieldPath:   "tags",
		expected:    "fields.tags.arrayValue.values",
		shouldError: false,
	},
		{
			name:        "Array field with items",
			fieldPath:   "items",
			expected:    "fields.items.arrayValue.values",
			shouldError: false,
		},
		{
			name:        "Nested field should error for arrays",
			fieldPath:   "customer.orders",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fieldPath := model.MustNewFieldPath(tc.fieldPath)

			result, err := resolver.ResolveArrayFieldPath(fieldPath)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMongoFieldPathResolver_InferValueTypeFromPath(t *testing.T) {
	resolver := NewMongoFieldPathResolver().(*MongoFieldPathResolver)

	testCases := []struct {
		name      string
		fieldPath string
		value     interface{}
		expected  model.FieldValueType
	}{
		{
			name:      "String value",
			fieldPath: "name",
			value:     "John Doe",
			expected:  model.FieldTypeString,
		},
		{
			name:      "Boolean value",
			fieldPath: "active",
			value:     true,
			expected:  model.FieldTypeBool,
		},
		{
			name:      "Integer value",
			fieldPath: "count",
			value:     42,
			expected:  model.FieldTypeInt,
		},
		{
			name:      "Float value",
			fieldPath: "price",
			value:     19.99,
			expected:  model.FieldTypeDouble,
		},
		{
			name:      "Array value",
			fieldPath: "tags",
			value:     []interface{}{"tag1", "tag2"},
			expected:  model.FieldTypeArray,
		},
		{
			name:      "Map value",
			fieldPath: "metadata",
			value:     map[string]interface{}{"key": "value"},
			expected:  model.FieldTypeMap,
		},
		{
			name:      "Nil value",
			fieldPath: "optional",
			value:     nil,
			expected:  model.FieldTypeNull,
		},
		{
			name:      "Unknown type defaults to string",
			fieldPath: "unknown",
			value:     struct{ Field string }{Field: "value"},
			expected:  model.FieldTypeString,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fieldPath := model.MustNewFieldPath(tc.fieldPath)

			result := resolver.InferValueTypeFromPath(fieldPath, tc.value)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMongoFieldPathResolver_Capabilities(t *testing.T) {
	resolver := NewMongoFieldPathResolver().(*MongoFieldPathResolver)

	// Test individual capability methods
	assert.True(t, resolver.SupportsNestedQueries())
	assert.True(t, resolver.SupportsArrayQueries())
	assert.Equal(t, 100, resolver.GetMaxNestingDepth())

	// Test full capabilities
	capabilities := resolver.GetCapabilities()
	assert.True(t, capabilities.SupportsNestedFields)
	assert.True(t, capabilities.SupportsArrayContains)
	assert.True(t, capabilities.SupportsArrayContainsAny)
	assert.True(t, capabilities.SupportsCompositeFilters)
	assert.True(t, capabilities.SupportsOrderBy)
	assert.True(t, capabilities.SupportsCursorPagination)
	assert.True(t, capabilities.SupportsOffsetPagination)
	assert.True(t, capabilities.SupportsProjection)
	assert.Equal(t, 100, capabilities.MaxFilterCount)
	assert.Equal(t, 32, capabilities.MaxOrderByCount)
	assert.Equal(t, 100, capabilities.MaxNestingDepth)
}

func TestMongoFieldPathResolver_ErrorCases(t *testing.T) {
	resolver := NewMongoFieldPathResolver()

	t.Run("Nil field path", func(t *testing.T) {
		result, err := resolver.ResolveFieldPath(nil, model.FieldTypeString)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNilFieldPath)
		assert.Empty(t, result)
	})

	t.Run("Very deep nesting", func(t *testing.T) {
		// Create a field path that exceeds maximum depth
		segments := make([]string, 101) // More than max depth of 100
		for i := range segments {
			segments[i] = "field"
		}

		// This should fail during creation
		_, err := model.NewFieldPath(strings.Join(segments, "."))
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkMongoFieldPathResolver_SimpleField(b *testing.B) {
	resolver := NewMongoFieldPathResolver()
	fieldPath := model.MustNewFieldPath("status")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.ResolveFieldPath(fieldPath, model.FieldTypeString)
	}
}

func BenchmarkMongoFieldPathResolver_NestedField(b *testing.B) {
	resolver := NewMongoFieldPathResolver()
	fieldPath := model.MustNewFieldPath("customer.address.city")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.ResolveFieldPath(fieldPath, model.FieldTypeString)
	}
}

func BenchmarkMongoFieldPathResolver_DeepNestedField(b *testing.B) {
	resolver := NewMongoFieldPathResolver()
	fieldPath := model.MustNewFieldPath("user.profile.preferences.theme.colors.primary")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.ResolveFieldPath(fieldPath, model.FieldTypeString)
	}
}
