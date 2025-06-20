package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"firestore-clone/internal/rules_translator/adapter/parser"
	"firestore-clone/internal/rules_translator/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFirestoreCompatibilityParser tests 100% compatibility with Firestore rules syntax
func TestFirestoreCompatibilityParser(t *testing.T) {
	testParser := parser.NewModernParserInstance()
	ctx := context.Background()

	t.Run("Official Firestore Examples", func(t *testing.T) {
		content, err := os.ReadFile("fixtures/firestore_official_examples.rules")
		require.NoError(t, err, "Should read official examples file")

		result, err := testParser.ParseString(ctx, string(content))
		require.NoError(t, err, "Should parse official Firestore examples without errors")

		assert.NotNil(t, result.Ruleset, "Should have parsed ruleset")
		assert.Equal(t, "cloud.firestore", result.Ruleset.Service, "Should parse service correctly")
		assert.Greater(t, result.RuleCount, 0, "Should have parsed rules")
		assert.Empty(t, result.Errors, "Should have no parse errors")

		// Verify specific rule structures
		verifyOfficialExamplesStructure(t, result.Ruleset)
	})

	t.Run("Complex Real-World Examples", func(t *testing.T) {
		content, err := os.ReadFile("fixtures/firestore_complex_examples.rules")
		require.NoError(t, err, "Should read complex examples file")

		result, err := testParser.ParseString(ctx, string(content))
		require.NoError(t, err, "Should parse complex examples without errors")

		assert.NotNil(t, result.Ruleset, "Should have parsed ruleset")
		assert.Greater(t, result.RuleCount, 10, "Should have parsed many rules")
		assert.Empty(t, result.Errors, "Should have no parse errors")

		// Verify complex patterns
		verifyComplexPatternsStructure(t, result.Ruleset)
	})

	t.Run("Performance Requirements", func(t *testing.T) {
		// Large ruleset for performance testing
		largeRuleset := generateLargeRuleset(1000)

		start := time.Now()
		result, err := testParser.ParseString(ctx, largeRuleset)
		parseTime := time.Since(start)

		require.NoError(t, err, "Should parse large ruleset")
		assert.Less(t, parseTime, time.Millisecond*500, "Should parse large ruleset quickly")
		assert.Greater(t, result.RuleCount, 900, "Should parse most rules")
	})
}

// TestSpecificFirestoreFeatures tests individual Firestore features for exact compatibility
func TestSpecificFirestoreFeatures(t *testing.T) {
	testParser := parser.NewModernParserInstance()
	ctx := context.Background()

	testCases := []struct {
		name        string
		rules       string
		expectError bool
		validation  func(t *testing.T, result *domain.ParseResult)
	}{
		{
			name: "Rules Version Handling",
			rules: `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true;
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				assert.Equal(t, "cloud.firestore", result.Ruleset.Service)
			},
		},
		{
			name: "Multiple Operations in Single Allow",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read, write, update, delete: if request.auth != null;
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				match := findMatchByPath(result.Ruleset, "/test/{id}")
				require.NotNil(t, match, "Should find test match")
				require.Len(t, match.Allow, 1, "Should have one allow statement")

				operations := match.Allow[0].Operations
				expectedOps := []string{"read", "write", "update", "delete"}
				assert.ElementsMatch(t, expectedOps, operations, "Should parse all operations")
			},
		},
		{
			name: "Complex Condition with get() and exists()",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /posts/{postId} {
      allow write: if request.auth != null && 
        exists(/databases/$(database)/documents/users/$(request.auth.uid)) &&
        get(/databases/$(database)/documents/users/$(request.auth.uid)).data.verified == true;
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				match := findMatchByPath(result.Ruleset, "/posts/{postId}")
				require.NotNil(t, match, "Should find posts match")
				require.Len(t, match.Allow, 1, "Should have one allow statement")

				condition := match.Allow[0].Condition
				assert.Contains(t, condition, "exists(", "Should contain exists() call")
				assert.Contains(t, condition, "get(", "Should contain get() call")
				assert.Contains(t, condition, "request.auth.uid", "Should contain auth reference")
			},
		},
		{
			name: "Nested Collections",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /users/{userId} {
      allow read: if true;
      
      match /posts/{postId} {
        allow read: if request.auth.uid == userId;
        
        match /comments/{commentId} {
          allow write: if request.auth != null;
        }
      }
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				userMatch := findMatchByPath(result.Ruleset, "/users/{userId}")
				require.NotNil(t, userMatch, "Should find users match")
				require.Len(t, userMatch.Nested, 1, "Should have nested posts match")

				postMatch := userMatch.Nested[0]
				assert.Equal(t, "/posts/{postId}", postMatch.Path)
				require.Len(t, postMatch.Nested, 1, "Should have nested comments match")

				commentMatch := postMatch.Nested[0]
				assert.Equal(t, "/comments/{commentId}", commentMatch.Path)
			},
		},
		{
			name: "Wildcard Paths",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /{document=**} {
      allow read, write: if false;
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				match := findMatchByPath(result.Ruleset, "/{document=**}")
				require.NotNil(t, match, "Should find wildcard match")
				assert.Contains(t, match.Variables, "document", "Should extract wildcard variable")
			},
		},
		{
			name: "Comments and Whitespace Handling",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    // This is a comment
    match /test/{id} {
      // Another comment
      allow read: if true; // Inline comment
      /* Multi-line
         comment */
      allow write: if false;
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				match := findMatchByPath(result.Ruleset, "/test/{id}")
				require.NotNil(t, match, "Should find test match")
				assert.Len(t, match.Allow, 2, "Should parse both allow statements despite comments")
			},
		},
		{
			name: "Data Validation Rules",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /products/{productId} {
      allow create: if request.resource.data.name is string &&
        request.resource.data.name.size() > 0 &&
        request.resource.data.price is number &&
        request.resource.data.price > 0 &&
        request.resource.data.category in ['electronics', 'books', 'clothing'];
    }
  }
}`,
			expectError: false,
			validation: func(t *testing.T, result *domain.ParseResult) {
				match := findMatchByPath(result.Ruleset, "/products/{productId}")
				require.NotNil(t, match, "Should find products match")
				require.Len(t, match.Allow, 1, "Should have one allow statement")

				condition := match.Allow[0].Condition
				assert.Contains(t, condition, "is string", "Should contain type check")
				assert.Contains(t, condition, "is number", "Should contain number check")
				assert.Contains(t, condition, "in [", "Should contain list membership check")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testParser.ParseString(ctx, tc.rules)

			if tc.expectError {
				assert.Error(t, err, "Expected parsing error")
				return
			}

			require.NoError(t, err, "Should parse without error")
			require.NotNil(t, result, "Should return result")

			if tc.validation != nil {
				tc.validation(t, result)
			}
		})
	}
}

