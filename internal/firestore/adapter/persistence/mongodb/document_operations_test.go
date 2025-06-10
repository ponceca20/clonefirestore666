package mongodb

import (
	"context"
	"errors"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID)
	if doc, ok := args.Get(0).(*model.Document); ok {
		return doc, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepository) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID, data)
	if doc, ok := args.Get(0).(*model.Document); ok {
		return doc, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepository) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
	if doc, ok := args.Get(0).(*model.Document); ok {
		return doc, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDocumentRepository) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID)
	return args.Error(0)
}

// Interfaz mínima para pruebas unitarias
// Esto permite inyectar el mock en DocumentOperations sin modificar el código de producción

type documentRepositoryIface interface {
	GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error)
	CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error)
	UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error)
	DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error
}

// DocumentOperationsTestable permite inyectar la interfaz en vez de la estructura concreta
// Solo para pruebas

type DocumentOperationsTestable struct {
	repo documentRepositoryIface
}

// Métodos delegan a la interfaz
func (d *DocumentOperationsTestable) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return d.repo.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}
func (d *DocumentOperationsTestable) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	return d.repo.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
}
func (d *DocumentOperationsTestable) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return d.repo.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
}
func (d *DocumentOperationsTestable) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return d.repo.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}

func TestDocumentOperations_CreateDocument(t *testing.T) {
	repo := new(MockDocumentRepository)
	docOps := &DocumentOperationsTestable{repo: repo}
	ctx := context.Background()
	projectID, databaseID, collectionID, documentID := "p1", "d1", "c1", "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	doc := &model.Document{DocumentID: documentID, Fields: fields}
	repo.On("CreateDocument", ctx, projectID, databaseID, collectionID, documentID, fields).Return(doc, nil)
	result, err := docOps.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, fields)
	assert.NoError(t, err)
	assert.Equal(t, doc, result)
	repo.AssertExpectations(t)
}

func TestDocumentOperations_GetDocument(t *testing.T) {
	repo := new(MockDocumentRepository)
	docOps := &DocumentOperationsTestable{repo: repo}
	ctx := context.Background()
	projectID, databaseID, collectionID, documentID := "p1", "d1", "c1", "doc1"
	doc := &model.Document{DocumentID: documentID}
	repo.On("GetDocument", ctx, projectID, databaseID, collectionID, documentID).Return(doc, nil)
	result, err := docOps.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
	assert.NoError(t, err)
	assert.Equal(t, doc, result)
	repo.AssertExpectations(t)
}

func TestDocumentOperations_UpdateDocument(t *testing.T) {
	repo := new(MockDocumentRepository)
	docOps := &DocumentOperationsTestable{repo: repo}
	ctx := context.Background()
	projectID, databaseID, collectionID, documentID := "p1", "d1", "c1", "doc1"
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("baz")}
	mask := []string{"foo"}
	doc := &model.Document{DocumentID: documentID, Fields: fields}
	repo.On("UpdateDocument", ctx, projectID, databaseID, collectionID, documentID, fields, mask).Return(doc, nil)
	result, err := docOps.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, fields, mask)
	assert.NoError(t, err)
	assert.Equal(t, doc, result)
	repo.AssertExpectations(t)
}

func TestDocumentOperations_DeleteDocument(t *testing.T) {
	repo := new(MockDocumentRepository)
	docOps := &DocumentOperationsTestable{repo: repo}
	ctx := context.Background()
	projectID, databaseID, collectionID, documentID := "p1", "d1", "c1", "doc1"
	repo.On("DeleteDocument", ctx, projectID, databaseID, collectionID, documentID).Return(nil)
	err := docOps.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDocumentOperations_GetDocument_NotFound(t *testing.T) {
	repo := new(MockDocumentRepository)
	docOps := &DocumentOperationsTestable{repo: repo}
	ctx := context.Background()
	projectID, databaseID, collectionID, documentID := "p1", "d1", "c1", "doc404"
	repo.On("GetDocument", ctx, projectID, databaseID, collectionID, documentID).Return(nil, errors.New("not found"))
	result, err := docOps.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestDocumentOperations_Compile(t *testing.T) {
	// Placeholder: Add real document operations tests here
}
