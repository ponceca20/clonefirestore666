package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/adapter"
	"firestore-clone/internal/rules_translator/domain"
	"firestore-clone/internal/rules_translator/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranslatorFirestoreCompatibility tests exact translation compatibility with Firestore
func TestTranslatorFirestoreCompatibility(t *testing.T) {
	ctx := context.Background()
	translator := setupTestTranslator(t)

	t.Run("Basic Authentication Rules", func(t *testing.T) {
		ruleset := createBasicAuthRuleset()

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should translate basic auth rules")
		require.NotNil(t, result.Rules, "Should have translated rules")

		rules := result.Rules.([]*repository.SecurityRule)

		// Find user rule
		userRule := findRuleByMatch(rules, "/users/{userId}")
		require.NotNil(t, userRule, "Should have user rule")

		// Verify read operation
		readCondition, hasRead := userRule.Allow[repository.OperationRead]
		assert.True(t, hasRead, "Should have read operation")
		assert.Contains(t, readCondition, "request.auth.uid == userId", "Should have correct auth condition")

		// Verify Firestore write operations (create, update, delete)
		for _, op := range []repository.OperationType{repository.OperationCreate, repository.OperationUpdate, repository.OperationDelete} {
			cond, ok := userRule.Allow[op]
			assert.True(t, ok, "Should have %s operation mapped from Firestore write", op)
			assert.Equal(t, readCondition, cond, "Read and %s should have same condition in this case", op)
		}
	})

	t.Run("Operation Mapping Compatibility", func(t *testing.T) {
		testCases := []struct {
			firestoreOp string
			expectedOps []repository.OperationType
			description string
		}{
			{
				firestoreOp: "read",
				expectedOps: []repository.OperationType{repository.OperationRead, repository.OperationList},
				description: "Firestore 'read' should map to both read and list",
			},
			{
				firestoreOp: "write",
				expectedOps: []repository.OperationType{repository.OperationCreate, repository.OperationUpdate, repository.OperationDelete},
				description: "Firestore 'write' should map to create, update, delete",
			},
			{
				firestoreOp: "create",
				expectedOps: []repository.OperationType{repository.OperationCreate},
				description: "Firestore 'create' should map to create only",
			},
			{
				firestoreOp: "update",
				expectedOps: []repository.OperationType{repository.OperationUpdate},
				description: "Firestore 'update' should map to update only",
			},
			{
				firestoreOp: "delete",
				expectedOps: []repository.OperationType{repository.OperationDelete},
				description: "Firestore 'delete' should map to delete only",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				ruleset := createRulesetWithOperation(tc.firestoreOp)

				result, err := translator.Translate(ctx, ruleset)
				require.NoError(t, err, "Should translate operation: %s", tc.firestoreOp)

				rules := result.Rules.([]*repository.SecurityRule)
				rule := findRuleByMatch(rules, "/test/{id}")
				require.NotNil(t, rule, "Should have test rule")

				// Verify all expected operations are present
				for _, expectedOp := range tc.expectedOps {
					_, hasOp := rule.Allow[expectedOp]
					assert.True(t, hasOp, "Should have operation %v for Firestore operation %s", expectedOp, tc.firestoreOp)
				}

				// Verify no unexpected operations
				assert.Len(t, rule.Allow, len(tc.expectedOps), "Should have exactly the expected number of operations")
			})
		}
	})

	t.Run("Complex Condition Translation", func(t *testing.T) {
		complexConditions := []struct {
			name               string
			firestoreCondition string
			expectedContains   []string
			description        string
		}{
			{
				name:               "Basic Auth Check",
				firestoreCondition: "request.auth != null",
				expectedContains:   []string{"request.auth"},
				description:        "Basic authentication check should be preserved",
			},
			{
				name:               "User Ownership",
				firestoreCondition: "request.auth.uid == resource.data.owner",
				expectedContains:   []string{"request.auth.uid", "resource.data.owner"},
				description:        "User ownership check should preserve field access",
			},
			{
				name:               "Get Document Call",
				firestoreCondition: "get(/databases/$(database)/documents/users/$(request.auth.uid)).data.admin == true",
				expectedContains:   []string{"get(", "request.auth.uid", ".data.admin"},
				description:        "Get() calls should be preserved with proper syntax",
			},
			{
				name:               "Exists Check",
				firestoreCondition: "exists(/databases/$(database)/documents/users/$(request.auth.uid))",
				expectedContains:   []string{"exists(", "request.auth.uid"},
				description:        "Exists() calls should be preserved",
			},
			{
				name:               "Complex Boolean Logic",
				firestoreCondition: "request.auth != null && (request.auth.uid == resource.data.owner || request.auth.uid in resource.data.editors)",
				expectedContains:   []string{"request.auth", "resource.data.owner", "resource.data.editors", "&&", "||", "in"},
				description:        "Complex boolean logic should be preserved",
			},
			{
				name:               "Data Validation",
				firestoreCondition: "request.resource.data.name is string && request.resource.data.name.size() > 0",
				expectedContains:   []string{"request.resource.data", "is string", ".size()", "> 0"},
				description:        "Data validation syntax should be preserved",
			},
		}

		for _, tc := range complexConditions {
			t.Run(tc.name, func(t *testing.T) {
				ruleset := createRulesetWithCondition(tc.firestoreCondition)

				result, err := translator.Translate(ctx, ruleset)
				require.NoError(t, err, "Should translate condition: %s", tc.name)

				rules := result.Rules.([]*repository.SecurityRule)
				rule := findRuleByMatch(rules, "/test/{id}")
				require.NotNil(t, rule, "Should have test rule")

				readCondition, hasRead := rule.Allow[repository.OperationRead]
				require.True(t, hasRead, "Should have read operation")

				// Verify all expected elements are present in translated condition
				for _, expected := range tc.expectedContains {
					assert.Contains(t, readCondition, expected,
						"Translated condition should contain '%s'. Got: %s", expected, readCondition)
				}
			})
		}
	})

	t.Run("Nested Collection Translation", func(t *testing.T) {
		ruleset := createNestedCollectionRuleset()

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should translate nested collections")

		rules := result.Rules.([]*repository.SecurityRule)

		// Should have rules for all levels
		userRule := findRuleByMatch(rules, "/users/{userId}")
		postRule := findRuleByMatch(rules, "/users/{userId}/posts/{postId}")
		commentRule := findRuleByMatch(rules, "/users/{userId}/posts/{postId}/comments/{commentId}")

		assert.NotNil(t, userRule, "Should have user rule")
		assert.NotNil(t, postRule, "Should have post rule")
		assert.NotNil(t, commentRule, "Should have comment rule")

		// Verify priorities (more specific = higher priority)
		assert.Greater(t, commentRule.Priority, postRule.Priority, "Comment rule should have higher priority")
		assert.Greater(t, postRule.Priority, userRule.Priority, "Post rule should have higher priority than user rule")
	})

	t.Run("Variable Extraction and Path Building", func(t *testing.T) {
		ruleset := createVariableExtractionRuleset()

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should translate with variables")

		rules := result.Rules.([]*repository.SecurityRule)
		rule := findRuleByMatch(rules, "/tenants/{tenantId}/users/{userId}")
		require.NotNil(t, rule, "Should have multi-variable rule")

		// Verify condition references variables correctly
		readCondition, hasRead := rule.Allow[repository.OperationRead]
		require.True(t, hasRead, "Should have read operation")
		assert.Contains(t, readCondition, "tenantId", "Should reference tenantId variable")
		assert.Contains(t, readCondition, "userId", "Should reference userId variable")
	})

	t.Run("Priority Calculation Compatibility", func(t *testing.T) {
		ruleset := createPriorityTestRuleset()

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should translate priority test rules")

		rules := result.Rules.([]*repository.SecurityRule)

		// Find rules
		wildcardRule := findRuleByMatch(rules, "/{document=**}")
		specificRule := findRuleByMatch(rules, "/users/{userId}")
		verySpecificRule := findRuleByMatch(rules, "/users/{userId}/posts/{postId}")

		require.NotNil(t, wildcardRule, "Should have wildcard rule")
		require.NotNil(t, specificRule, "Should have specific rule")
		require.NotNil(t, verySpecificRule, "Should have very specific rule")

		// Verify priority order (Firestore-compatible: more specific = higher priority)
		assert.Greater(t, verySpecificRule.Priority, specificRule.Priority,
			"Very specific rule should have higher priority")
		assert.Greater(t, specificRule.Priority, wildcardRule.Priority,
			"Specific rule should have higher priority than wildcard")
	})
}