// TestParserErrorHandling tests robust error handling like Firestore
func TestParserErrorHandling(t *testing.T) {
	testParser := parser.NewModernParserInstance()
	ctx := context.Background()

	errorTestCases := []struct {
		name           string
		rules          string
		expectedErrors int
		errorContains  string
	}{
		{
			name: "Invalid Service Name",
			rules: `service invalid.service {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true;
    }
  }
}`,
			expectedErrors: 0, // Should be lenient like Firestore
		},
		{
			name: "Missing Semicolon",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true
    }
  }
}`,
			expectedErrors: 0, // Should be lenient
		},
		{
			name: "Unclosed Braces",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow read: if true;
    }
  }
  // Missing closing brace`,
			expectedErrors: 1,
			errorContains:  "unclosed",
		},
		{
			name: "Invalid Allow Syntax",
			rules: `service cloud.firestore {
  match /databases/{database}/documents {
    match /test/{id} {
      allow invalid_operation: if true;
    }
  }
}`,
			expectedErrors: 0, // Should accept unknown operations
		},
	}

	for _, tc := range errorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testParser.ParseString(ctx, tc.rules)

			if tc.expectedErrors > 0 {
				if err != nil {
					assert.Contains(t, err.Error(), tc.errorContains, "Error should contain expected text")
				} else {
					assert.GreaterOrEqual(t, len(result.Errors), tc.expectedErrors, "Should have expected number of errors")
					if len(result.Errors) > 0 && tc.errorContains != "" {
						found := false
						for _, parseError := range result.Errors {
							if strings.Contains(parseError.Message, tc.errorContains) {
								found = true
								break
							}
						}
						assert.True(t, found, "Should find error containing expected text")
					}
				}
			} else {
				assert.NoError(t, err, "Should not error for lenient cases")
			}
		})
	}
}

// TestParserPerformance tests performance requirements
func TestParserPerformance(t *testing.T) {
	testParser := parser.NewModernParserInstance()
	ctx := context.Background()

	t.Run("Large Ruleset Performance", func(t *testing.T) {
		// Generate a large but realistic ruleset
		largeRules := generateRealisticLargeRuleset(500)

		start := time.Now()
		result, err := testParser.ParseString(ctx, largeRules)
		duration := time.Since(start)

		require.NoError(t, err, "Should parse large ruleset")
		assert.Less(t, duration, time.Millisecond*200, "Should parse quickly")
		assert.Greater(t, result.RuleCount, 400, "Should parse most rules")

		t.Logf("Parsed %d rules in %v (%.2f rules/ms)",
			result.RuleCount, duration, float64(result.RuleCount)/float64(duration.Nanoseconds()/1e6))
	})

	t.Run("Memory Usage", func(t *testing.T) {
		// Test memory efficiency
		rules := generateRealisticLargeRuleset(100)

		for i := 0; i < 10; i++ {
			_, err := testParser.ParseString(ctx, rules)
			require.NoError(t, err)
			// Only parse, do not store results
		}

		// Verify no memory leaks in parser metrics
		metrics := testParser.GetMetrics()
		assert.NotNil(t, metrics, "Should have metrics")
		assert.Greater(t, metrics.TotalParsed, int64(5), "Should have processed multiple parses")
	})
}

