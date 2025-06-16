// Package http provides integration tests for composite filter functionality
// in the Firestore clone HTTP adapter layer following hexagonal architecture.
//
// This test file validates the end-to-end functionality of:
// - Composite filter parsing from Firestore REST API format
// - AND/OR operations in query processing
// - Complex query scenarios mimicking real Firestore usage
// - Integration between HTTP layer and use case layer
//
// The tests ensure compatibility with Google Firestore query semantics
// and proper error handling throughout the request processing pipeline.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFirestoreUCForComposite implements FirestoreUsecaseInterface for testing composite filters
// following hexagonal architecture principles - provides complete interface implementation
type MockFirestoreUCForComposite struct {
	mock.Mock
}

// QueryDocuments is the main method we're testing - uses correct interface signature
func (m *MockFirestoreUCForComposite) QueryDocuments(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*model.Document), args.Error(1)
}

// RunQuery implements the query interface method
func (m *MockFirestoreUCForComposite) RunQuery(ctx context.Context, req usecase.QueryRequest) ([]*model.Document, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*model.Document), args.Error(1)
}

// Implement remaining interface methods as no-ops for compliance with usecase.FirestoreUsecaseInterface
func (m *MockFirestoreUCForComposite) CreateDocument(ctx context.Context, req usecase.CreateDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) GetDocument(ctx context.Context, req usecase.GetDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) UpdateDocument(ctx context.Context, req usecase.UpdateDocumentRequest) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) DeleteDocument(ctx context.Context, req usecase.DeleteDocumentRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListDocuments(ctx context.Context, req usecase.ListDocumentsRequest) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) CreateCollection(ctx context.Context, req usecase.CreateCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) GetCollection(ctx context.Context, req usecase.GetCollectionRequest) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) UpdateCollection(ctx context.Context, req usecase.UpdateCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListCollections(ctx context.Context, req usecase.ListCollectionsRequest) ([]*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) DeleteCollection(ctx context.Context, req usecase.DeleteCollectionRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListSubcollections(ctx context.Context, req usecase.ListSubcollectionsRequest) ([]model.Subcollection, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) CreateIndex(ctx context.Context, req usecase.CreateIndexRequest) (*model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) DeleteIndex(ctx context.Context, req usecase.DeleteIndexRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListIndexes(ctx context.Context, req usecase.ListIndexesRequest) ([]model.Index, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) RunBatchWrite(ctx context.Context, req usecase.BatchWriteRequest) (*model.BatchWriteResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) BeginTransaction(ctx context.Context, projectID string) (string, error) {
	return "", nil
}
func (m *MockFirestoreUCForComposite) CommitTransaction(ctx context.Context, projectID string, transactionID string) error {
	return nil
}
func (m *MockFirestoreUCForComposite) CreateProject(ctx context.Context, req usecase.CreateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) GetProject(ctx context.Context, req usecase.GetProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) UpdateProject(ctx context.Context, req usecase.UpdateProjectRequest) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) DeleteProject(ctx context.Context, req usecase.DeleteProjectRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListProjects(ctx context.Context, req usecase.ListProjectsRequest) ([]*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) CreateDatabase(ctx context.Context, req usecase.CreateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) GetDatabase(ctx context.Context, req usecase.GetDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) UpdateDatabase(ctx context.Context, req usecase.UpdateDatabaseRequest) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) DeleteDatabase(ctx context.Context, req usecase.DeleteDatabaseRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) ListDatabases(ctx context.Context, req usecase.ListDatabasesRequest) ([]*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) AtomicIncrement(ctx context.Context, req usecase.AtomicIncrementRequest) (*usecase.AtomicIncrementResponse, error) {
	return nil, nil
}
func (m *MockFirestoreUCForComposite) AtomicArrayUnion(ctx context.Context, req usecase.AtomicArrayUnionRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) AtomicArrayRemove(ctx context.Context, req usecase.AtomicArrayRemoveRequest) error {
	return nil
}
func (m *MockFirestoreUCForComposite) AtomicServerTimestamp(ctx context.Context, req usecase.AtomicServerTimestampRequest) error {
	return nil
}

// compositeTestLogger provides logging for composite filter tests
type compositeTestLogger struct{}

func (l compositeTestLogger) Debug(args ...interface{})                              {}
func (l compositeTestLogger) Info(args ...interface{})                               {}
func (l compositeTestLogger) Warn(args ...interface{})                               {}
func (l compositeTestLogger) Error(args ...interface{})                              {}
func (l compositeTestLogger) Fatal(args ...interface{})                              {}
func (l compositeTestLogger) Debugf(format string, args ...interface{})              {}
func (l compositeTestLogger) Infof(format string, args ...interface{})               {}
func (l compositeTestLogger) Warnf(format string, args ...interface{})               {}
func (l compositeTestLogger) Errorf(format string, args ...interface{})              {}
func (l compositeTestLogger) Fatalf(format string, args ...interface{})              {}
func (l compositeTestLogger) WithFields(fields map[string]interface{}) logger.Logger { return l }
func (l compositeTestLogger) WithContext(ctx context.Context) logger.Logger          { return l }
func (l compositeTestLogger) WithComponent(component string) logger.Logger           { return l }

func TestQueryDocuments_CompositeFilterAND_PriceRange(t *testing.T) {
	// Test the exact query that was failing: price >= 50 AND price <= 500
	app := fiber.New()
	mockUC := &MockFirestoreUCForComposite{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: compositeTestLogger{}}

	// Set up the route
	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)
	// Mock the expected response - return list of documents directly
	expectedDocuments := []*model.Document{
		{
			DocumentID: "doc1",
			Fields: map[string]*model.FieldValue{
				"price": {ValueType: model.FieldTypeDouble, Value: 100.0},
				"name":  {ValueType: model.FieldTypeString, Value: "Test Product"},
			},
		},
	} // Set up the mock expectation
	mockUC.On("RunQuery", mock.Anything, mock.MatchedBy(func(req usecase.QueryRequest) bool {
		// AND filters should be flattened, so we expect 2 separate field filters
		if req.StructuredQuery == nil || len(req.StructuredQuery.Filters) != 2 {
			return false
		}
		// Both should be field filters (not composite)
		for _, filter := range req.StructuredQuery.Filters {
			if filter.Composite != "" {
				return false
			}
		}
		return true
	})).Return(expectedDocuments, nil)

	// Create the request body (exact copy from the failing request)
	requestBody := `{
		"from": [
			{
				"collectionId": "products"
			}
		],
		"where": {
			"compositeFilter": {
				"op": "AND",
				"filters": [
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "price"
							},
							"op": "GREATER_THAN_OR_EQUAL",
							"value": {
								"doubleValue": 50.00
							}
						}
					},
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "price"
							},
							"op": "LESS_THAN_OR_EQUAL",
							"value": {
								"doubleValue": 500.00
							}
						}
					}
				]
			}
		}
	}`

	// Create the HTTP request
	req := httptest.NewRequest("POST", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/query/productos", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify the response
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify that we get documents back (not an error)
	assert.NotNil(t, result["documents"])
	assert.Equal(t, float64(1), result["count"])

	// Verify that the mock was called
	mockUC.AssertExpectations(t)
}

func TestQueryDocuments_CompositeFilterAND_ThreeConditions(t *testing.T) {
	// Test the three-condition AND query that was failing
	app := fiber.New()
	mockUC := &MockFirestoreUCForComposite{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: compositeTestLogger{}}

	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	expectedDocuments := []*model.Document{}
	mockUC.On("RunQuery", mock.Anything, mock.MatchedBy(func(req usecase.QueryRequest) bool {
		// AND filters should be flattened, so we expect 3 separate filters
		if req.StructuredQuery == nil || len(req.StructuredQuery.Filters) != 3 {
			return false
		}
		// All should be field filters (not composite)
		for _, filter := range req.StructuredQuery.Filters {
			if filter.Composite != "" {
				return false
			}
		}
		return true
	})).Return(expectedDocuments, nil)

	requestBody := `{
		"from": [
			{
				"collectionId": "products"
			}
		],
		"where": {
			"compositeFilter": {
				"op": "AND",
				"filters": [
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "price"
							},
							"op": "GREATER_THAN_OR_EQUAL",
							"value": {
								"doubleValue": 30.00
							}
						}
					},
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "price"
							},
							"op": "LESS_THAN_OR_EQUAL",
							"value": {
								"doubleValue": 500.00
							}
						}
					},
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "brand"
							},
							"op": "EQUAL",
							"value": {
								"stringValue": "MobileGenius"
							}
						}
					}
				]
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/query/productos", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Should not have an error
	assert.Nil(t, result["error"])
	assert.NotNil(t, result["documents"])

	mockUC.AssertExpectations(t)
}

func TestQueryDocuments_CompositeFilterAND_CategoryAndAvailable(t *testing.T) {
	// Test the category + available query that was failing
	app := fiber.New()
	mockUC := &MockFirestoreUCForComposite{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: compositeTestLogger{}}

	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	expectedDocuments := []*model.Document{}
	mockUC.On("RunQuery", mock.Anything, mock.MatchedBy(func(req usecase.QueryRequest) bool {
		if req.StructuredQuery == nil || len(req.StructuredQuery.Filters) != 2 {
			return false
		}
		// Check for category and available fields (now flattened)
		hasCategory := false
		hasAvailable := false
		for _, filter := range req.StructuredQuery.Filters {
			if filter.Field == "category" {
				hasCategory = true
			}
			if filter.Field == "available" {
				hasAvailable = true
			}
		}
		return hasCategory && hasAvailable
	})).Return(expectedDocuments, nil)

	requestBody := `{
		"from": [
			{
				"collectionId": "products"
			}
		],
		"where": {
			"compositeFilter": {
				"op": "AND",
				"filters": [
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "category"
							},
							"op": "EQUAL",
							"value": {
								"stringValue": "Peripherals"
							}
						}
					},
					{
						"fieldFilter": {
							"field": {
								"fieldPath": "available"
							},
							"op": "EQUAL",
							"value": {
								"booleanValue": true
							}
						}
					}
				]
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/query/productos", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Should not have an error
	assert.Nil(t, result["error"])
	assert.NotNil(t, result["documents"])

	mockUC.AssertExpectations(t)
}

func TestQueryDocuments_SimpleFieldFilter_StillWorks(t *testing.T) {
	// Verify that simple field filters still work (this was working before)
	app := fiber.New()
	mockUC := &MockFirestoreUCForComposite{}
	h := &HTTPHandler{FirestoreUC: mockUC, Log: compositeTestLogger{}}

	app.Post("/api/v1/organizations/:orgID/projects/:projectID/databases/:databaseID/query/:collectionID", h.QueryDocuments)

	expectedDocuments := []*model.Document{
		{
			DocumentID: "doc1",
			Fields: map[string]*model.FieldValue{
				"brand": {ValueType: model.FieldTypeString, Value: "TechMaster"},
				"name":  {ValueType: model.FieldTypeString, Value: "Test Product"},
			},
		},
	}
	mockUC.On("RunQuery", mock.Anything, mock.MatchedBy(func(req usecase.QueryRequest) bool {
		// Should have one simple filter
		if req.StructuredQuery == nil || len(req.StructuredQuery.Filters) != 1 {
			return false
		}
		filter := req.StructuredQuery.Filters[0]
		return filter.Field == "brand" && filter.Operator == model.OperatorEqual && filter.Value == "TechMaster"
	})).Return(expectedDocuments, nil)

	requestBody := `{
		"from": [
			{
				"collectionId": "products"
			}
		],
		"where": {
			"fieldFilter": {
				"field": {
					"fieldPath": "brand"
				},
				"op": "EQUAL",
				"value": {
					"stringValue": "TechMaster"
				}
			}
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/organizations/new-org-1749766807/projects/new-proj-from-postman/databases/Database-2026/query/productos", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotNil(t, result["documents"])
	assert.Equal(t, float64(1), result["count"])

	mockUC.AssertExpectations(t)
}
