package http

import (
	"fmt"
	"log"

	"firestore-clone/internal/firestore/usecase"

	"github.com/gofiber/fiber/v2"
)

// FirestoreAggregationWrapper represents the JSON structure for aggregation queries
type FirestoreAggregationWrapper struct {
	StructuredAggregationQuery *FirestoreStructuredAggregationQuery `json:"structuredAggregationQuery,omitempty"`
}

// FirestoreStructuredAggregationQuery represents the structured aggregation query in Firestore format
type FirestoreStructuredAggregationQuery struct {
	StructuredQuery *FirestoreStructuredQuery `json:"structuredQuery,omitempty"`
	GroupBy         []FirestoreGroupByField   `json:"groupBy,omitempty"`
	Aggregations    []FirestoreAggregation    `json:"aggregations"`
}

// FirestoreGroupByField represents a field to group by
type FirestoreGroupByField struct {
	FieldPath string `json:"fieldPath"`
}

// FirestoreAggregation represents an aggregation operation in Firestore format
type FirestoreAggregation struct {
	Alias string                     `json:"alias"`
	Count *FirestoreCountAggregation `json:"count,omitempty"`
	Sum   *FirestoreFieldAggregation `json:"sum,omitempty"`
	Avg   *FirestoreFieldAggregation `json:"avg,omitempty"`
	Min   *FirestoreFieldAggregation `json:"min,omitempty"` // [EXTENSIÓN]
	Max   *FirestoreFieldAggregation `json:"max,omitempty"` // [EXTENSIÓN]
}

// FirestoreCountAggregation represents a count aggregation
type FirestoreCountAggregation struct {
	// Empty struct for count aggregation - this is how Firestore defines it
}

// FirestoreFieldAggregation represents an aggregation that operates on a field
type FirestoreFieldAggregation struct {
	Field FirestoreFieldReference `json:"field"`
}

// RunAggregationQuery handles aggregation queries following Firestore API
func (h *HTTPHandler) RunAggregationQuery(c *fiber.Ctx) error {
	log.Printf("[DEBUG RunAggregationQuery] Handler called - Path: %s, Method: %s", c.Path(), c.Method())

	var req usecase.AggregationQueryRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")

	// Parse the Firestore JSON structured aggregation query from request body
	var firestoreAggWrapper FirestoreAggregationWrapper

	// Parse the request body as a wrapper containing structuredAggregationQuery
	if err := c.BodyParser(&firestoreAggWrapper); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_json",
			"message": "Failed to parse request body: " + err.Error(),
		})
	}

	// Validate that structuredAggregationQuery is present
	if firestoreAggWrapper.StructuredAggregationQuery == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_structured_aggregation_query",
			"message": "Request must contain a 'structuredAggregationQuery' field",
		})
	}

	firestoreAggQuery := firestoreAggWrapper.StructuredAggregationQuery

	// Validate that aggregations array is present and not empty
	if len(firestoreAggQuery.Aggregations) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_aggregations",
			"message": "structuredAggregationQuery must contain at least one aggregation",
		})
	}

	// Set parent path for RunAggregationQuery (should point to documents level only, per Firestore API)
	req.Parent = "projects/" + req.ProjectID + "/databases/" + req.DatabaseID + "/documents"

	// Convert Firestore JSON to internal usecase types
	log.Printf("[DEBUG RunAggregationQuery] About to convert firestoreAggQuery to usecase.StructuredAggregationQuery")

	structuredAggQuery, err := h.convertFirestoreAggregationToUsecase(*firestoreAggQuery)
	if err != nil {
		log.Printf("[ERROR RunAggregationQuery] Failed to convert Firestore aggregation query: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_aggregation_format",
			"message": "Failed to convert Firestore aggregation query: " + err.Error(),
		})
	}

	log.Printf("[DEBUG RunAggregationQuery] Successfully converted to usecase.StructuredAggregationQuery. Aggregations count: %d", len(structuredAggQuery.Aggregations))

	// Assign the structured aggregation query
	req.StructuredAggregationQuery = structuredAggQuery

	// Execute the aggregation query
	response, err := h.FirestoreUC.RunAggregationQuery(c.UserContext(), req)
	if err != nil {
		log.Printf("[ERROR RunAggregationQuery] Failed to execute aggregation query: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "aggregation_failed",
			"message": err.Error(),
		})
	}

	// Return results in Firestore-compatible format
	// The response is already in the correct format as per AggregationQueryResponse
	log.Printf("[DEBUG RunAggregationQuery] Returning %d aggregation results", len(response.Results))
	return c.JSON(response.Results) // Return the results array directly to match Firestore API
}

// convertFirestoreAggregationToUsecase converts Firestore aggregation JSON to internal usecase types
func (h *HTTPHandler) convertFirestoreAggregationToUsecase(firestoreAgg FirestoreStructuredAggregationQuery) (*usecase.StructuredAggregationQuery, error) {
	result := &usecase.StructuredAggregationQuery{
		Aggregations: make([]usecase.AggregationFunction, 0, len(firestoreAgg.Aggregations)),
	}
	// Convert structured query if present
	if firestoreAgg.StructuredQuery != nil {
		modelQuery, err := convertFirestoreJSONToModelQuery(*firestoreAgg.StructuredQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to convert structured query: %w", err)
		}
		result.StructuredQuery = modelQuery
	}

	// Convert groupBy fields if present
	if len(firestoreAgg.GroupBy) > 0 {
		result.GroupBy = make([]usecase.GroupByField, 0, len(firestoreAgg.GroupBy))
		for _, groupBy := range firestoreAgg.GroupBy {
			result.GroupBy = append(result.GroupBy, usecase.GroupByField{
				FieldPath: groupBy.FieldPath,
			})
		}
	}

	// Convert aggregations
	for _, agg := range firestoreAgg.Aggregations {
		usecaseAgg := usecase.AggregationFunction{
			Alias: agg.Alias,
		}

		// Check which type of aggregation this is
		aggregationCount := 0
		if agg.Count != nil {
			aggregationCount++
			usecaseAgg.Count = &usecase.CountAggregation{}
		}
		if agg.Sum != nil {
			aggregationCount++
			usecaseAgg.Sum = &usecase.FieldAggregation{
				Field: usecase.FieldReference{
					FieldPath: agg.Sum.Field.FieldPath,
				},
			}
		}
		if agg.Avg != nil {
			aggregationCount++
			usecaseAgg.Avg = &usecase.FieldAggregation{
				Field: usecase.FieldReference{
					FieldPath: agg.Avg.Field.FieldPath,
				},
			}
		}
		if agg.Min != nil {
			aggregationCount++
			usecaseAgg.Min = &usecase.FieldAggregation{
				Field: usecase.FieldReference{
					FieldPath: agg.Min.Field.FieldPath,
				},
			}
		}
		if agg.Max != nil {
			aggregationCount++
			usecaseAgg.Max = &usecase.FieldAggregation{
				Field: usecase.FieldReference{
					FieldPath: agg.Max.Field.FieldPath,
				},
			}
		}

		// Validate that exactly one aggregation type is specified
		if aggregationCount != 1 {
			return nil, fmt.Errorf("aggregation '%s' must have exactly one aggregation type (found %d)", agg.Alias, aggregationCount)
		}

		result.Aggregations = append(result.Aggregations, usecaseAgg)
	}

	return result, nil
}