// TestTranslatorPerformance tests performance requirements
func TestTranslatorPerformance(t *testing.T) {
	ctx := context.Background()
	translator := setupTestTranslator(t)

	t.Run("Large Ruleset Translation Performance", func(t *testing.T) {
		// Create large but realistic ruleset
		largeRuleset := createLargeRealisticRuleset(200)

		start := time.Now()
		result, err := translator.Translate(ctx, largeRuleset)
		duration := time.Since(start)

		require.NoError(t, err, "Should translate large ruleset")
		require.NotNil(t, result.Rules, "Should have translated rules")

		rules := result.Rules.([]*repository.SecurityRule)

		// Performance requirements
		assert.Less(t, duration, time.Millisecond*100, "Should translate quickly")
		assert.Greater(t, len(rules), 150, "Should translate most rules")
		assert.Equal(t, len(result.Errors), 0, "Should have no translation errors")

		t.Logf("Translated %d rules in %v (%.2f rules/ms)",
			len(rules), duration, float64(len(rules))/float64(duration.Nanoseconds()/1e6))
	})

	t.Run("Memory Efficiency", func(t *testing.T) {
		ruleset := createMediumRuleset(50)

		// Translate multiple times to test memory usage
		for i := 0; i < 20; i++ {
			result, err := translator.Translate(ctx, ruleset)
			require.NoError(t, err, "Translation %d should succeed", i)
			require.NotNil(t, result.Rules, "Should have rules")
		}

		// Verify translator metrics show good performance
		if metrics := translator.GetMetrics(); metrics != nil {
			assert.Greater(t, metrics.TotalTranslations, int64(15), "Should have processed multiple translations")
			assert.Less(t, metrics.ErrorRate, 0.1, "Should have low error rate")
		}
	})

	t.Run("Concurrent Translation Safety", func(t *testing.T) {
		ruleset := createMediumRuleset(30)

		// Test concurrent translations
		const goroutines = 10
		results := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				_, err := translator.Translate(ctx, ruleset)
				results <- err
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < goroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent translation %d should succeed", i)
		}
	})
}

