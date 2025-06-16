// Package http provides HTTP adapter tests for the Firestore clone service
// following hexagonal architecture principles.
//
// This test file validates the HTTP adapter layer that translates between
// Firestore REST API format and internal domain models. It ensures proper
// handling of:
// - Document CRUD operations via HTTP endpoints
// - Firestore query format parsing and conversion
// - Composite filter support (AND/OR operations)
// - Error handling and HTTP status codes
// - Value type conversion between Firestore and internal formats
//
// The tests follow the Firestore API specification to ensure compatibility
// with Google Firestore client libraries.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFirestoreUCForDocuments implements usecase interface for document handler tests
// following hexagonal architecture principles - only implements necessary methods
type MockFirestoreUCForDocuments struct {
	CreateDocumentFn func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error)
	GetDocumentFn    func(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error)
	UpdateDocumentFn func(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error)
	DeleteDocumentFn func(ctx context.Context, req usecase.DeleteDocumentRequest) error
	RunQueryFn       func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error)
}

func (m *MockFirestoreUCForDocuments) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	if m.CreateDocumentFn != nil {
		return m.CreateDocumentFn(ctx, req)
	}
	return &model.Document{DocumentID: "default"}, nil
}

func (m *MockFirestoreUCForDocuments) GetDocument(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error) {
	if m.GetDocumentFn != nil {
		return m.GetDocumentFn(ctx, req)
	}
	return &model.Document{DocumentID: req.DocumentID}, nil
}

func (m *MockFirestoreUCForDocuments) UpdateDocument(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error) {
	if m.UpdateDocumentFn != nil {
		return m.UpdateDocumentFn(ctx, req)
	}
	return &model.Document{DocumentID: req.DocumentID}, nil
}

func (m *MockFirestoreUCForDocuments) DeleteDocument(ctx context.Context, req usecase.DeleteDocumentRequest) error {
	if m.DeleteDocumentFn != nil {
		return m.DeleteDocumentFn(ctx, req)
	}
	return nil
}

func (m *MockFirestoreUCForDocuments) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	if m.RunQueryFn != nil {
		return m.RunQueryFn(ctx, req)
	}
	return []*model.Document{{DocumentID: "query_result"}}, nil
}

// Implement remaining interface methods as no-ops for compliance with usecase.FirestoreUsecaseInterface
func (m *MockFirestoreUCForDocuments) ListDocuments(context.Context, usecase.ListDocumentsRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) CreateCollection(context.Context, usecase.CreateCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) GetCollection(context.Context, usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) UpdateCollection(context.Context, usecase.UpdateCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) DeleteCollection(context.Context, usecase.DeleteCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) ListCollections(context.Context, usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) CreateIndex(context.Context, usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) DeleteIndex(context.Context, usecase.DeleteIndexRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) ListIndexes(context.Context, usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) CreateProject(context.Context, usecase.CreateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) GetProject(context.Context, usecase.GetProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) DeleteProject(context.Context, usecase.DeleteProjectRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) ListProjects(context.Context, usecase.ListProjectsRequest) ([]*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) CreateDatabase(context.Context, usecase.CreateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) GetDatabase(context.Context, usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) DeleteDatabase(context.Context, usecase.DeleteDatabaseRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) ListDatabases(context.Context, usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) ListSubcollections(context.Context, usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) QueryDocuments(context.Context, usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) BeginTransaction(context.Context, string) (string, error) {
	return "", nil
}
func (m *MockFirestoreUCForDocuments) CommitTransaction(context.Context, string, string) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) RunBatchWrite(context.Context, usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) AtomicIncrement(context.Context, usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) AtomicArrayUnion(context.Context, usecase.AtomicArrayUnionRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) AtomicArrayRemove(context.Context, usecase.AtomicArrayRemoveRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) AtomicServerTimestamp(context.Context, usecase.AtomicServerTimestampRequest) error {
	return nil
}
func (m *MockFirestoreUCForDocuments) UpdateProject(context.Context, usecase.UpdateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForDocuments) UpdateDatabase(context.Context, usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}

// customDocumentUC extends MockFirestoreUCForDocuments for test-specific behavior
type customDocumentUC struct {
	MockFirestoreUCForDocuments
	createDocumentFunc func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error)
}

func (m *customDocumentUC) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	if m.createDocumentFunc != nil {
		return m.createDocumentFunc(ctx, req)
	}
	return m.MockFirestoreUCForDocuments.CreateDocument(ctx, req)
}

func TestCreateDocumentHandler_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{
		createDocumentFunc: func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
			return &model.Document{DocumentID: "doc1"}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{"field":"value"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "doc1", result["documentID"])
}

