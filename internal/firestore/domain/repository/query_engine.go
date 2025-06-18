package repository

import (
	"context"
	"firestore-clone/internal/firestore/domain/model"
)

// FieldPathResolver defines the interface for resolving Firestore field paths to database-specific paths
// This follows the hexagonal architecture pattern as a domain service
type FieldPathResolver interface {
	// ResolveFieldPath converts a Firestore field path to the database-specific field path
	// Example: "customer.ruc" -> "fields.customer.value.ruc" (for MongoDB)
	ResolveFieldPath(fieldPath *model.FieldPath, valueType model.FieldValueType) (string, error)

	// ResolveOrderFieldPath resolves field paths for ordering operations
	ResolveOrderFieldPath(fieldPath *model.FieldPath, valueType model.FieldValueType) (string, error)

	// SupportsNestedQueries returns true if the underlying database supports nested field queries
	SupportsNestedQueries() bool

	// SupportsArrayQueries returns true if the underlying database supports array operations
	SupportsArrayQueries() bool

	// GetMaxNestingDepth returns the maximum supported nesting depth
	GetMaxNestingDepth() int
}

// QueryEngine defines the enhanced interface for executing queries with field path support
type QueryEngine interface {
	// ExecuteQuery executes a query with enhanced field path support
	ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error)

	// ExecuteQueryWithProjection executes a query with field projection
	ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error)

	// CountDocuments returns the count of documents matching the query
	CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error)

	// ValidateQuery validates if a query is supported by the engine
	ValidateQuery(query model.Query) error

	// GetQueryCapabilities returns the capabilities of this query engine
	GetQueryCapabilities() QueryCapabilities
}

// QueryCapabilities describes what features a query engine supports
type QueryCapabilities struct {
	SupportsNestedFields     bool
	SupportsArrayContains    bool
	SupportsArrayContainsAny bool
	SupportsCompositeFilters bool
	SupportsOrderBy          bool
	SupportsCursorPagination bool
	SupportsOffsetPagination bool
	SupportsProjection       bool
	MaxFilterCount           int
	MaxOrderByCount          int
	MaxNestingDepth          int
}

// QueryOptimizer defines interface for query optimization
type QueryOptimizer interface {
	// OptimizeQuery analyzes and optimizes a query for better performance
	OptimizeQuery(query model.Query) (model.Query, error)

	// SuggestIndexes suggests database indexes that would improve query performance
	SuggestIndexes(query model.Query) ([]IndexSuggestion, error)

	// EstimateQueryCost estimates the computational cost of executing a query
	EstimateQueryCost(query model.Query) (QueryCost, error)
}

// IndexSuggestion represents a suggested database index
type IndexSuggestion struct {
	Fields     []string      `json:"fields"`
	Type       IndexType     `json:"type"`
	Priority   IndexPriority `json:"priority"`
	Reason     string        `json:"reason"`
	Collection string        `json:"collection"`
}

// IndexType represents different types of database indexes
type IndexType string

const (
	IndexTypeSingle   IndexType = "single"
	IndexTypeCompound IndexType = "compound"
	IndexTypeText     IndexType = "text"
	IndexTypeGeo      IndexType = "geo"
	IndexTypeArray    IndexType = "array"
)

// IndexPriority represents the priority of creating an index
type IndexPriority string

const (
	IndexPriorityHigh   IndexPriority = "high"
	IndexPriorityMedium IndexPriority = "medium"
	IndexPriorityLow    IndexPriority = "low"
)

// QueryCost represents the estimated cost of executing a query
type QueryCost struct {
	DocumentsScanned   int64    `json:"documentsScanned"`
	IndexUsage         bool     `json:"indexUsage"`
	EstimatedTimeMS    int64    `json:"estimatedTimeMS"`
	ComplexityScore    float64  `json:"complexityScore"`
	RecommendedIndexes []string `json:"recommendedIndexes"`
}
