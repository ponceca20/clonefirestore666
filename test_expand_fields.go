package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// Simulamos la función expandFieldsFromMongoDB
func expandFieldsFromMongoDB(mongoFields interface{}) map[string]interface{} {
	fmt.Printf("[DEBUG test] Input mongoFields type: %T, value: %+v\n", mongoFields, mongoFields)

	if mongoFields == nil {
		return nil
	}

	fieldsMap, ok := mongoFields.(map[string]interface{})
	if !ok {
		log.Printf("[ERROR test] mongoFields is not a map[string]interface{}, got: %T", mongoFields)
		return nil
	}

	expandedFields := make(map[string]interface{})

	for fieldName, fieldValue := range fieldsMap {
		fmt.Printf("[DEBUG test] Processing field '%s' with value type: %T, value: %+v\n", fieldName, fieldValue, fieldValue)

		if fieldValue == nil {
			continue
		}

		// Verificar si es un array
		if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
			if arrayValue, hasArrayValue := fieldMap["arrayValue"]; hasArrayValue {
				fmt.Printf("[DEBUG test] Found arrayValue for field '%s': %+v\n", fieldName, arrayValue)
				expandedFields[fieldName] = fieldValue
				continue
			}
		}

		expandedFields[fieldName] = fieldValue
	}

	fmt.Printf("[DEBUG test] Final expandedFields: %+v\n", expandedFields)
	return expandedFields
}

func main() {
	// Simulamos datos de MongoDB con arrays
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
	}

	fmt.Println("=== TESTING expandFieldsFromMongoDB ===")
	result := expandFieldsFromMongoDB(mongoData)

	fmt.Println("\n=== RESULT ===")
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(resultJSON))

	// Verificar si los arrays están presentes
	fmt.Println("\n=== VERIFICATION ===")
	if _, hasName := result["name"]; hasName {
		fmt.Println("✅ name field present")
	} else {
		fmt.Println("❌ name field missing")
	}

	if _, hasPrice := result["price"]; hasPrice {
		fmt.Println("✅ price field present")
	} else {
		fmt.Println("❌ price field missing")
	}

	if _, hasTags := result["tags"]; hasTags {
		fmt.Println("✅ tags field present")
	} else {
		fmt.Println("❌ tags field missing")
	}

	if _, hasCategories := result["categories"]; hasCategories {
		fmt.Println("✅ categories field present")
	} else {
		fmt.Println("❌ categories field missing")
	}
}
