package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/options"

	"firestore-clone/internal/firestore/domain/model"
)

// MockProjectDatabaseProvider implements DatabaseProvider for testing with correct interface
type MockProjectDatabaseProvider struct {
	collections map[string]CollectionInterface
}

func NewMockProjectDatabaseProvider() *MockProjectDatabaseProvider {
	return &MockProjectDatabaseProvider{
		collections: make(map[string]CollectionInterface),
	}
}

func (m *MockProjectDatabaseProvider) Collection(name string) CollectionInterface {
	if col, exists := m.collections[name]; exists {
		return col
	}
	// Create a new mock collection if it doesn't exist
	col := &MockProjectCollection{}
	m.collections[name] = col
	return col
}

func (m *MockProjectDatabaseProvider) Client() interface{} {
	return nil // Return nil for mock
}

// MockProjectCollection implements CollectionInterface for project tests
type MockProjectCollection struct{}

var _ CollectionInterface = (*MockProjectCollection)(nil)

func (m *MockProjectCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil // No existing project
}
func (m *MockProjectCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return nil, nil // Successful insert
}
func (m *MockProjectCollection) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &MockSingleResult{}
}
func (m *MockProjectCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	return &MockUpdateResult{matched: 1}, nil
}
func (m *MockProjectCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	return &MockDeleteResult{deleted: 1}, nil
}
func (m *MockProjectCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	return &MockCursor{}, nil
}
func (m *MockProjectCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &MockCursor{}, nil
}
func (m *MockProjectCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return &MockUpdateResult{matched: 1}, nil
}
func (m *MockProjectCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &MockSingleResult{}
}

// newTestProjectDatabaseOperations creates a ProjectDatabaseOperations for testing
func newTestProjectDatabaseOperations() *ProjectDatabaseOperations {
	// Create a mock DocumentRepository with CollectionInterface implementations
	mockRepo := &DocumentRepository{
		documentsCol:   &MockProjectCollection{},
		collectionsCol: &MockProjectCollection{},
		db:             NewMockProjectDatabaseProvider(),
	}

	// Para pruebas, solo se requiere la dependencia repo
	ops := &ProjectDatabaseOperations{
		repo: mockRepo,
	}

	return ops
}

// MockDocumentRepositoryWithListProjects mocks only ListProjects for testing
// (other methods can panic or return zero values)
type MockDocumentRepositoryWithListProjects struct {
	ListProjectsFunc func(ctx context.Context, ownerEmail string) ([]*model.Project, error)
}

func (m *MockDocumentRepositoryWithListProjects) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return m.ListProjectsFunc(ctx, ownerEmail)
}

// Implement other methods as needed for compilation (can panic)

// MockProjectRepository implements ProjectRepository for unit tests
// Each method can be set to a custom function for flexible testing

type MockProjectRepository struct {
	CreateProjectFunc func(ctx context.Context, project *model.Project) error
	GetProjectFunc    func(ctx context.Context, projectID string) (*model.Project, error)
	UpdateProjectFunc func(ctx context.Context, project *model.Project) error
	DeleteProjectFunc func(ctx context.Context, projectID string) error
	ListProjectsFunc  func(ctx context.Context, ownerEmail string) ([]*model.Project, error)
}

func (m *MockProjectRepository) CreateProject(ctx context.Context, project *model.Project) error {
	return m.CreateProjectFunc(ctx, project)
}
func (m *MockProjectRepository) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return m.GetProjectFunc(ctx, projectID)
}
func (m *MockProjectRepository) UpdateProject(ctx context.Context, project *model.Project) error {
	return m.UpdateProjectFunc(ctx, project)
}
func (m *MockProjectRepository) DeleteProject(ctx context.Context, projectID string) error {
	return m.DeleteProjectFunc(ctx, projectID)
}
func (m *MockProjectRepository) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return m.ListProjectsFunc(ctx, ownerEmail)
}

func TestProjectDatabaseOperations_CreateProject(t *testing.T) {
	mockRepo := &MockProjectRepository{
		CreateProjectFunc: func(ctx context.Context, project *model.Project) error {
			return nil
		},
	}
	ops := NewProjectDatabaseOperations(mockRepo)
	ctx := context.Background()
	err := ops.CreateProject(ctx, &model.Project{ProjectID: "p1"})
	assert.NoError(t, err)
}

func TestProjectDatabaseOperations_GetProject(t *testing.T) {
	mockRepo := &MockProjectRepository{
		GetProjectFunc: func(ctx context.Context, projectID string) (*model.Project, error) {
			return &model.Project{ProjectID: projectID}, nil
		},
	}
	ops := NewProjectDatabaseOperations(mockRepo)
	ctx := context.Background()
	project, err := ops.GetProject(ctx, "p1")
	assert.NoError(t, err)
	assert.Equal(t, "p1", project.ProjectID)
}

