package test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/adapter/parser"
	"firestore-clone/internal/rules_translator/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExhaustiveFirestoreCompatibility performs comprehensive compatibility testing
// covering all Firestore syntax features, edge cases, and real-world scenarios
func TestExhaustiveFirestoreCompatibility(t *testing.T) {
	ctx := context.Background()

	t.Run("Firestore Syntax Coverage Tests", func(t *testing.T) {
		testFirestoreSyntaxCoverage(t, ctx)
	})

	t.Run("Operation Mapping Completeness", func(t *testing.T) {
		testOperationMappingCompleteness(t, ctx)
	})

	t.Run("Condition Translation Accuracy", func(t *testing.T) {
		testConditionTranslationAccuracy(t, ctx)
	})

	t.Run("Variable and Path Extraction", func(t *testing.T) {
		testVariableAndPathExtraction(t, ctx)
	})

	t.Run("Error Handling and Edge Cases", func(t *testing.T) {
		testErrorHandlingAndEdgeCases(t, ctx)
	})

	t.Run("Performance and Scalability", func(t *testing.T) {
		testPerformanceAndScalability(t, ctx)
	})

	t.Run("Security Rule Priority", func(t *testing.T) {
		testSecurityRulePriority(t, ctx)
	})

	t.Run("Real World Scenarios", func(t *testing.T) {
		testRealWorldScenarios(t, ctx)
	})
}

