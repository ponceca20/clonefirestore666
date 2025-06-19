package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/logger"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// CachedRule represents a compiled security rule with pre-compiled CEL programs
type CachedRule struct {
	Rule          *repository.SecurityRule
	AllowPrograms map[repository.OperationType]cel.Program
	DenyPrograms  map[repository.OperationType]cel.Program
	MatchRegex    *regexp.Regexp
	Variables     []string // Variable names extracted from the match pattern
}

// SecurityRulesEngine implements the repository.SecurityRulesEngine interface
type SecurityRulesEngine struct {
	collection       *mongo.Collection
	log              logger.Logger
	resourceAccessor repository.ResourceAccessor

	// Cache for compiled rules - maps "projectID:databaseID" to compiled rules
	rulesCache map[string][]*CachedRule
	cacheMu    sync.RWMutex

	// CEL environment for compiling expressions
	celEnv *cel.Env
}

// NewSecurityRulesEngine creates a new SecurityRulesEngine
func NewSecurityRulesEngine(db *mongo.Database, log logger.Logger) repository.SecurityRulesEngine {
	// Create CEL environment with security rule declarations
	celEnv, err := createCELEnvironment()
	if err != nil {
		log.Fatal("Failed to create CEL environment", zap.Error(err))
	}

	return &SecurityRulesEngine{
		collection: db.Collection("security_rules"),
		log:        log,
		rulesCache: make(map[string][]*CachedRule),
		celEnv:     celEnv,
	}
}

// createCELEnvironment creates and configures the CEL environment for security rules
func createCELEnvironment() (*cel.Env, error) {
	return cel.NewEnv(
		// Declare standard variables available in security rules
		cel.Declarations(
			decls.NewVar("auth", decls.Dyn),     // Allow auth to be null or map
			decls.NewVar("request", decls.Dyn),  // Allow request to be null or map
			decls.NewVar("resource", decls.Dyn), // Allow resource to be null or map
			decls.NewVar("path", decls.String),
			decls.NewVar("variables", decls.NewMapType(decls.String, decls.String)),
		),
	)
}

// SetResourceAccessor sets the resource accessor for CEL functions
func (e *SecurityRulesEngine) SetResourceAccessor(accessor repository.ResourceAccessor) {
	e.resourceAccessor = accessor
}

// LoadRules loads security rules from storage and compiles them for efficient evaluation
func (e *SecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)

	// Check cache first
	e.cacheMu.RLock()
	if cachedRules, exists := e.rulesCache[cacheKey]; exists {
		e.cacheMu.RUnlock()
		// Convert cached rules back to SecurityRule slice
		rules := make([]*repository.SecurityRule, len(cachedRules))
		for i, cached := range cachedRules {
			rules[i] = cached.Rule
		}
		e.log.Debug("Loaded security rules from cache",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Int("count", len(rules)))
		return rules, nil
	}
	e.cacheMu.RUnlock()

	filter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	cursor, err := e.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "priority", Value: -1}}))
	if err != nil {
		e.log.Error("Failed to load security rules",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Error(err))
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*repository.SecurityRule
	var cachedRules []*CachedRule

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			e.log.Error("Failed to decode security rule", zap.Error(err))
			continue
		}

		rule := &repository.SecurityRule{
			Match:    doc["match"].(string),
			Priority: int(doc["priority"].(int32)),
			Allow:    make(map[repository.OperationType]string),
			Deny:     make(map[repository.OperationType]string),
		}

		// Parse description if present
		if desc, ok := doc["description"].(string); ok {
			rule.Description = desc
		}

		// Parse allow conditions
		if allowDoc, ok := doc["allow"].(bson.M); ok {
			for op, condition := range allowDoc {
				rule.Allow[repository.OperationType(op)] = condition.(string)
			}
		}

		// Parse deny conditions
		if denyDoc, ok := doc["deny"].(bson.M); ok {
			for op, condition := range denyDoc {
				rule.Deny[repository.OperationType(op)] = condition.(string)
			}
		}

		// Compile the rule for caching
		cachedRule, err := e.compileRule(rule)
		if err != nil {
			e.log.Error("Failed to compile security rule",
				zap.String("match", rule.Match),
				zap.Error(err))
			continue // Skip invalid rules
		}

		rules = append(rules, rule)
		cachedRules = append(cachedRules, cachedRule)
	}

	if err := cursor.Err(); err != nil {
		e.log.Error("Cursor error while loading security rules", zap.Error(err))
		return nil, err
	}

	// Cache the compiled rules
	e.cacheMu.Lock()
	e.rulesCache[cacheKey] = cachedRules
	e.cacheMu.Unlock()

	e.log.Debug("Loaded and compiled security rules",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID),
		zap.Int("count", len(rules)))

	return rules, nil
}

