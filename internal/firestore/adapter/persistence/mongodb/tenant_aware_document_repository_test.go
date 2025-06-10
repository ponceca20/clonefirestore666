package mongodb

import "testing"

// --- UNIT TESTS: TenantAwareDocumentRepository (mocked tenant isolation) ---

func TestTenantAwareDocumentRepository_BasicInstantiation(t *testing.T) {
	_ = &TenantAwareDocumentRepository{}
}
