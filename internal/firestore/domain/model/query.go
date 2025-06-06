package model

// Query represents a Firestore query.
type Query struct {
	Path         string        // Path to the collection or subcollection
	Filters      []Filter      // List of where clauses
	Orders       []Order       // List of order by clauses
	Limit        int           // Limit number of documents
	Offset       int           // Offset for pagination
	StartAt      []interface{} // Values for startAt cursor
	EndAt        []interface{} // Values for endAt cursor
	SelectFields []string      // Fields to select (projection)
}

// Filter represents a single filter condition in a query (where clause).
type Filter struct {
	Field    string      // Document field to filter
	Operator string      // Comparison operator (==, !=, <, <=, >, >=, etc.)
	Value    interface{} // Value to compare against
}

// Order represents a single ordering condition in a query.
type Order struct {
	Field     string // Document field to order by
	Direction string // "asc" or "desc"
}

const (
	// Ascending is used for ordering in ascending order.
	Ascending = "asc"
	// Descending is used for ordering in descending order.
	Descending = "desc"
)

// Operator types for filters
const (
	OperatorEqual              = "=="
	OperatorNotEqual           = "!="
	OperatorLessThan           = "<"
	OperatorLessThanOrEqual    = "<="
	OperatorGreaterThan        = ">"
	OperatorGreaterThanOrEqual = ">="
	OperatorArrayContains      = "array-contains"
	OperatorArrayContainsAny   = "array-contains-any"
	OperatorIn                 = "in"
	OperatorNotIn              = "not-in"
)
