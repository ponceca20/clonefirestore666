package service

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"log"
)

// SchemaInferenceService provides field type inference capabilities
// following Firestore's type system for ordering operations
type SchemaInferenceService interface {
	InferFieldType(ctx context.Context, collectionPath string, fieldPath string) (model.FieldValueType, error)
}

// DocumentBasedSchemaInferenceService infers field types by examining existing documents
// This approach is similar to how Firestore handles dynamic schemas
type DocumentBasedSchemaInferenceService struct {
	queryEngine repository.QueryEngine
}

// NewDocumentBasedSchemaInferenceService creates a new schema inference service
func NewDocumentBasedSchemaInferenceService(queryEngine repository.QueryEngine) SchemaInferenceService {
	return &DocumentBasedSchemaInferenceService{
		queryEngine: queryEngine,
	}
}

// InferFieldType infers the field type by examining a sample of existing documents
// This follows Firestore's approach where field types are determined by actual data
func (s *DocumentBasedSchemaInferenceService) InferFieldType(ctx context.Context, collectionPath string, fieldPath string) (model.FieldValueType, error) {
	log.Printf("[SchemaInferenceService] Inferring type for field %s in collection %s", fieldPath, collectionPath)

	// Create a simple query to get a sample of documents
	// We only need a small sample to infer the type
	sampleQuery := model.Query{
		CollectionID: extractCollectionID(collectionPath),
		Path:         extractBasePath(collectionPath),
		Limit:        5,                // Small sample is sufficient for type inference
		Filters:      []model.Filter{}, // No filters, just get any documents
		Orders:       []model.Order{},  // No ordering needed for sampling
	}

	documents, err := s.queryEngine.ExecuteQuery(ctx, collectionPath, sampleQuery)
	if err != nil {
		log.Printf("[SchemaInferenceService] Error querying documents for type inference: %v", err)
		return model.FieldTypeString, nil // Default to string on error
	}

	if len(documents) == 0 {
		log.Printf("[SchemaInferenceService] No documents found, defaulting to stringValue")
		return model.FieldTypeString, nil
	}

	// Examine the field in the sample documents to infer its type
	for _, doc := range documents {
		if fieldValue := s.extractFieldValue(doc, fieldPath); fieldValue != nil {
			inferredType := model.DetermineValueType(fieldValue)
			log.Printf("[SchemaInferenceService] Inferred type %s for field %s from document sample", inferredType, fieldPath)
			return inferredType, nil
		}
	}

	log.Printf("[SchemaInferenceService] Field %s not found in sample documents, defaulting to stringValue", fieldPath)
	return model.FieldTypeString, nil
}

// extractFieldValue extracts the actual value of a field from a document
// This navigates through Firestore's nested field structure
func (s *DocumentBasedSchemaInferenceService) extractFieldValue(doc *model.Document, fieldPath string) interface{} {
	if doc.Fields == nil {
		return nil
	}

	// Navigate to the field in the Firestore document structure
	fieldValue, exists := doc.Fields[fieldPath]
	if !exists || fieldValue == nil {
		return nil
	}

	// Return the actual value from the FieldValue structure
	return fieldValue.Value
}

// extractPrimitiveValue extracts the primitive value from a Firestore field
// This handles the different value types in Firestore's format
func (s *DocumentBasedSchemaInferenceService) extractPrimitiveValue(fieldData interface{}) interface{} {
	if fieldData == nil {
		return nil
	}

	// Handle the Firestore field structure: {"stringValue": "text"}, {"doubleValue": 123}, etc.
	if fieldMap, ok := fieldData.(map[string]interface{}); ok {
		// Try different Firestore value types
		if val, exists := fieldMap["stringValue"]; exists {
			return val
		}
		if val, exists := fieldMap["doubleValue"]; exists {
			return val
		}
		if val, exists := fieldMap["integerValue"]; exists {
			return val
		}
		if val, exists := fieldMap["booleanValue"]; exists {
			return val
		}
		if val, exists := fieldMap["timestampValue"]; exists {
			return val
		}
		if val, exists := fieldMap["arrayValue"]; exists {
			return val
		}
		if val, exists := fieldMap["mapValue"]; exists {
			return val
		}
		if val, exists := fieldMap["nullValue"]; exists {
			return val
		}
		if val, exists := fieldMap["bytesValue"]; exists {
			return val
		}
		if val, exists := fieldMap["referenceValue"]; exists {
			return val
		}
		if val, exists := fieldMap["geoPointValue"]; exists {
			return val
		}
	}

	// If it's not in Firestore format, return the value as-is
	return fieldData
}

// Helper functions to extract collection info from path
func extractCollectionID(collectionPath string) string {
	// Extract the last segment as collection ID
	// e.g., "products" from "projects/proj/databases/db/documents/products"
	// This is a simple implementation, could be enhanced for nested collections
	return collectionPath
}

func extractBasePath(collectionPath string) string {
	// For now, return a default base path
	// This should be enhanced to properly parse Firestore paths
	return "projects/default/databases/default/documents"
}
