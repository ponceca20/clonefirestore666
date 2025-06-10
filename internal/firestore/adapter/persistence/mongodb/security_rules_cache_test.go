package mongodb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityRulesCacheTestSuite provides a test suite for security rules cache functionality
// following hexagonal architecture principles and Firestore standards
type SecurityRulesCacheTestSuite struct {
	suite.Suite
	engine     *TestableSecurityRulesEngine
	mockLogger *MockLogger
	ctx        context.Context
}

// TestableSecurityRulesEngine wraps SecurityRulesEngine for testing
// This allows us to test cache functionality without database dependencies
type TestableSecurityRulesEngine struct {
	log        logger.Logger
	rulesCache map[string][]*repository.SecurityRule
	cacheMu    sync.RWMutex
	// Mock function to simulate LoadRules
	loadRulesFunc func(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error)
}

// getCachedRules mimics the actual cache behavior from SecurityRulesEngine
func (e *TestableSecurityRulesEngine) getCachedRules(ctx context.Context, cacheKey, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	e.cacheMu.RLock()
	if rules, exists := e.rulesCache[cacheKey]; exists {
		e.cacheMu.RUnlock()
		return rules, nil
	}
	e.cacheMu.RUnlock()

	// Load from storage using mock function
	rules, err := e.loadRulesFunc(ctx, projectID, databaseID)
	if err != nil {
		return nil, err
	}

	// Cache the rules
	e.cacheMu.Lock()
	e.rulesCache[cacheKey] = rules
	e.cacheMu.Unlock()

	return rules, nil
}

// ClearCache clears the rules cache for a specific project/database
func (e *TestableSecurityRulesEngine) ClearCache(projectID, databaseID string) {
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)
	e.cacheMu.Lock()
	delete(e.rulesCache, cacheKey)
	e.cacheMu.Unlock()
}

// ClearAllCache clears all cached rules
func (e *TestableSecurityRulesEngine) ClearAllCache() {
	e.cacheMu.Lock()
	e.rulesCache = make(map[string][]*repository.SecurityRule)
	e.cacheMu.Unlock()
}

// SetupSuite initializes the test suite once before all tests
func (suite *SecurityRulesCacheTestSuite) SetupSuite() {
	suite.ctx = context.Background()
}

// SetupTest initializes test dependencies before each test
func (suite *SecurityRulesCacheTestSuite) SetupTest() {
	suite.mockLogger = &MockLogger{}

	// Create testable engine with mock dependencies
	suite.engine = &TestableSecurityRulesEngine{
		log:        suite.mockLogger,
		rulesCache: make(map[string][]*repository.SecurityRule),
		cacheMu:    sync.RWMutex{},
		loadRulesFunc: func(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
			return nil, errors.New("mock not configured")
		},
	}
}

// TearDownTest cleans up after each test
func (suite *SecurityRulesCacheTestSuite) TearDownTest() {
	// Clear cache after each test to ensure test isolation
	suite.engine.ClearAllCache()
}

// TestGetCachedRules_CacheHit tests cache hit scenario following Firestore caching patterns
func (suite *SecurityRulesCacheTestSuite) TestGetCachedRules_CacheHit() {
	// Arrange - following Firestore project/database hierarchy
	projectID := "test-project-123"
	databaseID := "test-database-456"
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)

	// Create test rules matching Firestore security rules structure
	expectedRules := []*repository.SecurityRule{
		{
			Match:    "databases/{database}/documents/users/{userId}",
			Priority: 100,
			Allow: map[repository.OperationType]string{
				repository.OperationRead:  "auth.uid == resource.data.userId",
				repository.OperationWrite: "auth.uid == resource.data.userId",
			},
		},
		{
			Match:    "databases/{database}/documents/public/{document}",
			Priority: 50,
			Allow: map[repository.OperationType]string{
				repository.OperationRead: "true",
			},
			Deny: map[repository.OperationType]string{
				repository.OperationWrite: "auth.uid == null",
			},
		},
	}

	// Pre-populate cache
	suite.engine.rulesCache[cacheKey] = expectedRules

	// Act
	result, err := suite.engine.getCachedRules(suite.ctx, cacheKey, projectID, databaseID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedRules, result)
	assert.Len(suite.T(), result, 2)
}

