package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PostmanLikeIntegrationTest simula exactamente el comportamiento de Postman
// para identificar dónde está fallando la cadena de filtros
func TestPostmanLikeIntegration_FilterComparison(t *testing.T) {
	// Crear un caso de uso mock que simule datos reales
	mockUC := &PostmanLikeMockUsecase{
		documents: createMockProductDocuments(),
	}

	// Crear un handler de prueba
	handler := createTestHandler(mockUC)

	t.Run("Step 1: Verify GET returns documents (like Postman GET)", func(t *testing.T) {
		// Simular GET que devuelve todos los documentos
		req := httptest.NewRequest("GET", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos", nil)
		req.Header.Set("Cookie", "auth_token=test-token")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Verificar que GET devuelve documentos (como en Postman)
		assert.Equal(t, http.StatusOK, w.Code)

		var getResult map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &getResult)
		require.NoError(t, err)

		// El GET debe devolver documentos
		documents, ok := getResult["documents"].([]interface{})
		require.True(t, ok, "GET response should have documents array")
		assert.Greater(t, len(documents), 0, "GET should return documents like in Postman")

		t.Logf("GET returned %d documents", len(documents))

		// Verificar que hay documentos con active=true
		activeDocuments := 0
		for _, doc := range documents {
			docMap, ok := doc.(map[string]interface{})
			if !ok {
				continue
			}
			fields, ok := docMap["fields"].(map[string]interface{})
			if !ok {
				continue
			}
			if active, exists := fields["active"]; exists {
				activeValue := extractFieldValue(active)
				if activeBool, ok := activeValue.(bool); ok && activeBool {
					activeDocuments++
				}
			}
		}
		t.Logf("Found %d documents with active=true", activeDocuments)
		assert.Greater(t, activeDocuments, 0, "Should have documents with active=true for filtering")
	})

	t.Run("Step 2: Test POST query with boolean filter (like Postman POST)", func(t *testing.T) {
		// JSON exacto como el que envía Postman
		postmanQuery := `{
			"from": [
				{
					"collectionId": "productos"
				}
			],
			"where": {
				"fieldFilter": {
					"field": {
						"fieldPath": "active"
					},
					"op": "EQUAL",
					"value": {
						"booleanValue": true
					}
				}
			},
			"orderBy": [
				{
					"field": {
						"fieldPath": "name"
					},
					"direction": "ASCENDING"
				}
			],
			"limit": 10
		}`

		req := httptest.NewRequest("POST", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/new-db-from-postman/query/productos", bytes.NewReader([]byte(postmanQuery)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "auth_token=test-token")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Este es el test crítico - debería devolver documentos filtrados, no count:0
		assert.Equal(t, http.StatusOK, w.Code)

		var postResult map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &postResult)
		require.NoError(t, err)

		t.Logf("POST Response: %s", w.Body.String())

		// Verificar que la respuesta contiene los campos esperados
		assert.Contains(t, postResult, "documents", "Response should contain 'documents' field")
		assert.Contains(t, postResult, "count", "Response should contain 'count' field")

		// ESTE ES EL PROBLEMA: En Postman devuelve count:0, documents:null
		// Pero debería devolver documentos filtrados
		count := postResult["count"]
		documents := postResult["documents"]

		if count == float64(0) && documents == nil {
			t.Errorf("❌ PROBLEM IDENTIFIED: POST query returns count:0, documents:null like in Postman")
			t.Errorf("   This means the filter is not working correctly")
			t.Errorf("   Expected: documents with active=true")
			t.Errorf("   Actual: no documents returned")
		} else {
			t.Logf("✅ Filter working: returned %v documents", count)
		}
	})

	t.Run("Step 3: Debug filter conversion", func(t *testing.T) {
		// Test directo de la conversión de filtros
		mockUC.enableDebug = true

		postmanQuery := `{
			"from": [{"collectionId": "productos"}],
			"where": {
				"fieldFilter": {
					"field": {"fieldPath": "active"},
					"op": "EQUAL",
					"value": {"booleanValue": true}
				}
			}
		}`

		req := httptest.NewRequest("POST", "/api/v1/organizations/test/projects/test/databases/test/query/productos", bytes.NewReader([]byte(postmanQuery)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Verificar que el mock recibió el filtro correctamente convertido
		assert.True(t, mockUC.lastQueryReceived != nil, "Should have received a query")
		if mockUC.lastQueryReceived != nil {
			require.Len(t, mockUC.lastQueryReceived.Filters, 1, "Should have exactly one filter")
			filter := mockUC.lastQueryReceived.Filters[0]

			t.Logf("Filter received: Field=%s, Operator=%s, Value=%v (type: %T)",
				filter.Field, filter.Operator, filter.Value, filter.Value)

			assert.Equal(t, "active", filter.Field, "Filter field should be 'active'")
			assert.Equal(t, model.OperatorEqual, filter.Operator, "Filter operator should be EQUAL")
			assert.Equal(t, true, filter.Value, "❌ CRITICAL: Filter value should be boolean true, not Firestore object")
		}
	})
}

// PostmanLikeMockUsecase only implements the methods needed for the integration test
type PostmanLikeMockUsecase struct {
	documents         []*model.Document
	enableDebug       bool
	lastQueryReceived *model.Query
}

// ListDocuments implements usecase.FirestoreUsecaseInterface
func (m *PostmanLikeMockUsecase) ListDocuments(ctx context.Context, req usecase.ListDocumentsRequest) ([]*model.Document, error) {
	// Simular GET que devuelve todos los documentos
	return m.documents, nil
}

// RunQuery implements usecase.FirestoreUsecaseInterface
func (m *PostmanLikeMockUsecase) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	if m.enableDebug {
		m.lastQueryReceived = req.StructuredQuery
	}

	var filteredDocs []*model.Document

	if req.StructuredQuery != nil && len(req.StructuredQuery.Filters) > 0 {
		for _, doc := range m.documents {
			matchesAllFilters := true

			for _, filter := range req.StructuredQuery.Filters {
				if docField, exists := doc.Fields[filter.Field]; exists {
					// Extract primitive value from FieldValue
					docValue := docField.Value
					filterValue := filter.Value // Ya es un valor primitivo después de convertFirestoreQueryToInternal

					// Compare values directly (both are now primitive values)
					matches := compareValuesDirect(docValue, filterValue, filter.Operator)
					if !matches {
						matchesAllFilters = false
						break
					}
				} else {
					matchesAllFilters = false
					break
				}
			}

			if matchesAllFilters {
				filteredDocs = append(filteredDocs, doc)
			}
		}

		return filteredDocs, nil
	}

	// No filters, return all documents
	return m.documents, nil
}

// Unused interface methods return nil
func (m *PostmanLikeMockUsecase) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) GetDocument(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) UpdateDocument(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) DeleteDocument(ctx context.Context, req usecase.DeleteDocumentRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) QueryDocuments(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) CreateProject(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) GetProject(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) CreateDatabase(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) GetDatabase(ctx context.Context, req usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) GetCollection(ctx context.Context, req usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, fmt.Errorf("not implemented")
}