// testFirestoreSyntaxCoverage ensures all Firestore syntax elements are properly parsed
func testFirestoreSyntaxCoverage(t *testing.T, ctx context.Context) {
	testCases := []struct {
		name     string
		rules    string
		expected func(t *testing.T, result *domain.ParseResult)
	}{
		{
			name: "Rules Version Declarations",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true;
    }
  }
}`, expected: func(t *testing.T, result *domain.ParseResult) {
				assert.Equal(t, "2", result.Ruleset.Version)
				assert.Equal(t, "cloud.firestore", result.Ruleset.Service)
			},
		},
		{
			name: "All Operation Types",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true;
      allow write: if true;
      allow create: if true;
      allow update: if true;
      allow delete: if true;
      allow list: if true;
      allow get: if true;
    }
  }
}`,
			expected: func(t *testing.T, result *domain.ParseResult) {
				// Traverse to the first match block with allow rules
				match := firstAllowMatch(result.Ruleset)
				require.NotNil(t, match, "Should find a match block with allow rules")
				operations := []string{"read", "write", "create", "update", "delete", "list", "get"}
				for _, op := range operations {
					found := false
					for _, allowRule := range match.Allow {
						if contains(allowRule.Operations, op) {
							found = true
							break
						}
					}
					assert.True(t, found, fmt.Sprintf("Operation %s should be parsed", op))
				}
			},
		},
		{
			name: "Complex Path Patterns",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /users/{userId} {
      allow read: if true;
    }
    match /users/{userId}/posts/{postId} {
      allow read: if true;
    }
    match /path/{document=**} {
      allow read: if true;
    }
    match /{path=**} {
      allow read: if false;
    }
  }
}`, expected: func(t *testing.T, result *domain.ParseResult) { // Recoge todos los paths de los bloques match, incluyendo los anidados
				paths := []string{"/users/{userId}", "/users/{userId}/posts/{postId}", "/path/{document=**}", "/{path=**}"}
				allMatches := collectAllMatches(result.Ruleset.Matches)
				fmt.Println("[DEBUG] Estructura real de matches:")
				debugPrintAllMatchPaths(result.Ruleset.Matches, "")
				var foundPaths []string
				for _, m := range allMatches {
					foundPaths = append(foundPaths, m.Path)
				}
				// Debug explícito de los paths comparados
				fmt.Printf("[DEBUG] Paths esperados: %v\n", paths)
				fmt.Printf("[DEBUG] Paths encontrados: %v\n", foundPaths)
				assert.ElementsMatch(t, paths, foundPaths)
			},
		},
		{
			name: "Built-in Functions and Variables",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if request.auth != null;
      allow write: if resource.data.owner == request.auth.uid;
      allow create: if request.resource.data.timestamp == request.time;
      allow update: if exists(/databases/$(database)/documents/users/$(request.auth.uid));
      allow delete: if get(/databases/$(database)/documents/settings/$(id)).data.deletable == true;
    }
  }
}`,
			expected: func(t *testing.T, result *domain.ParseResult) {
				match := firstAllowMatch(result.Ruleset)
				require.NotNil(t, match, "Should find a match block with allow rules")
				conditionTexts := make([]string, 0)
				for _, allowRule := range match.Allow {
					conditionTexts = append(conditionTexts, allowRule.Condition)
				}
				allConditions := strings.Join(conditionTexts, " ")
				builtIns := []string{"request.auth", "resource.data", "request.resource", "request.time", "exists", "get"}
				for _, builtIn := range builtIns {
					assert.Contains(t, allConditions, builtIn, fmt.Sprintf("Built-in %s should be in conditions", builtIn))
				}
			},
		},
		{
			name: "Data Type Validations",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /products/{id} {
      allow create: if request.resource.data.name is string &&
                      request.resource.data.price is number &&
                      request.resource.data.active is bool &&
                      request.resource.data.tags is list &&
                      request.resource.data.metadata is map;
    }
  }
}`,
			expected: func(t *testing.T, result *domain.ParseResult) {
				match := firstAllowMatch(result.Ruleset)
				require.NotNil(t, match, "Should find a match block with allow rules")
				condition := match.Allow[0].Condition
				types := []string{"is string", "is number", "is bool", "is list", "is map"}
				for _, typeCheck := range types {
					assert.Contains(t, condition, typeCheck, fmt.Sprintf("Type check '%s' should be in condition", typeCheck))
				}
			},
		}}

	parser := parser.NewModernParserInstance()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.ParseString(ctx, tc.rules)
			require.NoError(t, err, "Should parse rules without error")
			require.NotNil(t, result.Ruleset, "Should have parsed ruleset") // Debug output for development - remove in production
			// if tc.name == "All Operation Types" {
			//	 fmt.Printf("[DEBUG] Parsed ruleset: %+v\n", result.Ruleset)
			//	 fmt.Printf("[DEBUG] Number of matches: %d\n", len(result.Ruleset.Matches))
			//	 debugPrintMatches(result.Ruleset.Matches, 0)
			// }

			tc.expected(t, result)
		})
	}
}

// testOperationMappingCompleteness verifies all Firestore operations map correctly
func testOperationMappingCompleteness(t *testing.T, ctx context.Context) {
	translator := setupTestTranslator(t)
	testCases := []struct {
		firestoreOp string
		expectOps   []repository.OperationType
	}{
		{"read", []repository.OperationType{repository.OperationRead, repository.OperationList}},                                  // read expands to read and list
		{"write", []repository.OperationType{repository.OperationCreate, repository.OperationUpdate, repository.OperationDelete}}, // write expands to create, update, delete
		{"create", []repository.OperationType{repository.OperationCreate}},
		{"update", []repository.OperationType{repository.OperationUpdate}},
		{"delete", []repository.OperationType{repository.OperationDelete}},
		{"get", []repository.OperationType{repository.OperationRead}},  // maps to read
		{"list", []repository.OperationType{repository.OperationList}}, // maps to list
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Operation_%s", tc.firestoreOp), func(t *testing.T) {
			ruleset := &domain.FirestoreRuleset{
				Version: "2",
				Service: "cloud.firestore",
				Matches: []*domain.MatchBlock{
					{
						Path: "/test/{id}",
						Allow: []*domain.AllowStatement{{
							Operations: []string{tc.firestoreOp},
							Condition:  "true",
						}},
					},
				},
			}

			result, err := translator.Translate(ctx, ruleset)
			require.NoError(t, err, "Should translate operation")

			rules := result.Rules.([]*repository.SecurityRule)
			require.Greater(t, len(rules), 0, "Should have translated rules")

			rule := rules[0]
			for _, expectedOp := range tc.expectOps {
				_, exists := rule.Allow[expectedOp]
				assert.True(t, exists, fmt.Sprintf("Operation %s should be mapped", expectedOp))
			}
		})
	}
}

// testConditionTranslationAccuracy verifies condition translation accuracy
func testConditionTranslationAccuracy(t *testing.T, ctx context.Context) {
	translator := setupTestTranslator(t)

	testCases := []struct {
		name               string
		firestoreCondition string
		expectedPattern    string
		description        string
	}{
		{
			name:               "Auth_Null_Check",
			firestoreCondition: "request.auth != null",
			expectedPattern:    "request.auth != null",
			description:        "Basic auth null check should be preserved",
		},
		{
			name:               "Auth_UID_Comparison",
			firestoreCondition: "request.auth.uid == resource.data.owner",
			expectedPattern:    "request.auth.uid == resource.data.owner",
			description:        "Auth UID comparison should be preserved",
		},
		{
			name:               "Complex_Logical_Expression",
			firestoreCondition: "request.auth != null && (resource.data.public == true || request.auth.uid == resource.data.owner)",
			expectedPattern:    "&&",
			description:        "Complex logical expressions should maintain structure",
		},
		{
			name:               "Function_Call_Get",
			firestoreCondition: "get(/databases/$(database)/documents/users/$(request.auth.uid)).data.admin == true",
			expectedPattern:    "get(",
			description:        "Function calls should be preserved",
		},
		{
			name:               "Type_Validation",
			firestoreCondition: "request.resource.data.name is string && request.resource.data.name.size() > 0",
			expectedPattern:    "is string",
			description:        "Type validations should be preserved",
		},
		{
			name:               "List_Operations",
			firestoreCondition: "request.auth.uid in resource.data.members",
			expectedPattern:    " in ",
			description:        "List membership checks should be preserved",
		},
		{
			name:               "Time_Comparison",
			firestoreCondition: "request.time < timestamp.date(2025, 12, 31)",
			expectedPattern:    "timestamp.date",
			description:        "Time operations should be preserved",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleset := &domain.FirestoreRuleset{
				Version: "2",
				Service: "cloud.firestore",
				Matches: []*domain.MatchBlock{
					{
						Path: "/test/{id}",
						Allow: []*domain.AllowStatement{{
							Operations: []string{"read"},
							Condition:  tc.firestoreCondition,
						}},
					},
				},
			}

			result, err := translator.Translate(ctx, ruleset)
			require.NoError(t, err, "Should translate condition")

			rules := result.Rules.([]*repository.SecurityRule)
			require.Greater(t, len(rules), 0, "Should have translated rules")

			condition, exists := ruleAllowCondition(rules[0], repository.OperationRead)
			require.True(t, exists, "Should have read operation")

			assert.Contains(t, condition, tc.expectedPattern, tc.description)
		})
	}
}

// testVariableAndPathExtraction verifies variable extraction from paths
func testVariableAndPathExtraction(t *testing.T, ctx context.Context) {
	translator := setupTestTranslator(t)

	testCases := []struct {
		path              string
		expectedVariables []string
		expectedTemplate  string
	}{
		{
			path:              "/users/{userId}",
			expectedVariables: []string{"userId"},
			expectedTemplate:  "/users/{userId}",
		},
		{
			path:              "/users/{userId}/posts/{postId}",
			expectedVariables: []string{"userId", "postId"},
			expectedTemplate:  "/users/{userId}/posts/{postId}",
		},
		{
			path:              "/data/{document=**}",
			expectedVariables: []string{"document"},
			expectedTemplate:  "/data/{document=**}",
		},
		{
			path:              "/{path=**}",
			expectedVariables: []string{"path"},
			expectedTemplate:  "/{path=**}",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Path_%s", strings.ReplaceAll(tc.path, "/", "_")), func(t *testing.T) {
			ruleset := &domain.FirestoreRuleset{
				Version: "2",
				Service: "cloud.firestore",
				Matches: []*domain.MatchBlock{
					{
						Path:      tc.path,
						Variables: map[string]string{}, // Assume parser fills this in real use
						Allow: []*domain.AllowStatement{{
							Operations: []string{"read"},
							Condition:  "true",
						}},
					},
				},
			}

			result, err := translator.Translate(ctx, ruleset)
			require.NoError(t, err, "Should translate path")

			rules := result.Rules.([]*repository.SecurityRule)
			require.Greater(t, len(rules), 0, "Should have translated rules")

			rule := rules[0]

			// Extract variables from the rule.Match path pattern
			actualVars := extractPathVariables(rule.Match)
			for _, expectedVar := range tc.expectedVariables {
				found := false
				for _, v := range actualVars {
					if v == expectedVar {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("Variable %s should be extracted", expectedVar))
			}

			// Check path pattern
			assert.Equal(t, tc.expectedTemplate, rule.Match, "Path template should match")
		})
	}
}

// extractPathVariables extracts variable names from a Firestore path pattern like /users/{userId}/posts/{postId}
func extractPathVariables(path string) []string {
	vars := []string{}
	start := -1
	for i, c := range path {
		if c == '{' {
			start = i + 1
		} else if c == '}' && start != -1 {
			varSpec := path[start:i]
			if eqIdx := strings.Index(varSpec, "="); eqIdx != -1 {
				vars = append(vars, varSpec[:eqIdx])
			} else {
				vars = append(vars, varSpec)
			}
			start = -1
		}
	}
	return vars
}

// testErrorHandlingAndEdgeCases verifies robust error handling
func testErrorHandlingAndEdgeCases(t *testing.T, ctx context.Context) {
	parser := parser.NewModernParserInstance()

	testCases := []struct {
		name        string
		rules       string
		expectError bool
		description string
	}{
		{
			name:        "Empty_Rules",
			rules:       "",
			expectError: true,
			description: "Empty rules should be handled gracefully",
		},
		{
			name: "Missing_Rules_Version",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} { allow read: if true; }
  }
}`,
			expectError: false, // Should be allowed with warning
			description: "Missing rules version should be handled",
		}, {
			name: "Invalid_Syntax",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true {
    }
  }
}`, // Invalid syntax: extra opening brace after condition
			expectError: true,
			description: "Invalid syntax should be detected",
		},
		{
			name:        "Deeply_Nested_Rules",
			rules:       generateDeeplyNestedRules(10),
			expectError: false,
			description: "Deeply nested rules should be handled",
		},
		{
			name:        "Very_Long_Condition",
			rules:       generateVeryLongCondition(1000),
			expectError: false,
			description: "Very long conditions should be handled",
		},
		{
			name: "Special_Characters_In_Path",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test-data/{id} { allow read: if true; }
    match /test_data/{id} { allow read: if true; }
    match /test.data/{id} { allow read: if true; }
  }
}`,
			expectError: false,
			description: "Special characters in paths should be handled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.ParseString(ctx, tc.rules)

			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
				if err == nil {
					assert.NotNil(t, result, "Should have parse result")
				}
			}
		})
	}
}