// TestTranslatorEdgeCases tests edge cases and error conditions
func TestTranslatorEdgeCases(t *testing.T) {
	ctx := context.Background()
	translator := setupTestTranslator(t)

	t.Run("Empty Ruleset", func(t *testing.T) {
		ruleset := &domain.FirestoreRuleset{
			Service: "cloud.firestore",
			Matches: []*domain.MatchBlock{},
		}

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should handle empty ruleset")

		rules := result.Rules.([]*repository.SecurityRule)
		assert.Len(t, rules, 0, "Should have no rules")
	})

	t.Run("Rules with Only Deny Statements", func(t *testing.T) {
		ruleset := createDenyOnlyRuleset()

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should handle deny-only rules")

		rules := result.Rules.([]*repository.SecurityRule)
		rule := findRuleByMatch(rules, "/admin/{document}")
		require.NotNil(t, rule, "Should have admin rule")

		// Should have deny operations but no allow operations
		assert.Len(t, rule.Allow, 0, "Should have no allow operations")
		assert.Greater(t, len(rule.Deny), 0, "Should have deny operations")
	})

	t.Run("Very Long Condition Strings", func(t *testing.T) {
		longCondition := createVeryLongCondition(1000) // 1000 character condition
		ruleset := createRulesetWithCondition(longCondition)

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should handle very long conditions")

		rules := result.Rules.([]*repository.SecurityRule)
		rule := findRuleByMatch(rules, "/test/{id}")
		require.NotNil(t, rule, "Should have test rule")

		readCondition, hasRead := rule.Allow[repository.OperationRead]
		require.True(t, hasRead, "Should have read operation")
		assert.Greater(t, len(readCondition), 500, "Should preserve long condition")
	})

	t.Run("Deep Nesting", func(t *testing.T) {
		ruleset := createDeeplyNestedRuleset(10) // 10 levels deep

		result, err := translator.Translate(ctx, ruleset)
		require.NoError(t, err, "Should handle deep nesting")

		rules := result.Rules.([]*repository.SecurityRule)
		assert.Greater(t, len(rules), 5, "Should translate nested rules")

		// Find the deepest rule
		var deepestRule *repository.SecurityRule
		maxPriority := 0
		for _, rule := range rules {
			if rule.Priority > maxPriority {
				maxPriority = rule.Priority
				deepestRule = rule
			}
		}

		require.NotNil(t, deepestRule, "Should have a deepest rule")
		assert.Greater(t, deepestRule.Priority, 500, "Deepest rule should have high priority")
	})
}