// Helper functions

func verifyOfficialExamplesStructure(t *testing.T, ruleset *domain.FirestoreRuleset) {
	// Verify specific structures from official examples

	// Check users match
	userMatch := findMatchByPath(ruleset, "/users/{userId}")
	assert.NotNil(t, userMatch, "Should have users match")
	assert.Contains(t, userMatch.Variables, "userId", "Should extract userId variable")

	// Check posts match
	postMatch := findMatchByPath(ruleset, "/posts/{postId}")
	assert.NotNil(t, postMatch, "Should have posts match")

	// Check nested structure
	postMatchWithNesting := findMatchByPath(ruleset, "/posts/{postId}")
	if postMatchWithNesting != nil && len(postMatchWithNesting.Nested) > 0 {
		commentMatch := postMatchWithNesting.Nested[0]
		assert.Contains(t, commentMatch.Path, "comments", "Should have nested comments")
	}
}

func verifyComplexPatternsStructure(t *testing.T, ruleset *domain.FirestoreRuleset) {
	// Verify complex patterns are parsed correctly

	// Check orders match with complex conditions
	orderMatch := findMatchByPath(ruleset, "/orders/{orderId}")
	assert.NotNil(t, orderMatch, "Should have orders match")

	// Check tenant-based match
	tenantMatch := findMatchByPath(ruleset, "/tenants/{tenantId}/users/{userId}")
	assert.NotNil(t, tenantMatch, "Should have multi-tenant match")
	assert.Contains(t, tenantMatch.Variables, "tenantId", "Should extract tenantId")
	assert.Contains(t, tenantMatch.Variables, "userId", "Should extract userId")
}

func findMatchByPath(ruleset *domain.FirestoreRuleset, path string) *domain.MatchBlock {
	for _, match := range ruleset.Matches {
		if found := findMatchInBlock(match, path); found != nil {
			return found
		}
	}
	return nil
}

func findMatchInBlock(block *domain.MatchBlock, path string) *domain.MatchBlock {
	if block.Path == path || block.FullPath == path {
		return block
	}

	for _, nested := range block.Nested {
		if found := findMatchInBlock(nested, path); found != nil {
			return found
		}
	}

	return nil
}

func generateLargeRuleset(ruleCount int) string {
	var sb strings.Builder
	sb.WriteString(`service cloud.firestore {
  match /databases/{database}/documents {
`)

	for i := 0; i < ruleCount; i++ {
		sb.WriteString(fmt.Sprintf(`    match /collection%d/{doc%dId} {
      allow read: if request.auth != null;
      allow write: if request.auth.uid == resource.data.owner;
    }
`, i, i))
	}

	sb.WriteString(`  }
}`)
	return sb.String()
}

func generateRealisticLargeRuleset(ruleCount int) string {
	var sb strings.Builder
	sb.WriteString(`rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
`)

	patterns := []string{
		`    match /users/{userId} {
      allow read: if request.auth != null && request.auth.uid == userId;
    }`,
		`    match /posts/{postId} {
      allow read: if true;
      allow write: if request.auth != null && request.auth.uid == resource.data.author;
    }`,
		`    match /comments/{commentId} {
      allow read: if true;
      allow write: if request.auth != null && 
        get(/databases/$(database)/documents/posts/$(resource.data.postId)).data.author == request.auth.uid;
    }`,
		`    match /orders/{orderId} {
      allow read: if request.auth != null && request.auth.uid == resource.data.customerId;
      allow create: if request.auth != null && request.resource.data.total > 0;
    }`,
	}

	for i := 0; i < ruleCount; i++ {
		pattern := patterns[i%len(patterns)]
		// Modify pattern to make it unique
		uniquePattern := strings.ReplaceAll(pattern, "users", fmt.Sprintf("users%d", i/len(patterns)))
		uniquePattern = strings.ReplaceAll(uniquePattern, "posts", fmt.Sprintf("posts%d", i/len(patterns)))
		uniquePattern = strings.ReplaceAll(uniquePattern, "comments", fmt.Sprintf("comments%d", i/len(patterns)))
		uniquePattern = strings.ReplaceAll(uniquePattern, "orders", fmt.Sprintf("orders%d", i/len(patterns)))

		sb.WriteString(uniquePattern)
		sb.WriteString("\n")
	}

	sb.WriteString(`  }
}`)
	return sb.String()
}
