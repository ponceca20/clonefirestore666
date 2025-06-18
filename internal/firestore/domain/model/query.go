package model

// Firestore query model definitions
import (
	"errors"
	"time"
)

// Query represents a Firestore query with all supported operations
type Query struct {
	// Collection path
	Path         string `json:"path" bson:"path"`                  // Path to the collection or subcollection
	CollectionID string `json:"collectionId" bson:"collection_id"` // Collection identifier

	// Basic filters and ordering
	Filters []Filter `json:"filters,omitempty" bson:"filters,omitempty"` // List of where clauses
	Orders  []Order  `json:"orders,omitempty" bson:"orders,omitempty"`   // List of order by clauses

	// Pagination
	Limit  int `json:"limit,omitempty" bson:"limit,omitempty"`   // Limit number of documents
	Offset int `json:"offset,omitempty" bson:"offset,omitempty"` // Offset for pagination

	// Cursor-based pagination
	StartAt    []interface{} `json:"startAt,omitempty" bson:"start_at,omitempty"`       // Values for startAt cursor
	StartAfter []interface{} `json:"startAfter,omitempty" bson:"start_after,omitempty"` // Values for startAfter cursor
	EndAt      []interface{} `json:"endAt,omitempty" bson:"end_at,omitempty"`           // Values for endAt cursor
	EndBefore  []interface{} `json:"endBefore,omitempty" bson:"end_before,omitempty"`   // Values for endBefore cursor

	// Field selection
	SelectFields []string `json:"selectFields,omitempty" bson:"select_fields,omitempty"` // Fields to select (projection)

	// Query options
	AllDescendants bool `json:"allDescendants,omitempty" bson:"all_descendants,omitempty"` // Include subcollections

	// LimitToLast for reverse pagination
	LimitToLast bool `json:"limitToLast,omitempty" bson:"limit_to_last,omitempty"`
}

// Filter represents a single filter condition in a query (where clause)
type Filter struct {
	Field     string         `json:"field" bson:"field"`                              // Document field to filter (legacy)
	FieldPath *FieldPath     `json:"fieldPath,omitempty" bson:"field_path,omitempty"` // Enhanced field path support
	Operator  Operator       `json:"operator" bson:"operator"`                        // Comparison operator
	Value     interface{}    `json:"value" bson:"value"`                              // Value to compare against
	ValueType FieldValueType `json:"valueType,omitempty" bson:"value_type,omitempty"` // Type hint for MongoDB mapping

	// For composite filters (AND/OR)
	Composite  string   `json:"composite,omitempty" bson:"composite,omitempty"` // "and" or "or"
	SubFilters []Filter `json:"subFilters,omitempty" bson:"sub_filters,omitempty"`
}

// Order represents a single ordering condition in a query
type Order struct {
	Field     string    `json:"field" bson:"field"`         // Document field to order by
	Direction Direction `json:"direction" bson:"direction"` // Sort direction
}

// Operator represents query filter operators
type Operator string

const (
	// Comparison operators
	OperatorEqual              Operator = "=="
	OperatorNotEqual           Operator = "!="
	OperatorLessThan           Operator = "<"
	OperatorLessThanOrEqual    Operator = "<="
	OperatorGreaterThan        Operator = ">"
	OperatorGreaterThanOrEqual Operator = ">="

	// Array operators
	OperatorArrayContains    Operator = "array-contains"
	OperatorArrayContainsAny Operator = "array-contains-any"
	OperatorIn               Operator = "in"
	OperatorNotIn            Operator = "not-in"
)

// Direction represents sort direction
type Direction string

const (
	DirectionAscending  Direction = "asc"
	DirectionDescending Direction = "desc"
)

// QueryCursor represents a cursor for pagination
type QueryCursor struct {
	Values    []interface{} `json:"values" bson:"values"`
	Before    bool          `json:"before" bson:"before"`
	Timestamp time.Time     `json:"timestamp" bson:"timestamp"`
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Documents []Document   `json:"documents" bson:"documents"`
	Cursor    *QueryCursor `json:"cursor,omitempty" bson:"cursor,omitempty"`
	HasMore   bool         `json:"hasMore" bson:"has_more"`
	ReadTime  time.Time    `json:"readTime" bson:"read_time"`
}

// CompositeQuery represents a compound query with multiple conditions
type CompositeQuery struct {
	Queries  []Query         `json:"queries" bson:"queries"`
	Operator LogicalOperator `json:"operator" bson:"operator"`
}

// LogicalOperator represents logical operators for composite queries
type LogicalOperator string

const (
	LogicalOperatorAnd LogicalOperator = "AND"
	LogicalOperatorOr  LogicalOperator = "OR"
)

