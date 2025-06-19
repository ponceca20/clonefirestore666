package mongodb

import (
	"fmt"

	"go.uber.org/zap"
)

// ClearCache clears the rules cache for a specific project/database
func (e *SecurityRulesEngine) ClearCache(projectID, databaseID string) {
	cacheKey := fmt.Sprintf("%s:%s", projectID, databaseID)
	e.cacheMu.Lock()
	delete(e.rulesCache, cacheKey)
	e.cacheMu.Unlock()

	e.log.Debug("Cleared security rules cache",
		zap.String("projectID", projectID),
		zap.String("databaseID", databaseID))
}

// ClearAllCache clears all cached rules
func (e *SecurityRulesEngine) ClearAllCache() {
	e.cacheMu.Lock()
	e.rulesCache = make(map[string][]*CachedRule)
	e.cacheMu.Unlock()

	e.log.Debug("Cleared all security rules cache")
}
