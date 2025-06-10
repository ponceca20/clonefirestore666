package firestore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFirestorePath_Valid(t *testing.T) {
	path := "projects/proj1/databases/db1/documents/col1/doc1"
	info, err := ParseFirestorePath(path)
	assert.NoError(t, err)
	assert.Equal(t, "proj1", info.ProjectID)
	assert.Equal(t, "db1", info.DatabaseID)
	assert.Equal(t, "col1/doc1", info.DocumentPath)
	assert.True(t, info.IsDocument)
	assert.False(t, info.IsCollection)
	assert.Equal(t, []string{"col1", "doc1"}, info.Segments)
}

func TestParseFirestorePath_InvalidFormat(t *testing.T) {
	_, err := ParseFirestorePath("invalid/path")
	assert.Error(t, err)
}

func TestIsValidID(t *testing.T) {
	assert.True(t, IsValidID("abc-123_X"))
	assert.False(t, IsValidID(""))
	assert.False(t, IsValidID("a@b"))
}

func TestIsDocumentPath_IsCollectionPath(t *testing.T) {
	assert.True(t, IsDocumentPath("col1/doc1"))
	assert.False(t, IsDocumentPath("col1"))
	assert.True(t, IsCollectionPath("col1"))
	assert.False(t, IsCollectionPath("col1/doc1"))
}

func TestBuildAndParseDocumentPath(t *testing.T) {
	segments := []string{"col1", "doc1", "col2", "doc2"}
	path := BuildDocumentPath(segments...)
	parsed := ParseDocumentPath(path)
	assert.Equal(t, segments, parsed)
}

func TestGetDocumentID_And_CollectionID(t *testing.T) {
	docID, err := GetDocumentID("col1/doc1")
	assert.NoError(t, err)
	assert.Equal(t, "doc1", docID)

	colID, err := GetCollectionID("col1/doc1")
	assert.NoError(t, err)
	assert.Equal(t, "col1", colID)
}

func TestValidateDocumentPath(t *testing.T) {
	assert.NoError(t, ValidateDocumentPath("col1/doc1"))
	assert.Error(t, ValidateDocumentPath("col1"))
}

func TestValidateCollectionPath(t *testing.T) {
	assert.NoError(t, ValidateCollectionPath("col1"))
	assert.Error(t, ValidateCollectionPath("col1/doc1"))
}

func TestPathParser_Compile(t *testing.T) {
	// Placeholder: Add real path parser tests here
}
