package http

import (
	"encoding/json"
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// FirestoreStructuredQuery represents the exact JSON format that Google Firestore expects
type FirestoreStructuredQuery struct {
	From    []FirestoreCollectionSelector `json:"from,omitempty"`
	Where   *FirestoreFilter              `json:"where,omitempty"`
	OrderBy []FirestoreOrder              `json:"orderBy,omitempty"`
	Limit   int                           `json:"limit,omitempty"`
	Offset  int                           `json:"offset,omitempty"`
}

type FirestoreCollectionSelector struct {
	CollectionID   string `json:"collectionId"`
	AllDescendants bool   `json:"allDescendants,omitempty"`
}

type FirestoreFilter struct {
	FieldFilter     *FirestoreFieldFilter     `json:"fieldFilter,omitempty"`
	CompositeFilter *FirestoreCompositeFilter `json:"compositeFilter,omitempty"`
}

type FirestoreFieldFilter struct {
	Field FirestoreFieldReference `json:"field"`
	Op    string                  `json:"op"`
	Value interface{}             `json:"value"`
}

type FirestoreCompositeFilter struct {
	Op      string            `json:"op"`
	Filters []FirestoreFilter `json:"filters"`
}

type FirestoreFieldReference struct {
	FieldPath string `json:"fieldPath"`
}

type FirestoreOrder struct {
	Field     FirestoreFieldReference `json:"field"`
	Direction string                  `json:"direction"`
}

// convertFirestoreValue extracts the actual value from Firestore typed values
func convertFirestoreValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Check if value is a map (Firestore typed value)
	if valueMap, ok := value.(map[string]interface{}); ok {
		// Handle different Firestore value types
		if boolVal, exists := valueMap["booleanValue"]; exists {
			return boolVal
		}
		if intVal, exists := valueMap["integerValue"]; exists {
			// Firestore sends integers as strings, convert back
			if strVal, ok := intVal.(string); ok {
				// Try parsing as int64 first, then float64
				if i, err := fmt.Sscanf(strVal, "%d", new(int64)); err == nil && i == 1 {
					var result int64
					fmt.Sscanf(strVal, "%d", &result)
					return result
				}
			}
			return intVal
		}
		if doubleVal, exists := valueMap["doubleValue"]; exists {
			return doubleVal
		}
		if strVal, exists := valueMap["stringValue"]; exists {
			return strVal
		}
		if timestampVal, exists := valueMap["timestampValue"]; exists {
			// Handle timestamp values - can be string or actual time value
			if timestampStr, ok := timestampVal.(string); ok {
				// Try to parse the timestamp string
				if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
					return t
				} else if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
					return t
				}
				// If parsing fails, return as string
				return timestampStr
			}
			return timestampVal
		}
		if bytesVal, exists := valueMap["bytesValue"]; exists {
			return bytesVal
		}
		if refVal, exists := valueMap["referenceValue"]; exists {
			return refVal
		}
		if geoVal, exists := valueMap["geoPointValue"]; exists {
			return geoVal
		}
		if arrayVal, exists := valueMap["arrayValue"]; exists {
			// Handle array values
			if arrayMap, ok := arrayVal.(map[string]interface{}); ok {
				if values, exists := arrayMap["values"]; exists {
					if valuesSlice, ok := values.([]interface{}); ok {
						// Recursively convert array elements
						convertedValues := make([]interface{}, len(valuesSlice))
						for i, v := range valuesSlice {
							convertedValues[i] = convertFirestoreValue(v)
						}
						return convertedValues
					}
				}
			}
			return arrayVal
		}
		if mapVal, exists := valueMap["mapValue"]; exists {
			// Handle map values
			if mapFields, ok := mapVal.(map[string]interface{}); ok {
				if fields, exists := mapFields["fields"]; exists {
					if fieldsMap, ok := fields.(map[string]interface{}); ok {
						// Recursively convert map fields
						convertedMap := make(map[string]interface{})
						for key, val := range fieldsMap {
							convertedMap[key] = convertFirestoreValue(val)
						}
						return convertedMap
					}
				}
			}
			return mapVal
		}
		if nullVal, exists := valueMap["nullValue"]; exists {
			_ = nullVal // nullValue is typically "NULL_VALUE"
			return nil
		}
	}

	// If it's not a Firestore typed value, return as is
	return value
}

