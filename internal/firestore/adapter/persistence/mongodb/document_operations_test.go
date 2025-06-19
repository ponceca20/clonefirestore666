// Package mongodb provides MongoDB implementation of Firestore operations
package mongodb

import (
	"context"
	"fmt"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
)

// testContext proporciona un contexto compartido para las pruebas
var testContext = context.Background()

// testDocumentData representa datos de prueba comunes
type testDocumentData struct {
	projectID    string
	databaseID   string
	collectionID string
	documentID   string
	fields       map[string]*model.FieldValue
}

// newTestData crea una nueva instancia de datos de prueba con valores por defecto
func newTestData() testDocumentData {
	return testDocumentData{
		projectID:    "test-project",
		databaseID:   "test-database",
		collectionID: "test-collection",
		documentID:   "test-doc",
		fields: map[string]*model.FieldValue{
			"stringField": model.NewFieldValue("test value"),
			"numberField": {
				ValueType: model.FieldTypeDouble,
				Value:     42.0,
			},
			"boolField": {
				ValueType: model.FieldTypeBool,
				Value:     true,
			},
		},
	}
}

// newComplexTestData crea datos de prueba con estructuras anidadas
func newComplexTestData() map[string]*model.FieldValue {
	return map[string]*model.FieldValue{
		"array": {
			ValueType: model.FieldTypeArray,
			Value: &model.ArrayValue{
				Values: []*model.FieldValue{
					model.NewFieldValue("item1"),
					model.NewFieldValue("item2"),
				},
			},
		},
		"map": {
			ValueType: model.FieldTypeMap,
			Value: &model.MapValue{
				Fields: map[string]*model.FieldValue{
					"key1": model.NewFieldValue("value1"),
					"key2": {
						ValueType: model.FieldTypeInt,
						Value:     "42",
					},
				},
			},
		}}
}

// TestFlattenFields verifica la función de aplanamiento de campos
func TestFlattenFields(t *testing.T) {
	t.Run("flatten basic fields", func(t *testing.T) {
		fields := map[string]*model.FieldValue{
			"string": model.NewFieldValue("test"),
			"number": {ValueType: model.FieldTypeDouble, Value: 123.45},
			"bool":   {ValueType: model.FieldTypeBool, Value: true},
			"int":    {ValueType: model.FieldTypeInt, Value: "42"},
		}

		flattened := flattenFieldsForMongoDB(fields)

		testCases := []struct {
			field    string
			key      string
			expected interface{}
		}{
			{"string", "stringValue", "test"},
			{"number", "doubleValue", 123.45},
			{"bool", "booleanValue", true},
			{"int", "integerValue", "42"},
		}

		for _, tc := range testCases {
			field, ok := flattened[tc.field].(map[string]interface{})
			if !ok {
				t.Errorf("field %s: expected map[string]interface{}, got %T", tc.field, flattened[tc.field])
				continue
			}
			if field[tc.key] != tc.expected {
				t.Errorf("field %s: expected %v, got %v", tc.field, tc.expected, field[tc.key])
			}
		}
	})

	t.Run("flatten complex fields", func(t *testing.T) {
		fields := newComplexTestData()
		flattened := flattenFieldsForMongoDB(fields)

		// Verificar estructura de array
		if arrayField, ok := flattened["array"].(map[string]interface{}); ok {
			if arrayValue, ok := arrayField["arrayValue"].(map[string]interface{}); ok {
				values := arrayValue["values"].([]map[string]interface{})
				if len(values) != 2 {
					t.Errorf("expected 2 array values, got %d", len(values))
				}
				if val, ok := values[0]["stringValue"]; !ok || val != "item1" {
					t.Errorf("expected first array value to be 'item1', got %v", val)
				}
			} else {
				t.Error("array field structure is invalid")
			}
		} else {
			t.Error("array field is not a map")
		}

		// Verificar estructura de map
		if mapField, ok := flattened["map"].(map[string]interface{}); ok {
			if mapValue, ok := mapField["mapValue"].(map[string]interface{}); ok {
				fields := mapValue["fields"].(map[string]interface{})
				if val, ok := fields["key2"].(map[string]interface{})["integerValue"]; !ok || val != "42" {
					t.Errorf("expected map.key2 value to be '42', got %v", val)
				}
			} else {
				t.Error("map field structure is invalid")
			}
		} else {
			t.Error("map field is not a map")
		}
	})
}

