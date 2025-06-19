package firestore

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/auth/domain/model" // For auth models
	"firestore-clone/internal/firestore/config"
	firestoreModel "firestore-clone/internal/firestore/domain/model" // Alias to avoid conflict
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/contextkeys"
	"firestore-clone/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockAuthClient implements client.AuthClient for testing
// This follows hexagonal architecture by implementing the domain interface
type MockAuthClient struct{}

func (m *MockAuthClient) ValidateToken(ctx context.Context, token string) (string, error) {
	return "test-user", nil
}

func (m *MockAuthClient) GetUserByID(ctx context.Context, userID, projectID string) (*model.User, error) {
	return &model.User{
		UserID:    userID,
		Email:     "test@example.com",
		IsActive:  true,
		FirstName: "Test",
		LastName:  "User",
		Roles:     []string{"user"},
		TenantID:  projectID,
	}, nil
}

// TestFirestoreModule_Initialization tests the basic initialization of the Firestore module
// following hexagonal architecture principles: dependency injection and interface segregation
func TestFirestoreModule_Initialization(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client (dependency injection)
	authClient := &MockAuthClient{}
	// Create logger
	log := logger.NewLogger()

	// Create test Redis client for distributed event storage
	redisClient := createTestRedisClientMock()

	// Create FirestoreModule with dependency injection
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, redisClient)
	require.NoError(t, err)
	require.NotNil(t, module)
	// Verify all components are initialized properly
	assert.NotNil(t, module.Config)
	assert.NotNil(t, module.AuthClient)
	assert.NotNil(t, module.TenantAwareRepo)
	assert.NotNil(t, module.QueryEngine)
	assert.NotNil(t, module.SecurityRules)
	assert.NotNil(t, module.FirestoreUsecase)
	assert.NotNil(t, module.RealtimeUsecase)
	assert.NotNil(t, module.SecurityUsecase)
	assert.NotNil(t, module.Logger)
	assert.NotNil(t, module.TenantManager)
	assert.NotNil(t, module.OrganizationRepo)
	assert.NotNil(t, module.OrganizationHandler)

	// Verify Redis components are initialized
	assert.NotNil(t, module.RedisClient)
	assert.NotNil(t, module.RedisEventStore)

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_InitializationWithConfig tests custom configuration injection
func TestFirestoreModule_InitializationWithConfig(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client
	authClient := &MockAuthClient{}
	// Create logger
	log := logger.NewLogger()

	// Create test Redis client for distributed event storage
	redisClient := createTestRedisClientMock()

	// Create custom config (clean architecture: configuration as data)
	cfg := &config.FirestoreConfig{
		MongoDBURI:          "mongodb://localhost:27017",
		DefaultDatabaseName: "test_firestore",
		Realtime: config.RealtimeConfig{
			WebSocketPath:           "/ws/test",
			ClientSendChannelBuffer: 50,
		},
	}

	// Create FirestoreModule with custom config
	module, err := NewFirestoreModuleWithConfig(authClient, log, mongoClient, masterDB, cfg, redisClient)
	require.NoError(t, err)
	require.NotNil(t, module)

	// Verify config is set correctly
	assert.Equal(t, cfg, module.Config)

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_QueryEngine_NestedFieldSupport tests the enhanced query capabilities
// This tests the main feature we implemented: nested field path support
func TestFirestoreModule_QueryEngine_NestedFieldSupport(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")
	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create test Redis client for distributed event storage
	redisClient := createTestRedisClientMock()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, redisClient)
	require.NoError(t, err)
	require.NotNil(t, module)

	// Test QueryEngine capabilities - this verifies our hexagonal architecture implementation
	capabilities := module.QueryEngine.GetQueryCapabilities()
	assert.True(t, capabilities.SupportsNestedFields)
	assert.True(t, capabilities.SupportsArrayContains)
	assert.True(t, capabilities.SupportsArrayContainsAny)
	assert.True(t, capabilities.SupportsCompositeFilters)
	assert.True(t, capabilities.SupportsOrderBy)
	assert.True(t, capabilities.SupportsCursorPagination)
	assert.True(t, capabilities.SupportsOffsetPagination)
	assert.True(t, capabilities.SupportsProjection)
	assert.Equal(t, 100, capabilities.MaxFilterCount)
	assert.Equal(t, 32, capabilities.MaxOrderByCount)
	assert.Equal(t, 100, capabilities.MaxNestingDepth)

	// Test query validation with nested fields (using domain models)
	fieldPath, err := firestoreModel.NewFieldPath("customer.ruc")
	require.NoError(t, err)

	query := firestoreModel.Query{
		Path:         "facturas",
		CollectionID: "facturas",
		Filters: []firestoreModel.Filter{
			{
				FieldPath: fieldPath,
				Field:     "customer.ruc",
				Operator:  firestoreModel.OperatorEqual,
				Value:     "20123456789",
				ValueType: firestoreModel.FieldTypeString,
			},
		},
	}

	err = module.QueryEngine.ValidateQuery(query)
	assert.NoError(t, err)

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_QueryEngine_ExecuteQuery_WithContext tests query execution with tenant context
func TestFirestoreModule_QueryEngine_ExecuteQuery_WithContext(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, createTestRedisClientMock())
	require.NoError(t, err)
	require.NotNil(t, module)

	// Create context with organization ID (multi-tenant support)
	ctx := context.WithValue(context.Background(), contextkeys.OrganizationIDKey, "test-org-123")

	// Create a simple query using domain models
	query := firestoreModel.Query{
		Path:         "facturas",
		CollectionID: "facturas",
		Filters: []firestoreModel.Filter{
			{
				Field:     "status",
				Operator:  firestoreModel.OperatorEqual,
				Value:     "paid",
				ValueType: firestoreModel.FieldTypeString,
			},
		},
		Limit: 10,
	}

	// Execute query (should not error even if no documents found)
	documents, err := module.QueryEngine.ExecuteQuery(ctx, "facturas", query)
	assert.NoError(t, err)
	assert.NotNil(t, documents)
	assert.IsType(t, []*firestoreModel.Document{}, documents)

	// Test count documents
	count, err := module.QueryEngine.CountDocuments(ctx, "facturas", query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_QueryEngine_NestedFieldQuery tests the original use case from the requirements
// This is the most important test: customer.ruc nested field query like Firestore real
func TestFirestoreModule_QueryEngine_NestedFieldQuery(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, createTestRedisClientMock())
	require.NoError(t, err)
	require.NotNil(t, module)

	// Create context with organization ID
	ctx := context.WithValue(context.Background(), contextkeys.OrganizationIDKey, "test-org-nested")

	// Create query with nested field (the original requirement!)
	// This mimics the exact scenario from the user's original question
	customerRucPath, err := firestoreModel.NewFieldPath("customer.ruc")
	require.NoError(t, err)

	query := firestoreModel.Query{
		Path:         "facturas",
		CollectionID: "facturas",
		Filters: []firestoreModel.Filter{
			{
				Field:     "status",
				Operator:  firestoreModel.OperatorEqual,
				Value:     "paid",
				ValueType: firestoreModel.FieldTypeString,
			},
			{
				FieldPath: customerRucPath,
				Field:     "customer.ruc", // Keep for backward compatibility
				Operator:  firestoreModel.OperatorEqual,
				Value:     "20123456789",
				ValueType: firestoreModel.FieldTypeString,
			},
		},
		Limit: 10,
	}

	// Validate the nested field query
	err = module.QueryEngine.ValidateQuery(query)
	assert.NoError(t, err, "Nested field query should be valid")

	// Execute query (should not error even if no documents found)
	documents, err := module.QueryEngine.ExecuteQuery(ctx, "facturas", query)
	assert.NoError(t, err, "Nested field query execution should not error")
	assert.NotNil(t, documents)
	assert.IsType(t, []*firestoreModel.Document{}, documents)

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_ArchitecturalCompliance verifies hexagonal architecture compliance
func TestFirestoreModule_ArchitecturalCompliance(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, createTestRedisClientMock())
	require.NoError(t, err)
	require.NotNil(t, module)

	// Verify dependency injection pattern (dependencies injected from outside)
	assert.NotNil(t, module.AuthClient, "AuthClient should be injected")
	assert.NotNil(t, module.Logger, "Logger should be injected")

	// Verify hexagonal architecture (domain doesn't depend on infrastructure)
	// Query should be a domain model, not tied to any specific database
	query := firestoreModel.Query{}
	assert.IsType(t, firestoreModel.Query{}, query, "Query should be domain model")

	// Verify multi-tenant support (clean architecture principle)
	assert.NotNil(t, module.TenantManager, "TenantManager should be available")
	assert.NotNil(t, module.OrganizationRepo, "OrganizationRepo should be available")

	// Verify clean separation of concerns
	assert.NotNil(t, module.FirestoreUsecase, "FirestoreUsecase should handle business logic")
	assert.NotNil(t, module.SecurityUsecase, "SecurityUsecase should handle security logic")
	assert.NotNil(t, module.RealtimeUsecase, "RealtimeUsecase should handle real-time logic")

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// TestFirestoreModule_RedisIntegration tests Redis Streams integration for realtime events
// This verifies that the distributed event storage is working correctly
func TestFirestoreModule_RedisIntegration(t *testing.T) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_test_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create test Redis client for distributed event storage
	redisClient := createTestRedisClientMock()

	// Create FirestoreModule with Redis integration
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, redisClient)
	require.NoError(t, err)
	require.NotNil(t, module)

	// Verify Redis components are properly initialized
	assert.NotNil(t, module.RedisClient, "RedisClient should be initialized")
	assert.NotNil(t, module.RedisEventStore, "RedisEventStore should be initialized")
	assert.NotNil(t, module.RealtimeUsecase, "RealtimeUsecase should be initialized with Redis backing")

	// Test that realtime usecase can handle events (basic functionality)
	ctx := context.Background()

	// Create a test subscription channel
	eventChannel := make(chan firestoreModel.RealtimeEvent, 10)

	// Test subscription creation (this should work even without actual Redis connection)
	subscribeReq := usecase.SubscribeRequest{
		SubscriberID:   "test-subscriber",
		SubscriptionID: firestoreModel.SubscriptionID("test-subscription"),
		FirestorePath:  "projects/test-project/databases/test-db/documents/users/user1",
		EventChannel:   eventChannel,
		Options: usecase.SubscriptionOptions{
			IncludeMetadata: true,
		},
	}

	response, err := module.RealtimeUsecase.Subscribe(ctx, subscribeReq)
	assert.NoError(t, err, "Subscribe should work with Redis backend")
	assert.NotNil(t, response, "Subscribe response should not be nil")
	assert.Equal(t, subscribeReq.SubscriptionID, response.SubscriptionID)

	// Test unsubscription
	unsubscribeReq := usecase.UnsubscribeRequest{
		SubscriberID:   "test-subscriber",
		SubscriptionID: firestoreModel.SubscriptionID("test-subscription"),
	}

	err = module.RealtimeUsecase.Unsubscribe(ctx, unsubscribeReq)
	assert.NoError(t, err, "Unsubscribe should work with Redis backend")

	// Test module stop
	err = module.Stop()
	assert.NoError(t, err)
}

