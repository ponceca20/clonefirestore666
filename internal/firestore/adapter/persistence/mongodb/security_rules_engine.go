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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// SecurityRulesEngine implements the repository.SecurityRulesEngine interface
type SecurityRulesEngine struct {
	collection *mongo.Collection
	log        logger.Logger
	// Cache for compiled rules
	rulesCache map[string][]*repository.SecurityRule
	cacheMu    sync.RWMutex
}

// NewSecurityRulesEngine creates a new SecurityRulesEngine
func NewSecurityRulesEngine(db *mongo.Database, log logger.Logger) repository.SecurityRulesEngine {
	return &SecurityRulesEngine{
		collection: db.Collection("security_rules"),
		log:        log,
		rulesCache: make(map[string][]*repository.SecurityRule),
	}
}

// LoadRules loads security rules from storage
func (e *SecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
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

		rules = append(rules, rule)
	}

	if err := cursor.Err(); err != nil {
		e.log.Error("Cursor error while loading security rules", zap.Error(err))
		return nil, err
	}

	e.log.Debug("Loaded security rules",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID),
		zap.Int("count", len(rules)))

	return rules, nil
}

// SaveRules saves security rules to storage
func (e *SecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
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
				documents[i] = bson.M{
					"project_id":  projectID,
					"database_id": databaseID,
					"match":       rule.Match,
					"allow":       rule.Allow,
					"deny":        rule.Deny,
					"priority":    rule.Priority,
					"created_at":  time.Now(),
					"updated_at":  time.Now(),
				}
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
		e.log.Error("Failed to save security rules",
			zap.String("projectID", projectID),
			zap.String("databaseID", databaseID),
			zap.Error(err))
		return err
	}

	// Clear cache for this project/database
	e.ClearCache(projectID, databaseID)

	e.log.Info("Saved security rules",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID),
		zap.Int("count", len(rules)))

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
