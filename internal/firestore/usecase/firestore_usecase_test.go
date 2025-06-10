package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/firestore/usecase"

	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFirestoreRepo mocks the FirestoreRepository interface
// Only the methods needed for the tests are implemented

type MockFirestoreRepo struct{ mock.Mock }

func (m *MockFirestoreRepo) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID, data)
	return args.Get(0).(*model.Document), args.Error(1)
}
func (m *MockFirestoreRepo) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID)
	return args.Get(0).(*model.Document), args.Error(1)
}
func (m *MockFirestoreRepo) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	args := m.Called(ctx, projectID, databaseID, collectionID, documentID, data, merge)
	return args.Get(0).(*model.Document), args.Error(1)
}

// DummyLogger is un logger compatible con logger.Logger

type DummyLogger struct{}

func (d *DummyLogger) Info(args ...interface{})                               {}
func (d *DummyLogger) Error(args ...interface{})                              {}
func (d *DummyLogger) Debug(args ...interface{})                              {}
func (d *DummyLogger) Warn(args ...interface{})                               {}
func (d *DummyLogger) Fatal(args ...interface{})                              {}
func (d *DummyLogger) Infof(format string, args ...interface{})               {}
func (d *DummyLogger) Errorf(format string, args ...interface{})              {}
func (d *DummyLogger) Debugf(format string, args ...interface{})              {}
func (d *DummyLogger) Warnf(format string, args ...interface{})               {}
func (d *DummyLogger) Fatalf(format string, args ...interface{})              {}
func (d *DummyLogger) WithFields(fields map[string]interface{}) logger.Logger { return d }
func (d *DummyLogger) WithContext(ctx context.Context) logger.Logger          { return d }
func (d *DummyLogger) WithComponent(component string) logger.Logger           { return d }

// --- Add missing FirestoreRepository methods (no-op) ---
func (m *MockFirestoreRepo) CreateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockFirestoreRepo) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteProject(ctx context.Context, projectID string) error { return nil }
func (m *MockFirestoreRepo) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *MockFirestoreRepo) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) DeleteDocumentByPath(ctx context.Context, path string) error { return nil }
func (m *MockFirestoreRepo) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return nil, "", nil
}
func (m *MockFirestoreRepo) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	return fn(&DummyTransaction{})
}
func (m *MockFirestoreRepo) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockFirestoreRepo) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	return nil
}
func (m *MockFirestoreRepo) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	return nil
}
func (m *MockFirestoreRepo) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID string, indexID string) error {
	return nil
}
func (m *MockFirestoreRepo) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return nil, nil
}
func (m *MockFirestoreRepo) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (m *MockFirestoreRepo) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}

// --- SecurityRulesEngine mock ---
type MockSecurityRulesEngine struct{}

func (m *MockSecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	return &repository.RuleEvaluationResult{Allowed: true}, nil
}
func (m *MockSecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return nil, nil
}
func (m *MockSecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	return nil
}
func (m *MockSecurityRulesEngine) ValidateRules(rules []*repository.SecurityRule) error {
	return nil
}

// --- QueryEngine mock ---
type MockQueryEngine struct{}

func (m *MockQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	return nil, nil
}

func newTestFirestoreUsecase(repo *MockFirestoreRepo) usecase.FirestoreUsecaseInterface {
	security := &MockSecurityRulesEngine{}
	query := &MockQueryEngine{}
	logger := &DummyLogger{}
	return usecase.NewFirestoreUsecase(repo, security, query, logger)
}

func TestCreateDocument_Success(t *testing.T) {
	repo := new(MockFirestoreRepo)
	uc := newTestFirestoreUsecase(repo)
	ctx := context.Background()
	request := usecase.CreateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         map[string]interface{}{"foo": "bar"},
	}
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	doc := &model.Document{DocumentID: "doc1", Fields: fields}
	repo.On("CreateDocument", ctx, "p1", "d1", "c1", "doc1", mock.Anything).Return(doc, nil)
	result, err := uc.CreateDocument(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, doc, result)
}

func TestCreateDocument_Error(t *testing.T) {
	repo := new(MockFirestoreRepo)
	uc := newTestFirestoreUsecase(repo)
	ctx := context.Background()
	request := usecase.CreateDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
		Data:         map[string]interface{}{"foo": "bar"},
	}
	repo.On("CreateDocument", ctx, "p1", "d1", "c1", "doc1", mock.Anything).Return((*model.Document)(nil), errors.New("fail"))
	result, err := uc.CreateDocument(ctx, request)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetDocument_Success(t *testing.T) {
	repo := new(MockFirestoreRepo)
	uc := newTestFirestoreUsecase(repo)
	ctx := context.Background()
	request := usecase.GetDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	}
	fields := map[string]*model.FieldValue{"foo": model.NewFieldValue("bar")}
	doc := &model.Document{DocumentID: "doc1", Fields: fields}
	repo.On("GetDocument", ctx, "p1", "d1", "c1", "doc1").Return(doc, nil)
	result, err := uc.GetDocument(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, doc, result)
}

func TestGetDocument_NotFound(t *testing.T) {
	repo := new(MockFirestoreRepo)
	uc := newTestFirestoreUsecase(repo)
	ctx := context.Background()
	request := usecase.GetDocumentRequest{
		ProjectID:    "p1",
		DatabaseID:   "d1",
		CollectionID: "c1",
		DocumentID:   "doc1",
	}
	repo.On("GetDocument", ctx, "p1", "d1", "c1", "doc1").Return((*model.Document)(nil), errors.New("not found"))
	result, err := uc.GetDocument(ctx, request)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- Minimal dummy Transaction for RunTransaction ---
type DummyTransaction struct{}

func (d *DummyTransaction) Get(projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return nil, nil
}
func (d *DummyTransaction) GetByPath(path string) (*model.Document, error) { return nil, nil }
func (d *DummyTransaction) Create(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) error {
	return nil
}
func (d *DummyTransaction) CreateByPath(path string, data map[string]*model.FieldValue) error {
	return nil
}
func (d *DummyTransaction) Update(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) error {
	return nil
}
func (d *DummyTransaction) UpdateByPath(path string, data map[string]*model.FieldValue, updateMask []string) error {
	return nil
}
func (d *DummyTransaction) Set(projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) error {
	return nil
}
func (d *DummyTransaction) SetByPath(path string, data map[string]*model.FieldValue, merge bool) error {
	return nil
}
func (d *DummyTransaction) Delete(projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (d *DummyTransaction) DeleteByPath(path string) error { return nil }
func (d *DummyTransaction) Query(projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (d *DummyTransaction) GetStartTime() time.Time { return time.Now() }
func (d *DummyTransaction) IsReadOnly() bool        { return false }
