package contextkeys

// contextKey is an unexported type to prevent collisions with context keys defined in
// other packages.
type contextKey string

// String makes contextKey satisfy the Stringer interface to assist with debugging.
func (c contextKey) String() string {
	return "firestore-clone context key " + string(c)
}

// Context keys for Firestore clone application
const (
	// User-related context keys
	UserIDKey    = contextKey("userID")
	UserEmailKey = contextKey("userEmail")
	TenantIDKey  = contextKey("tenantID")

	// Request-related context keys
	RequestIDKey = contextKey("requestID")

	// Firestore-specific context keys
	ProjectIDKey  = contextKey("projectID")
	DatabaseIDKey = contextKey("databaseID")

	// Authentication context keys
	TokenKey  = contextKey("token")
	ClaimsKey = contextKey("claims")

	// Component context keys
	ComponentKey = contextKey("component")
	OperationKey = contextKey("operation")
)
