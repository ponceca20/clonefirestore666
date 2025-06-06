package contextkeys

// contextKey is an unexported type to prevent collisions with context keys defined in
// other packages.
type contextKey string

// String makes contextKey satisfy the Stringer interface to assist with debugging.
func (c contextKey) String() string {
	return "firestore-clone context key " + string(c)
}

// TenantIDKey is the key for TenantID in context.Context
const TenantIDKey = contextKey("tenantID")

// UserIDKey is the key for UserID in context.Context (example, might be useful)
// const UserIDKey = contextKey("userID")

// UserEmailKey is the key for UserEmail in context.Context (example, might be useful)
// const UserEmailKey = contextKey("userEmail")