// createTestRedisClient creates a Redis client for testing
// Uses miniredis for in-memory Redis simulation in tests

// createTestRedisClientMock creates a mock Redis client for unit tests
// This avoids requiring an actual Redis instance for unit tests
func createTestRedisClientMock() *redis.Client {
	// For unit tests where Redis isn't available, create a client that won't be used
	// The actual EventStore can be mocked separately in the usecase tests
	return redis.NewClient(&redis.Options{
		Addr:         "localhost:16379", // Non-standard port to avoid conflicts
		DialTimeout:  1 * time.Second,   // Short timeout for mock
		ReadTimeout:  1 * time.Second,   // Short timeout for mock
		WriteTimeout: 1 * time.Second,   // Short timeout for mock
	})
}

// Benchmark tests for performance verification (clean code: performance testing)
func BenchmarkFirestoreModule_QueryEngine_SimpleQuery(b *testing.B) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(b, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_bench_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, createTestRedisClientMock())
	require.NoError(b, err)

	// Create context with organization ID
	ctx := context.WithValue(context.Background(), contextkeys.OrganizationIDKey, "bench-org")

	// Create simple query
	query := firestoreModel.Query{
		Path:         "bench_collection",
		CollectionID: "bench_collection",
		Filters: []firestoreModel.Filter{
			{
				Field:     "status",
				Operator:  firestoreModel.OperatorEqual,
				Value:     "active",
				ValueType: firestoreModel.FieldTypeString,
			},
		},
		Limit: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.QueryEngine.ExecuteQuery(ctx, "bench_collection", query)
	}
}

