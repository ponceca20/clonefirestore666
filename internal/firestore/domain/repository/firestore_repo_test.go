package repository

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"

	"github.com/stretchr/testify/assert"
)

// mockFirestoreRepository is a minimal mock for interface compliance
// Only implements one method for demonstration
type mockFirestoreRepository struct{}

func (m *mockFirestoreRepository) CreateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockFirestoreRepository) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ProjectID: projectID}, nil
}
func (m *mockFirestoreRepository) UpdateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockFirestoreRepository) DeleteProject(ctx context.Context, projectID string) error {
	return nil
}
func (m *mockFirestoreRepository) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *mockFirestoreRepository) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *mockFirestoreRepository) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return nil
}
func (m *mockFirestoreRepository) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *mockFirestoreRepository) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *mockFirestoreRepository) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return nil
}
func (m *mockFirestoreRepository) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (m *mockFirestoreRepository) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) DeleteDocumentByPath(ctx context.Context, path string) error {
	return nil
}
func (m *mockFirestoreRepository) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return nil, "", nil
}
func (m *mockFirestoreRepository) RunTransaction(ctx context.Context, fn func(tx Transaction) error) error {
	return nil
}
func (m *mockFirestoreRepository) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	return nil
}
func (m *mockFirestoreRepository) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *mockFirestoreRepository) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *mockFirestoreRepository) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	return nil
}
func (m *mockFirestoreRepository) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	return nil
}
func (m *mockFirestoreRepository) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID string, indexID string) error {
	return nil
}
func (m *mockFirestoreRepository) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return nil, nil
}
func (m *mockFirestoreRepository) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return nil, nil
}

func TestFirestoreRepository_InterfaceCompliance(t *testing.T) {
	var _ FirestoreRepository = &mockFirestoreRepository{}
}

func TestFirestoreRepository_GetProject(t *testing.T) {
	repo := &mockFirestoreRepository{}
	ctx := context.Background()
	projectID := "test-proj"
	project, err := repo.GetProject(ctx, projectID)
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, projectID, project.ProjectID)
}