func TestProjectDatabaseOperations_UpdateProject(t *testing.T) {
	mockRepo := &MockProjectRepository{
		UpdateProjectFunc: func(ctx context.Context, project *model.Project) error {
			return nil
		},
	}
	ops := NewProjectDatabaseOperations(mockRepo)
	ctx := context.Background()
	err := ops.UpdateProject(ctx, &model.Project{ProjectID: "p1"})
	assert.NoError(t, err)
}

func TestProjectDatabaseOperations_DeleteProject(t *testing.T) {
	mockRepo := &MockProjectRepository{
		DeleteProjectFunc: func(ctx context.Context, projectID string) error {
			return nil
		},
	}
	ops := NewProjectDatabaseOperations(mockRepo)
	ctx := context.Background()
	err := ops.DeleteProject(ctx, "p1")
	assert.NoError(t, err)
}

func TestProjectDatabaseOperations_ListProjects_FilterByOwnerEmail(t *testing.T) {
	mockRepo := &MockProjectRepository{
		ListProjectsFunc: func(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
			return []*model.Project{{ProjectID: "p1", OwnerEmail: ownerEmail}}, nil
		},
	}
	ops := NewProjectDatabaseOperations(mockRepo)
	ctx := context.Background()
	ownerEmail := "admin@example.com"
	projects, err := ops.ListProjects(ctx, ownerEmail)
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, ownerEmail, projects[0].OwnerEmail)
}

// --- DatabaseOperations tests ---

type MockDatabaseRepository struct {
	CreateDatabaseFunc func(ctx context.Context, projectID string, database *model.Database) error
	GetDatabaseFunc    func(ctx context.Context, projectID, databaseID string) (*model.Database, error)
	UpdateDatabaseFunc func(ctx context.Context, projectID string, database *model.Database) error
	DeleteDatabaseFunc func(ctx context.Context, projectID, databaseID string) error
	ListDatabasesFunc  func(ctx context.Context, projectID string) ([]*model.Database, error)
}

func (m *MockDatabaseRepository) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return m.CreateDatabaseFunc(ctx, projectID, database)
}
func (m *MockDatabaseRepository) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	return m.GetDatabaseFunc(ctx, projectID, databaseID)
}
func (m *MockDatabaseRepository) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	return m.UpdateDatabaseFunc(ctx, projectID, database)
}
func (m *MockDatabaseRepository) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	return m.DeleteDatabaseFunc(ctx, projectID, databaseID)
}
func (m *MockDatabaseRepository) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	return m.ListDatabasesFunc(ctx, projectID)
}

func TestDatabaseOperations_CreateDatabase(t *testing.T) {
	mockRepo := &MockDatabaseRepository{
		CreateDatabaseFunc: func(ctx context.Context, projectID string, database *model.Database) error {
			return nil
		},
	}
	ops := NewDatabaseOperations(mockRepo)
	err := ops.CreateDatabase(context.Background(), "p1", &model.Database{DatabaseID: "d1"})
	assert.NoError(t, err)
}

func TestDatabaseOperations_GetDatabase(t *testing.T) {
	mockRepo := &MockDatabaseRepository{
		GetDatabaseFunc: func(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
			return &model.Database{DatabaseID: databaseID}, nil
		},
	}
	ops := NewDatabaseOperations(mockRepo)
	db, err := ops.GetDatabase(context.Background(), "p1", "d1")
	assert.NoError(t, err)
	assert.Equal(t, "d1", db.DatabaseID)
}

func TestDatabaseOperations_UpdateDatabase(t *testing.T) {
	mockRepo := &MockDatabaseRepository{
		UpdateDatabaseFunc: func(ctx context.Context, projectID string, database *model.Database) error {
			return nil
		},
	}
	ops := NewDatabaseOperations(mockRepo)
	err := ops.UpdateDatabase(context.Background(), "p1", &model.Database{DatabaseID: "d1"})
	assert.NoError(t, err)
}

func TestDatabaseOperations_DeleteDatabase(t *testing.T) {
	mockRepo := &MockDatabaseRepository{
		DeleteDatabaseFunc: func(ctx context.Context, projectID, databaseID string) error {
			return nil
		},
	}
	ops := NewDatabaseOperations(mockRepo)
	err := ops.DeleteDatabase(context.Background(), "p1", "d1")
	assert.NoError(t, err)
}

func TestDatabaseOperations_ListDatabases(t *testing.T) {
	mockRepo := &MockDatabaseRepository{
		ListDatabasesFunc: func(ctx context.Context, projectID string) ([]*model.Database, error) {
			return []*model.Database{{DatabaseID: "d1"}}, nil
		},
	}
	ops := NewDatabaseOperations(mockRepo)
	dbs, err := ops.ListDatabases(context.Background(), "p1")
	assert.NoError(t, err)
	assert.Len(t, dbs, 1)
	assert.Equal(t, "d1", dbs[0].DatabaseID)
}
