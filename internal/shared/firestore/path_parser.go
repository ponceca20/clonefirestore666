package firestore

import (
	"fmt"
	"regexp"
	"strings"

	"firestore-clone/internal/shared/errors"
)

// PathInfo represents parsed Firestore path information
type PathInfo struct {
	ProjectID    string
	DatabaseID   string
	DocumentPath string
	IsDocument   bool
	IsCollection bool
	Segments     []string
}

// CollectionInfo represents a collection in a path
type CollectionInfo struct {
	ID   string
	Path string
}

// DocumentInfo represents a document in a path
type DocumentInfo struct {
	ID   string
	Path string
}

var (
	// Firestore path pattern: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{DOCUMENT_PATH}
	firestorePathRegex = regexp.MustCompile(`^projects/([^/]+)/databases/([^/]+)/documents/(.*)$`)

	// Valid ID pattern (alphanumeric, hyphens, underscores)
	validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ParseFirestorePath parses a complete Firestore path
func ParseFirestorePath(path string) (*PathInfo, error) {
	if path == "" {
		return nil, errors.NewValidationError("path cannot be empty")
	}

	// Remove leading/trailing slashes and normalize
	path = strings.Trim(path, "/")

	matches := firestorePathRegex.FindStringSubmatch(path)
	if len(matches) != 4 {
		return nil, errors.NewValidationError("invalid Firestore path format").
			WithDetail("expected_format", "projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{DOCUMENT_PATH}").
			WithDetail("provided_path", path)
	}

	projectID := matches[1]
	databaseID := matches[2]
	documentPath := matches[3]

	// Validate project ID
	if !IsValidID(projectID) {
		return nil, errors.NewValidationError("invalid project ID").
			WithDetail("project_id", projectID)
	}

	// Validate database ID
	if !IsValidID(databaseID) {
		return nil, errors.NewValidationError("invalid database ID").
			WithDetail("database_id", databaseID)
	}

	// Parse document path segments
	segments := ParseDocumentPath(documentPath)
	if len(segments) == 0 {
		return nil, errors.NewValidationError("document path cannot be empty")
	}

	// Validate all segments
	for i, segment := range segments {
		if !IsValidID(segment) {
			return nil, errors.NewValidationError("invalid path segment").
				WithDetail("segment", segment).
				WithDetail("position", i)
		}
	}

	isDocument := len(segments)%2 == 0
	isCollection := len(segments)%2 == 1

	return &PathInfo{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		DocumentPath: documentPath,
		IsDocument:   isDocument,
		IsCollection: isCollection,
		Segments:     segments,
	}, nil
}

// ParseDocumentPath parses just the document path part (after /documents/)
func ParseDocumentPath(documentPath string) []string {
	if documentPath == "" {
		return []string{}
	}

	// Split by / and filter out empty segments
	segments := strings.Split(documentPath, "/")
	var result []string
	for _, segment := range segments {
		if segment != "" {
			result = append(result, segment)
		}
	}

	return result
}

// BuildFirestorePath constructs a Firestore path from components
func BuildFirestorePath(projectID, databaseID, documentPath string) string {
	return fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, documentPath)
}

// BuildDocumentPath constructs a document path from segments
func BuildDocumentPath(segments ...string) string {
	return strings.Join(segments, "/")
}

// GetCollectionPath returns the collection path for a document
func GetCollectionPath(documentPath string) (string, error) {
	segments := ParseDocumentPath(documentPath)
	if len(segments) == 0 {
		return "", errors.NewValidationError("empty document path")
	}

	if len(segments)%2 == 1 {
		// Already a collection path
		return documentPath, nil
	}

	// Remove the last segment (document ID) to get collection path
	collectionSegments := segments[:len(segments)-1]
	return BuildDocumentPath(collectionSegments...), nil
}

// GetParentPath returns the parent path (collection for document, parent collection for subcollection)
func GetParentPath(path string) (string, error) {
	segments := ParseDocumentPath(path)
	if len(segments) <= 1 {
		return "", errors.NewValidationError("path has no parent")
	}

	parentSegments := segments[:len(segments)-1]
	return BuildDocumentPath(parentSegments...), nil
}

