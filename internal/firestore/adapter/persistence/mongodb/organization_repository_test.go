package mongodb

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockOrganizationCollection implements CollectionInterface for organization tests.
type MockOrganizationCollection struct{}

var _ CollectionInterface = (*MockOrganizationCollection)(nil)

func (m *MockOrganizationCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}
func (m *MockOrganizationCollection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
	return primitive.NewObjectID(), nil
}
func (m *MockOrganizationCollection) FindOne(ctx context.Context, filter interface{}) SingleResultInterface {
	return &MockSingleResult{}
}
func (m *MockOrganizationCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (UpdateResultInterface, error) {
	return &MockUpdateResult{matched: 1}, nil
}
func (m *MockOrganizationCollection) DeleteOne(ctx context.Context, filter interface{}) (DeleteResultInterface, error) {
	return &MockDeleteResult{deleted: 1}, nil
}
func (m *MockOrganizationCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorInterface, error) {
	return &MockCursor{}, nil
}
func (m *MockOrganizationCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (CursorInterface, error) {
	return &MockCursor{}, nil
}
func (m *MockOrganizationCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (UpdateResultInterface, error) {
	return &MockUpdateResult{matched: 1}, nil
}
func (m *MockOrganizationCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) SingleResultInterface {
	return &MockSingleResult{}
}

// MockSingleResult implements SingleResultInterface.
type MockSingleResult struct{}

func (m *MockSingleResult) Decode(v interface{}) error {
	return mongo.ErrNoDocuments
}

// MockUpdateResult implements UpdateResultInterface.
type MockUpdateResult struct{ matched int64 }

func (m *MockUpdateResult) Matched() int64 { return m.matched }

// MockDeleteResult implements DeleteResultInterface.
type MockDeleteResult struct{ deleted int64 }

func (m *MockDeleteResult) Deleted() int64 { return m.deleted }

// MockCursor implements CursorInterface.
type MockCursor struct{}

func (m *MockCursor) Next(ctx context.Context) bool   { return false }
func (m *MockCursor) Decode(val interface{}) error    { return nil }
func (m *MockCursor) Close(ctx context.Context) error { return nil }
func (m *MockCursor) Err() error                      { return nil }

// MockMongoClient implements a minimal session client for testing.
type MockMongoClient struct{}

var _ ClientInterface = (*MockMongoClient)(nil)

func (m *MockMongoClient) StartSession(opts ...*options.SessionOptions) (mongo.Session, error) {
	// For unit tests, return an error to simulate session unavailability
	// This tests the error handling path in the repository
	return nil, mongo.CommandError{Code: 40415, Message: "session not supported in test environment"}
}

// MockTenantManager implements TenantManagerInterface for testing.
type MockTenantManager struct{}

var _ TenantManagerInterface = (*MockTenantManager)(nil)

func (m *MockTenantManager) CreateOrganizationDatabase(ctx context.Context, organizationID string) error {
	return nil
}
func (m *MockTenantManager) DeleteOrganizationDatabase(ctx context.Context, organizationID string) error {
	return nil
}
func (m *MockTenantManager) GetDatabaseForOrganization(ctx context.Context, organizationID string) (*mongo.Database, error) {
	return nil, nil
}

// MockLogger implements logger.Logger for testing.
type MockLogger struct{}

func (m *MockLogger) Debug(args ...interface{})                              {}
func (m *MockLogger) Info(args ...interface{})                               {}
func (m *MockLogger) Warn(args ...interface{})                               {}
func (m *MockLogger) Error(args ...interface{})                              {}
func (m *MockLogger) Fatal(args ...interface{})                              {}
func (m *MockLogger) Debugf(format string, args ...interface{})              {}
func (m *MockLogger) Infof(format string, args ...interface{})               {}
func (m *MockLogger) Warnf(format string, args ...interface{})               {}
func (m *MockLogger) Errorf(format string, args ...interface{})              {}
func (m *MockLogger) Fatalf(format string, args ...interface{})              {}
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *MockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *MockLogger) WithComponent(component string) logger.Logger           { return m }

// newTestOrganizationRepository returns a repository with all dependencies mocked.
func newTestOrganizationRepository() *OrganizationRepository {
	return &OrganizationRepository{
		client:        &MockMongoClient{},
		collection:    &MockOrganizationCollection{},
		tenantManager: &MockTenantManager{},
		logger:        &MockLogger{},
	}
}

// --- Test Cases ---

func TestOrganizationRepository_CreateOrganization(t *testing.T) {
	t.Parallel()
	repo := newTestOrganizationRepository()
	org := &model.Organization{
		OrganizationID: "firestore-org-001",
		DisplayName:    "Test Firestore Organization",
		BillingEmail:   "admin@testorg.com",
		AdminEmails:    []string{"admin@testorg.com"},
		State:          model.OrganizationStateActive,
	}
	err := repo.CreateOrganization(context.Background(), org)
	// Since our mock client returns a session error, we expect this to fail
	if err == nil {
		t.Error("Expected error due to session not being available in test environment")
	}
	// Verify that the error is session-related
	if err != nil && err.Error() != "failed to start session: session not supported in test environment" {
		t.Errorf("Expected session error, got: %v", err)
	}
}

func TestOrganizationRepository_CreateOrganization_InvalidID(t *testing.T) {
	t.Parallel()
	repo := newTestOrganizationRepository()
	org := &model.Organization{
		OrganizationID: "",
		DisplayName:    "Test Org",
	}
	err := repo.CreateOrganization(context.Background(), org)
	if err == nil {
		t.Error("Expected error for invalid organization ID")
	}
}

func TestOrganizationRepository_GetOrganization_NotFound(t *testing.T) {
	t.Parallel()
	repo := newTestOrganizationRepository()
	org, err := repo.GetOrganization(context.Background(), "non-existent-org")
	if err == nil {
		t.Error("Expected error for non-existent organization")
	}
	if org != nil {
		t.Error("Expected nil organization for non-existent ID")
	}
}

func TestOrganizationRepository_UpdateOrganization_Success(t *testing.T) {
	t.Parallel()
	repo := newTestOrganizationRepository()
	org := &model.Organization{
		OrganizationID: "firestore-org-001",
		DisplayName:    "Updated Organization",
		BillingEmail:   "new-admin@testorg.com",
	}
	err := repo.UpdateOrganization(context.Background(), org)
	if err != nil {
		t.Errorf("UpdateOrganization failed: %v", err)
	}
	if org.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after update")
	}
}