// Helper functions for creating test rulesets

func setupTestTranslator(t *testing.T) domain.RulesTranslator {
	cache := adapter.NewMemoryCache(adapter.DefaultCacheConfig())

	// Configuración de optimizador sin transformaciones agresivas para tests de compatibilidad
	optimizerConfig := adapter.DefaultOptimizerConfig()
	optimizerConfig.EnableAggressiveOptim = false
	optimizer := adapter.NewRulesOptimizer(optimizerConfig)

	config := usecase.DefaultTranslatorConfig()
	config.EnableMetrics = true
	config.OptimizeConditions = false // Deshabilitamos optimización de condiciones para tests

	return usecase.NewFastTranslator(cache, optimizer, config)
}

func createBasicAuthRuleset() *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:      "/users/{userId}",
				FullPath:  "/users/{userId}",
				Variables: map[string]string{"userId": "{userId}"},
				Allow: []*domain.AllowStatement{
					{
						Operations: []string{"read", "write"},
						Condition:  "request.auth != null && request.auth.uid == userId",
					},
				},
			},
		},
	}
}

func createRulesetWithOperation(operation string) *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:      "/test/{id}",
				FullPath:  "/test/{id}",
				Variables: map[string]string{"id": "{id}"},
				Allow: []*domain.AllowStatement{
					{
						Operations: []string{operation},
						Condition:  "request.auth != null",
					},
				},
			},
		},
	}
}

func createRulesetWithCondition(condition string) *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:      "/test/{id}",
				FullPath:  "/test/{id}",
				Variables: map[string]string{"id": "{id}"},
				Allow: []*domain.AllowStatement{
					{
						Operations: []string{"read"},
						Condition:  condition,
					},
				},
			},
		},
	}
}

func createNestedCollectionRuleset() *domain.FirestoreRuleset {
	commentMatch := &domain.MatchBlock{
		Path:      "/comments/{commentId}",
		FullPath:  "/users/{userId}/posts/{postId}/comments/{commentId}",
		Variables: map[string]string{"commentId": "{commentId}"},
		Depth:     3,
		Allow: []*domain.AllowStatement{
			{Operations: []string{"read"}, Condition: "true"},
		},
	}

	postMatch := &domain.MatchBlock{
		Path:      "/posts/{postId}",
		FullPath:  "/users/{userId}/posts/{postId}",
		Variables: map[string]string{"postId": "{postId}"},
		Depth:     2,
		Nested:    []*domain.MatchBlock{commentMatch},
		Allow: []*domain.AllowStatement{
			{Operations: []string{"read"}, Condition: "request.auth.uid == userId"},
		},
	}

	userMatch := &domain.MatchBlock{
		Path:      "/users/{userId}",
		FullPath:  "/users/{userId}",
		Variables: map[string]string{"userId": "{userId}"},
		Depth:     1,
		Nested:    []*domain.MatchBlock{postMatch},
		Allow: []*domain.AllowStatement{
			{Operations: []string{"read"}, Condition: "true"},
		},
	}

	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{userMatch},
	}
}

func createVariableExtractionRuleset() *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:     "/tenants/{tenantId}/users/{userId}",
				FullPath: "/tenants/{tenantId}/users/{userId}",
				Variables: map[string]string{
					"tenantId": "{tenantId}",
					"userId":   "{userId}",
				},
				Allow: []*domain.AllowStatement{
					{
						Operations: []string{"read"},
						Condition:  "request.auth.uid == userId && exists(/databases/$(database)/documents/tenants/$(tenantId)/members/$(request.auth.uid))",
					},
				},
			},
		},
	}
}