// mapFirestoreOperator converts Firestore operator strings to internal operator types
// following Firestore API standards and ensuring compatibility with Google Firestore semantics
func mapFirestoreOperator(firestoreOp string) model.Operator {
	switch firestoreOp {
	case "EQUAL":
		return model.OperatorEqual
	case "NOT_EQUAL":
		return model.OperatorNotEqual
	case "LESS_THAN":
		return model.OperatorLessThan
	case "LESS_THAN_OR_EQUAL":
		return model.OperatorLessThanOrEqual
	case "GREATER_THAN":
		return model.OperatorGreaterThan
	case "GREATER_THAN_OR_EQUAL":
		return model.OperatorGreaterThanOrEqual
	case "IN":
		return model.OperatorIn
	case "NOT_IN":
		return model.OperatorNotIn
	case "ARRAY_CONTAINS":
		return model.OperatorArrayContains
	case "ARRAY_CONTAINS_ANY":
		return model.OperatorArrayContainsAny
	default:
		// Return the input string cast to Operator for unknown operators
		// This maintains backward compatibility while allowing for extensibility
		return model.Operator(firestoreOp)
	}
}

// convertFirestoreFilter converts a Firestore filter (simple or composite) to model filters
func convertFirestoreFilter(filter FirestoreFilter) ([]model.Filter, error) {
	var filters []model.Filter

	if filter.FieldFilter != nil { // Handle simple field filter
		ff := filter.FieldFilter

		// Convert Firestore operator to internal operator using extracted function
		op := mapFirestoreOperator(ff.Op)

		// Validate that we support this operator
		if string(op) == ff.Op && ff.Op != "EQUAL" && ff.Op != "NOT_EQUAL" &&
			ff.Op != "LESS_THAN" && ff.Op != "LESS_THAN_OR_EQUAL" &&
			ff.Op != "GREATER_THAN" && ff.Op != "GREATER_THAN_OR_EQUAL" &&
			ff.Op != "IN" && ff.Op != "NOT_IN" &&
			ff.Op != "ARRAY_CONTAINS" && ff.Op != "ARRAY_CONTAINS_ANY" {
			return nil, fmt.Errorf("unsupported operator: %s", ff.Op)
		}

		// Convert Firestore typed value to actual value
		convertedValue := convertFirestoreValue(ff.Value)

		filter := model.Filter{
			Field:    ff.Field.FieldPath,
			Operator: op,
			Value:    convertedValue,
		}
		filters = append(filters, filter)

	} else if filter.CompositeFilter != nil {
		// Handle composite filter (AND/OR)
		cf := filter.CompositeFilter

		// Check if there are no filters
		if len(cf.Filters) == 0 {
			return nil, fmt.Errorf("composite filter must have at least one sub-filter")
		}
		if cf.Op == "AND" {
			// For AND operations, flatten sub-filters into separate individual filters
			// This is the correct Firestore behavior - AND filters should be flattened
			for _, subFilter := range cf.Filters {
				convertedSubFilters, err := convertFirestoreFilter(subFilter)
				if err != nil {
					return nil, fmt.Errorf("failed to convert AND sub-filter: %w", err)
				}
				filters = append(filters, convertedSubFilters...)
			}
		} else if cf.Op == "OR" {
			// For OR operations, we create a composite filter with sub-filters
			var subFilters []model.Filter
			for _, subFilter := range cf.Filters {
				convertedSubFilters, err := convertFirestoreFilter(subFilter)
				if err != nil {
					return nil, fmt.Errorf("failed to convert OR sub-filter: %w", err)
				}
				subFilters = append(subFilters, convertedSubFilters...)
			}

			// Create a composite OR filter
			orFilter := model.Filter{
				Composite:  "or",
				SubFilters: subFilters,
			}
			filters = append(filters, orFilter)
		} else {
			return nil, fmt.Errorf("unsupported composite filter operator: %s", cf.Op)
		}
	} else {
		return nil, fmt.Errorf("filter must have either fieldFilter or compositeFilter")
	}

	return filters, nil
}

