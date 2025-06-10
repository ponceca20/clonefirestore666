package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestSecurityRulesEngine_LoadRules(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		mockLogger := new(MockLogger) // Assumes MockLogger is defined elsewhere
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)

		projectID := "test-project"
		databaseID := "test-db"

		expectedRules := []*repository.SecurityRule{
			{Match: "/users/{userId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "request.auth != null && request.auth.uid == userId"}, Deny: make(map[repository.OperationType]string)},
			{Match: "/posts/{postId}", Priority: 0, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}, Deny: make(map[repository.OperationType]string)},
		}

		first := mtest.CreateCursorResponse(1, fmt.Sprintf("%s.%s", databaseID, "security_rules"), mtest.FirstBatch, bson.D{
			{Key: "project_id", Value: projectID},
			{Key: "database_id", Value: databaseID},
			{Key: "match", Value: "/users/{userId}"},
			{Key: "priority", Value: int32(1)},
			{Key: "allow", Value: bson.M{string(repository.OperationRead): "request.auth != null && request.auth.uid == userId"}},
		})
		second := mtest.CreateCursorResponse(1, fmt.Sprintf("%s.%s", databaseID, "security_rules"), mtest.NextBatch, bson.D{
			{Key: "project_id", Value: projectID},
			{Key: "database_id", Value: databaseID},
			{Key: "match", Value: "/posts/{postId}"},
			{Key: "priority", Value: int32(0)},
			{Key: "allow", Value: bson.M{string(repository.OperationRead): "true"}},
		})
		killCursors := mtest.CreateCursorResponse(0, fmt.Sprintf("%s.%s", databaseID, "security_rules"), mtest.NextBatch)
		mt.AddMockResponses(first, second, killCursors)

		rules, err := engine.LoadRules(context.Background(), projectID, databaseID)

		assert.NoError(t, err)
		assert.Equal(t, expectedRules, rules)
		mockLogger.AssertExpectations(t)
	})

	mt.Run("find_error", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "Failed to load security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Return()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 1, Message: "find error"}))

		_, err := engine.LoadRules(context.Background(), "proj", "db")
		assert.Error(t, err)
		mockLogger.AssertCalled(t, "Error", "Failed to load security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"))
	})
}

func TestSecurityRulesEngine_SaveRules(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Info", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "test-project"
		databaseID := "test-db"
		rulesToSave := []*repository.SecurityRule{
			{Match: "/users/{userId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "request.auth != null && request.auth.uid == userId"}},
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse()) // For DeleteMany
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // For InsertMany

		err := engine.SaveRules(context.Background(), projectID, databaseID, rulesToSave)
		assert.NoError(t, err)
		mockLogger.AssertCalled(t, "Info", "Saved security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"))

		// Verify cache is cleared
		engine.cacheMu.RLock()
		_, exists := engine.rulesCache[projectID+":"+databaseID]
		engine.cacheMu.RUnlock()
		assert.False(t, exists, "Cache should be cleared after saving rules")
	})

	mt.Run("delete_many_error", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "Failed to save security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Return()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 1, Message: "delete error"}))

		err := engine.SaveRules(context.Background(), "proj", "db", []*repository.SecurityRule{})
		assert.Error(t, err)
		mockLogger.AssertCalled(t, "Error", "Failed to save security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"))
	})

	mt.Run("insert_many_error", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "Failed to save security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Return()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		rulesToSave := []*repository.SecurityRule{
			{Match: "/test", Priority: 1},
		}
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // For DeleteMany
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 1, Message: "insert error"}))

		err := engine.SaveRules(context.Background(), "proj", "db", rulesToSave)
		assert.Error(t, err)
		mockLogger.AssertCalled(t, "Error", "Failed to save security rules", mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"))
	})
}