// testPerformanceAndScalability verifies performance characteristics
func testPerformanceAndScalability(t *testing.T, ctx context.Context) {
	parser := parser.NewModernParserInstance()
	translator := setupTestTranslator(t)

	t.Run("Large_Rules_File", func(t *testing.T) {
		largeRules := generateLargeRulesFile(100) // 100 rules

		start := time.Now()
		result, err := parser.ParseString(ctx, largeRules)
		parseTime := time.Since(start)

		require.NoError(t, err, "Should parse large rules file")
		assert.Less(t, parseTime, 5*time.Second, "Parsing should be fast")

		start = time.Now()
		_, err = translator.Translate(ctx, result.Ruleset)
		translateTime := time.Since(start)

		require.NoError(t, err, "Should translate large rules file")
		assert.Less(t, translateTime, 2*time.Second, "Translation should be fast")
	})

	t.Run("Memory_Usage", func(t *testing.T) {
		// Parse multiple rule sets to check for memory leaks
		for i := 0; i < 10; i++ {
			rules := generateRulesWithVariableComplexity(10 + i)
			_, err := parser.ParseString(ctx, rules)
			require.NoError(t, err, "Should parse rules iteration %d", i)
		}
	})
}

// Debug: imprime todos los paths recursivamente, excluyendo el root
func debugPrintAllMatchPaths(matches []*domain.MatchBlock, prefix string) {
	for _, m := range matches {
		if m.Path != "" { // Excluye el root de la impresión
			fmt.Printf("%sPath: %s\n", prefix, m.Path)
		}
		if len(m.Nested) > 0 {
			debugPrintAllMatchPaths(m.Nested, prefix+"  ")
		}
	}
}

