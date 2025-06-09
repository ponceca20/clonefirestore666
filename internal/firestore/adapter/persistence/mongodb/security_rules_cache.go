package mongodb

import (
	"context"
	"fmt"

	"firestore-clone/internal/firestore/domain/repository"
)

// getCachedRules gets rules from cache or loads them from storage
func (e *SecurityRulesEngine) getCachedRules(ctx context.Context, cacheKey, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	e.cacheMu.RLock()
	if rules, exists := e.rulesCache[cacheKey]; exists {
		e.cacheMu.RUnlock()
		return rules, nil
	}
	e.cacheMu.RUnlock()

	// Load from storage
	rules, err := e.LoadRules(ctx, projectID, databaseID)
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
func (e *SecurityRulesEngine) ClearCache(projectID, databaseID string) {
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)
	e.cacheMu.Lock()
	delete(e.rulesCache, cacheKey)
	e.cacheMu.Unlock()
}

// ClearAllCache clears all cached rules
func (e *SecurityRulesEngine) ClearAllCache() {
	e.cacheMu.Lock()
	e.rulesCache = make(map[string][]*repository.SecurityRule)
	e.cacheMu.Unlock()
}