// Additional methods to fully implement FirestoreUsecaseInterface
func (m *PostmanLikeMockUsecase) UpdateCollection(ctx context.Context, req usecase.UpdateCollectionRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) ListCollections(ctx context.Context, req usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) DeleteCollection(ctx context.Context, req usecase.DeleteCollectionRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) ListSubcollections(ctx context.Context, req usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) ListProjects(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) UpdateProject(ctx context.Context, req usecase.UpdateProjectRequest) (*model.Project, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) DeleteProject(ctx context.Context, req usecase.DeleteProjectRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) UpdateDatabase(ctx context.Context, req usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) DeleteDatabase(ctx context.Context, req usecase.DeleteDatabaseRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) ListDatabases(ctx context.Context, req usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) ListIndexes(ctx context.Context, req usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) CreateIndex(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) AtomicIncrement(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) AtomicArrayUnion(ctx context.Context, req usecase.AtomicArrayUnionRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) AtomicArrayRemove(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) AtomicServerTimestamp(ctx context.Context, req usecase.AtomicServerTimestampRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) DeleteIndex(ctx context.Context, req usecase.DeleteIndexRequest) error {
	return fmt.Errorf("not implemented")
}
func (m *PostmanLikeMockUsecase) RunBatchWrite(ctx context.Context, req usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// extractPrimitiveValue extracts the primitive value from a Firestore-style value
func extractPrimitiveValue(value interface{}) interface{} {
	if valueMap, ok := value.(map[string]interface{}); ok {
		// Handle Firestore-style values
		if boolVal, exists := valueMap["booleanValue"]; exists {
			return boolVal
		}
		if strVal, exists := valueMap["stringValue"]; exists {
			return strVal
		}
		if intVal, exists := valueMap["integerValue"]; exists {
			// Handle both string and float64 integer values
			switch v := intVal.(type) {
			case string:
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
					return parsed
				}
			case float64:
				return int64(v)
			}
		}
		if numVal, exists := valueMap["doubleValue"]; exists {
			if parsed, ok := numVal.(float64); ok {
				return parsed
			}
		}
		// Return original value if no special handling needed
		return value
	}
	// Return primitive values as-is
	return value
}

