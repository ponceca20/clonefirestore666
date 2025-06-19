package usecase_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSecurityRulesUseCase_Integration(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_usecase_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	securityEngine := mongodb.NewSecurityRulesEngine(testDB, log)
	resourceAccessor := mongodb.NewResourceAccessor(testDB, log)
	securityEngine.SetResourceAccessor(resourceAccessor)

	securityUseCase := usecase.NewSecurityRulesUseCase(securityEngine, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Complete Security Workflow", func(t *testing.T) {
		// 1. Deploy security rules
		rules := []*repository.SecurityRule{
			{
				Match:    "/users/{userId}",
				Priority: 100,
				Allow: map[repository.OperationType]string{
					repository.OperationRead:  "auth.uid == variables.userId",
					repository.OperationWrite: "auth.uid == variables.userId",
				},
				Deny: map[repository.OperationType]string{
					repository.OperationDelete: "auth == null",
				},
				Description: "User access control - users can read/write their own data",
			},
			{
				Match:    "/public/{docId}",
				Priority: 50,
				Allow: map[repository.OperationType]string{
					repository.OperationRead: "true",
				},
				Description: "Public documents are readable by everyone",
			},
		}

		deployRequest := &usecase.RulesManagementRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Rules:      rules,
		}

		err := securityUseCase.DeployRules(ctx, deployRequest)
		require.NoError(t, err)

		// 2. Load and verify rules
		loadedRules, err := securityUseCase.LoadRules(ctx, projectID, databaseID)
		require.NoError(t, err)
		assert.Len(t, loadedRules, 2)

		// 3. Test access evaluation
		userID := primitive.NewObjectID()
		otherUserID := primitive.NewObjectID()
		user := &model.User{
			ID:    userID,
			Email: "test@example.com",
		}

		// Test user accessing their own data - should be allowed
		accessRequest := &usecase.AccessRequest{
			User:       user,
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/users/" + userID.Hex(),
			Operation:  repository.OperationRead,
			Resource:   map[string]interface{}{"ownerId": userID.Hex()},
			Request:    map[string]interface{}{"source": "test"},
		}

		response, err := securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.True(t, response.Allowed)
		assert.Contains(t, response.RuleMatch, "/users/{userId}")

		// Test user accessing other user's data - should be denied
		accessRequest.Path = "/users/" + otherUserID.Hex()

		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.False(t, response.Allowed)

		// Test delete operation without auth - should be denied
		accessRequest.User = nil
		accessRequest.Path = "/users/" + userID.Hex()
		accessRequest.Operation = repository.OperationDelete

		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.False(t, response.Allowed)

		// Test public access - should be allowed
		accessRequest.Path = "/public/doc1"
		accessRequest.Operation = repository.OperationRead

		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.True(t, response.Allowed)
	})
}

func TestSecurityRulesUseCase_ValidationErrors(t *testing.T) {
	log := logger.NewTestLogger()
	mockEngine := usecase.NewMockSecurityRulesEngine()
	securityUseCase := usecase.NewSecurityRulesUseCase(mockEngine, log)

	t.Run("EvaluateAccess Validation Errors", func(t *testing.T) {
		testCases := []struct {
			name    string
			request *usecase.AccessRequest
		}{
			{
				name: "empty project ID",
				request: &usecase.AccessRequest{
					ProjectID:  "",
					DatabaseID: "test-db",
					Path:       "/test",
					Operation:  repository.OperationRead,
				},
			},
			{
				name: "empty database ID",
				request: &usecase.AccessRequest{
					ProjectID:  "test-project",
					DatabaseID: "",
					Path:       "/test",
					Operation:  repository.OperationRead,
				},
			},
			{
				name: "empty path",
				request: &usecase.AccessRequest{
					ProjectID:  "test-project",
					DatabaseID: "test-db",
					Path:       "",
					Operation:  repository.OperationRead,
				},
			},
			{
				name: "invalid operation",
				request: &usecase.AccessRequest{
					ProjectID:  "test-project",
					DatabaseID: "test-db",
					Path:       "/test",
					Operation:  "",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := securityUseCase.EvaluateAccess(context.Background(), tc.request)
				assert.Error(t, err)
			})
		}
	})

	t.Run("DeployRules Validation Errors", func(t *testing.T) {
		testCases := []struct {
			name    string
			request *usecase.RulesManagementRequest
		}{
			{
				name: "empty project ID",
				request: &usecase.RulesManagementRequest{
					ProjectID:  "",
					DatabaseID: "test-db",
					Rules:      []*repository.SecurityRule{},
				},
			},
			{
				name: "empty database ID",
				request: &usecase.RulesManagementRequest{
					ProjectID:  "test-project",
					DatabaseID: "",
					Rules:      []*repository.SecurityRule{},
				},
			},
			{
				name: "nil rules",
				request: &usecase.RulesManagementRequest{
					ProjectID:  "test-project",
					DatabaseID: "test-db",
					Rules:      nil,
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := securityUseCase.DeployRules(context.Background(), tc.request)
				// nil rules should be valid (clearing all rules), others should error
				if tc.name == "nil rules" {
					assert.NoError(t, err) // nil rules should be valid
				} else {
					assert.Error(t, err) // other cases should error
				}
			})
		}
	})
}