// SaveRules saves security rules to storage
func (e *SecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	// Try to use transactions if available, otherwise use normal operations
	err := e.saveRulesWithTransaction(ctx, projectID, databaseID, rules)
	if err != nil {
		// If transaction fails (e.g., no replica set), fall back to non-transactional operations
		if strings.Contains(err.Error(), "Transaction numbers are only allowed") ||
			strings.Contains(err.Error(), "IllegalOperation") {
			e.log.Warn("Transactions not supported, using non-transactional operations",
				zap.String("projectID", projectID),
				zap.String("databaseID", databaseID))
			err = e.saveRulesWithoutTransaction(ctx, projectID, databaseID, rules)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Clear cache for this project/database (regardless of transaction method)
	e.ClearCache(projectID, databaseID)

	e.log.Info("Saved security rules",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID),
		zap.Int("count", len(rules)))

	return nil
}

// saveRulesWithTransaction saves rules using MongoDB transactions
func (e *SecurityRulesEngine) saveRulesWithTransaction(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	// Start a transaction to ensure atomicity
	session, err := e.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	callback := func(sessionContext mongo.SessionContext) (interface{}, error) {
		// Delete existing rules for this project/database
		deleteFilter := bson.M{
			"project_id":  projectID,
			"database_id": databaseID,
		}

		_, err := e.collection.DeleteMany(sessionContext, deleteFilter)
		if err != nil {
			return nil, err
		}

		// Insert new rules
		if len(rules) > 0 {
			documents := make([]interface{}, len(rules))
			for i, rule := range rules {
				doc := bson.M{
					"project_id":  projectID,
					"database_id": databaseID,
					"match":       rule.Match,
					"allow":       rule.Allow,
					"deny":        rule.Deny,
					"priority":    rule.Priority,
					"created_at":  time.Now(),
					"updated_at":  time.Now(),
				}

				// Add description if present
				if rule.Description != "" {
					doc["description"] = rule.Description
				}

				documents[i] = doc
			}

			_, err = e.collection.InsertMany(sessionContext, documents)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		e.log.Error("Failed to save security rules with transaction",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Error(err))
		return err
	}

	return nil
}

// saveRulesWithoutTransaction saves rules without using transactions (fallback)
func (e *SecurityRulesEngine) saveRulesWithoutTransaction(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	// Delete existing rules for this project/database
	deleteFilter := bson.M{
		"project_id":  projectID,
		"database_id": databaseID,
	}

	_, err := e.collection.DeleteMany(ctx, deleteFilter)
	if err != nil {
		e.log.Error("Failed to delete existing security rules",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Error(err))
		return err
	}

	// Insert new rules
	if len(rules) > 0 {
		documents := make([]interface{}, len(rules))
		for i, rule := range rules {
			doc := bson.M{
				"project_id":  projectID,
				"database_id": databaseID,
				"match":       rule.Match,
				"allow":       rule.Allow,
				"deny":        rule.Deny,
				"priority":    rule.Priority,
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			}

			// Add description if present
			if rule.Description != "" {
				doc["description"] = rule.Description
			}

			documents[i] = doc
		}

		_, err = e.collection.InsertMany(ctx, documents)
		if err != nil {
			e.log.Error("Failed to insert security rules",
				zap.String("projectID", projectID),
				zap.String("databaseID", databaseID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// ValidateRules validates the syntax and logic of security rules
func (e *SecurityRulesEngine) ValidateRules(rules []*repository.SecurityRule) error {
	if len(rules) == 0 {
		return nil
	}

	// Check for duplicate priorities
	priorityMap := make(map[int]bool)
	for _, rule := range rules {
		if priorityMap[rule.Priority] {
			return fmt.Errorf("duplicate priority %d found in rules", rule.Priority)
		}
		priorityMap[rule.Priority] = true
	}

	// Sort rules by priority for validation
	sortedRules := make([]*repository.SecurityRule, len(rules))
	copy(sortedRules, rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	for _, rule := range sortedRules {
		// Validate match pattern
		if err := e.validateMatchPattern(rule.Match); err != nil {
			return fmt.Errorf("invalid match pattern '%s': %w", rule.Match, err)
		}

		// Validate allow conditions
		for op, condition := range rule.Allow {
			if err := e.validateCondition(string(op), condition); err != nil {
				return fmt.Errorf("invalid allow condition for operation '%s': %w", op, err)
			}
		}

		// Validate deny conditions
		for op, condition := range rule.Deny {
			if err := e.validateCondition(string(op), condition); err != nil {
				return fmt.Errorf("invalid deny condition for operation '%s': %w", op, err)
			}
		}

		// Check that rule has at least one allow or deny condition
		if len(rule.Allow) == 0 && len(rule.Deny) == 0 {
			return fmt.Errorf("rule with match pattern '%s' has no allow or deny conditions", rule.Match)
		}
	}

	return nil
}

// validateMatchPattern validates a Firestore path pattern
func (e *SecurityRulesEngine) validateMatchPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("match pattern cannot be empty")
	}

	// Check for valid Firestore path characters
	validPathRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-/{}*]+$`)
	if !validPathRegex.MatchString(pattern) {
		return fmt.Errorf("pattern contains invalid characters")
	}

	// Check for balanced braces
	braceCount := 0
	for _, char := range pattern {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount < 0 {
				return fmt.Errorf("unmatched closing brace")
			}
		}
	}
	if braceCount != 0 {
		return fmt.Errorf("unmatched opening brace")
	}

	// Validate variable names in braces
	braceRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := braceRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		varName := match[1]
		if varName == "" {
			return fmt.Errorf("empty variable name in braces")
		}
		// Variable names should be valid identifiers
		validVarRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		if !validVarRegex.MatchString(varName) {
			return fmt.Errorf("invalid variable name '%s'", varName)
		}
	}

	return nil
}

// validateCondition validates a rule condition
func (e *SecurityRulesEngine) validateCondition(operation, condition string) error {
	if condition == "" {
		return fmt.Errorf("condition cannot be empty")
	}

	// Validate operation type
	validOps := map[string]bool{
		"read":   true,
		"write":  true,
		"delete": true,
		"create": true,
		"update": true,
	}
	if !validOps[operation] {
		return fmt.Errorf("invalid operation type '%s'", operation)
	}

	// Basic condition validation (this could be expanded with a full expression parser)
	condition = strings.TrimSpace(condition)

	// Check for common patterns
	commonPatterns := []string{
		"true",
		"false",
		"auth != null",
		"auth == null",
	}

	for _, pattern := range commonPatterns {
		if condition == pattern {
			return nil // Valid common pattern
		}
	}

	// Allow more complex conditions (in a real implementation, this would use a proper parser)
	if strings.Contains(condition, "auth") ||
		strings.Contains(condition, "resource") ||
		strings.Contains(condition, "request") {
		return nil // Assume valid for now
	}

	// If we get here, the condition might be invalid
	e.log.Warn("Unknown condition pattern, allowing for now",
		zap.String("condition", condition))

	return nil
}

// compileRule compiles a security rule into a CachedRule with pre-compiled CEL programs
func (e *SecurityRulesEngine) compileRule(rule *repository.SecurityRule) (*CachedRule, error) {
	cached := &CachedRule{
		Rule:          rule,
		AllowPrograms: make(map[repository.OperationType]cel.Program),
		DenyPrograms:  make(map[repository.OperationType]cel.Program),
	}

	// Compile match pattern to regex and extract variables
	matchRegex, variables, err := e.compileMatchPattern(rule.Match)
	if err != nil {
		return nil, fmt.Errorf("failed to compile match pattern '%s': %w", rule.Match, err)
	}
	cached.MatchRegex = matchRegex
	cached.Variables = variables

	// Compile allow conditions
	for op, condition := range rule.Allow {
		program, err := e.compileCELExpression(condition)
		if err != nil {
			return nil, fmt.Errorf("failed to compile allow condition for operation '%s': %w", op, err)
		}
		cached.AllowPrograms[op] = program
	}

	// Compile deny conditions
	for op, condition := range rule.Deny {
		program, err := e.compileCELExpression(condition)
		if err != nil {
			return nil, fmt.Errorf("failed to compile deny condition for operation '%s': %w", op, err)
		}
		cached.DenyPrograms[op] = program
	}

	return cached, nil
}

// compileMatchPattern converts a Firestore match pattern to a regex and extracts variable names
func (e *SecurityRulesEngine) compileMatchPattern(pattern string) (*regexp.Regexp, []string, error) {
	// Extract variable names from pattern
	varRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := varRegex.FindAllStringSubmatch(pattern, -1)
	variables := make([]string, len(matches))
	for i, match := range matches {
		variables[i] = match[1]
	}

	// Convert Firestore pattern to regex
	// Replace {variableName} with named capture groups
	// Replace {variableName=**} with recursive wildcard matching
	regexPattern := regexp.QuoteMeta(pattern)

	// Handle recursive wildcards first (=**)
	regexPattern = regexp.MustCompile(`\\{([^}]+)=\\\*\\\*\\}`).ReplaceAllString(regexPattern, `(?P<$1>.*)`)

	// Handle regular wildcards
	regexPattern = regexp.MustCompile(`\\{([^}]+)\\}`).ReplaceAllString(regexPattern, `(?P<$1>[^/]+)`)

	// Anchor the pattern
	regexPattern = "^" + regexPattern + "$"

	compiledRegex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile regex pattern: %w", err)
	}

	return compiledRegex, variables, nil
}

// compileCELExpression compiles a CEL expression string into a program
func (e *SecurityRulesEngine) compileCELExpression(expression string) (cel.Program, error) {
	ast, issues := e.celEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	program, err := e.celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	return program, nil
}

// EvaluateAccess evaluates security rules to determine if an operation is allowed
func (e *SecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	startTime := time.Now()

	// Validate input parameters
	if securityContext == nil {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid security context: securityContext is required",
		}, fmt.Errorf("securityContext is required")
	}

	if securityContext.ProjectID == "" {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid security context: projectID is required",
		}, fmt.Errorf("projectID is required")
	}

	if securityContext.DatabaseID == "" {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid security context: databaseID is required",
		}, fmt.Errorf("databaseID is required")
	}

	result := &repository.RuleEvaluationResult{
		Allowed: false,
		Reason:  "No matching rule found (default deny)",
	}

	// Load cached rules
	cacheKey := fmt.Sprintf("%s:%s", securityContext.ProjectID, securityContext.DatabaseID)
	e.cacheMu.RLock()
	cachedRules, exists := e.rulesCache[cacheKey]
	e.cacheMu.RUnlock()

	if !exists {
		// Load rules if not in cache
		_, err := e.LoadRules(ctx, securityContext.ProjectID, securityContext.DatabaseID)
		if err != nil {
			return nil, fmt.Errorf("failed to load security rules: %w", err)
		}

		// Try again after loading
		e.cacheMu.RLock()
		cachedRules, exists = e.rulesCache[cacheKey]
		e.cacheMu.RUnlock()

		if !exists {
			result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
			return result, nil
		}
	}

	// Find the first matching rule (rules are sorted by priority)
	for _, cachedRule := range cachedRules {
		if e.matchesPath(cachedRule, securityContext.Path) {
			// Extract variables from path
			variables := e.extractVariables(cachedRule, securityContext.Path)

			// Create updated security context with variables
			contextWithVars := *securityContext
			contextWithVars.Variables = variables

			// Check deny rules first (deny takes precedence)
			if denyProgram, exists := cachedRule.DenyPrograms[operation]; exists {
				denied, reason, err := e.evaluateCondition(ctx, denyProgram, &contextWithVars)
				if err != nil {
					e.log.Error("Failed to evaluate deny condition",
						zap.String("rule", cachedRule.Rule.Match),
						zap.String("operation", string(operation)),
						zap.Error(err))
					continue
				}

				if denied {
					result.Allowed = false
					result.DeniedBy = cachedRule.Rule.Match
					result.Reason = reason
					result.RuleMatch = cachedRule.Rule.Match
					result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
					return result, nil
				}
			}

			// Check allow rules
			if allowProgram, exists := cachedRule.AllowPrograms[operation]; exists {
				allowed, reason, err := e.evaluateCondition(ctx, allowProgram, &contextWithVars)
				if err != nil {
					e.log.Error("Failed to evaluate allow condition",
						zap.String("rule", cachedRule.Rule.Match),
						zap.String("operation", string(operation)),
						zap.Error(err))
					continue
				}

				if allowed {
					result.Allowed = true
					result.AllowedBy = cachedRule.Rule.Match
					result.Reason = reason
					result.RuleMatch = cachedRule.Rule.Match
					result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
					return result, nil
				}
			}

			// If this rule matched but didn't produce a definitive result,
			// continue to next rule instead of breaking
			// Only set the first matching rule if no rule has been set yet
			if result.RuleMatch == "" {
				result.RuleMatch = cachedRule.Rule.Match
			}
		}
	}

	// If we get here, no rule produced a definitive allow/deny result
	if result.RuleMatch != "" {
		result.Reason = fmt.Sprintf("Rule matched but no condition for operation '%s'", operation)
	}

	result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// matchesPath checks if a cached rule's regex matches the given path
func (e *SecurityRulesEngine) matchesPath(cachedRule *CachedRule, path string) bool {
	return cachedRule.MatchRegex.MatchString(path)
}

// extractVariables extracts path variables using the cached rule's regex
func (e *SecurityRulesEngine) extractVariables(cachedRule *CachedRule, path string) map[string]string {
	variables := make(map[string]string)

	matches := cachedRule.MatchRegex.FindStringSubmatch(path)
	if matches == nil {
		return variables
	}

	subexpNames := cachedRule.MatchRegex.SubexpNames()
	for i, name := range subexpNames {
		if i != 0 && name != "" && i < len(matches) {
			variables[name] = matches[i]
		}
	}

	return variables
}

// evaluateCondition evaluates a CEL program with the given security context
func (e *SecurityRulesEngine) evaluateCondition(ctx context.Context, program cel.Program, securityContext *repository.SecurityContext) (bool, string, error) {
	// Prepare evaluation variables
	vars := map[string]interface{}{
		"request":   securityContext.Request,
		"resource":  securityContext.Resource,
		"path":      securityContext.Path,
		"variables": securityContext.Variables,
	}

	// Add auth information if user is present
	if securityContext.User != nil {
		authMap := map[string]interface{}{
			"uid": securityContext.User.ID.Hex(),
		}
		if securityContext.User.Email != "" {
			authMap["token"] = map[string]interface{}{
				"email": securityContext.User.Email,
			}
		}
		vars["auth"] = authMap
	} else {
		vars["auth"] = nil
	}

	// Evaluate the CEL program
	out, _, err := program.Eval(vars)
	if err != nil {
		return false, "", fmt.Errorf("CEL evaluation error: %w", err)
	}

	// Convert result to boolean
	result, ok := out.Value().(bool)
	if !ok {
		return false, "", fmt.Errorf("CEL expression did not return boolean value")
	}
	reason := fmt.Sprintf("CEL expression evaluated to %v", result)
	return result, reason, nil
}
