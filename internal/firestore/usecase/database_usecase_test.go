package usecase_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	. "firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDatabaseRepo is a specialized mock for database tests
type MockDatabaseRepo struct {
	projects             map[string]*model.Project
	databases            map[string]map[string]*model.Database // projectID -> databaseID -> Database
	shouldFailOnProject  bool
	shouldFailOnDatabase bool
}

func NewMockDatabaseRepo() *MockDatabaseRepo {
	return &MockDatabaseRepo{
		projects:  make(map[string]*model.Project),
		databases: make(map[string]map[string]*model.Database),
	}
}

func (m *MockDatabaseRepo) AddProject(project *model.Project) {
	m.projects[project.ProjectID] = project
}

func (m *MockDatabaseRepo) AddDatabase(projectID string, database *model.Database) {
	if m.databases[projectID] == nil {
		m.databases[projectID] = make(map[string]*model.Database)
	}
	m.databases[projectID][database.DatabaseID] = database
}

func (m *MockDatabaseRepo) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	if m.shouldFailOnProject {
		return nil, errors.NewInternalError("Repository error")
	}
	if project, exists := m.projects[projectID]; exists {
		return project, nil
	}
	return nil, errors.NewNotFoundError("Project not found")
}

func (m *MockDatabaseRepo) GetDatabase(ctx context.Context, projectID, databaseID string) (*model.Database, error) {
	if m.shouldFailOnDatabase {
		return nil, errors.NewInternalError("Repository error")
	}
	if projectDBs, exists := m.databases[projectID]; exists {
		if database, exists := projectDBs[databaseID]; exists {
			return database, nil
		}
	}
	return nil, errors.NewNotFoundError("Database not found")
}

func (m *MockDatabaseRepo) CreateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	if m.shouldFailOnDatabase {
		return errors.NewInternalError("Repository error")
	}
	m.AddDatabase(projectID, database)
	return nil
}

func (m *MockDatabaseRepo) UpdateDatabase(ctx context.Context, projectID string, database *model.Database) error {
	if m.shouldFailOnDatabase {
		return errors.NewInternalError("Repository error")
	}
	if projectDBs, exists := m.databases[projectID]; exists {
		if _, exists := projectDBs[database.DatabaseID]; exists {
			projectDBs[database.DatabaseID] = database
			return nil
		}
	}
	return errors.NewNotFoundError("Database not found")
}

func (m *MockDatabaseRepo) DeleteDatabase(ctx context.Context, projectID, databaseID string) error {
	if m.shouldFailOnDatabase {
		return errors.NewInternalError("Repository error")
	}
	if projectDBs, exists := m.databases[projectID]; exists {
		if _, exists := projectDBs[databaseID]; exists {
			delete(projectDBs, databaseID)
			return nil
		}
	}
	return errors.NewNotFoundError("Database not found")
}

func (m *MockDatabaseRepo) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, error) {
	if m.shouldFailOnDatabase {
		return nil, errors.NewInternalError("Repository error")
	}
	var databases []*model.Database
	if projectDBs, exists := m.databases[projectID]; exists {
		for _, db := range projectDBs {
			databases = append(databases, db)
		}
	}
	return databases, nil
}

// Implement minimal required methods for FirestoreRepository interface
func (m *MockDatabaseRepo) CreateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockDatabaseRepo) UpdateProject(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *MockDatabaseRepo) DeleteProject(ctx context.Context, projectID string) error { return nil }
func (m *MockDatabaseRepo) ListProjects(ctx context.Context, ownerEmail string) ([]*model.Project, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) CreateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockDatabaseRepo) GetCollection(ctx context.Context, projectID, databaseID, collectionID string) (*model.Collection, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) UpdateCollection(ctx context.Context, projectID, databaseID string, collection *model.Collection) error {
	return nil
}
func (m *MockDatabaseRepo) DeleteCollection(ctx context.Context, projectID, databaseID, collectionID string) error {
	return nil
}
func (m *MockDatabaseRepo) ListCollections(ctx context.Context, projectID, databaseID string) ([]*model.Collection, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	return nil
}
func (m *MockDatabaseRepo) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) DeleteDocumentByPath(ctx context.Context, path string) error { return nil }
func (m *MockDatabaseRepo) RunQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) RunCollectionGroupQuery(ctx context.Context, projectID, databaseID string, collectionID string, query *model.Query) ([]*model.Document, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) RunAggregationQuery(ctx context.Context, projectID, databaseID, collectionID string, query *model.Query) (*model.AggregationResult, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	return nil, "", nil
}
func (m *MockDatabaseRepo) RunTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	return nil
}
func (m *MockDatabaseRepo) RunBatchWrite(ctx context.Context, projectID, databaseID string, writes []*model.WriteOperation) ([]*model.WriteResult, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	return nil
}
func (m *MockDatabaseRepo) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockDatabaseRepo) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	return nil
}
func (m *MockDatabaseRepo) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	return nil
}
func (m *MockDatabaseRepo) CreateIndex(ctx context.Context, projectID, databaseID, collectionID string, index *model.CollectionIndex) error {
	return nil
}
func (m *MockDatabaseRepo) DeleteIndex(ctx context.Context, projectID, databaseID, collectionID string, indexID string) error {
	return nil
}
func (m *MockDatabaseRepo) ListIndexes(ctx context.Context, projectID, databaseID, collectionID string) ([]*model.CollectionIndex, error) {
	return nil, nil
}
func (m *MockDatabaseRepo) ListSubcollections(ctx context.Context, projectID, databaseID, collectionID, documentID string) ([]string, error) {
	return nil, nil
}