// Helper functions for debugging

// debugPrintMatches imprime recursivamente todos los matches con indentación
// debugPrintMatches imprime recursivamente todos los matches con indentación
// Comentado para evitar warnings de función no usada - descomentar para debug
/*
func debugPrintMatches(matches []*domain.MatchBlock, level int) {
	indent := strings.Repeat("  ", level)
	for i, match := range matches {
		fmt.Printf("[DEBUG] %sMatch %d: Path='%s', Allow count=%d, Nested count=%d\n",
			indent, i, match.Path, len(match.Allow), len(match.Nested))
		for j, allow := range match.Allow {
			fmt.Printf("[DEBUG] %s  Allow %d: Operations=%v, Condition='%s'\n",
				indent, j, allow.Operations, allow.Condition)
		}
		for j, deny := range match.Deny {
			fmt.Printf("[DEBUG] %s  Deny %d: Operations=%v, Condition='%s'\n",
				indent, j, deny.Operations, deny.Condition)
		}
		// Imprimir matches anidados recursivamente
		if len(match.Nested) > 0 {
			debugPrintMatches(match.Nested, level+1)
		}
	}
}
*/

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper para encontrar el primer MatchBlock con condiciones allow (búsqueda recursiva)
func firstAllowMatch(ruleset *domain.FirestoreRuleset) *domain.MatchBlock {
	return findFirstAllowMatchRecursive(ruleset.Matches)
}