// convertFirestoreJSONToModelQuery converts the Firestore JSON format to our internal model
func convertFirestoreJSONToModelQuery(firestoreQuery FirestoreStructuredQuery) (*model.Query, error) {
	query := &model.Query{}

	// Handle From clause
	if len(firestoreQuery.From) > 0 {
		query.CollectionID = firestoreQuery.From[0].CollectionID
		// Handle collection group queries
		if firestoreQuery.From[0].AllDescendants {
			// For collection group queries, mark it appropriately
			// This might need adjustment based on your model.Query structure
		}
	}
	// Handle Where clause
	if firestoreQuery.Where != nil {
		filters, err := convertFirestoreFilter(*firestoreQuery.Where)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filters: %w", err)
		}
		query.Filters = filters
	}

	// Handle OrderBy clause
	for _, order := range firestoreQuery.OrderBy {
		var direction model.Direction
		switch order.Direction {
		case "ASCENDING":
			direction = model.DirectionAscending
		case "DESCENDING":
			direction = model.DirectionDescending
		default:
			direction = model.DirectionAscending // Default
		}
		query.AddOrder(order.Field.FieldPath, direction)
	}

	// Handle Limit
	if firestoreQuery.Limit > 0 {
		query.SetLimit(firestoreQuery.Limit)
	}

	// Handle Offset
	if firestoreQuery.Offset > 0 {
		query.SetOffset(firestoreQuery.Offset)
	}

	return query, nil
}

// Document handlers implementation following single responsibility principle
func (h *HTTPHandler) CreateDocument(c *fiber.Ctx) error {
	h.Log.Debug("Creating document via HTTP", "collection", c.Params("collectionID"))

	var req usecase.CreateDocumentRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")
	req.CollectionID = c.Params("collectionID")
	req.DocumentID = c.Query("documentId") // Optional from query params
	// Parse request body
	if err := c.BodyParser(&req.Data); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	// Validate required fields - check if data is nil or empty map
	if req.Data == nil || len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_data",
			"message": "Document data is required",
		})
	}

	// Call usecase
	document, err := h.FirestoreUC.CreateDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to create document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document created successfully", "documentID", document.DocumentID)
	return c.Status(fiber.StatusCreated).JSON(document)
}

func (h *HTTPHandler) GetDocument(c *fiber.Ctx) error {
	h.Log.Debug("Getting document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	req := usecase.GetDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
	}

	document, err := h.FirestoreUC.GetDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to get document", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "document_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(document)
}

func (h *HTTPHandler) UpdateDocument(c *fiber.Ctx) error {
	h.Log.Debug("Updating document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	var reqData map[string]any
	if err := c.BodyParser(&reqData); err != nil {
		h.Log.Error("Failed to parse request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request_body",
			"message": "Failed to parse request body",
		})
	}

	req := usecase.UpdateDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
		Data:         reqData,
		Mask:         parseUpdateMaskQuery(c), // Parse update mask from query as []string
	}

	document, err := h.FirestoreUC.UpdateDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to update document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document updated successfully", "documentID", document.DocumentID)
	return c.JSON(document)
}

