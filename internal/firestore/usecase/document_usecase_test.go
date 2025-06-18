package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFirestoreDocumentUsecase crea un caso de uso con mocks centralizados y logger dummy para tests de documentos.
func newTestFirestoreDocumentUsecase() usecase.FirestoreUsecaseInterface {
	return usecase.NewFirestoreUsecase(
		usecase.NewMockFirestoreRepo(), // Mock repo hexagonal
		nil,                            // Security repo (no necesario aquí)
		nil,                            // Query engine (no necesario aquí)
		nil,                            // Projection service (no necesario aquí)
		&usecase.MockLogger{},          // Logger dummy
	)
}

func TestCreateDocument(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	data := map[string]interface{}{"foo": "bar"}
	resp, err := firestoreUC.CreateDocument(ctx, usecase.CreateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         data,
	})
	require.NoError(t, err, "CreateDocument debe ejecutarse sin error")
	assert.Equal(t, "doc1", resp.DocumentID, "El ID del documento debe coincidir")
}

func TestGetDocument(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	resp, err := firestoreUC.GetDocument(ctx, usecase.GetDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	require.NoError(t, err, "GetDocument debe ejecutarse sin error")
	assert.Equal(t, "doc1", resp.DocumentID, "El ID del documento debe coincidir")
	assert.Contains(t, resp.Fields, "count", "El mock debe devolver el campo 'count'")
}

func TestUpdateDocument(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	resp, err := firestoreUC.UpdateDocument(ctx, usecase.UpdateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         map[string]interface{}{"foo": "baz"},
		Mask:         []string{"foo"},
	})
	require.NoError(t, err, "UpdateDocument debe ejecutarse sin error")
	assert.Equal(t, "doc1", resp.DocumentID, "El ID del documento debe coincidir")
}

func TestDeleteDocument(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	err := firestoreUC.DeleteDocument(ctx, usecase.DeleteDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	})
	assert.NoError(t, err, "DeleteDocument debe ejecutarse sin error")
}

func TestListDocuments(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	docs, err := firestoreUC.ListDocuments(ctx, usecase.ListDocumentsRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
	})
	require.NoError(t, err, "ListDocuments debe ejecutarse sin error")
	assert.Len(t, docs, 1, "Debe retornar un documento")
	assert.Equal(t, "doc1", docs[0].DocumentID, "El ID del documento debe coincidir")
}

func TestQueryDocuments(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	// El mock no implementa lógica real de queries, pero se prueba la integración
	_, err := firestoreUC.QueryDocuments(ctx, usecase.QueryRequest{
		ProjectID:       "p1",
		DatabaseID:      "d1",
		Parent:          "projects/p1/databases/d1/documents/c1",
		StructuredQuery: nil, // nil para simular error
	})
	assert.Error(t, err, "Debe fallar si el query es nil")
	// Ahora con un query válido (el mock lo acepta)
	resp, err := firestoreUC.QueryDocuments(ctx, usecase.QueryRequest{
		ProjectID:       "p1",
		DatabaseID:      "d1",
		Parent:          "projects/p1/databases/d1/documents/c1",
		StructuredQuery: &model.Query{}, // Query válido
	})
	assert.NoError(t, err, "QueryDocuments debe ejecutarse sin error con query válido")
	assert.NotNil(t, resp, "La respuesta no debe ser nil")
}

func TestRunQuery(t *testing.T) {
	firestoreUC := newTestFirestoreDocumentUsecase()
	ctx := context.Background()
	resp, err := firestoreUC.RunQuery(ctx, usecase.QueryRequest{
		ProjectID:       "p1",
		DatabaseID:      "d1",
		Parent:          "projects/p1/databases/d1/documents/c1",
		StructuredQuery: &model.Query{},
	})
	assert.NoError(t, err, "RunQuery debe ejecutarse sin error")
	assert.NotNil(t, resp, "La respuesta no debe ser nil")
}