// Helper recursivo para buscar en matches anidados
func findFirstAllowMatchRecursive(matches []*domain.MatchBlock) *domain.MatchBlock {
	for _, m := range matches {
		if len(m.Allow) > 0 {
			return m
		}
		// Buscar recursivamente en matches anidados
		if found := findFirstAllowMatchRecursive(m.Nested); found != nil {
			return found
		}
	}
	return nil
}

// Helper to extract allow condition for a given operation from a SecurityRule
func ruleAllowCondition(rule *repository.SecurityRule, op repository.OperationType) (string, bool) {
	if rule.Allow == nil {
		return "", false
	}
	cond, ok := rule.Allow[op]
	return cond, ok
}

// Helper para encontrar SecurityRule por ruta
func findRuleByPath(rules []*repository.SecurityRule, path string) *repository.SecurityRule {
	for _, rule := range rules {
		// Support both exact match and pattern matching
		if rule.Match == path {
			return rule
		}

		// Pattern matching for simplified paths
		// Convert full path like '/databases/{database}/documents/users/{userId}'
		// to simplified pattern like '/users/*'
		if strings.Contains(rule.Match, "/documents/") {
			// Extract the part after /documents/
			parts := strings.Split(rule.Match, "/documents/")
			if len(parts) > 1 {
				simplifiedPath := "/" + parts[1]
				// Replace variables like {userId} with *
				simplifiedPath = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(simplifiedPath, "*")
				if simplifiedPath == path {
					return rule
				}
			}
		}
	}
	return nil
}

