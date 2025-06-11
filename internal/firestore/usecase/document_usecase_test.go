package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDocument(t *testing.T) {
	uc := newTestFirestoreUsecase()
	doc, err := uc.CreateDocument(context.Background(), usecase.CreateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         map[string]interface{}{"foo": "bar"},
	})
	require.NoError(t, err)
	assert.Equal(t, "doc1", doc.DocumentID)
}

func TestGetDocument(t *testing.T) {
	uc := newTestFirestoreUsecase()
	doc, err := uc.GetDocument(context.Background(), usecase.GetDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	require.NoError(t, err)
	assert.Equal(t, "doc1", doc.DocumentID)
}

func TestUpdateDocument(t *testing.T) {
	uc := newTestFirestoreUsecase()
	doc, err := uc.UpdateDocument(context.Background(), usecase.UpdateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         map[string]interface{}{"foo": "baz"},
		Mask:         []string{"foo"},
	})
	require.NoError(t, err)
	assert.Equal(t, "doc1", doc.DocumentID)
}

func TestDeleteDocument(t *testing.T) {
	uc := newTestFirestoreUsecase()
	err := uc.DeleteDocument(context.Background(), usecase.DeleteDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	assert.NoError(t, err)
}

func TestListDocuments(t *testing.T) {
	uc := newTestFirestoreUsecase()
	docs, err := uc.ListDocuments(context.Background(), usecase.ListDocumentsRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].DocumentID)
}