func newTestFirestoreUsecaseWithDatabaseMock(repo repository.FirestoreRepository) FirestoreUsecaseInterface {
	return NewFirestoreUsecase(
		repo,
		nil, // securityRepo
		nil, // queryEngine
		nil, // projectionService
		&MockLogger{},
	)
}

func TestCreateDatabase(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		// Add a project first
		project := &model.Project{
			ProjectID:      "p1",
			OrganizationID: "org1",
			DisplayName:    "Test Project",
			CreatedAt:      time.Now(),
		}
		repo.AddProject(project)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		database := &model.Database{
			DatabaseID:      "d1",
			Type:            model.DatabaseTypeFirestoreNative,
			ConcurrencyMode: model.ConcurrencyModeOptimistic,
			LocationID:      "us-central1",
		}

		db, err := uc.CreateDatabase(context.Background(), CreateDatabaseRequest{
			ProjectID: "p1",
			Database:  database,
		})

		require.NoError(t, err)
		assert.Equal(t, "d1", db.DatabaseID)
		assert.Equal(t, "p1", db.ProjectID)
		assert.Equal(t, model.DatabaseStateCreating, db.State)
		assert.NotZero(t, db.CreatedAt)
		assert.NotZero(t, db.UpdatedAt)
	})

	t.Run("project not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		database := &model.Database{
			DatabaseID: "d1",
		}

		_, err := uc.CreateDatabase(context.Background(), CreateDatabaseRequest{
			ProjectID: "nonexistent",
			Database:  database,
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Project 'nonexistent' not found")
	})

	t.Run("database already exists", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project and existing database
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		existingDB := &model.Database{DatabaseID: "d1", ProjectID: "p1"}
		repo.AddDatabase("p1", existingDB)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		database := &model.Database{DatabaseID: "d1"}

		_, err := uc.CreateDatabase(context.Background(), CreateDatabaseRequest{
			ProjectID: "p1",
			Database:  database,
		})

		require.Error(t, err)
		assert.True(t, errors.IsConflict(err))
		assert.Contains(t, err.Error(), "Database 'd1' already exists")
	})

	t.Run("validation errors", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		testCases := []struct {
			name        string
			request     CreateDatabaseRequest
			expectedErr string
		}{
			{
				name: "missing project ID",
				request: CreateDatabaseRequest{
					Database: &model.Database{DatabaseID: "d1"},
				},
				expectedErr: "Project ID is required",
			},
			{
				name: "missing database config",
				request: CreateDatabaseRequest{
					ProjectID: "p1",
				},
				expectedErr: "Database configuration is required",
			},
			{
				name: "missing database ID",
				request: CreateDatabaseRequest{
					ProjectID: "p1",
					Database:  &model.Database{},
				},
				expectedErr: "Database ID is required",
			},
			{
				name: "invalid database ID format",
				request: CreateDatabaseRequest{
					ProjectID: "p1",
					Database:  &model.Database{DatabaseID: "invalid@id"},
				},
				expectedErr: "Database ID must contain only alphanumeric characters",
			},
			{
				name: "invalid location ID",
				request: CreateDatabaseRequest{
					ProjectID: "p1",
					Database:  &model.Database{DatabaseID: "d1", LocationID: "invalid-location"},
				},
				expectedErr: "Invalid location ID format",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := uc.CreateDatabase(context.Background(), tc.request)
				require.Error(t, err)
				assert.True(t, errors.IsValidation(err))
				assert.Contains(t, err.Error(), tc.expectedErr)
			})
		}
	})
}