// compareValues compares two values using Firestore operators
func compareValues(docValue interface{}, filterValue interface{}, operator model.Operator) bool {
	// Handle Firestore-style values
	docPrimitive := extractFieldValue(docValue)
	filterPrimitive := extractFieldValue(filterValue)

	// Handle nil values
	if docPrimitive == nil || filterPrimitive == nil {
		return operator == model.OperatorEqual && docPrimitive == filterPrimitive
	}

	// Special handling for booleans since they can't be compared numerically
	dBool, dIsBool := docPrimitive.(bool)
	fBool, fIsBool := filterPrimitive.(bool)
	if dIsBool && fIsBool {
		switch operator {
		case model.OperatorEqual:
			return dBool == fBool
		case model.OperatorNotEqual:
			return dBool != fBool
		default:
			return false
		}
	}

	// Try numeric comparison
	if dNum, dok := toFloat64(docPrimitive); dok {
		if fNum, fok := toFloat64(filterPrimitive); fok {
			switch operator {
			case model.OperatorLessThan:
				return dNum < fNum
			case model.OperatorLessThanOrEqual:
				return dNum <= fNum
			case model.OperatorGreaterThan:
				return dNum > fNum
			case model.OperatorGreaterThanOrEqual:
				return dNum >= fNum
			case model.OperatorEqual:
				return dNum == fNum
			case model.OperatorNotEqual:
				return dNum != fNum
			}
		}
	}

	// Fall back to string comparison
	dStr := fmt.Sprintf("%v", docPrimitive)
	fStr := fmt.Sprintf("%v", filterPrimitive)

	switch operator {
	case model.OperatorEqual:
		return dStr == fStr
	case model.OperatorNotEqual:
		return dStr != fStr
	case model.OperatorLessThan:
		return dStr < fStr
	case model.OperatorLessThanOrEqual:
		return dStr <= fStr
	case model.OperatorGreaterThan:
		return dStr > fStr
	case model.OperatorGreaterThanOrEqual:
		return dStr >= fStr
	default:
		return false
	}
}

