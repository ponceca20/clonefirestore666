package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimestampParser_IsTimestampString(t *testing.T) {
	parser := NewTimestampParser()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "ISO 8601 with Z",
			input:    "2025-02-01T15:00:00Z",
			expected: true,
		},
		{
			name:     "ISO 8601 with timezone",
			input:    "2025-02-01T15:00:00-05:00",
			expected: true,
		},
		{
			name:     "ISO 8601 with milliseconds",
			input:    "2025-02-01T15:00:00.123Z",
			expected: true,
		},
		{
			name:     "Simple date",
			input:    "2025-02-01",
			expected: true,
		},
		{
			name:     "Simple datetime",
			input:    "2025-02-01 15:00:00",
			expected: true,
		},
		{
			name:     "Not a timestamp - plain string",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "Not a timestamp - number",
			input:    "12345",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Too short",
			input:    "2025",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsTimestampString(tt.input)
			assert.Equal(t, tt.expected, result, "Input: %s", tt.input)
		})
	}
}

func TestTimestampParser_ParseTimestamp(t *testing.T) {
	parser := NewTimestampParser()

	tests := []struct {
		name          string
		input         string
		shouldSucceed bool
		expectedYear  int
	}{
		{
			name:          "ISO 8601 with Z",
			input:         "2025-02-01T15:00:00Z",
			shouldSucceed: true,
			expectedYear:  2025,
		},
		{
			name:          "ISO 8601 with timezone",
			input:         "2025-02-01T15:00:00-05:00",
			shouldSucceed: true,
			expectedYear:  2025,
		},
		{
			name:          "Date only",
			input:         "2025-02-01",
			shouldSucceed: true,
			expectedYear:  2025,
		},
		{
			name:          "Invalid format",
			input:         "not a date",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseTimestamp(tt.input)
			if tt.shouldSucceed {
				assert.NoError(t, err, "Input: %s", tt.input)
				assert.Equal(t, tt.expectedYear, result.Year(), "Input: %s", tt.input)
			} else {
				assert.Error(t, err, "Input: %s", tt.input)
			}
		})
	}
}

// DEPRECATED TEST - field name detection no longer used
// All timestamp detection is now value-based only
/*
func TestIsLikelyTimestampField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{
			name:      "fechaFabricacion - Spanish date field",
			fieldName: "fechaFabricacion",
			expected:  true,
		},
		{
			name:      "createdAt - English date field",
			fieldName: "createdAt",
			expected:  true,
		},
		{
			name:      "timestamp - generic timestamp",
			fieldName: "timestamp",
			expected:  true,
		},
		{
			name:      "productId - not a date field",
			fieldName: "productId",
			expected:  false,
		},
		{
			name:      "name - not a date field",
			fieldName: "name",
			expected:  false,
		},
		{
			name:      "Date - capitalized",
			fieldName: "Date",
			expected:  true,
		},
		{
			name:      "modified - contains date keyword",
			fieldName: "modified",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLikelyTimestampField(tt.fieldName)
			assert.Equal(t, tt.expected, result, "Field name: %s", tt.fieldName)		})
	}
}
*/

// DEPRECATED TEST - field name detection no longer used
/*
func TestTimestampParser_SmartTimestampDetection(t *testing.T) {
	parser := NewTimestampParser()

	tests := []struct {
		name         string
		fieldName    string
		value        interface{}
		expectTime   bool
		expectedYear int
	}{
		{
			name:         "fechaFabricacion with ISO timestamp",
			fieldName:    "fechaFabricacion",
			value:        "2025-02-01T15:00:00Z",
			expectTime:   true,
			expectedYear: 2025,
		},
		{
			name:         "productId with timestamp-like string",
			fieldName:    "productId",
			value:        "2025-02-01T15:00:00Z",
			expectTime:   true, // Should still detect based on content
			expectedYear: 2025,
		},
		{
			name:       "fechaFabricacion with non-timestamp string",
			fieldName:  "fechaFabricacion",
			value:      "not a timestamp",
			expectTime: false,
		},
		{
			name:       "name with regular string",
			fieldName:  "name",
			value:      "Product Name",
			expectTime: false,
		},
		{
			name:       "non-string value",
			fieldName:  "fechaFabricacion",
			value:      123,
			expectTime: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isTime := parser.SmartTimestampDetection(tt.fieldName, tt.value)
			assert.Equal(t, tt.expectTime, isTime, "Field: %s, Value: %v", tt.fieldName, tt.value)
			if tt.expectTime {
				assert.Equal(t, tt.expectedYear, result.Year(), "Field: %s, Value: %v", tt.fieldName, tt.value)
			}
		})
	}
}
*/

// NewFieldValueWithContext test still works because it now just calls NewFieldValue
func TestNewFieldValueWithContext(t *testing.T) {
	tests := []struct {
		name         string
		fieldName    string
		value        interface{}
		expectedType FieldValueType
	}{
		{
			name:         "fechaFabricacion with timestamp string",
			fieldName:    "fechaFabricacion",
			value:        "2025-02-01T15:00:00Z",
			expectedType: FieldTypeTimestamp,
		},
		{
			name:         "normal string field",
			fieldName:    "name",
			value:        "Product Name",
			expectedType: FieldTypeString,
		},
		{
			name:         "number field",
			fieldName:    "price",
			value:        199.99,
			expectedType: FieldTypeDouble,
		},
		{
			name:         "boolean field",
			fieldName:    "available",
			value:        true,
			expectedType: FieldTypeBool,
		},
		{
			name:         "actual time.Time value",
			fieldName:    "created",
			value:        time.Now(),
			expectedType: FieldTypeTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewFieldValueWithContext(tt.fieldName, tt.value)
			assert.Equal(t, tt.expectedType, result.ValueType, "Field: %s, Value: %v", tt.fieldName, tt.value)

			// Additional validation for timestamp fields
			if tt.expectedType == FieldTypeTimestamp {
				_, ok := result.Value.(time.Time)
				assert.True(t, ok, "Expected time.Time value for timestamp field")
			}
		})
	}
}

func TestNewFieldValue_BackwardCompatibility(t *testing.T) {
	// Test that the enhanced NewFieldValue still works for automatic detection
	tests := []struct {
		name         string
		value        interface{}
		expectedType FieldValueType
	}{
		{
			name:         "timestamp string auto-detected",
			value:        "2025-02-01T15:00:00Z",
			expectedType: FieldTypeTimestamp,
		},
		{
			name:         "regular string",
			value:        "not a timestamp",
			expectedType: FieldTypeString,
		},
		{
			name:         "time.Time value",
			value:        time.Now(),
			expectedType: FieldTypeTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewFieldValue(tt.value)
			assert.Equal(t, tt.expectedType, result.ValueType, "Value: %v", tt.value)
		})
	}
}