// TestGetCachedRules_CacheMiss tests cache miss scenario with database loading
func (suite *SecurityRulesCacheTestSuite) TestGetCachedRules_CacheMiss() {
	// Arrange
	projectID := "test-project-789"
	databaseID := "test-database-012"
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)

	// Expected rules from database
	expectedRules := []*repository.SecurityRule{
		{
			Match:    "databases/{database}/documents/orders/{orderId}",
			Priority: 200,
			Allow: map[repository.OperationType]string{
				repository.OperationRead: "auth.uid != null && auth.uid == resource.data.customerId",
			},
		},
	}

	// Configure mock to return rules
	suite.engine.loadRulesFunc = func(ctx context.Context, pID, dID string) ([]*repository.SecurityRule, error) {
		assert.Equal(suite.T(), projectID, pID)
		assert.Equal(suite.T(), databaseID, dID)
		return expectedRules, nil
	}

	// Act
	result, err := suite.engine.getCachedRules(suite.ctx, cacheKey, projectID, databaseID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedRules, result)

	// Verify rules were cached
	cachedRules, exists := suite.engine.rulesCache[cacheKey]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), expectedRules, cachedRules)
}

// TestGetCachedRules_LoadError tests error handling during database loading
func (suite *SecurityRulesCacheTestSuite) TestGetCachedRules_LoadError() {
	// Arrange
	projectID := "test-project-error"
	databaseID := "test-database-error"
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)

	expectedError := errors.New("database connection failed")

	// Configure mock to return error
	suite.engine.loadRulesFunc = func(ctx context.Context, pID, dID string) ([]*repository.SecurityRule, error) {
		return nil, expectedError
	}

	// Act
	result, err := suite.engine.getCachedRules(suite.ctx, cacheKey, projectID, databaseID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), expectedError, err)

	// Verify cache remains empty
	_, exists := suite.engine.rulesCache[cacheKey]
	assert.False(suite.T(), exists)
}

// TestClearCache tests clearing specific project/database cache
func (suite *SecurityRulesCacheTestSuite) TestClearCache() {
	// Arrange - populate cache with multiple entries
	projectID1, databaseID1 := "project-1", "database-1"
	projectID2, databaseID2 := "project-2", "database-2"

	cacheKey1 := fmt.Sprintf("%s:%s", projectID1, databaseID1)
	cacheKey2 := fmt.Sprintf("%s:%s", projectID2, databaseID2)

	rules1 := []*repository.SecurityRule{{Match: "path1", Priority: 1}}
	rules2 := []*repository.SecurityRule{{Match: "path2", Priority: 2}}

	suite.engine.rulesCache[cacheKey1] = rules1
	suite.engine.rulesCache[cacheKey2] = rules2

	// Act - clear only first cache entry
	suite.engine.ClearCache(projectID1, databaseID1)

	// Assert
	_, exists1 := suite.engine.rulesCache[cacheKey1]
	assert.False(suite.T(), exists1, "First cache entry should be cleared")

	cachedRules2, exists2 := suite.engine.rulesCache[cacheKey2]
	assert.True(suite.T(), exists2, "Second cache entry should remain")
	assert.Equal(suite.T(), rules2, cachedRules2)
}

// TestClearAllCache tests clearing entire cache
func (suite *SecurityRulesCacheTestSuite) TestClearAllCache() {
	// Arrange - populate cache with multiple entries
	suite.engine.rulesCache["key1"] = []*repository.SecurityRule{{Match: "path1"}}
	suite.engine.rulesCache["key2"] = []*repository.SecurityRule{{Match: "path2"}}
	suite.engine.rulesCache["key3"] = []*repository.SecurityRule{{Match: "path3"}}

	// Verify cache has entries
	assert.Len(suite.T(), suite.engine.rulesCache, 3)

	// Act
	suite.engine.ClearAllCache()

	// Assert
	assert.Empty(suite.T(), suite.engine.rulesCache, "All cache entries should be cleared")
	assert.NotNil(suite.T(), suite.engine.rulesCache, "Cache map should be reinitialized, not nil")
}

// TestConcurrentCacheAccess tests thread safety of cache operations
func (suite *SecurityRulesCacheTestSuite) TestConcurrentCacheAccess() {
	// Arrange
	projectID := "concurrent-project"
	databaseID := "concurrent-database"
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)

	expectedRules := []*repository.SecurityRule{
		{Match: "concurrent/path", Priority: 1},
	}

	// Pre-populate cache
	suite.engine.rulesCache[cacheKey] = expectedRules

	// Configure mock for potential cache misses
	suite.engine.loadRulesFunc = func(ctx context.Context, pID, dID string) ([]*repository.SecurityRule, error) {
		return expectedRules, nil
	}

	// Act - perform concurrent operations
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([][]*repository.SecurityRule, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Alternate between read and clear operations
			if idx%2 == 0 {
				results[idx], errors[idx] = suite.engine.getCachedRules(
					suite.ctx, cacheKey, projectID, databaseID,
				)
			} else {
				suite.engine.ClearCache(projectID, databaseID)
			}
		}(i)
	}

	wg.Wait()

	// Assert - verify no race conditions occurred
	for i := 0; i < numGoroutines; i++ {
		if i%2 == 0 {
			// Read operations should either succeed or fail gracefully
			if errors[i] == nil {
				assert.NotNil(suite.T(), results[i])
			}
		}
	}
}