func TestGetDatabase(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project and database
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		database := &model.Database{
			DatabaseID: "d1",
			ProjectID:  "p1",
			Type:       model.DatabaseTypeFirestoreNative,
		}
		repo.AddDatabase("p1", database)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		db, err := uc.GetDatabase(context.Background(), GetDatabaseRequest{
			ProjectID:  "p1",
			DatabaseID: "d1",
		})

		require.NoError(t, err)
		assert.Equal(t, "d1", db.DatabaseID)
		assert.Equal(t, "p1", db.ProjectID)
	})

	t.Run("database not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		_, err := uc.GetDatabase(context.Background(), GetDatabaseRequest{
			ProjectID:  "p1",
			DatabaseID: "nonexistent",
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Database 'nonexistent' not found in project 'p1'")
	})
}

func TestUpdateDatabase(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project and database
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		database := &model.Database{
			DatabaseID: "d1",
			ProjectID:  "p1",
			Type:       model.DatabaseTypeFirestoreNative,
		}
		repo.AddDatabase("p1", database)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		updatedDB := &model.Database{
			DatabaseID:  "d1",
			ProjectID:   "p1",
			DisplayName: "Updated Database",
		}

		db, err := uc.UpdateDatabase(context.Background(), UpdateDatabaseRequest{
			ProjectID: "p1",
			Database:  updatedDB,
		})

		require.NoError(t, err)
		assert.Equal(t, "d1", db.DatabaseID)
		assert.Equal(t, "Updated Database", db.DisplayName)
	})

	t.Run("project not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		database := &model.Database{DatabaseID: "d1"}

		_, err := uc.UpdateDatabase(context.Background(), UpdateDatabaseRequest{
			ProjectID: "nonexistent",
			Database:  database,
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Project 'nonexistent' not found")
	})
}

func TestDeleteDatabase(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project and database
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		database := &model.Database{DatabaseID: "d1", ProjectID: "p1"}
		repo.AddDatabase("p1", database)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		err := uc.DeleteDatabase(context.Background(), DeleteDatabaseRequest{
			ProjectID:  "p1",
			DatabaseID: "d1",
		})

		assert.NoError(t, err)
	})

	t.Run("project not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		err := uc.DeleteDatabase(context.Background(), DeleteDatabaseRequest{
			ProjectID:  "nonexistent",
			DatabaseID: "d1",
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Project 'nonexistent' not found")
	})

	t.Run("database not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project but no database
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		err := uc.DeleteDatabase(context.Background(), DeleteDatabaseRequest{
			ProjectID:  "p1",
			DatabaseID: "nonexistent",
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Database 'nonexistent' not found in project 'p1'")
	})
}

func TestListDatabases(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project and databases
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		db1 := &model.Database{DatabaseID: "d1", ProjectID: "p1"}
		db2 := &model.Database{DatabaseID: "d2", ProjectID: "p1"}
		repo.AddDatabase("p1", db1)
		repo.AddDatabase("p1", db2)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		dbs, err := uc.ListDatabases(context.Background(), ListDatabasesRequest{
			ProjectID: "p1",
		})

		require.NoError(t, err)
		assert.Len(t, dbs, 2)

		// Check that both databases are returned
		dbIDs := make([]string, len(dbs))
		for i, db := range dbs {
			dbIDs[i] = db.DatabaseID
		}
		assert.Contains(t, dbIDs, "d1")
		assert.Contains(t, dbIDs, "d2")
	})

	t.Run("project not found", func(t *testing.T) {
		repo := NewMockDatabaseRepo()
		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		_, err := uc.ListDatabases(context.Background(), ListDatabasesRequest{
			ProjectID: "nonexistent",
		})

		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
		assert.Contains(t, err.Error(), "Project 'nonexistent' not found")
	})

	t.Run("empty list for existing project", func(t *testing.T) {
		repo := NewMockDatabaseRepo()

		// Add project but no databases
		project := &model.Project{ProjectID: "p1", OrganizationID: "org1"}
		repo.AddProject(project)

		uc := newTestFirestoreUsecaseWithDatabaseMock(repo)

		dbs, err := uc.ListDatabases(context.Background(), ListDatabasesRequest{
			ProjectID: "p1",
		})

		require.NoError(t, err)
		assert.Len(t, dbs, 0)
	})
}