func TestCreateDocumentHandler_MissingData(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "missing_data", result["error"])
}

func TestCreateDocumentHandler_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customDocumentUC{
		createDocumentFunc: func(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
			return nil, errors.New("internal error")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/test/:projectID/:databaseID/:collectionID", h.CreateDocument)

	body := []byte(`{"field":"value"}`)
	req := httptest.NewRequest("POST", "/test/p1/d1/c1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "create_document_failed", result["error"])
}

// Mock para RunQuery method
type customQueryUC struct {
	MockFirestoreUC
	runQueryFunc func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error)
}

func (m *customQueryUC) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	if m.runQueryFunc != nil {
		return m.runQueryFunc(ctx, req)
	}
	return m.MockFirestoreUC.RunQuery(ctx, req)
}

func TestQueryDocumentsHandler_FirestoreJSON_Success(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que el request está correctamente configurado
			assert.Equal(t, "test-project", req.ProjectID)
			assert.Equal(t, "test-database", req.DatabaseID)
			assert.Equal(t, "projects/test-project/databases/test-database/documents/productos", req.Parent)
			assert.NotNil(t, req.StructuredQuery)
			assert.Equal(t, "productos", req.StructuredQuery.CollectionID)

			// Retornar documentos mock
			return []*model.Document{
				{DocumentID: "doc1"},
				{DocumentID: "doc2"},
			}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	// JSON exacto del formato Firestore como el que envías desde Postman
	firestoreQuery := `{
		"from": [
			{
				"collectionId": "productos"
			}
		],
		"where": {
			"fieldFilter": {
				"field": {
					"fieldPath": "born"
				},
				"op": "LESS_THAN",
				"value": 1900
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

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verificar la respuesta
	assert.Contains(t, result, "documents")
	assert.Contains(t, result, "count")
	documents := result["documents"].([]interface{})
	assert.Len(t, documents, 2)
	assert.Equal(t, float64(2), result["count"])
}

func TestQueryDocumentsHandler_FirestoreJSON_InvalidBody(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	// JSON inválido
	invalidJSON := `{"from": [invalid json}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(invalidJSON)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_json", result["error"])
}

func TestQueryDocumentsHandler_FirestoreJSON_UnsupportedOperator(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	// JSON con operador no soportado
	queryWithUnsupportedOp := `{
		"from": [
			{
				"collectionId": "productos"
			}
		],
		"where": {
			"fieldFilter": {
				"field": {
					"fieldPath": "price"
				},
				"op": "INVALID_OPERATOR",
				"value": 100
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(queryWithUnsupportedOp)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_query_format", result["error"])
	assert.Contains(t, result["message"], "unsupported operator")
}

func TestQueryDocumentsHandler_FirestoreJSON_UsecaseError(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			return nil, errors.New("database connection failed")
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	firestoreQuery := `{
		"from": [
			{
				"collectionId": "productos"
			}
		],
		"where": {
			"fieldFilter": {
				"field": {
					"fieldPath": "status"
				},
				"op": "EQUAL",
				"value": "active"
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "query_failed", result["error"])
}

func TestQueryDocumentsHandler_FirestoreJSON_ComplexQuery(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que la query compleja se parsea correctamente
			assert.NotNil(t, req.StructuredQuery)
			assert.Equal(t, "users", req.StructuredQuery.CollectionID)

			return []*model.Document{
				{DocumentID: "user1"},
			}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	// Query compleja con IN operator y ordenamiento múltiple
	complexQuery := `{
		"from": [
			{
				"collectionId": "users"
			}
		],
		"where": {
			"fieldFilter": {
				"field": {
					"fieldPath": "status"
				},
				"op": "IN",
				"value": ["active", "premium"]
			}
		},
		"orderBy": [
			{
				"field": {
					"fieldPath": "priority"
				},
				"direction": "DESCENDING"
			},
			{
				"field": {
					"fieldPath": "name"
				},
				"direction": "ASCENDING"
			}
		],
		"limit": 50,
		"offset": 10
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/users", bytes.NewReader([]byte(complexQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result, "documents")
	assert.Equal(t, float64(1), result["count"])
}

func TestConvertFirestoreJSONToModelQuery_AllOperators(t *testing.T) {
	tests := []struct {
		name        string
		operator    string
		expectedOp  model.Operator
		value       interface{}
		expectError bool
	}{
		{"EQUAL operator", "EQUAL", model.OperatorEqual, "test", false},
		{"NOT_EQUAL operator", "NOT_EQUAL", model.OperatorNotEqual, "test", false},
		{"LESS_THAN operator", "LESS_THAN", model.OperatorLessThan, 100, false},
		{"LESS_THAN_OR_EQUAL operator", "LESS_THAN_OR_EQUAL", model.OperatorLessThanOrEqual, 100, false},
		{"GREATER_THAN operator", "GREATER_THAN", model.OperatorGreaterThan, 50, false},
		{"GREATER_THAN_OR_EQUAL operator", "GREATER_THAN_OR_EQUAL", model.OperatorGreaterThanOrEqual, 50, false},
		{"IN operator", "IN", model.OperatorIn, []interface{}{"a", "b", "c"}, false},
		{"NOT_IN operator", "NOT_IN", model.OperatorNotIn, []interface{}{"x", "y"}, false},
		{"ARRAY_CONTAINS operator", "ARRAY_CONTAINS", model.OperatorArrayContains, "tag", false},
		{"ARRAY_CONTAINS_ANY operator", "ARRAY_CONTAINS_ANY", model.OperatorArrayContainsAny, []interface{}{"tag1", "tag2"}, false},
		{"Unsupported operator", "INVALID_OP", model.OperatorEqual, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firestoreQuery := FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "test"}},
				Where: &FirestoreFilter{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "testField"},
						Op:    tt.operator,
						Value: tt.value,
					},
				},
			}

			query, err := convertFirestoreJSONToModelQuery(firestoreQuery)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, query)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, query)
				assert.Equal(t, "test", query.CollectionID)
			}
		})
	}
}

func TestConvertFirestoreJSONToModelQuery_OrderByDirections(t *testing.T) {
	firestoreQuery := FirestoreStructuredQuery{
		From: []FirestoreCollectionSelector{{CollectionID: "test"}},
		OrderBy: []FirestoreOrder{
			{
				Field:     FirestoreFieldReference{FieldPath: "field1"},
				Direction: "ASCENDING",
			},
			{
				Field:     FirestoreFieldReference{FieldPath: "field2"},
				Direction: "DESCENDING",
			},
			{
				Field:     FirestoreFieldReference{FieldPath: "field3"},
				Direction: "INVALID", // Should default to ASCENDING
			},
		},
	}

	query, err := convertFirestoreJSONToModelQuery(firestoreQuery)
	assert.NoError(t, err)
	assert.NotNil(t, query)
	assert.Equal(t, "test", query.CollectionID)
}

func TestConvertFirestoreJSONToModelQuery_TypedValues(t *testing.T) {
	tests := []struct {
		name           string
		firestoreValue interface{}
		expectedValue  interface{}
		description    string
	}{
		{
			name:           "boolean true value",
			firestoreValue: map[string]interface{}{"booleanValue": true},
			expectedValue:  true,
			description:    "Should extract boolean true from Firestore typed value",
		},
		{
			name:           "boolean false value",
			firestoreValue: map[string]interface{}{"booleanValue": false},
			expectedValue:  false,
			description:    "Should extract boolean false from Firestore typed value",
		},
		{
			name:           "string value",
			firestoreValue: map[string]interface{}{"stringValue": "active"},
			expectedValue:  "active",
			description:    "Should extract string from Firestore typed value",
		},
		{
			name:           "integer value as string",
			firestoreValue: map[string]interface{}{"integerValue": "123"},
			expectedValue:  int64(123),
			description:    "Should parse integer string to int64",
		},
		{
			name:           "integer value as number",
			firestoreValue: map[string]interface{}{"integerValue": 456},
			expectedValue:  456,
			description:    "Should keep integer as is when already a number",
		},
		{
			name:           "double value",
			firestoreValue: map[string]interface{}{"doubleValue": 3.14},
			expectedValue:  3.14,
			description:    "Should extract double from Firestore typed value",
		},
		{
			name:           "null value",
			firestoreValue: map[string]interface{}{"nullValue": "NULL_VALUE"},
			expectedValue:  nil,
			description:    "Should convert null value to nil",
		},
		{
			name:           "plain value",
			firestoreValue: "plainString",
			expectedValue:  "plainString",
			description:    "Should pass through plain values unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firestoreQuery := FirestoreStructuredQuery{
				From: []FirestoreCollectionSelector{{CollectionID: "products"}},
				Where: &FirestoreFilter{
					FieldFilter: &FirestoreFieldFilter{
						Field: FirestoreFieldReference{FieldPath: "status"},
						Op:    "EQUAL",
						Value: tt.firestoreValue,
					},
				},
			}

			query, err := convertFirestoreJSONToModelQuery(firestoreQuery)
			assert.NoError(t, err, tt.description)
			assert.NotNil(t, query)
			assert.Equal(t, "products", query.CollectionID)

			// Verify the filter was created correctly
			require.Len(t, query.Filters, 1, "Should have exactly one filter")
			filter := query.Filters[0]
			assert.Equal(t, "status", filter.Field)
			assert.Equal(t, model.OperatorEqual, filter.Operator)
			assert.Equal(t, tt.expectedValue, filter.Value, tt.description)
		})
	}
}

func TestQueryDocumentsHandler_FirestoreJSON_BooleanFilter(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que el filtro boolean se convirtió correctamente
			assert.NotNil(t, req.StructuredQuery)
			assert.Len(t, req.StructuredQuery.Filters, 1)
			filter := req.StructuredQuery.Filters[0]
			assert.Equal(t, "active", filter.Field)
			assert.Equal(t, model.OperatorEqual, filter.Operator)
			assert.Equal(t, true, filter.Value, "Boolean value should be extracted from Firestore type")

			// Simular documentos que coinciden con el filtro
			return []*model.Document{
				{DocumentID: "product1"},
				{DocumentID: "product2"},
			}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	// JSON exacto como el que envías desde Postman con valores tipados de Firestore
	firestoreQuery := `{
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

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verificar que la respuesta contiene documentos (no count:0)
	assert.Contains(t, result, "documents")
	assert.Contains(t, result, "count")
	documents := result["documents"].([]interface{})
	assert.Len(t, documents, 2, "Should return filtered documents, not empty result")
	assert.Equal(t, float64(2), result["count"])
}

func TestQueryDocumentsHandler_FirestoreJSON_StringFilter(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que el filtro string se convirtió correctamente
			assert.NotNil(t, req.StructuredQuery)
			assert.Len(t, req.StructuredQuery.Filters, 1)
			filter := req.StructuredQuery.Filters[0]
			assert.Equal(t, "category", filter.Field)
			assert.Equal(t, model.OperatorEqual, filter.Operator)
			assert.Equal(t, "electronics", filter.Value, "String value should be extracted from Firestore type")

			return []*model.Document{{DocumentID: "product1"}}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	firestoreQuery := `{
		"from": [{"collectionId": "productos"}],
		"where": {
			"fieldFilter": {
				"field": {"fieldPath": "category"},
				"op": "EQUAL",
				"value": {"stringValue": "electronics"}
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestQueryDocumentsHandler_FirestoreJSON_IntegerFilter(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que el filtro integer se convirtió correctamente
			assert.NotNil(t, req.StructuredQuery)
			assert.Len(t, req.StructuredQuery.Filters, 1)
			filter := req.StructuredQuery.Filters[0]
			assert.Equal(t, "price", filter.Field)
			assert.Equal(t, model.OperatorGreaterThan, filter.Operator)
			assert.Equal(t, int64(100), filter.Value, "Integer value should be parsed from string")

			return []*model.Document{{DocumentID: "expensive_product"}}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	firestoreQuery := `{
		"from": [{"collectionId": "productos"}],
		"where": {
			"fieldFilter": {
				"field": {"fieldPath": "price"},
				"op": "GREATER_THAN",
				"value": {"integerValue": "100"}
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestQueryDocumentsHandler_FirestoreJSON_ArrayFilter(t *testing.T) {
	app := fiber.New()
	mockUC := &customQueryUC{
		runQueryFunc: func(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
			// Verificar que el filtro array se convirtió correctamente
			assert.NotNil(t, req.StructuredQuery)
			assert.Len(t, req.StructuredQuery.Filters, 1)
			filter := req.StructuredQuery.Filters[0]
			assert.Equal(t, "tags", filter.Field)
			assert.Equal(t, model.OperatorIn, filter.Operator)

			// Verificar que los valores del array se convirtieron correctamente
			expectedArray := []interface{}{"electronics", "mobile", "smartphone"}
			assert.Equal(t, expectedArray, filter.Value, "Array values should be extracted from Firestore types")

			return []*model.Document{{DocumentID: "tagged_product"}}, nil
		},
	}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: testLogger{}}
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	firestoreQuery := `{
		"from": [{"collectionId": "productos"}],
		"where": {
			"fieldFilter": {
				"field": {"fieldPath": "tags"},
				"op": "IN",
				"value": {
					"arrayValue": {
						"values": [
							{"stringValue": "electronics"},
							{"stringValue": "mobile"},
							{"stringValue": "smartphone"}
						]
					}
				}
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/query/productos", bytes.NewReader([]byte(firestoreQuery)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
