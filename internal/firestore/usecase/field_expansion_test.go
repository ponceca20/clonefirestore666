package usecase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// expandFieldsFromMongoDB converts MongoDB field format to Firestore format
// This function handles the conversion of MongoDB's type-specific field structure
// to Firestore's expected format, particularly for arrays and complex types
func expandFieldsFromMongoDB(mongoFields interface{}) map[string]interface{} {
	if mongoFields == nil {
		return nil
	}

	fieldsMap, ok := mongoFields.(map[string]interface{})
	if !ok {
		return nil
	}

	expandedFields := make(map[string]interface{})

	for fieldName, fieldValue := range fieldsMap {
		if fieldValue == nil {
			continue
		}
		// Verify if it's an array with arrayValue structure (Firestore format)
		if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
			if _, hasArrayValue := fieldMap["arrayValue"]; hasArrayValue {
				expandedFields[fieldName] = fieldValue
				continue
			}
		}

		expandedFields[fieldName] = fieldValue
	}

	return expandedFields
}

// TestExpandFieldsFromMongoDB verifies the conversion of MongoDB fields to Firestore format
// This test ensures proper handling of arrays and different field types following Firestore specifications
func TestExpandFieldsFromMongoDB(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "invalid type input",
			input:    "not a map",
			expected: nil,
		},
		{
			name: "simple fields",
			input: map[string]interface{}{
				"name": map[string]interface{}{
					"stringValue": "Test Product",
				},
				"price": map[string]interface{}{
					"doubleValue": 299.99,
				},
			},
			expected: map[string]interface{}{
				"name": map[string]interface{}{
					"stringValue": "Test Product",
				},
				"price": map[string]interface{}{
					"doubleValue": 299.99,
				},
			},
		},
		{
			name: "array fields with Firestore format",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "gaming"},
							map[string]interface{}{"stringValue": "monitor"},
						},
					},
				},
				"categories": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "electronics"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"tags": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "gaming"},
							map[string]interface{}{"stringValue": "monitor"},
						},
					},
				},
				"categories": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "electronics"},
						},
					},
				},
			},
		},
		{
			name: "mixed fields with arrays and simple values",
			input: map[string]interface{}{
				"name": map[string]interface{}{
					"stringValue": "Monitor Gaming",
				},
				"price": map[string]interface{}{
					"doubleValue": 599.99,
				},
				"tags": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "gaming"},
							map[string]interface{}{"stringValue": "curved"},
						},
					},
				},
				"inStock": map[string]interface{}{
					"booleanValue": true,
				},
			},
			expected: map[string]interface{}{
				"name": map[string]interface{}{
					"stringValue": "Monitor Gaming",
				},
				"price": map[string]interface{}{
					"doubleValue": 599.99,
				},
				"tags": map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": []interface{}{
							map[string]interface{}{"stringValue": "gaming"},
							map[string]interface{}{"stringValue": "curved"},
						},
					},
				},
				"inStock": map[string]interface{}{
					"booleanValue": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandFieldsFromMongoDB(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExpandFieldsFromMongoDB_ComplexScenario tests a comprehensive scenario
// that combines multiple Firestore field types to ensure full compatibility
func TestExpandFieldsFromMongoDB_ComplexScenario(t *testing.T) {
	mongoData := map[string]interface{}{
		"name": map[string]interface{}{
			"stringValue": "Monitor Gaming",
		},
		"price": map[string]interface{}{
			"doubleValue": 599.99,
		},
		"tags": map[string]interface{}{
			"arrayValue": map[string]interface{}{
				"values": []interface{}{
					map[string]interface{}{"stringValue": "gaming"},
					map[string]interface{}{"stringValue": "monitor"},
					map[string]interface{}{"stringValue": "curved"},
				},
			},
		},
		"categories": map[string]interface{}{
			"arrayValue": map[string]interface{}{
				"values": []interface{}{
					map[string]interface{}{"stringValue": "electronics"},
					map[string]interface{}{"stringValue": "gaming"},
				},
			},
		},
		"nullField": nil,
	}

	result := expandFieldsFromMongoDB(mongoData)

	// Verify all expected fields are present
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "price")
	assert.Contains(t, result, "tags")
	assert.Contains(t, result, "categories")

	// Verify null fields are excluded
	assert.NotContains(t, result, "nullField")

	// Verify array structure is preserved
	tagsField := result["tags"].(map[string]interface{})
	assert.Contains(t, tagsField, "arrayValue")

	categoriesField := result["categories"].(map[string]interface{})
	assert.Contains(t, categoriesField, "arrayValue")

	// Verify the result can be marshaled to JSON (important for Firestore compatibility)
	_, err := json.Marshal(result)
	assert.NoError(t, err, "Result should be JSON serializable for Firestore compatibility")
}

// BenchmarkExpandFieldsFromMongoDB benchmarks the field expansion performance
// Important for ensuring the conversion doesn't become a bottleneck in the Firestore clone
func BenchmarkExpandFieldsFromMongoDB(b *testing.B) {
	mongoData := map[string]interface{}{
		"name": map[string]interface{}{
			"stringValue": "Monitor Gaming",
		},
		"price": map[string]interface{}{
			"doubleValue": 599.99,
		},
		"tags": map[string]interface{}{
			"arrayValue": map[string]interface{}{
				"values": []interface{}{
					map[string]interface{}{"stringValue": "gaming"},
					map[string]interface{}{"stringValue": "monitor"},
					map[string]interface{}{"stringValue": "curved"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expandFieldsFromMongoDB(mongoData)
	}
}
