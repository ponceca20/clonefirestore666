package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	authModel "firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/logger"
)

type MockFirestoreRepo struct {
	projects  map[string]*model.Project
	dbMu      sync.Mutex
	databases map[string]map[string]*model.Database // projectID -> databaseID -> *Database
}

func NewMockFirestoreRepo() *MockFirestoreRepo {
	return &MockFirestoreRepo{
		projects:  make(map[string]*model.Project),
		databases: make(map[string]map[string]*model.Database),
	}
}

func (m *MockFirestoreRepo) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields map[string]*model.FieldValue) (*model.Document, error) {
	return &model.Document{DocumentID: documentID}, nil
}
func (m *MockFirestoreRepo) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	// Return a document with all fields needed for atomic tests
	return &model.Document{
		DocumentID: documentID,
		Fields: map[string]*model.FieldValue{
			"count":     model.NewFieldValue(int64(42)),
			"counter":   model.NewFieldValue(int64(1)),           // For atomic increment tests
			"stock":     model.NewFieldValue(int64(10)),          // Para TestAtomicIncrement_Valid
			"tags":      model.NewFieldValue([]interface{}{"a"}), // For array operations
			"updatedAt": model.NewFieldValue(time.Now()),         // For server timestamp
		},
	}, nil
}
func (m *MockFirestoreRepo) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields map[string]*model.FieldValue, mask []string) (*model.Document, error) {
	return &model.Document{DocumentID: documentID}, nil
}
func (m *MockFirestoreRepo) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return []*model.Document{{DocumentID: "doc1"}}, "", nil
}
func (m *MockFirestoreRepo) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	if field == "" {
		return fmt.Errorf("field is required")
	}
	if field != "stock" && field != "count" && field != "counter" {
		return fmt.Errorf("field not found after increment: %s", field)
	}
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	if field == "" {
		return fmt.Errorf("field is required")
	}
	if len(elements) == 0 {
		return fmt.Errorf("elements required")
	}
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	if field == "" {
		return fmt.Errorf("field is required")
	}
	if len(elements) == 0 {
		return fmt.Errorf("elements required")
	}
	return nil
}
func (m *MockFirestoreRepo) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("field is required")
	}
	return nil
}

// Add missing methods to MockFirestoreRepo to fully implement FirestoreRepository
func (m *MockFirestoreRepo) CreateProject(ctx context.Context, project *model.Project) error {
	if _, exists := m.projects[project.ProjectID]; exists {
		return fmt.Errorf("project already exists")
	}
	m.projects[project.ProjectID] = project
	return nil
}
func (m *MockFirestoreRepo) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	if p, ok := m.projects[projectID]; ok {
		return p, nil
	}
	// Cambia el error plano por uno de tipo AppError compatible con errors.IsNotFound
	return nil, errors.NewNotFoundError("project")
}
func (m *MockFirestoreRepo) UpdateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteProject(ctx context.Context, projectID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return []*model.Project{
		{
			ProjectID: "p1",
		},
	}, nil
}
func (m *MockFirestoreRepo) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	m.dbMu.Lock()
	defer m.dbMu.Unlock()
	if m.databases[projectID] == nil {
		m.databases[projectID] = make(map[string]*model.Database)
	}
	if _, exists := m.databases[projectID][database.DatabaseID]; exists {
		return errors.NewConflictError("database already exists")
	}
	m.databases[projectID][database.DatabaseID] = database
	return nil
}
func (m *MockFirestoreRepo) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	m.dbMu.Lock()
	defer m.dbMu.Unlock()
	if dbs, ok := m.databases[projectID]; ok {
		if db, ok := dbs[databaseID]; ok {
			return db, nil
		}
	}
	return nil, errors.NewNotFoundError("database")
}
func (m *MockFirestoreRepo) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	m.dbMu.Lock()
	defer m.dbMu.Unlock()
	if m.databases[projectID] == nil {
		return errors.NewNotFoundError("database")
	}
	if _, exists := m.databases[projectID][database.DatabaseID]; !exists {
		return errors.NewNotFoundError("database")
	}
	m.databases[projectID][database.DatabaseID] = database
	return nil
}
func (m *MockFirestoreRepo) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	m.dbMu.Lock()
	defer m.dbMu.Unlock()
	if m.databases[projectID] == nil {
		return errors.NewNotFoundError("database")
	}
	if _, exists := m.databases[projectID][databaseID]; !exists {
		return errors.NewNotFoundError("database")
	}
	delete(m.databases[projectID], databaseID)
	return nil
}
func (m *MockFirestoreRepo) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	m.dbMu.Lock()
	defer m.dbMu.Unlock()
	var out []*model.Database
	if dbs, ok := m.databases[projectID]; ok {
		for _, db := range dbs {
			out = append(out, db)
		}
	}
	return out, nil
}
func (m *MockFirestoreRepo) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return &model.Collection{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
	}, nil
}