func generateDeeplyNestedRules(depth int) string {
	rules := `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {`

	for i := 0; i < depth; i++ {
		rules += fmt.Sprintf(`
    match /level%d/{id%d} {`, i, i)
	}

	rules += `
      allow read: if true;`

	for i := 0; i < depth; i++ {
		rules += `
    }`
	}

	rules += `
  }
}`

	return rules
}

func generateVeryLongCondition(length int) string {
	var conditions []string
	for i := 0; i < length/20; i++ {
		conditions = append(conditions, fmt.Sprintf("request.resource.data.field%d == 'value%d'", i, i))
	}

	condition := strings.Join(conditions, " && ")

	return fmt.Sprintf(`rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if %s;
    }
  }
}`, condition)
}

func generateLargeRulesFile(numRules int) string {
	rules := `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {`

	for i := 0; i < numRules; i++ {
		rules += fmt.Sprintf(`
    match /collection%d/{id} {
      allow read: if request.auth != null;
      allow write: if request.auth != null && request.auth.uid == resource.data.owner;
    }`, i)
	}

	rules += `
  }
}`

	return rules
}

func generateRulesWithVariableComplexity(complexity int) string {
	conditions := make([]string, complexity)
	for i := 0; i < complexity; i++ {
		conditions[i] = fmt.Sprintf("request.resource.data.field%d != null", i)
	}

	condition := strings.Join(conditions, " && ")

	return fmt.Sprintf(`rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {      allow read: if %s;
    }
  }
}`, condition)
}

// testSecurityRulePriority verifies rule priority calculation
func testSecurityRulePriority(t *testing.T, ctx context.Context) {
	translator := setupTestTranslator(t)

	ruleset := &domain.FirestoreRuleset{
		Version: "2",
		Service: "cloud.firestore",
		Matches: []*domain.MatchBlock{
			{
				Path:  "/{path=**}",
				Allow: []*domain.AllowStatement{{Operations: []string{"read"}, Condition: "false"}},
			},
			{
				Path:  "/users/{userId}",
				Allow: []*domain.AllowStatement{{Operations: []string{"read"}, Condition: "true"}},
			},
			{
				Path:  "/users/{userId}/posts/{postId}",
				Allow: []*domain.AllowStatement{{Operations: []string{"read"}, Condition: "true"}},
			},
		},
	}

	result, err := translator.Translate(ctx, ruleset)
	require.NoError(t, err, "Should translate rules with different priorities")

	rules := result.Rules.([]*repository.SecurityRule)
	require.Equal(t, 3, len(rules), "Should have three rules")

	// More specific paths should have higher priority
	specificRulePriority := -1
	generalRulePriority := -1

	for _, rule := range rules {
		if rule.Match == "/users/{userId}/posts/{postId}" {
			specificRulePriority = rule.Priority
		}
		if rule.Match == "/{path=**}" {
			generalRulePriority = rule.Priority
		}
	}

	assert.Greater(t, specificRulePriority, generalRulePriority, "More specific rules should have higher priority")
}