// TestExpandFields verifica la función de expansión de campos
func TestExpandFields(t *testing.T) {
	t.Run("expand basic fields", func(t *testing.T) {
		flattened := map[string]interface{}{
			"string": map[string]interface{}{"stringValue": "test"},
			"number": map[string]interface{}{"doubleValue": 123.45},
			"bool":   map[string]interface{}{"booleanValue": true},
			"int":    map[string]interface{}{"integerValue": "42"},
		}

		expanded := expandFieldsFromMongoDB(flattened)

		testCases := []struct {
			field        string
			expectedType model.FieldValueType
			expectedVal  interface{}
		}{
			{"string", model.FieldTypeString, "test"},
			{"number", model.FieldTypeDouble, 123.45},
			{"bool", model.FieldTypeBool, true},
			{"int", model.FieldTypeInt, "42"},
		}

		for _, tc := range testCases {
			field := expanded[tc.field]
			if field == nil {
				t.Errorf("field %s is missing", tc.field)
				continue
			}
			if field.ValueType != tc.expectedType {
				t.Errorf("field %s: expected type %v, got %v", tc.field, tc.expectedType, field.ValueType)
			}
			if field.Value != tc.expectedVal {
				t.Errorf("field %s: expected value %v, got %v", tc.field, tc.expectedVal, field.Value)
			}
		}
	})
	t.Run("expand complex fields", func(t *testing.T) {
		original := newComplexTestData()
		flattened := flattenFieldsForMongoDB(original)

		// Debug: imprimir la estructura aplanada
		t.Logf("Flattened: %+v", flattened)

		expanded := expandFieldsFromMongoDB(flattened)

		// Debug: imprimir la estructura expandida
		t.Logf("Expanded: %+v", expanded)

		// Verificar array
		if array := expanded["array"]; array == nil {
			t.Error("array field is missing")
		} else if array.ValueType != model.FieldTypeArray {
			t.Error("array field has wrong type")
		} else if arr, ok := array.Value.(*model.ArrayValue); !ok {
			t.Error("array field value is not ArrayValue")
		} else if len(arr.Values) != 2 {
			t.Errorf("expected 2 array values, got %d", len(arr.Values))
		}

		// Verificar map
		if mapField := expanded["map"]; mapField == nil {
			t.Error("map field is missing")
		} else if mapField.ValueType != model.FieldTypeMap {
			t.Error("map field has wrong type")
		} else if m, ok := mapField.Value.(*model.MapValue); !ok {
			t.Error("map field value is not MapValue")
		} else if val := m.Fields["key2"]; val == nil || val.Value != "42" {
			t.Errorf("expected map.key2 value to be '42', got %v", val)
		}
	})
}

// TestDocumentOperations_CRUD verifica las operaciones CRUD básicas
func TestDocumentOperations_CRUD(t *testing.T) {
	repo := NewTestDocumentRepositoryForOps()
	docs := NewDocumentOperations(repo)
	data := newTestData()

	t.Run("create document", func(t *testing.T) {
		doc, err := docs.CreateDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, data.documentID, data.fields)
		if err != nil {
			t.Fatalf("CreateDocument failed: %v", err)
		}
		if doc.DocumentID != data.documentID {
			t.Errorf("expected documentID %s, got %s", data.documentID, doc.DocumentID)
		}
		validateFields(t, doc.Fields, data.fields)
	})

	t.Run("get document", func(t *testing.T) {
		doc, err := docs.GetDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, data.documentID)
		if err != nil {
			t.Fatalf("GetDocument failed: %v", err)
		}
		if doc == nil {
			t.Fatal("expected document, got nil")
		}
		validateFields(t, doc.Fields, data.fields)
	})

	t.Run("update document", func(t *testing.T) {
		updateFields := map[string]*model.FieldValue{
			"stringField": model.NewFieldValue("updated value"),
			"numberField": {ValueType: model.FieldTypeDouble, Value: 99.9},
		}

		doc, err := docs.UpdateDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, data.documentID, updateFields, nil)
		if err != nil {
			t.Fatalf("UpdateDocument failed: %v", err)
		}

		validateFields(t, doc.Fields, updateFields)
	})

	t.Run("delete document", func(t *testing.T) {
		err := docs.DeleteDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, data.documentID)
		if err != nil {
			t.Fatalf("DeleteDocument failed: %v", err)
		}

		// Verificar que el documento fue eliminado
		doc, err := docs.GetDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, data.documentID)
		if err == nil || doc != nil {
			t.Error("expected error or nil document after deletion")
		}
	})
}

