package service

import (
	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/errors"
)

// ProjectionService provides functional field projection capabilities
// Following functional programming principles for composable and testable field filtering
type ProjectionService interface {
	// ApplyProjection filters document fields based on the provided field list
	ApplyProjection(docs []*model.Document, fields []string) []*model.Document

	// ValidateProjectionFields validates that projection fields are valid field paths
	ValidateProjectionFields(fields []string) error

	// IsProjectionRequired returns true if projection should be applied
	IsProjectionRequired(fields []string) bool
}

// projectionService implements ProjectionService using functional programming principles
type projectionService struct{}

// NewProjectionService creates a new projection service
func NewProjectionService() ProjectionService {
	return &projectionService{}
}

// ApplyProjection implements field filtering using pure functions
func (s *projectionService) ApplyProjection(docs []*model.Document, fields []string) []*model.Document {
	if !s.IsProjectionRequired(fields) {
		return docs
	}

	return applyProjectionToDocuments(docs, normalizeFieldPaths(fields))
}

// ValidateProjectionFields validates projection field paths
func (s *projectionService) ValidateProjectionFields(fields []string) error {
	return validateFieldPaths(fields)
}

// IsProjectionRequired checks if projection should be applied
func (s *projectionService) IsProjectionRequired(fields []string) bool {
	return len(fields) > 0
}

// Pure functions for field projection logic

// applyProjectionToDocuments applies projection to a list of documents
func applyProjectionToDocuments(docs []*model.Document, fields []string) []*model.Document {
	if docs == nil {
		return nil
	}

	projected := make([]*model.Document, len(docs))
	for i, doc := range docs {
		projected[i] = applyProjectionToDocument(doc, fields)
	}
	return projected
}

// applyProjectionToDocument creates a new document with only the specified fields
func applyProjectionToDocument(doc *model.Document, fields []string) *model.Document {
	if doc == nil {
		return nil
	}

	// Create a new document with the same metadata but filtered fields
	projected := &model.Document{
		ID:                doc.ID,
		ProjectID:         doc.ProjectID,
		DatabaseID:        doc.DatabaseID,
		CollectionID:      doc.CollectionID,
		DocumentID:        doc.DocumentID,
		Path:              doc.Path,
		ParentPath:        doc.ParentPath,
		Fields:            filterFields(doc.Fields, fields),
		CreateTime:        doc.CreateTime,
		UpdateTime:        doc.UpdateTime,
		ReadTime:          doc.ReadTime,
		Version:           doc.Version,
		Exists:            doc.Exists,
		HasSubcollections: doc.HasSubcollections,
	}

	return projected
}

// filterFields filters document fields based on projection
func filterFields(originalFields map[string]*model.FieldValue, projectionFields []string) map[string]*model.FieldValue {
	if originalFields == nil {
		return nil
	}

	filtered := make(map[string]*model.FieldValue)

	for _, fieldPath := range projectionFields {
		if value, exists := getFieldByPath(originalFields, fieldPath); exists {
			setFieldByPath(filtered, fieldPath, value)
		}
	}

	return filtered
}

// getFieldByPath retrieves a field value by its path (supporting nested fields)
func getFieldByPath(fields map[string]*model.FieldValue, fieldPath string) (*model.FieldValue, bool) {
	// For simple fields, direct lookup
	if value, exists := fields[fieldPath]; exists {
		return value, true
	}

	// TODO: Support nested field paths like "address.city" in future versions
	// For now, we only support direct field names

	return nil, false
}

// setFieldByPath sets a field value by its path
func setFieldByPath(fields map[string]*model.FieldValue, fieldPath string, value *model.FieldValue) {
	// For simple fields, direct assignment
	fields[fieldPath] = value

	// TODO: Support nested field paths like "address.city" in future versions
}

// normalizeFieldPaths normalizes field paths for consistent processing
func normalizeFieldPaths(fields []string) []string {
	if fields == nil {
		return nil
	}

	// For now, return as-is. In the future, we could handle normalization
	// like trimming whitespace, converting to lowercase, etc.
	normalized := make([]string, len(fields))
	copy(normalized, fields)
	return normalized
}

// validateFieldPaths validates that field paths are valid
func validateFieldPaths(fields []string) error { // Basic validation - ensure no empty field paths
	for _, field := range fields {
		if field == "" {
			return errors.NewValidationError("field path cannot be empty")
		}
	}

	return nil
}