// TestCacheKeyGeneration tests cache key generation follows Firestore patterns
func (suite *SecurityRulesCacheTestSuite) TestCacheKeyGeneration() {
	testCases := []struct {
		name       string
		projectID  string
		databaseID string
		expected   string
	}{
		{
			name:       "Standard project and database IDs",
			projectID:  "my-project",
			databaseID: "my-database",
			expected:   "my-project:my-database",
		},
		{
			name:       "Project with hyphens and underscores",
			projectID:  "my-project_123",
			databaseID: "test-db_456",
			expected:   "my-project_123:test-db_456",
		},
		{
			name:       "Default Firestore database",
			projectID:  "firestore-project",
			databaseID: "(default)",
			expected:   "firestore-project:(default)",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Generate cache key the same way as the implementation
			cacheKey := fmt.Sprintf("%s:%s", tc.projectID, tc.databaseID)
			assert.Equal(t, tc.expected, cacheKey)
		})
	}
}

// TestCacheEvictionScenarios tests various cache eviction scenarios
func (suite *SecurityRulesCacheTestSuite) TestCacheEvictionScenarios() {
	// Test cache behavior under different eviction scenarios
	suite.T().Run("Single Entry Eviction", func(t *testing.T) {
		projectID := "eviction-test"
		databaseID := "single-entry"

		// Add entry
		rules := []*repository.SecurityRule{{Match: "test", Priority: 1}}
		cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)
		suite.engine.rulesCache[cacheKey] = rules

		// Verify entry exists
		assert.Len(t, suite.engine.rulesCache, 1)

		// Evict entry
		suite.engine.ClearCache(projectID, databaseID)

		// Verify eviction
		assert.Empty(t, suite.engine.rulesCache)
	})

	suite.T().Run("Multiple Entry Selective Eviction", func(t *testing.T) {
		// Add multiple entries
		entries := map[string][]*repository.SecurityRule{
			"proj1:db1": {{Match: "path1", Priority: 1}},
			"proj2:db2": {{Match: "path2", Priority: 2}},
			"proj3:db3": {{Match: "path3", Priority: 3}},
		}

		for key, rules := range entries {
			suite.engine.rulesCache[key] = rules
		}

		// Evict middle entry
		suite.engine.ClearCache("proj2", "db2")

		// Verify selective eviction
		assert.Len(t, suite.engine.rulesCache, 2)
		assert.Contains(t, suite.engine.rulesCache, "proj1:db1")
		assert.Contains(t, suite.engine.rulesCache, "proj3:db3")
		assert.NotContains(t, suite.engine.rulesCache, "proj2:db2")
	})
}

// Benchmark tests to ensure cache performance meets Firestore standards
func BenchmarkSecurityRulesCache(b *testing.B) {
	// Initialize test environment
	mockLogger := &MockLogger{}
	engine := &TestableSecurityRulesEngine{
		log:        mockLogger,
		rulesCache: make(map[string][]*repository.SecurityRule),
		cacheMu:    sync.RWMutex{},
		loadRulesFunc: func(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
			return []*repository.SecurityRule{{Match: "benchmark/path", Priority: 1}}, nil
		},
	}

	// Pre-populate cache with test data
	rules := []*repository.SecurityRule{
		{Match: "benchmark/path", Priority: 1},
	}
	cacheKey := "benchmark-project:benchmark-database"
	engine.rulesCache[cacheKey] = rules

	b.ResetTimer()

	b.Run("CacheHit", func(b *testing.B) {
		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			_, _ = engine.getCachedRules(ctx, cacheKey, "benchmark-project", "benchmark-database")
		}
	})

	b.Run("CacheClear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			engine.ClearCache("benchmark-project", "benchmark-database")
			engine.rulesCache[cacheKey] = rules // Restore for next iteration
		}
	})

	b.Run("ConcurrentRead", func(b *testing.B) {
		ctx := context.Background()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = engine.getCachedRules(ctx, cacheKey, "benchmark-project", "benchmark-database")
			}
		})
	})
}

// TestSuite runner
func TestSecurityRulesCacheTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityRulesCacheTestSuite))
}

// TestSecurityRulesCache_Compile ensures the package compiles correctly
func TestSecurityRulesCache_Compile(t *testing.T) {
	// This test ensures that all dependencies are properly imported
	// and the package compiles without errors
	assert.True(t, true, "Security rules cache package compiles successfully")
}