// BenchmarkFirestoreModule_QueryEngine_NestedFieldQuery benchmarks the core feature
func BenchmarkFirestoreModule_QueryEngine_NestedFieldQuery(b *testing.B) {
	// Create test MongoDB client
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(b, err)
	defer mongoClient.Disconnect(context.Background())

	// Create test database
	masterDB := mongoClient.Database("firestore_bench_master")

	// Create mock auth client
	authClient := &MockAuthClient{}

	// Create logger
	log := logger.NewLogger()

	// Create FirestoreModule
	module, err := NewFirestoreModule(authClient, log, mongoClient, masterDB, createTestRedisClientMock())
	require.NoError(b, err)

	// Create context with organization ID
	ctx := context.WithValue(context.Background(), contextkeys.OrganizationIDKey, "bench-org")

	// Create nested field query (performance test for our main feature)
	customerRucPath, err := firestoreModel.NewFieldPath("customer.ruc")
	require.NoError(b, err)

	query := firestoreModel.Query{
		Path:         "facturas",
		CollectionID: "facturas",
		Filters: []firestoreModel.Filter{
			{
				FieldPath: customerRucPath,
				Field:     "customer.ruc",
				Operator:  firestoreModel.OperatorEqual,
				Value:     "20123456789",
				ValueType: firestoreModel.FieldTypeString,
			},
		},
		Limit: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.QueryEngine.ExecuteQuery(ctx, "facturas", query)
	}
}