func (m *MockFirestoreRepo) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return []*model.Collection{
		{
			ProjectID:    projectID,
			DatabaseID:   databaseID,
			CollectionID: "c1",
		},
	}, nil
}
func (m *MockFirestoreRepo) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) DeleteDocumentByPath(ctx context.Context, path string) error {
	return nil
}
func (m *MockFirestoreRepo) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	// Devuelve un documento simulado para pruebas de integración
	return []*model.Document{{DocumentID: "doc1"}}, nil
}
func (m *MockFirestoreRepo) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunTransaction(ctx context.Context, fn func(repository.Transaction) error) error {
	return nil
}

// Add missing RunBatchWrite mock for usecase interface
// Corrijo la firma del mock para que cumpla la interfaz del repositorio
func (m *MockFirestoreRepo) RunBatchWrite(ctx context.Context, projectID string, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return []*model.WriteResult{{UpdateTime: time.Now()}}, nil
}

// Add missing CreateIndex, DeleteIndex, ListIndexes to MockFirestoreRepo
func (m *MockFirestoreRepo) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, idx *model.CollectionIndex) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID, indexName string) error {
	return nil
}

// Fix ListIndexes to return []*model.CollectionIndex
func (m *MockFirestoreRepo) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return []*model.CollectionIndex{
		{
			Name:   "idx1",
			Fields: []model.IndexField{{Path: "f1", Order: model.IndexFieldOrderAscending}},
			State:  "READY",
		},
	}, nil
}
func (m *MockFirestoreRepo) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return []string{"sub1"}, nil
}

// Add other required methods for other usecases as needed

// Update MockLogger to return Logger interface for WithFields, WithContext, WithComponent
type MockLogger struct{}

func (m *MockLogger) Info(args ...interface{})                               {}
func (m *MockLogger) Error(args ...interface{})                              {}
func (m *MockLogger) Debug(args ...interface{})                              {}
func (m *MockLogger) Warn(args ...interface{})                               {}
func (m *MockLogger) Fatal(args ...interface{})                              {}
func (m *MockLogger) Debugf(format string, args ...interface{})              {}
func (m *MockLogger) Infof(format string, args ...interface{})               {}
func (m *MockLogger) Warnf(format string, args ...interface{})               {}
func (m *MockLogger) Errorf(format string, args ...interface{})              {}
func (m *MockLogger) Fatalf(format string, args ...interface{})              {}
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *MockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *MockLogger) WithComponent(component string) logger.Logger           { return m }

// MockRealtimeUsecase implements RealtimeUsecase for testing with full synchronization
type MockRealtimeUsecase struct {
	subscriptions map[string]map[model.SubscriptionID]*Subscription // subscriberID -> subscriptionID -> subscription
	mu            sync.RWMutex
	events        []model.RealtimeEvent // Store events for verification
}

func NewMockRealtimeUsecase() *MockRealtimeUsecase {
	return &MockRealtimeUsecase{
		subscriptions: make(map[string]map[model.SubscriptionID]*Subscription),
		events:        make([]model.RealtimeEvent, 0),
	}
}

func (m *MockRealtimeUsecase) Subscribe(ctx context.Context, req SubscribeRequest) (*SubscribeResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscriptions[req.SubscriberID] == nil {
		m.subscriptions[req.SubscriberID] = make(map[model.SubscriptionID]*Subscription)
	}

	subscription := &Subscription{
		SubscriberID:   req.SubscriberID,
		SubscriptionID: req.SubscriptionID,
		FirestorePath:  req.FirestorePath,
		EventChannel:   req.EventChannel,
		CreatedAt:      time.Now(),
		LastHeartbeat:  time.Now(),
		ResumeToken:    req.ResumeToken,
		Query:          req.Query,
		IsActive:       true,
		Options:        req.Options,
	}

	m.subscriptions[req.SubscriberID][req.SubscriptionID] = subscription

	return &SubscribeResponse{
		SubscriptionID:  req.SubscriptionID,
		InitialSnapshot: true,
		ResumeToken:     req.ResumeToken,
		CreatedAt:       subscription.CreatedAt,
	}, nil
}

func (m *MockRealtimeUsecase) Unsubscribe(ctx context.Context, req UnsubscribeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscriptions[req.SubscriberID] != nil {
		delete(m.subscriptions[req.SubscriberID], req.SubscriptionID)
		if len(m.subscriptions[req.SubscriberID]) == 0 {
			delete(m.subscriptions, req.SubscriberID)
		}
	}
	return nil
}

