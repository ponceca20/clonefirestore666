package mongodb

import (
	"firestore-clone/internal/firestore/domain/repository"
	"testing"
)

func TestSecurityRulesEngine_ClearCache(t *testing.T) {
	engine := &SecurityRulesEngine{rulesCache: make(map[string][]*repository.SecurityRule)}
	engine.ClearCache("p1", "d1")
}