func TestSecurityRulesEngine_ValidateRules(t *testing.T) {
	mockLogger := new(MockLogger)                                            // Real logger can be used if preferred for this simple validation
	engine := NewSecurityRulesEngine(nil, mockLogger).(*SecurityRulesEngine) // DB not needed for validation logic

	tests := []struct {
		name      string
		rules     []*repository.SecurityRule
		expectErr bool
		errSubstr string
	}{
		{
			name:      "valid_rules",
			rules:     []*repository.SecurityRule{{Match: "/users/{userId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}}},
			expectErr: false,
		},
		{
			name:      "empty_rules",
			rules:     []*repository.SecurityRule{},
			expectErr: false,
		},
		{
			name:      "nil_rules",
			rules:     nil,
			expectErr: false,
		},
		{
			name:      "duplicate_priority",
			rules:     []*repository.SecurityRule{{Match: "/a", Priority: 1}, {Match: "/b", Priority: 1}},
			expectErr: true,
			errSubstr: "duplicate priority 1",
		},
		{
			name:      "invalid_match_pattern_empty",
			rules:     []*repository.SecurityRule{{Match: "", Priority: 1}},
			expectErr: true,
			errSubstr: "invalid match pattern ''",
		},
		{
			name:      "invalid_match_pattern_no_leading_slash",
			rules:     []*repository.SecurityRule{{Match: "users/{userId}", Priority: 1}},
			expectErr: true,
			errSubstr: "invalid match pattern 'users/{userId}'",
		},
		{
			name:      "invalid_match_pattern_double_slash",
			rules:     []*repository.SecurityRule{{Match: "//users", Priority: 1}},
			expectErr: true,
			errSubstr: "invalid match pattern '//users'",
		},
		{
			name:      "invalid_match_pattern_trailing_slash_non_recursive",
			rules:     []*repository.SecurityRule{{Match: "/users/", Priority: 1}},
			expectErr: true,
			errSubstr: "invalid match pattern '/users/'",
		},
		{
			name:      "valid_match_pattern_recursive_wildcard",
			rules:     []*repository.SecurityRule{{Match: "/users/{document=**}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}}},
			expectErr: false,
		},
		{
			name:      "invalid_condition_empty_allow",
			rules:     []*repository.SecurityRule{{Match: "/users/{userId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: ""}}},
			expectErr: true,
			errSubstr: "invalid allow condition for operation 'read': condition cannot be empty",
		},
		{
			name:      "invalid_condition_empty_deny",
			rules:     []*repository.SecurityRule{{Match: "/users/{userId}", Priority: 1, Deny: map[repository.OperationType]string{repository.OperationWrite: ""}}},
			expectErr: true,
			errSubstr: "invalid deny condition for operation 'write': condition cannot be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.ValidateRules(tc.rules)
			if tc.expectErr {
				assert.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityRulesEngine_EvaluateAccess(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("allow_by_simple_true_rule", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe()        // Assuming one field for error logs in EvaluateAccess
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe() // Adjusted for 3 fields: path, operation, authContext

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj1"
		databaseID := "db1"

		// Pre-cache rules
		engine.cacheMu.Lock()
		engine.rulesCache[projectID+":"+databaseID] = []*repository.SecurityRule{
			{Match: "/documents/{docId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}},
		}
		engine.cacheMu.Unlock()

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/documents/doc123",
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.Equal(t, "/documents/{docId}", result.AllowedBy)
	})

	mt.Run("deny_by_simple_false_rule", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj1"
		databaseID := "db1"

		engine.cacheMu.Lock()
		engine.rulesCache[projectID+":"+databaseID] = []*repository.SecurityRule{
			{Match: "/documents/{docId}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "false"}},
		}
		engine.cacheMu.Unlock()

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/documents/doc123",
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, "No matching allow rule or explicit deny.", result.Reason) // Updated expected reason
	})

	mt.Run("deny_by_explicit_deny_rule", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj1"
		databaseID := "db1"

		engine.cacheMu.Lock()
		engine.rulesCache[projectID+":"+databaseID] = []*repository.SecurityRule{
			{Match: "/documents/{docId}", Priority: 1, Deny: map[repository.OperationType]string{repository.OperationRead: "true"}},
			{Match: "/documents/{docId}", Priority: 0, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}}, // Lower priority allow
		}
		engine.cacheMu.Unlock()

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/documents/doc123",
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, "/documents/{docId}", result.DeniedBy)
	})

	mt.Run("load_rules_from_db_if_not_cached", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		// For LoadRules call within EvaluateAccess
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field"), mock.AnythingOfType("zapcore.Field")).Maybe()
		// For EvaluateAccess's own debug logs
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe() // For potential error in LoadRules

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj-load"
		databaseID := "db-load"

		first := mtest.CreateCursorResponse(1, fmt.Sprintf("%s.%s", databaseID, "security_rules"), mtest.FirstBatch, bson.D{
			{Key: "project_id", Value: projectID},
			{Key: "database_id", Value: databaseID},
			{Key: "match", Value: "/load/{id}"},
			{Key: "priority", Value: int32(1)},
			{Key: "allow", Value: bson.M{string(repository.OperationRead): "true"}},
		})
		killCursors := mtest.CreateCursorResponse(0, fmt.Sprintf("%s.%s", databaseID, "security_rules"), mtest.NextBatch)
		mt.AddMockResponses(first, killCursors)

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/load/item1",
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.Equal(t, "/load/{id}", result.AllowedBy)

		// Verify rules are now cached
		engine.cacheMu.RLock()
		cachedRules, exists := engine.rulesCache[projectID+":"+databaseID]
		engine.cacheMu.RUnlock()
		assert.True(t, exists)
		assert.Len(t, cachedRules, 1)
		assert.Equal(t, "/load/{id}", cachedRules[0].Match)
	})

	mt.Run("path_matching_recursive_wildcard", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj-recursive"
		databaseID := "db-recursive"

		engine.cacheMu.Lock()
		engine.rulesCache[projectID+":"+databaseID] = []*repository.SecurityRule{
			{Match: "/users/{userId}/{document=**}", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}},
		}
		engine.cacheMu.Unlock()

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/users/user123/profile/settings", // Matches recursive wildcard
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.Equal(t, "/users/{userId}/{document=**}", result.AllowedBy)
	})

	mt.Run("no_matching_rule", func(mt *mtest.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string"), mock.AnythingOfType("zapcore.Field")).Maybe()

		engine := NewSecurityRulesEngine(mt.DB, mockLogger).(*SecurityRulesEngine)
		projectID := "proj-nomatch"
		databaseID := "db-nomatch"

		engine.cacheMu.Lock()
		engine.rulesCache[projectID+":"+databaseID] = []*repository.SecurityRule{
			{Match: "/specific/path", Priority: 1, Allow: map[repository.OperationType]string{repository.OperationRead: "true"}},
		}
		engine.cacheMu.Unlock()

		securityContext := &repository.SecurityContext{
			ProjectID:  projectID,
			DatabaseID: databaseID,
			Path:       "/other/path",
			Timestamp:  time.Now().Unix(),
		}

		result, err := engine.EvaluateAccess(context.Background(), repository.OperationRead, securityContext)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Equal(t, "Default deny: No rules matched the path.", result.Reason)
	})

}

// TestSecurityRulesEngine_ClearCache is a simple test for cache clearing.
func TestSecurityRulesEngine_ClearCache(t *testing.T) {
	mockLogger := new(MockLogger)
	engine := NewSecurityRulesEngine(nil, mockLogger).(*SecurityRulesEngine) // DB not needed

	projectID1 := "proj1"
	databaseID1 := "db1"
	projectID2 := "proj2"
	databaseID2 := "db2"

	// Populate cache
	engine.cacheMu.Lock()
	engine.rulesCache[projectID1+":"+databaseID1] = []*repository.SecurityRule{{Match: "/a"}}
	engine.rulesCache[projectID2+":"+databaseID2] = []*repository.SecurityRule{{Match: "/b"}}
	engine.cacheMu.Unlock()

	// Clear cache for project1/database1
	engine.ClearCache(projectID1, databaseID1)

	engine.cacheMu.RLock()
	_, exists1 := engine.rulesCache[projectID1+":"+databaseID1]
	_, exists2 := engine.rulesCache[projectID2+":"+databaseID2]
	engine.cacheMu.RUnlock()

	assert.False(t, exists1, "Cache for proj1/db1 should be cleared")
	assert.True(t, exists2, "Cache for proj2/db2 should remain")

	// Clear all cache
	engine.ClearAllCache()
	engine.cacheMu.RLock()
	_, exists2AfterClearAll := engine.rulesCache[projectID2+":"+databaseID2]
	engine.cacheMu.RUnlock()
	assert.False(t, exists2AfterClearAll, "All cache should be cleared")
}

// Note: Full CEL evaluation testing is complex and would require a CEL mock or a dedicated CEL evaluation test suite.
// These tests focus on the interaction with storage, cache, rule structure validation, and basic path matching logic within EvaluateAccess.