// AggregateQuery represents an aggregation query
type AggregateQuery struct {
	CollectionPath string        `json:"collectionPath" bson:"collection_path"`
	Aggregations   []Aggregation `json:"aggregations" bson:"aggregations"`
	Filters        []Filter      `json:"filters,omitempty" bson:"filters,omitempty"`
}

// Aggregation represents a single aggregation operation
type Aggregation struct {
	Type  AggregationType `json:"type" bson:"type"`
	Field string          `json:"field,omitempty" bson:"field,omitempty"`
	Alias string          `json:"alias,omitempty" bson:"alias,omitempty"`
}

// AggregationType represents types of aggregation operations
type AggregationType string

const (
	AggregationCount   AggregationType = "count"
	AggregationSum     AggregationType = "sum"
	AggregationAverage AggregationType = "average"
	AggregationMin     AggregationType = "min"
	AggregationMax     AggregationType = "max"
)

// ValidateQuery validates a query for correctness
func (q *Query) ValidateQuery() error {
	if q.Path == "" {
		return ErrInvalidQueryPath
	}

	if q.Limit < 0 {
		return ErrInvalidQueryLimit
	}

	if q.Offset < 0 {
		return ErrInvalidQueryOffset
	}

	// Validate filters
	for _, filter := range q.Filters {
		if filter.Field == "" {
			return ErrInvalidFilterField
		}
		if !isValidOperator(filter.Operator) {
			return ErrInvalidFilterOperator
		}
	}

	// Validate orders
	for _, order := range q.Orders {
		if order.Field == "" {
			return ErrInvalidOrderField
		}
		if order.Direction != DirectionAscending && order.Direction != DirectionDescending {
			return ErrInvalidOrderDirection
		}
	}

	return nil
}

// Helper function to validate operators
func isValidOperator(op Operator) bool {
	validOps := []Operator{
		OperatorEqual, OperatorNotEqual, OperatorLessThan, OperatorLessThanOrEqual,
		OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorArrayContains,
		OperatorArrayContainsAny, OperatorIn, OperatorNotIn,
	}

	for _, validOp := range validOps {
		if op == validOp {
			return true
		}
	}
	return false
}

// AddFilter adds a new filter to the query
func (q *Query) AddFilter(field string, operator Operator, value interface{}) *Query {
	q.Filters = append(q.Filters, Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return q
}

// AddFilterWithFieldPath adds a new filter using a FieldPath to the query
func (q *Query) AddFilterWithFieldPath(fieldPath *FieldPath, operator Operator, value interface{}, valueType FieldValueType) *Query {
	q.Filters = append(q.Filters, Filter{
		FieldPath: fieldPath,
		Field:     fieldPath.Raw(), // Keep legacy field for backward compatibility
		Operator:  operator,
		Value:     value,
		ValueType: valueType,
	})
	return q
}

// GetEffectiveFieldPath returns the FieldPath if available, otherwise creates one from Field
func (f *Filter) GetEffectiveFieldPath() (*FieldPath, error) {
	if f.FieldPath != nil {
		return f.FieldPath, nil
	}

	if f.Field != "" {
		return NewFieldPath(f.Field)
	}

	return nil, ErrInvalidFilterField
}

// IsNestedField returns true if the filter targets a nested field
func (f *Filter) IsNestedField() bool {
	if fp, err := f.GetEffectiveFieldPath(); err == nil {
		return fp.IsNested()
	}
	return false
}

// IsComposite returns true if this is a composite filter (AND/OR)
func (f *Filter) IsComposite() bool {
	return f.Composite != "" && len(f.SubFilters) > 0
}

// IsArrayOperation returns true if the operator works on arrays
func (f *Filter) IsArrayOperation() bool {
	return f.Operator == OperatorArrayContains ||
		f.Operator == OperatorArrayContainsAny ||
		f.Operator == OperatorIn ||
		f.Operator == OperatorNotIn
}

// AddOrder adds a new ordering to the query
func (q *Query) AddOrder(field string, direction Direction) *Query {
	q.Orders = append(q.Orders, Order{
		Field:     field,
		Direction: direction,
	})
	return q
}

// SetLimit sets the limit for the query
func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
}

// SetOffset sets the offset for the query
func (q *Query) SetOffset(offset int) *Query {
	q.Offset = offset
	return q
}

// Query validation errors
var (
	ErrInvalidQueryPath      = errors.New("invalid query path")
	ErrInvalidQueryLimit     = errors.New("invalid query limit")
	ErrInvalidQueryOffset    = errors.New("invalid query offset")
	ErrInvalidFilterField    = errors.New("invalid filter field")
	ErrInvalidFilterOperator = errors.New("invalid filter operator")
	ErrInvalidOrderField     = errors.New("invalid order field")
	ErrInvalidOrderDirection = errors.New("invalid order direction")
	ErrQueryTooComplex       = errors.New("query too complex")
)