func (m *MockRealtimeUsecase) UnsubscribeAll(ctx context.Context, subscriberID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscriptions, subscriberID)
	return nil
}

func (m *MockRealtimeUsecase) PublishEvent(ctx context.Context, event model.RealtimeEvent) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, subscriptions := range m.subscriptions {
		for _, subscription := range subscriptions {
			if subscription.FirestorePath == event.FullPath || strings.HasPrefix(event.FullPath, subscription.FirestorePath) {
				select {
				case subscription.EventChannel <- event:
					// Event sent successfully
				default:
					// Channel is full, skip
				}
			}
		}
	}
	return nil
}

func (m *MockRealtimeUsecase) GetEventsSince(ctx context.Context, firestorePath string, resumeToken model.ResumeToken) ([]model.RealtimeEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple implementation for testing - return all events
	return m.events, nil
}

func (m *MockRealtimeUsecase) SendHeartbeat(ctx context.Context) error {
	// Mock implementation - no-op
	return nil
}

func (m *MockRealtimeUsecase) UpdateLastHeartbeat(subscriberID string, subscriptionID model.SubscriptionID) error {
	// Mock implementation - no-op
	return nil
}

func (m *MockRealtimeUsecase) CleanupStaleConnections(ctx context.Context, timeout time.Duration) error {
	// Mock implementation - no-op
	return nil
}

func (m *MockRealtimeUsecase) GetSubscriberCount(firestorePath string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, subscriptions := range m.subscriptions {
		for _, subscription := range subscriptions {
			if subscription.FirestorePath == firestorePath {
				count++
			}
		}
	}
	return count
}

func (m *MockRealtimeUsecase) GetActiveSubscriptions(subscriberID string) map[model.SubscriptionID]*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[model.SubscriptionID]*Subscription)
	if subscriptions, exists := m.subscriptions[subscriberID]; exists {
		for subscriptionID, subscription := range subscriptions {
			result[subscriptionID] = subscription
		}
	}
	return result
}

func (m *MockRealtimeUsecase) ValidatePermissions(ctx context.Context, subscriberID string, permissionValidator PermissionValidator) error {
	// Mock implementation - always pass
	return nil
}

func (m *MockRealtimeUsecase) GetHealthStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSubscriptions := 0
	for _, subscriptions := range m.subscriptions {
		totalSubscriptions += len(subscriptions)
	}

	return HealthStatus{
		IsHealthy:           true,
		ActiveSubscriptions: totalSubscriptions,
		ActiveConnections:   len(m.subscriptions),
		LastHealthCheck:     time.Now(),
		EventStoreSize:      len(m.events),
	}
}

func (m *MockRealtimeUsecase) GetMetrics() RealtimeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSubscriptions := 0
	for _, subscriptions := range m.subscriptions {
		totalSubscriptions += len(subscriptions)
	}

	return RealtimeMetrics{
		TotalSubscriptions: int64(totalSubscriptions),
		TotalEvents:        int64(len(m.events)),
		ActiveSubscribers:  len(m.subscriptions),
		EventsPerSecond:    0.0, // Mock value
		AverageLatency:     time.Millisecond,
		LastMetricsUpdate:  time.Now(),
	}
}

func (m *MockRealtimeUsecase) GetEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events)
}

// MockSecurityUsecase implements SecurityUsecase for testing with configurable responses
type MockSecurityUsecase struct {
	shouldValidate bool
	validationErr  error
	mu             sync.RWMutex
}

func NewMockSecurityUsecase() *MockSecurityUsecase {
	return &MockSecurityUsecase{
		shouldValidate: true,
		validationErr:  nil,
	}
}

func (m *MockSecurityUsecase) SetValidationResult(shouldValidate bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldValidate = shouldValidate
	m.validationErr = err
}