func (h *HTTPHandler) DeleteDocument(c *fiber.Ctx) error {
	h.Log.Debug("Deleting document via HTTP",
		"collection", c.Params("collectionID"),
		"document", c.Params("documentID"))

	req := usecase.DeleteDocumentRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
		DocumentID:   c.Params("documentID"),
	}

	err := h.FirestoreUC.DeleteDocument(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to delete document", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_document_failed",
			"message": err.Error(),
		})
	}

	h.Log.Info("Document deleted successfully", "documentID", req.DocumentID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandler) QueryDocuments(c *fiber.Ctx) error {
	var req usecase.QueryRequest

	// Parse path parameters
	req.ProjectID = c.Params("projectID")
	req.DatabaseID = c.Params("databaseID")
	collectionTarget := c.Params("collectionID") // Collection from URL path

	// Set parent path for RunQuery (should point to documents level + collection)
	req.Parent = "projects/" + req.ProjectID + "/databases/" + req.DatabaseID + "/documents/" + collectionTarget
	// Parse the Firestore JSON structured query from request body
	var firestoreQuery FirestoreStructuredQuery

	// First, try to parse as raw JSON to handle different formats
	var rawBody map[string]interface{}
	if err := c.BodyParser(&rawBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_json",
			"message": "Failed to parse request body: " + err.Error(),
		})
	}

	// Check if the request has a compositeFilter or fieldFilter at root level
	// This handles Postman-style queries that don't wrap in "where"
	if compositeFilter, exists := rawBody["compositeFilter"]; exists {
		firestoreQuery.Where = &FirestoreFilter{
			CompositeFilter: &FirestoreCompositeFilter{},
		}
		// Convert the raw data back to proper structure using JSON marshaling
		compositeBytes, err := json.Marshal(compositeFilter)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_composite_filter",
				"message": "Failed to marshal compositeFilter: " + err.Error(),
			})
		}
		if err := json.Unmarshal(compositeBytes, firestoreQuery.Where.CompositeFilter); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_composite_filter",
				"message": "Failed to unmarshal compositeFilter: " + err.Error(),
			})
		}
	} else if fieldFilter, exists := rawBody["fieldFilter"]; exists {
		firestoreQuery.Where = &FirestoreFilter{
			FieldFilter: &FirestoreFieldFilter{},
		}
		// Convert the raw data back to proper structure using JSON marshaling
		fieldBytes, err := json.Marshal(fieldFilter)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_field_filter",
				"message": "Failed to marshal fieldFilter: " + err.Error(),
			})
		}
		if err := json.Unmarshal(fieldBytes, firestoreQuery.Where.FieldFilter); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_field_filter",
				"message": "Failed to unmarshal fieldFilter: " + err.Error(),
			})
		}
	} else {
		// Try normal parsing for properly structured queries
		if err := c.BodyParser(&firestoreQuery); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_structured_query",
				"message": "Failed to parse Firestore structured query: " + err.Error(),
			})
		}
	}

	// Convert Firestore JSON to internal model.Query
	query, err := convertFirestoreJSONToModelQuery(firestoreQuery)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_query_format",
			"message": "Failed to convert Firestore query: " + err.Error()})
	}

	// Set the collection info in the query
	query.CollectionID = collectionTarget
	query.Path = req.Parent

	// Assign the structured query
	req.StructuredQuery = query

	// Use RunQuery for Firestore structured queries (not QueryDocuments)
	documents, err := h.FirestoreUC.RunQuery(c.UserContext(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "query_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

// ListDocuments lists all documents in a collection with pagination
func (h *HTTPHandler) ListDocuments(c *fiber.Ctx) error {
	h.Log.Debug("Listing documents via HTTP", "collection", c.Params("collectionID"))

	req := usecase.ListDocumentsRequest{
		ProjectID:    c.Params("projectID"),
		DatabaseID:   c.Params("databaseID"),
		CollectionID: c.Params("collectionID"),
	}

	// Parse optional query parameters
	if pageSize := c.QueryInt("pageSize"); pageSize > 0 {
		req.PageSize = int32(pageSize)
	}
	req.PageToken = c.Query("pageToken")
	req.OrderBy = c.Query("orderBy")
	req.ShowMissing = c.QueryBool("showMissing")

	documents, err := h.FirestoreUC.ListDocuments(c.UserContext(), req)
	if err != nil {
		h.Log.Error("Failed to list documents", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_documents_failed",
			"message": err.Error(),
		})
	}

	h.Log.Debug("Documents listed successfully", "count", len(documents))
	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

// Helper to parse updateMask query param as []string (comma-separated)
func parseUpdateMaskQuery(c *fiber.Ctx) []string {
	maskParam := c.Query("updateMask")
	if maskParam == "" {
		return nil
	}
	// Firestore API expects comma-separated field paths
	fields := strings.Split(maskParam, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}