// compareValuesDirect compares two primitive values directly without extracting from Firestore format
func compareValuesDirect(docValue, filterValue interface{}, operator model.Operator) bool {
	// Handle nil values
	if docValue == nil || filterValue == nil {
		return operator == model.OperatorEqual && docValue == filterValue
	}

	// Special handling for booleans (most common case)
	dBool, dIsBool := docValue.(bool)
	fBool, fIsBool := filterValue.(bool)
	if dIsBool && fIsBool {
		switch operator {
		case model.OperatorEqual:
			return dBool == fBool
		case model.OperatorNotEqual:
			return dBool != fBool
		default:
			return false
		}
	}

	// Try numeric comparison
	if dNum, dok := toFloat64(docValue); dok {
		if fNum, fok := toFloat64(filterValue); fok {
			switch operator {
			case model.OperatorLessThan:
				return dNum < fNum
			case model.OperatorLessThanOrEqual:
				return dNum <= fNum
			case model.OperatorGreaterThan:
				return dNum > fNum
			case model.OperatorGreaterThanOrEqual:
				return dNum >= fNum
			case model.OperatorEqual:
				return dNum == fNum
			case model.OperatorNotEqual:
				return dNum != fNum
			}
		}
	}

	// Fall back to string comparison
	dStr := fmt.Sprintf("%v", docValue)
	fStr := fmt.Sprintf("%v", filterValue)

	switch operator {
	case model.OperatorEqual:
		return dStr == fStr
	case model.OperatorNotEqual:
		return dStr != fStr
	case model.OperatorLessThan:
		return dStr < fStr
	case model.OperatorLessThanOrEqual:
		return dStr <= fStr
	case model.OperatorGreaterThan:
		return dStr > fStr
	case model.OperatorGreaterThanOrEqual:
		return dStr >= fStr
	default:
		return false
	}
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// createMockProductDocuments crea documentos de productos de prueba
func createMockProductDocuments() []*model.Document {
	now := time.Now()
	return []*model.Document{
		{
			DocumentID:   "product1",
			ProjectID:    "new-proj-from-postman",
			DatabaseID:   "new-db-from-postman",
			CollectionID: "productos",
			Path:         "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos/product1",
			ParentPath:   "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos",
			Fields: map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Smartphone Pro",
				},
				"active": {
					ValueType: model.FieldTypeBool,
					Value:     true,
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     599.99,
				},
				"category": {
					ValueType: model.FieldTypeString,
					Value:     "electronics",
				},
			},
			CreateTime: now,
			UpdateTime: now,
			Exists:     true,
		},
		{
			DocumentID:   "product2",
			ProjectID:    "new-proj-from-postman",
			DatabaseID:   "new-db-from-postman",
			CollectionID: "productos",
			Path:         "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos/product2",
			ParentPath:   "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos",
			Fields: map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Laptop Ultra",
				},
				"active": {
					ValueType: model.FieldTypeBool,
					Value:     true,
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     1299.99,
				},
				"category": {
					ValueType: model.FieldTypeString,
					Value:     "electronics",
				},
			},
			CreateTime: now,
			UpdateTime: now,
			Exists:     true,
		},
		{
			DocumentID:   "product3",
			ProjectID:    "new-proj-from-postman",
			DatabaseID:   "new-db-from-postman",
			CollectionID: "productos",
			Path:         "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos/product3",
			ParentPath:   "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos",
			Fields: map[string]*model.FieldValue{
				"name": {
					ValueType: model.FieldTypeString,
					Value:     "Old Model",
				},
				"active": {
					ValueType: model.FieldTypeBool,
					Value:     false,
				},
				"price": {
					ValueType: model.FieldTypeDouble,
					Value:     199.99,
				},
				"category": {
					ValueType: model.FieldTypeString,
					Value:     "electronics",
				},
			},
			CreateTime: now,
			UpdateTime: now,
			Exists:     true,
		},
	}
}