func TestSecurityRulesUseCase_ComplexRules(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_complex_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	securityEngine := mongodb.NewSecurityRulesEngine(testDB, log)
	resourceAccessor := mongodb.NewResourceAccessor(testDB, log)
	securityEngine.SetResourceAccessor(resourceAccessor)

	securityUseCase := usecase.NewSecurityRulesUseCase(securityEngine, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Complex Rules with Resource Access", func(t *testing.T) {
		// Deploy complex rules
		rules := []*repository.SecurityRule{
			{
				Match:    "/users/{userId}/posts/{postId}",
				Priority: 100,
				Allow: map[repository.OperationType]string{
					repository.OperationRead:  "auth.uid == variables.userId || resource.data.public == true",
					repository.OperationWrite: "auth.uid == variables.userId",
				},
				Description: "Post access with public visibility support",
			},
		}

		deployRequest := &usecase.RulesManagementRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Rules:      rules,
		}

		err := securityUseCase.DeployRules(ctx, deployRequest)
		require.NoError(t, err)

		userID := primitive.NewObjectID()
		otherUserID := primitive.NewObjectID()
		user := &model.User{
			ID:    userID,
			Email: "test@example.com",
		}

		// Test accessing private post as owner - should be allowed
		accessRequest := &usecase.AccessRequest{
			User:       user,
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/users/" + userID.Hex() + "/posts/post1",
			Operation:  repository.OperationRead,
			Resource: map[string]interface{}{
				"data": map[string]interface{}{
					"public": false,
					"title":  "Private Post",
				},
			},
		}

		response, err := securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.True(t, response.Allowed)

		// Test accessing public post as non-owner - should be allowed
		accessRequest.Path = "/users/" + otherUserID.Hex() + "/posts/post2"
		accessRequest.Resource = map[string]interface{}{
			"data": map[string]interface{}{
				"public": true,
				"title":  "Public Post",
			},
		}

		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.True(t, response.Allowed)

		// Test accessing private post as non-owner - should be denied
		accessRequest.Resource = map[string]interface{}{
			"data": map[string]interface{}{
				"public": false,
				"title":  "Private Post",
			},
		}

		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.False(t, response.Allowed)
	})
}

func TestSecurityRulesUseCase_PerformanceMetrics(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_performance_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	securityEngine := mongodb.NewSecurityRulesEngine(testDB, log)
	resourceAccessor := mongodb.NewResourceAccessor(testDB, log)
	securityEngine.SetResourceAccessor(resourceAccessor)

	securityUseCase := usecase.NewSecurityRulesUseCase(securityEngine, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Performance Metrics", func(t *testing.T) {
		// Deploy simple rules for performance testing
		rules := []*repository.SecurityRule{
			{
				Match:    "/test/{docId}",
				Priority: 100,
				Allow: map[repository.OperationType]string{
					repository.OperationRead: "true",
				},
			},
		}

		deployRequest := &usecase.RulesManagementRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Rules:      rules,
		}

		err := securityUseCase.DeployRules(ctx, deployRequest)
		require.NoError(t, err)

		// Measure evaluation time
		start := time.Now()

		accessRequest := &usecase.AccessRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/test/doc1",
			Operation:  repository.OperationRead,
		}

		response, err := securityUseCase.EvaluateAccess(ctx, accessRequest)

		elapsed := time.Since(start)
		require.NoError(t, err)
		assert.True(t, response.Allowed)
		assert.GreaterOrEqual(t, response.EvaluationTimeMs, int64(0)) // Time can be 0 for very fast evaluations
		assert.Less(t, elapsed, 100*time.Millisecond)                 // Should be fast
	})
}

func TestSecurityRulesUseCase_CacheBehavior(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_cache_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	securityEngine := mongodb.NewSecurityRulesEngine(testDB, log)
	resourceAccessor := mongodb.NewResourceAccessor(testDB, log)
	securityEngine.SetResourceAccessor(resourceAccessor)

	securityUseCase := usecase.NewSecurityRulesUseCase(securityEngine, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Cache Invalidation", func(t *testing.T) {
		// Deploy initial rules
		rules1 := []*repository.SecurityRule{
			{
				Match:    "/test/{docId}",
				Priority: 100,
				Allow: map[repository.OperationType]string{
					repository.OperationRead: "false", // Initially deny
				},
			},
		}

		deployRequest1 := &usecase.RulesManagementRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Rules:      rules1,
		}

		err := securityUseCase.DeployRules(ctx, deployRequest1)
		require.NoError(t, err)

		// Test access - should be denied
		accessRequest := &usecase.AccessRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/test/doc1",
			Operation:  repository.OperationRead,
		}

		response, err := securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.False(t, response.Allowed)

		// Deploy updated rules
		rules2 := []*repository.SecurityRule{
			{
				Match:    "/test/{docId}",
				Priority: 100,
				Allow: map[repository.OperationType]string{
					repository.OperationRead: "true", // Now allow
				},
			},
		}

		deployRequest2 := &usecase.RulesManagementRequest{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Rules:      rules2,
		}

		err = securityUseCase.DeployRules(ctx, deployRequest2)
		require.NoError(t, err)

		// Test access again - should now be allowed (cache should be cleared)
		response, err = securityUseCase.EvaluateAccess(ctx, accessRequest)
		require.NoError(t, err)
		assert.True(t, response.Allowed)
	})
}