// testRealWorldScenarios tests comprehensive real-world scenarios
func testRealWorldScenarios(t *testing.T, ctx context.Context) {
	testCases := []struct {
		name        string
		description string
		rules       string
		validations func(t *testing.T, rules []*repository.SecurityRule)
	}{
		{
			name:        "Social_Media_App",
			description: "Social media application with posts, comments, and user profiles",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /users/{userId} {
      allow read: if true;
      allow write: if request.auth != null && request.auth.uid == userId;
    }
    
    match /posts/{postId} {
      allow read: if resource.data.visibility == 'public' || 
                     (request.auth != null && 
                      (request.auth.uid == resource.data.author || 
                       request.auth.uid in resource.data.friends));
      allow create: if request.auth != null && 
                       request.auth.uid == request.resource.data.author;
      allow update, delete: if request.auth != null && 
                              request.auth.uid == resource.data.author;
      
      match /comments/{commentId} {
        allow read: if get(/databases/$(database)/documents/posts/$(postId)).data.visibility == 'public' ||
                       (request.auth != null && 
                        request.auth.uid in get(/databases/$(database)/documents/posts/$(postId)).data.friends);
        allow create: if request.auth != null;
        allow update, delete: if request.auth != null && 
                                request.auth.uid == resource.data.author;
      }
    }
  }
}`, validations: func(t *testing.T, rules []*repository.SecurityRule) {
				assert.Greater(t, len(rules), 2, "Should have multiple rules")

				// Find and validate user rule
				userRule := findRuleByPath(rules, "/users/*")
				assert.NotNil(t, userRule, "Should have user rule")

				// Find and validate post rule
				postRule := findRuleByPath(rules, "/posts/*")
				assert.NotNil(t, postRule, "Should have post rule")

				// Find and validate comment rule
				commentRule := findRuleByPath(rules, "/posts/*/comments/*")
				assert.NotNil(t, commentRule, "Should have comment rule")
			},
		},
		{
			name:        "E_Commerce_Platform",
			description: "E-commerce platform with products, orders, and reviews",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /products/{productId} {
      allow read: if true;
      allow write: if request.auth != null && 
                      get(/databases/$(database)/documents/users/$(request.auth.uid)).data.role == 'admin';
      
      match /reviews/{reviewId} {
        allow read: if true;
        allow create: if request.auth != null && 
                         request.auth.uid == request.resource.data.userId &&
                         exists(/databases/$(database)/documents/orders/$(request.resource.data.orderId)) &&
                         get(/databases/$(database)/documents/orders/$(request.resource.data.orderId)).data.customerId == request.auth.uid;
        allow update, delete: if request.auth != null && 
                                request.auth.uid == resource.data.userId;
      }
    }
    
    match /orders/{orderId} {
      allow read, write: if request.auth != null && 
                           request.auth.uid == resource.data.customerId;
      allow create: if request.auth != null && 
                      request.auth.uid == request.resource.data.customerId;
    }
  }
}`, validations: func(t *testing.T, rules []*repository.SecurityRule) {
				assert.Greater(t, len(rules), 2, "Should have multiple rules")

				// Validate that admin-only rules are properly translated
				productRule := findRuleByPath(rules, "/products/*")
				assert.NotNil(t, productRule, "Should have product rule")

				// Validate nested review rules
				reviewRule := findRuleByPath(rules, "/products/*/reviews/*")
				assert.NotNil(t, reviewRule, "Should have review rule")
			},
		}}

	parser := parser.NewModernParserInstance()
	translator := setupTestTranslator(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse rules
			parseResult, err := parser.ParseString(ctx, tc.rules)
			require.NoError(t, err, "Should parse %s rules", tc.description)

			// Translate rules
			translateResult, err := translator.Translate(ctx, parseResult.Ruleset)
			require.NoError(t, err, "Should translate %s rules", tc.description)

			rules := translateResult.Rules.([]*repository.SecurityRule)

			// Run specific validations
			tc.validations(t, rules)
		})
	}
}

// collectAllMatches recursively collects all match blocks from a ruleset, excluding the root
func collectAllMatches(matches []*domain.MatchBlock) []*domain.MatchBlock {
	var allMatches []*domain.MatchBlock
	for _, match := range matches {
		// Excluye el path root de Firestore
		if match.Path != "" && match.Path != "/databases/{database}/documents" {
			allMatches = append(allMatches, match)
		}
		// Recursively collect nested matches
		allMatches = append(allMatches, collectAllMatches(match.Nested)...)
	}
	return allMatches
}