// GetDocumentID returns the document ID from a document path
func GetDocumentID(documentPath string) (string, error) {
	segments := ParseDocumentPath(documentPath)
	if len(segments) == 0 {
		return "", errors.NewValidationError("empty document path")
	}

	if len(segments)%2 == 1 {
		return "", errors.NewValidationError("path is a collection, not a document")
	}

	return segments[len(segments)-1], nil
}

// GetCollectionID returns the collection ID from a path
func GetCollectionID(path string) (string, error) {
	segments := ParseDocumentPath(path)
	if len(segments) == 0 {
		return "", errors.NewValidationError("empty path")
	}

	if len(segments)%2 == 0 {
		// Document path - get the collection ID
		if len(segments) < 2 {
			return "", errors.NewValidationError("invalid document path")
		}
		return segments[len(segments)-2], nil
	}

	// Collection path - get the collection ID
	return segments[len(segments)-1], nil
}

// SplitIntoCollectionsAndDocuments splits a path into alternating collections and documents
func SplitIntoCollectionsAndDocuments(path string) ([]CollectionInfo, []DocumentInfo, error) {
	segments := ParseDocumentPath(path)
	if len(segments) == 0 {
		return nil, nil, errors.NewValidationError("empty path")
	}

	var collections []CollectionInfo
	var documents []DocumentInfo

	currentPath := ""
	for i, segment := range segments {
		if i > 0 {
			currentPath += "/"
		}
		currentPath += segment

		if i%2 == 0 {
			// Collection
			collections = append(collections, CollectionInfo{
				ID:   segment,
				Path: currentPath,
			})
		} else {
			// Document
			documents = append(documents, DocumentInfo{
				ID:   segment,
				Path: currentPath,
			})
		}
	}

	return collections, documents, nil
}

// IsValidID checks if an ID is valid for Firestore
func IsValidID(id string) bool {
	if id == "" {
		return false
	}

	// Check length (Firestore has limits)
	if len(id) > 1500 {
		return false
	}

	// Check for valid characters
	return validIDPattern.MatchString(id)
}

// IsDocumentPath checks if a path represents a document
func IsDocumentPath(path string) bool {
	segments := ParseDocumentPath(path)
	return len(segments) > 0 && len(segments)%2 == 0
}

// IsCollectionPath checks if a path represents a collection
func IsCollectionPath(path string) bool {
	segments := ParseDocumentPath(path)
	return len(segments) > 0 && len(segments)%2 == 1
}

// ValidateDocumentPath validates a document path
func ValidateDocumentPath(path string) error {
	if path == "" {
		return errors.NewValidationError("document path cannot be empty")
	}

	segments := ParseDocumentPath(path)
	if len(segments) == 0 {
		return errors.NewValidationError("document path cannot be empty")
	}

	if len(segments)%2 != 0 {
		return errors.NewValidationError("invalid document path: must have even number of segments")
	}

	for i, segment := range segments {
		if !IsValidID(segment) {
			return errors.NewValidationError("invalid segment in document path").
				WithDetail("segment", segment).
				WithDetail("position", i)
		}
	}

	return nil
}

// ValidateCollectionPath validates a collection path
func ValidateCollectionPath(path string) error {
	if path == "" {
		return errors.NewValidationError("collection path cannot be empty")
	}

	segments := ParseDocumentPath(path)
	if len(segments) == 0 {
		return errors.NewValidationError("collection path cannot be empty")
	}

	if len(segments)%2 != 1 {
		return errors.NewValidationError("invalid collection path: must have odd number of segments")
	}

	for i, segment := range segments {
		if !IsValidID(segment) {
			return errors.NewValidationError("invalid segment in collection path").
				WithDetail("segment", segment).
				WithDetail("position", i)
		}
	}

	return nil
}

// JoinPaths joins multiple path segments
func JoinPaths(segments ...string) string {
	var validSegments []string
	for _, segment := range segments {
		if segment != "" {
			validSegments = append(validSegments, strings.Trim(segment, "/"))
		}
	}
	return strings.Join(validSegments, "/")
}

// AppendToPath appends a segment to an existing path
func AppendToPath(basePath, segment string) string {
	if basePath == "" {
		return segment
	}
	return basePath + "/" + segment
}
