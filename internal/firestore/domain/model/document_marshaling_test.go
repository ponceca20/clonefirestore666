package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFieldValue_MarshalJSON_Timestamp(t *testing.T) {
	// Test timestamp marshaling
	testTime := time.Date(2025, 6, 10, 11, 0, 0, 0, time.UTC)
	fv := &FieldValue{
		ValueType: FieldTypeTimestamp,
		Value:     testTime,
	}

	data, err := json.Marshal(fv)
	assert.NoError(t, err)

	expected := `{"timestampValue":"2025-06-10T11:00:00Z"}`
	assert.JSONEq(t, expected, string(data))
}

func TestFieldValue_MarshalJSON_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		fv       *FieldValue
		expected string
	}{
		{
			name: "boolean",
			fv: &FieldValue{
				ValueType: FieldTypeBool,
				Value:     true,
			},
			expected: `{"booleanValue":true}`,
		},
		{
			name: "string",
			fv: &FieldValue{
				ValueType: FieldTypeString,
				Value:     "test",
			},
			expected: `{"stringValue":"test"}`,
		},
		{
			name: "integer",
			fv: &FieldValue{
				ValueType: FieldTypeInt,
				Value:     int64(42),
			},
			expected: `{"integerValue":"42"}`,
		},
		{
			name: "double",
			fv: &FieldValue{
				ValueType: FieldTypeDouble,
				Value:     3.14,
			},
			expected: `{"doubleValue":3.14}`,
		},
		{
			name: "null",
			fv: &FieldValue{
				ValueType: FieldTypeNull,
				Value:     nil,
			},
			expected: `{"nullValue":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.fv)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestFieldValue_UnmarshalJSON_Timestamp(t *testing.T) {
	// Test timestamp unmarshaling
	jsonData := `{"timestampValue":"2025-06-10T11:00:00Z"}`

	var fv FieldValue
	err := json.Unmarshal([]byte(jsonData), &fv)
	assert.NoError(t, err)

	assert.Equal(t, FieldTypeTimestamp, fv.ValueType)

	timeVal, ok := fv.Value.(time.Time)
	assert.True(t, ok)

	expected := time.Date(2025, 6, 10, 11, 0, 0, 0, time.UTC)
	assert.True(t, timeVal.Equal(expected))
}

func TestFieldValue_UnmarshalJSON_AllTypes(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedType  FieldValueType
		expectedValue interface{}
	}{
		{
			name:          "boolean",
			jsonData:      `{"booleanValue":true}`,
			expectedType:  FieldTypeBool,
			expectedValue: true,
		},
		{
			name:          "string",
			jsonData:      `{"stringValue":"test"}`,
			expectedType:  FieldTypeString,
			expectedValue: "test",
		},
		{
			name:          "integer",
			jsonData:      `{"integerValue":"42"}`,
			expectedType:  FieldTypeInt,
			expectedValue: "42",
		},
		{
			name:          "double",
			jsonData:      `{"doubleValue":3.14}`,
			expectedType:  FieldTypeDouble,
			expectedValue: 3.14,
		},
		{
			name:          "null",
			jsonData:      `{"nullValue":null}`,
			expectedType:  FieldTypeNull,
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fv FieldValue
			err := json.Unmarshal([]byte(tt.jsonData), &fv)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedType, fv.ValueType)
			assert.Equal(t, tt.expectedValue, fv.Value)
		})
	}
}

func TestFieldValue_RoundTrip(t *testing.T) {
	// Test that marshal + unmarshal preserves the original data
	original := &FieldValue{
		ValueType: FieldTypeTimestamp,
		Value:     time.Date(2025, 6, 10, 11, 0, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Unmarshal
	var restored FieldValue
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)

	// Compare
	assert.Equal(t, original.ValueType, restored.ValueType)

	originalTime := original.Value.(time.Time)
	restoredTime := restored.Value.(time.Time)
	assert.True(t, originalTime.Equal(restoredTime))
}