// validateFields ayuda a verificar que los campos coincidan con los esperados
func validateFields(t *testing.T, got, want map[string]*model.FieldValue) {
	t.Helper()
	for k, wantField := range want {
		gotField, exists := got[k]
		if !exists {
			t.Errorf("field %s missing", k)
			continue
		}
		if gotField.ValueType != wantField.ValueType {
			t.Errorf("field %s: type mismatch. Want %v, got %v",
				k, wantField.ValueType, gotField.ValueType)
		}
		if gotField.Value != wantField.Value {
			t.Errorf("field %s: value mismatch. Want %v, got %v",
				k, wantField.Value, gotField.Value)
		}
	}
}

// TestDocumentOperations_ListDocuments verifica el listado de documentos
func TestDocumentOperations_ListDocuments(t *testing.T) {
	repo, cleanup := NewTestDocumentRepositoryForOpsWithCleanup()
	defer cleanup() // Clean up at the end

	docs := NewDocumentOperations(repo)
	data := newTestData()

	// Crear documentos de prueba
	for i := 1; i <= 3; i++ {
		docID := fmt.Sprintf("doc%d", i)
		data.documentID = docID
		_, err := docs.CreateDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, docID, data.fields)
		if err != nil {
			t.Fatalf("failed to create test document %s: %v", docID, err)
		}
	}

	t.Run("list all documents", func(t *testing.T) {
		docsList, _, err := docs.ListDocuments(testContext, data.projectID,
			data.databaseID, data.collectionID, 10, "", "", false)
		if err != nil {
			t.Fatalf("ListDocuments failed: %v", err)
		}
		if len(docsList) != 3 {
			t.Errorf("expected 3 documents, got %d", len(docsList))
		}
		for _, doc := range docsList {
			validateFields(t, doc.Fields, data.fields)
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		// Obtener primera página
		docsList, nextPageToken, err := docs.ListDocuments(testContext,
			data.projectID, data.databaseID, data.collectionID, 2, "", "", false)
		if err != nil {
			t.Fatalf("ListDocuments (page 1) failed: %v", err)
		}
		if len(docsList) != 2 {
			t.Errorf("expected 2 documents in first page, got %d", len(docsList))
		}
		if nextPageToken == "" {
			t.Error("expected nextPageToken for pagination")
		}

		// Obtener segunda página
		remainingDocs, _, err := docs.ListDocuments(testContext,
			data.projectID, data.databaseID, data.collectionID,
			2, nextPageToken, "", false)
		if err != nil {
			t.Fatalf("ListDocuments (page 2) failed: %v", err)
		}
		if len(remainingDocs) != 1 {
			t.Errorf("expected 1 document in second page, got %d", len(remainingDocs))
		}
	})
}

// TestDocumentOperations_Errors verifica el manejo de errores
func TestDocumentOperations_Errors(t *testing.T) {
	repo := NewTestDocumentRepositoryForOps()
	docs := NewDocumentOperations(repo)
	data := newTestData()

	t.Run("get non-existent document", func(t *testing.T) {
		doc, err := docs.GetDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, "non-existent")
		if err == nil {
			t.Error("expected error when getting non-existent document")
		}
		if doc != nil {
			t.Error("expected nil document for non-existent ID")
		}
	})

	t.Run("update non-existent document", func(t *testing.T) {
		_, err := docs.UpdateDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, "non-existent",
			map[string]*model.FieldValue{"field": model.NewFieldValue("value")}, nil)
		if err == nil {
			t.Error("expected error when updating non-existent document")
		}
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		err := docs.DeleteDocument(testContext, data.projectID, data.databaseID,
			data.collectionID, "non-existent")
		if err == nil {
			t.Error("expected error when deleting non-existent document")
		}
	})

	t.Run("invalid document path", func(t *testing.T) {
		_, err := docs.GetDocumentByPath(testContext, "invalid/path")
		if err == nil {
			t.Error("expected error for invalid document path")
		}
	})
}