func (m *MockSecurityUsecase) ValidateRead(ctx context.Context, user *authModel.User, path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.shouldValidate {
		if m.validationErr != nil {
			return m.validationErr
		}
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockSecurityUsecase) ValidateWrite(ctx context.Context, user *authModel.User, path string, data map[string]interface{}) error {
	return m.ValidateRead(ctx, user, path)
}

func (m *MockSecurityUsecase) ValidateCreate(ctx context.Context, user *authModel.User, path string, data map[string]interface{}) error {
	return m.ValidateRead(ctx, user, path)
}

func (m *MockSecurityUsecase) ValidateUpdate(ctx context.Context, user *authModel.User, path string, data map[string]interface{}, existingData map[string]interface{}) error {
	return m.ValidateRead(ctx, user, path)
}

func (m *MockSecurityUsecase) ValidateDelete(ctx context.Context, user *authModel.User, path string) error {
	return m.ValidateRead(ctx, user, path)
}

// MockAuthClient implements AuthClient for testing with configurable user
type MockAuthClient struct {
	user *authModel.User
	mu   sync.RWMutex
}

func NewMockAuthClient() *MockAuthClient {
	return &MockAuthClient{}
}

func (m *MockAuthClient) SetUser(user *authModel.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.user = user
}

func (m *MockAuthClient) ValidateToken(ctx context.Context, token string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.user == nil {
		return "", fmt.Errorf("no user configured")
	}
	return m.user.UserID, nil
}

func (m *MockAuthClient) GetUserByID(ctx context.Context, userID string, projectID string) (*authModel.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.user != nil && m.user.UserID == userID {
		return m.user, nil
	}
	return nil, fmt.Errorf("user not found")
}

// MockQueryEngine implements QueryEngine for testing
type MockQueryEngine struct{}

func NewMockQueryEngine() *MockQueryEngine {
	return &MockQueryEngine{}
}

func (m *MockQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	// Return a mock document that matches the query
	return []*model.Document{
		{
			DocumentID: "mock-doc-1",
			Fields: map[string]*model.FieldValue{
				"name":      model.NewFieldValue("Test Product"),
				"available": model.NewFieldValue(true),
				"price":     model.NewFieldValue(99.99),
			},
		},
	}, nil
}

func (m *MockQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	return m.ExecuteQuery(ctx, collectionPath, query)
}

func (m *MockQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	return 1, nil
}

func (m *MockQueryEngine) ValidateQuery(query model.Query) error {
	return nil
}

func (m *MockQueryEngine) GetQueryCapabilities() repository.QueryCapabilities {
	return repository.QueryCapabilities{
		SupportsNestedFields:     true,
		SupportsArrayContains:    true,
		SupportsArrayContainsAny: true,
		SupportsCompositeFilters: true,
		SupportsOrderBy:          true,
		SupportsCursorPagination: true,
		SupportsOffsetPagination: true,
		SupportsProjection:       true,
		MaxFilterCount:           30,
		MaxOrderByCount:          3,
		MaxNestingDepth:          10,
	}
}

func (m *MockQueryEngine) ExecuteAggregationPipeline(ctx context.Context, projectID, databaseID, collectionPath string, pipeline []interface{}) ([]map[string]interface{}, error) {
	// Return mock aggregation results
	return []map[string]interface{}{
		{
			"count": 42,
			"sum":   1234.56,
			"avg":   78.9,
		},
	}, nil
}

func (m *MockQueryEngine) BuildMongoFilter(filters []model.Filter) (interface{}, error) {
	// Return a simple mock MongoDB filter
	return map[string]interface{}{
		"mockFilter": true,
	}, nil
}

// MockProjectionService implements ProjectionService for testing
type MockProjectionService struct{}

func NewMockProjectionService() *MockProjectionService {
	return &MockProjectionService{}
}

func (m *MockProjectionService) ApplyProjection(documents []*model.Document, selectFields []string) []*model.Document {
	if len(selectFields) == 0 {
		return documents
	}

	// Apply projection by filtering fields
	projectedDocs := make([]*model.Document, len(documents))
	for i, doc := range documents {
		projectedDoc := &model.Document{
			DocumentID: doc.DocumentID,
			Fields:     make(map[string]*model.FieldValue),
		}

		// Copy only selected fields
		for _, field := range selectFields {
			if value, exists := doc.Fields[field]; exists {
				projectedDoc.Fields[field] = value
			}
		}

		projectedDocs[i] = projectedDoc
	}

	return projectedDocs
}

func (m *MockProjectionService) ValidateProjectionFields(fields []string) error {
	return nil
}

func (m *MockProjectionService) IsProjectionRequired(fields []string) bool {
	return len(fields) > 0
}

// MockSecurityRulesEngine implements SecurityRulesEngine for testing
type MockSecurityRulesEngine struct{}

func NewMockSecurityRulesEngine() *MockSecurityRulesEngine {
	return &MockSecurityRulesEngine{}
}

func (m *MockSecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	return &repository.RuleEvaluationResult{
		Allowed: true,
		Reason:  "Mock allowed",
	}, nil
}

func (m *MockSecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return []*repository.SecurityRule{}, nil
}

func (m *MockSecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	return nil
}

func (m *MockSecurityRulesEngine) ValidateRules(rules []*repository.SecurityRule) error {
	return nil
}

// ClearCache clears the rules cache for a specific project/database (mock implementation)
func (m *MockSecurityRulesEngine) ClearCache(projectID, databaseID string) {
	// Mock implementation - no-op
}

// SetResourceAccessor sets the resource accessor for CEL functions (mock implementation)
func (m *MockSecurityRulesEngine) SetResourceAccessor(accessor repository.ResourceAccessor) {
	// Mock implementation - no-op
}
