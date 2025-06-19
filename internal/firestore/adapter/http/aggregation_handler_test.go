package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFirestoreUCForAggregation is a mock implementation for testing aggregation functionality
type MockFirestoreUCForAggregation struct {
	mock.Mock
}

func (m *MockFirestoreUCForAggregation) RunAggregationQuery(ctx context.Context, req usecase.AggregationQueryRequest) (*usecase.AggregationQueryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AggregationQueryResponse), args.Error(1)
}

// Implement other required methods as no-ops for testing
func (m *MockFirestoreUCForAggregation) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) GetDocument(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) UpdateDocument(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) DeleteDocument(ctx context.Context, req usecase.DeleteDocumentRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListDocuments(ctx context.Context, req usecase.ListDocumentsRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) GetCollection(ctx context.Context, req usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) UpdateCollection(ctx context.Context, req usecase.UpdateCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListCollections(ctx context.Context, req usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) DeleteCollection(ctx context.Context, req usecase.DeleteCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListSubcollections(ctx context.Context, req usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) CreateIndex(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) DeleteIndex(ctx context.Context, req usecase.DeleteIndexRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListIndexes(ctx context.Context, req usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) QueryDocuments(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) RunBatchWrite(ctx context.Context, req usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	return "", nil
}
func (m *MockFirestoreUCForAggregation) CommitTransaction(ctx context.Context, projectID, transactionID string) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) CreateProject(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) GetProject(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) UpdateProject(ctx context.Context, req usecase.UpdateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) DeleteProject(ctx context.Context, req usecase.DeleteProjectRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListProjects(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) CreateDatabase(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) GetDatabase(ctx context.Context, req usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) UpdateDatabase(ctx context.Context, req usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) DeleteDatabase(ctx context.Context, req usecase.DeleteDatabaseRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) ListDatabases(ctx context.Context, req usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) AtomicIncrement(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForAggregation) AtomicArrayUnion(ctx context.Context, req usecase.AtomicArrayUnionRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) AtomicArrayRemove(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error {
	return nil
}
func (m *MockFirestoreUCForAggregation) AtomicServerTimestamp(ctx context.Context, req usecase.AtomicServerTimestampRequest) error {
	return nil
}

func TestRunAggregationQuery_BasicCount(t *testing.T) {
	// Setup
	mockUC := &MockFirestoreUCForAggregation{}
	handler := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Mock response
	expectedResponse := &usecase.AggregationQueryResponse{
		Results: []usecase.AggregationResult{
			{
				Result: usecase.AggregationResultData{
					AggregateFields: map[string]interface{}{
						"conteo_total_productos": map[string]interface{}{"integerValue": "2"},
					},
				},
				ReadTime: "2025-06-18T12:00:00.000Z",
			},
		},
	}

	mockUC.On("RunAggregationQuery", mock.Anything, mock.MatchedBy(func(req usecase.AggregationQueryRequest) bool {
		return req.ProjectID == "test-project" &&
			req.DatabaseID == "test-database" &&
			len(req.StructuredAggregationQuery.Aggregations) == 1 &&
			req.StructuredAggregationQuery.Aggregations[0].Alias == "conteo_total_productos" &&
			req.StructuredAggregationQuery.Aggregations[0].Count != nil
	})).Return(expectedResponse, nil)

	// Create Fiber app
	app := fiber.New()

	// Setup route with parameters
	app.Post("/api/v1/organizations/:organizationID/projects/:projectID/databases/:databaseID/documents:runAggregationQuery", handler.RunAggregationQuery)

	// Create request body
	requestBody := map[string]interface{}{
		"structuredAggregationQuery": map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"from": []map[string]interface{}{
					{
						"collectionId": "productos",
					},
				},
			},
			"aggregations": []map[string]interface{}{
				{
					"alias": "conteo_total_productos",
					"count": map[string]interface{}{},
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/documents:runAggregationQuery", bytes.NewBuffer(requestBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify mock was called
	mockUC.AssertExpectations(t)
}

func TestRunAggregationQuery_CountAndSum(t *testing.T) {
	// Setup
	mockUC := &MockFirestoreUCForAggregation{}
	handler := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Mock response
	expectedResponse := &usecase.AggregationQueryResponse{
		Results: []usecase.AggregationResult{
			{
				Result: usecase.AggregationResultData{
					AggregateFields: map[string]interface{}{
						"conteo_total_productos": map[string]interface{}{"integerValue": "2"},
						"stock_total_inventario": map[string]interface{}{"doubleValue": 150.0},
					},
				},
				ReadTime: "2025-06-18T12:00:00.000Z",
			},
		},
	}

	mockUC.On("RunAggregationQuery", mock.Anything, mock.MatchedBy(func(req usecase.AggregationQueryRequest) bool {
		return req.ProjectID == "test-project" &&
			req.DatabaseID == "test-database" &&
			len(req.StructuredAggregationQuery.Aggregations) == 2
	})).Return(expectedResponse, nil)

	// Create Fiber app
	app := fiber.New()

	// Setup route with parameters
	app.Post("/api/v1/organizations/:organizationID/projects/:projectID/databases/:databaseID/documents:runAggregationQuery", handler.RunAggregationQuery)

	// Create request body (matching the failing request from the logs)
	requestBody := map[string]interface{}{
		"structuredAggregationQuery": map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"from": []map[string]interface{}{
					{
						"collectionId": "productos",
					},
				},
			},
			"aggregations": []map[string]interface{}{
				{
					"alias": "conteo_total_productos",
					"count": map[string]interface{}{},
				},
				{
					"alias": "stock_total_inventario",
					"sum": map[string]interface{}{
						"field": map[string]interface{}{
							"fieldPath": "stock",
						},
					},
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/documents:runAggregationQuery", bytes.NewBuffer(requestBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify mock was called
	mockUC.AssertExpectations(t)
}

func TestRunAggregationQuery_WithGroupBy(t *testing.T) {
	// Setup
	mockUC := &MockFirestoreUCForAggregation{}
	handler := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Mock response with grouped results
	expectedResponse := &usecase.AggregationQueryResponse{
		Results: []usecase.AggregationResult{
			{
				Result: usecase.AggregationResultData{
					AggregateFields: map[string]interface{}{
						"category":         map[string]interface{}{"stringValue": "Electronics"},
						"conteo_productos": map[string]interface{}{"integerValue": "5"},
						"precio_promedio":  map[string]interface{}{"doubleValue": 899.99},
					},
				},
				ReadTime: "2025-06-18T12:00:00.000Z",
			},
			{
				Result: usecase.AggregationResultData{
					AggregateFields: map[string]interface{}{
						"category":         map[string]interface{}{"stringValue": "Books"},
						"conteo_productos": map[string]interface{}{"integerValue": "3"},
						"precio_promedio":  map[string]interface{}{"doubleValue": 29.99},
					},
				},
				ReadTime: "2025-06-18T12:00:00.000Z",
			},
		},
	}

	mockUC.On("RunAggregationQuery", mock.Anything, mock.MatchedBy(func(req usecase.AggregationQueryRequest) bool {
		return len(req.StructuredAggregationQuery.GroupBy) == 1 &&
			req.StructuredAggregationQuery.GroupBy[0].FieldPath == "category"
	})).Return(expectedResponse, nil)

	// Create Fiber app
	app := fiber.New()

	// Setup route with parameters
	app.Post("/api/v1/organizations/:organizationID/projects/:projectID/databases/:databaseID/documents:runAggregationQuery", handler.RunAggregationQuery)

	// Create request body with groupBy
	requestBody := map[string]interface{}{
		"structuredAggregationQuery": map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"from": []map[string]interface{}{
					{
						"collectionId": "productos",
					},
				},
			},
			"groupBy": []map[string]interface{}{
				{
					"fieldPath": "category",
				},
			},
			"aggregations": []map[string]interface{}{
				{
					"alias": "conteo_productos",
					"count": map[string]interface{}{},
				},
				{
					"alias": "precio_promedio",
					"avg": map[string]interface{}{
						"field": map[string]interface{}{
							"fieldPath": "price",
						},
					},
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/documents:runAggregationQuery", bytes.NewBuffer(requestBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify mock was called
	mockUC.AssertExpectations(t)
}

func TestRunAggregationQuery_ExtendedOperators(t *testing.T) {
	// Setup
	mockUC := &MockFirestoreUCForAggregation{}
	handler := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Mock response with extended operators (min, max)
	expectedResponse := &usecase.AggregationQueryResponse{
		Results: []usecase.AggregationResult{
			{
				Result: usecase.AggregationResultData{
					AggregateFields: map[string]interface{}{
						"precio_mas_bajo": map[string]interface{}{"doubleValue": 19.99},
						"precio_mas_alto": map[string]interface{}{"doubleValue": 1999.99},
					},
				},
				ReadTime: "2025-06-18T12:00:00.000Z",
			},
		},
	}

	mockUC.On("RunAggregationQuery", mock.Anything, mock.MatchedBy(func(req usecase.AggregationQueryRequest) bool {
		return len(req.StructuredAggregationQuery.Aggregations) == 2 &&
			req.StructuredAggregationQuery.Aggregations[0].Min != nil &&
			req.StructuredAggregationQuery.Aggregations[1].Max != nil
	})).Return(expectedResponse, nil)

	// Create Fiber app
	app := fiber.New()

	// Setup route with parameters
	app.Post("/api/v1/organizations/:organizationID/projects/:projectID/databases/:databaseID/documents:runAggregationQuery", handler.RunAggregationQuery)

	// Create request body with extended operators
	requestBody := map[string]interface{}{
		"structuredAggregationQuery": map[string]interface{}{
			"structuredQuery": map[string]interface{}{
				"from": []map[string]interface{}{
					{
						"collectionId": "productos",
					},
				},
			},
			"aggregations": []map[string]interface{}{
				{
					"alias": "precio_mas_bajo",
					"min": map[string]interface{}{
						"field": map[string]interface{}{
							"fieldPath": "price",
						},
					},
				},
				{
					"alias": "precio_mas_alto",
					"max": map[string]interface{}{
						"field": map[string]interface{}{
							"fieldPath": "price",
						},
					},
				},
			},
		},
	}

	requestBodyBytes, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/documents:runAggregationQuery", bytes.NewBuffer(requestBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify mock was called
	mockUC.AssertExpectations(t)
}

func TestRunAggregationQuery_InvalidRequest(t *testing.T) {
	// Setup
	mockUC := &MockFirestoreUCForAggregation{}
	handler := &HTTPHandler{FirestoreUC: mockUC, Log: TestLogger{}}

	// Create Fiber app
	app := fiber.New()

	// Setup route with parameters
	app.Post("/api/v1/organizations/:organizationID/projects/:projectID/databases/:databaseID/documents:runAggregationQuery", handler.RunAggregationQuery)

	// Create invalid request body (missing structuredAggregationQuery)
	requestBody := map[string]interface{}{
		"invalidField": "test",
	}

	requestBodyBytes, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/organizations/test-org/projects/test-project/databases/test-database/documents:runAggregationQuery", bytes.NewBuffer(requestBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 400, resp.StatusCode)

	// Parse response
	var respBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	assert.Equal(t, "missing_structured_aggregation_query", respBody["error"])
}