// createTestHandler crea un handler de prueba para simular el servidor
func createTestHandler(uc usecase.FirestoreUsecaseInterface) http.Handler {
	// Simular router básico para las rutas de prueba
	mux := http.NewServeMux()

	// GET /documents/productos
	mux.HandleFunc("/api/v1/organizations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && contains(r.URL.Path, "/documents/productos") {
			// Simular ListDocuments
			docs, err := uc.ListDocuments(r.Context(), usecase.ListDocumentsRequest{
				ProjectID:    "new-proj-from-postman",
				DatabaseID:   "new-db-from-postman",
				CollectionID: "productos",
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Convert internal document format to Firestore JSON format
			response := map[string]interface{}{
				"documents": convertDocumentsToFirestoreJSON(docs),
				"count":     len(docs),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "POST" && contains(r.URL.Path, "/query/productos") {
			// Parse query request
			var firestoreQuery map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&firestoreQuery); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Convert Firestore query to internal query format
			query, err := convertFirestoreQueryToInternal(firestoreQuery)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Execute query
			docs, err := uc.RunQuery(r.Context(), usecase.QueryRequest{
				ProjectID:       "new-proj-from-postman",
				DatabaseID:      "new-db-from-postman",
				Parent:          "projects/new-proj-from-postman/databases/new-db-from-postman/documents/productos",
				StructuredQuery: query,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Convert results to Firestore JSON format
			response := map[string]interface{}{
				"documents": convertDocumentsToFirestoreJSON(docs),
				"count":     len(docs),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		http.NotFound(w, r)
	})

	return mux
}

// convertDocumentsToFirestoreJSON converts internal document models to Firestore JSON format
func convertDocumentsToFirestoreJSON(docs []*model.Document) []map[string]interface{} {
	result := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		docJSON := map[string]interface{}{
			"name":       doc.Path,
			"createTime": doc.CreateTime.Format(time.RFC3339Nano),
			"updateTime": doc.UpdateTime.Format(time.RFC3339Nano),
			"fields":     make(map[string]interface{}),
		}

		// Convert FieldValues to Firestore format
		fields := make(map[string]interface{})
		for key, fieldValue := range doc.Fields {
			fields[key] = convertFieldValueToFirestore(fieldValue)
		}
		docJSON["fields"] = fields

		result[i] = docJSON
	}
	return result
}

// convertFieldValueToFirestore converts internal FieldValue to Firestore JSON format
func convertFieldValueToFirestore(fieldValue *model.FieldValue) map[string]interface{} {
	if fieldValue == nil {
		return map[string]interface{}{
			"nullValue": nil,
		}
	}

	switch fieldValue.ValueType {
	case model.FieldTypeBool:
		return map[string]interface{}{
			"booleanValue": fieldValue.Value,
		}
	case model.FieldTypeInt:
		return map[string]interface{}{
			"integerValue": fmt.Sprintf("%d", fieldValue.Value),
		}
	case model.FieldTypeDouble:
		return map[string]interface{}{
			"doubleValue": fieldValue.Value,
		}
	case model.FieldTypeString:
		return map[string]interface{}{
			"stringValue": fieldValue.Value,
		}
	case model.FieldTypeTimestamp:
		if t, ok := fieldValue.Value.(time.Time); ok {
			return map[string]interface{}{
				"timestampValue": t.Format(time.RFC3339Nano),
			}
		}
		return map[string]interface{}{
			"nullValue": nil,
		}
	default:
		// Handle other types as needed
		return map[string]interface{}{
			"nullValue": nil,
		}
	}
}

// convertFirestoreQueryToInternal converts Firestore query JSON to internal Query model
func convertFirestoreQueryToInternal(firestoreQuery map[string]interface{}) (*model.Query, error) {
	query := &model.Query{
		Filters: make([]model.Filter, 0),
	}

	// Extract filters
	if where, ok := firestoreQuery["where"].(map[string]interface{}); ok {
		if fieldFilter, ok := where["fieldFilter"].(map[string]interface{}); ok {
			filter := model.Filter{}

			// Extract field path
			if field, ok := fieldFilter["field"].(map[string]interface{}); ok {
				if fieldPath, ok := field["fieldPath"].(string); ok {
					filter.Field = fieldPath
				}
			}

			// Extract operator
			if op, ok := fieldFilter["op"].(string); ok {
				filter.Operator = mapFirestoreOperator(op)
			} // Extract and convert Firestore-style value to primitive value
			if value, ok := fieldFilter["value"].(map[string]interface{}); ok {
				filter.Value = extractFieldValue(value)
			}

			query.Filters = append(query.Filters, filter)
		}
	}

	return query, nil
}

// mapFirestoreOperator maps Firestore operator strings to model.Operator
func mapFirestoreOperator(op string) model.Operator {
	switch op {
	case "EQUAL":
		return model.OperatorEqual
	case "NOT_EQUAL":
		return model.OperatorNotEqual
	case "GREATER_THAN":
		return model.OperatorGreaterThan
	case "GREATER_THAN_OR_EQUAL":
		return model.OperatorGreaterThanOrEqual
	case "LESS_THAN":
		return model.OperatorLessThan
	case "LESS_THAN_OR_EQUAL":
		return model.OperatorLessThanOrEqual
	case "ARRAY_CONTAINS":
		return model.OperatorArrayContains
	case "ARRAY_CONTAINS_ANY":
		return model.OperatorArrayContainsAny
	case "IN":
		return model.OperatorIn
	case "NOT_IN":
		return model.OperatorNotIn
	default:
		return model.OperatorEqual
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		s == substr
}

// extractFieldValue extracts a primitive value from a Firestore-style field
func extractFieldValue(field interface{}) interface{} {
	if fieldMap, ok := field.(map[string]interface{}); ok {
		// Handle each Firestore value type
		if val, exists := fieldMap["booleanValue"]; exists {
			return val
		}
		if val, exists := fieldMap["stringValue"]; exists {
			return val
		}
		if val, exists := fieldMap["integerValue"]; exists {
			switch v := val.(type) {
			case string:
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
					return parsed
				}
			case float64:
				return int64(v)
			}
		}
		if val, exists := fieldMap["doubleValue"]; exists {
			return val
		}
		// Add more types as needed
	}
	return nil
}