func createPriorityTestRuleset() *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:      "/{document=**}",
				FullPath:  "/{document=**}",
				Variables: map[string]string{"document": "{document=**}"},
				Depth:     0,
				Allow: []*domain.AllowStatement{
					{Operations: []string{"read", "write"}, Condition: "false"},
				},
			},
			{
				Path:      "/users/{userId}",
				FullPath:  "/users/{userId}",
				Variables: map[string]string{"userId": "{userId}"},
				Depth:     1,
				Allow: []*domain.AllowStatement{
					{Operations: []string{"read"}, Condition: "request.auth.uid == userId"},
				},
			},
			{
				Path:      "/users/{userId}/posts/{postId}",
				FullPath:  "/users/{userId}/posts/{postId}",
				Variables: map[string]string{"userId": "{userId}", "postId": "{postId}"},
				Depth:     2,
				Allow: []*domain.AllowStatement{
					{Operations: []string{"read"}, Condition: "request.auth.uid == userId"},
				},
			},
		},
	}
}

func createDenyOnlyRuleset() *domain.FirestoreRuleset {
	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:      "/admin/{document}",
				FullPath:  "/admin/{document}",
				Variables: map[string]string{"document": "{document}"},
				Deny: []*domain.DenyStatement{
					{
						Operations: []string{"read", "write"},
						Condition:  "request.auth == null",
					},
				},
			},
		},
	}
}

func createLargeRealisticRuleset(size int) *domain.FirestoreRuleset {
	matches := make([]*domain.MatchBlock, 0, size)

	for i := 0; i < size; i++ {
		collection := []string{"users", "posts", "comments", "orders", "products"}[i%5]
		path := fmt.Sprintf("/%s%d/{id}", collection, i)

		match := &domain.MatchBlock{
			Path:      path,
			FullPath:  path,
			Variables: map[string]string{"id": "{id}"},
			Allow: []*domain.AllowStatement{
				{
					Operations: []string{"read"},
					Condition:  "request.auth != null",
				},
				{
					Operations: []string{"write"},
					Condition:  "request.auth.uid == resource.data.owner",
				},
			},
		}
		matches = append(matches, match)
	}

	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: matches,
	}
}

func createMediumRuleset(size int) *domain.FirestoreRuleset {
	matches := make([]*domain.MatchBlock, 0, size)

	for i := 0; i < size; i++ {
		path := fmt.Sprintf("/collection%d/{id}", i)
		match := &domain.MatchBlock{
			Path:      path,
			FullPath:  path,
			Variables: map[string]string{"id": "{id}"},
			Allow: []*domain.AllowStatement{
				{Operations: []string{"read"}, Condition: "true"},
			},
		}
		matches = append(matches, match)
	}

	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: matches,
	}
}

func createVeryLongCondition(length int) string {
	base := "request.auth != null && request.auth.uid == resource.data.owner"
	for len(base) < length {
		base += " && request.resource.data.validField" + fmt.Sprintf("%d", len(base)%100) + " is string"
	}
	return base[:length]
}

func createDeeplyNestedRuleset(depth int) *domain.FirestoreRuleset {
	var createNested func(level int, parentPath string) *domain.MatchBlock

	createNested = func(level int, parentPath string) *domain.MatchBlock {
		path := fmt.Sprintf("/level%d/{id%d}", level, level)
		fullPath := parentPath + path

		match := &domain.MatchBlock{
			Path:      path,
			FullPath:  fullPath,
			Variables: map[string]string{fmt.Sprintf("id%d", level): fmt.Sprintf("{id%d}", level)},
			Depth:     level,
			Allow: []*domain.AllowStatement{
				{Operations: []string{"read"}, Condition: "true"},
			},
		}

		if level < depth {
			match.Nested = []*domain.MatchBlock{createNested(level+1, fullPath)}
		}

		return match
	}

	rootMatch := createNested(1, "")

	return &domain.FirestoreRuleset{
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{rootMatch},
	}
}

func findRuleByMatch(rules []*repository.SecurityRule, match string) *repository.SecurityRule {
	for _, rule := range rules {
		if rule.Match == match {
			return rule
		}
	}
	return nil
}
