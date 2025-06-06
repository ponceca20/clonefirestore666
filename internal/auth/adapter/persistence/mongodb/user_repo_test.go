package mongodb_test

import (
	"context"
	"testing"
	"time"

	"firestore-clone/internal/auth/adapter/persistence/mongodb"
	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepoTestSuite struct {
	suite.Suite
	client     *mongo.Client
	database   *mongo.Database
	repository *mongodb.MongoAuthRepository
}

func (suite *MongoRepoTestSuite) SetupSuite() {
	// Setup test MongoDB connection
	// This should connect to a test database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use MongoDB test container or local test instance
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(suite.T(), err)

	suite.client = client
	suite.database = client.Database("test_auth_db")
}

func (suite *MongoRepoTestSuite) SetupTest() {
	// Clean database before each test
	ctx := context.Background()
	err := suite.database.Drop(ctx)
	require.NoError(suite.T(), err)

	// Create repository
	repo, err := mongodb.NewMongoAuthRepository(suite.database)
	require.NoError(suite.T(), err)
	suite.repository = repo
}

func (suite *MongoRepoTestSuite) TearDownSuite() {
	if suite.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.client.Disconnect(ctx)
	}
}

func (suite *MongoRepoTestSuite) TestCreateUser_Success() {
	// Arrange
	ctx := context.Background()
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	// Act
	err := suite.repository.CreateUser(ctx, user)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), user.ID)
	assert.False(suite.T(), user.CreatedAt.IsZero())
	assert.False(suite.T(), user.UpdatedAt.IsZero())
}

func (suite *MongoRepoTestSuite) TestCreateUser_DuplicateEmail() {
	// Arrange
	ctx := context.Background()
	email := "duplicate@example.com"

	user1 := &model.User{
		Email:        email,
		PasswordHash: "password1",
	}
	user2 := &model.User{
		Email:        email,
		PasswordHash: "password2",
	}

	// Act
	err1 := suite.repository.CreateUser(ctx, user1)
	require.NoError(suite.T(), err1)

	err2 := suite.repository.CreateUser(ctx, user2)

	// Assert
	assert.Error(suite.T(), err2)
	assert.Equal(suite.T(), usecase.ErrEmailTaken, err2)
}

func (suite *MongoRepoTestSuite) TestGetUserByEmail_Success() {
	// Arrange
	ctx := context.Background()
	email := "test@example.com"
	originalUser := &model.User{
		Email:        email,
		PasswordHash: "hashed_password",
	}

	err := suite.repository.CreateUser(ctx, originalUser)
	require.NoError(suite.T(), err)

	// Act
	retrievedUser, err := suite.repository.GetUserByEmail(ctx, email)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), originalUser.ID, retrievedUser.ID)
	assert.Equal(suite.T(), originalUser.Email, retrievedUser.Email)
	assert.Equal(suite.T(), originalUser.PasswordHash, retrievedUser.PasswordHash)
}

func (suite *MongoRepoTestSuite) TestGetUserByEmail_NotFound() {
	// Arrange
	ctx := context.Background()
	email := "nonexistent@example.com"

	// Act
	user, err := suite.repository.GetUserByEmail(ctx, email)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrUserNotFound, err)
	assert.Nil(suite.T(), user)
}

func (suite *MongoRepoTestSuite) TestGetUserByID_Success() {
	// Arrange
	ctx := context.Background()
	originalUser := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	err := suite.repository.CreateUser(ctx, originalUser)
	require.NoError(suite.T(), err)

	// Act
	retrievedUser, err := suite.repository.GetUserByID(ctx, originalUser.ID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), originalUser.ID, retrievedUser.ID)
	assert.Equal(suite.T(), originalUser.Email, retrievedUser.Email)
}

func (suite *MongoRepoTestSuite) TestGetUserByID_NotFound() {
	// Arrange
	ctx := context.Background()
	nonExistentID := "nonexistent-id"

	// Act
	user, err := suite.repository.GetUserByID(ctx, nonExistentID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrUserNotFound, err)
	assert.Nil(suite.T(), user)
}

func (suite *MongoRepoTestSuite) TestCreateSession_Success() {
	// Arrange
	ctx := context.Background()
	session := &model.Session{
		UserID:    "user-123",
		Token:     "session-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Act
	err := suite.repository.CreateSession(ctx, session)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), session.ID)
	assert.False(suite.T(), session.CreatedAt.IsZero())
}

func (suite *MongoRepoTestSuite) TestGetSessionByID_Success() {
	// Arrange
	ctx := context.Background()
	originalSession := &model.Session{
		UserID:    "user-123",
		Token:     "session-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := suite.repository.CreateSession(ctx, originalSession)
	require.NoError(suite.T(), err)

	// Act
	retrievedSession, err := suite.repository.GetSessionByID(ctx, originalSession.ID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), originalSession.ID, retrievedSession.ID)
	assert.Equal(suite.T(), originalSession.UserID, retrievedSession.UserID)
	assert.Equal(suite.T(), originalSession.Token, retrievedSession.Token)
}

func (suite *MongoRepoTestSuite) TestGetSessionByID_NotFound() {
	// Arrange
	ctx := context.Background()
	nonExistentID := "nonexistent-session-id"

	// Act
	session, err := suite.repository.GetSessionByID(ctx, nonExistentID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrSessionNotFound, err)
	assert.Nil(suite.T(), session)
}

func (suite *MongoRepoTestSuite) TestDeleteSession_Success() {
	// Arrange
	ctx := context.Background()
	session := &model.Session{
		UserID:    "user-123",
		Token:     "session-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := suite.repository.CreateSession(ctx, session)
	require.NoError(suite.T(), err)

	// Act
	err = suite.repository.DeleteSession(ctx, session.ID)

	// Assert
	require.NoError(suite.T(), err)

	// Verify session is deleted
	_, err = suite.repository.GetSessionByID(ctx, session.ID)
	assert.Equal(suite.T(), usecase.ErrSessionNotFound, err)
}

func (suite *MongoRepoTestSuite) TestDeleteSession_NotFound() {
	// Arrange
	ctx := context.Background()
	nonExistentID := "nonexistent-session-id"

	// Act
	err := suite.repository.DeleteSession(ctx, nonExistentID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), usecase.ErrSessionNotFound, err)
}

func TestMongoRepoTestSuite(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping MongoDB integration tests in short mode")
	}

	suite.Run(t, new(MongoRepoTestSuite))
}

// Benchmark tests
func BenchmarkCreateUser(b *testing.B) {
	// Setup
	ctx := context.Background()
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	database := client.Database("benchmark_test_db")
	repo, _ := mongodb.NewMongoAuthRepository(database)

	defer func() {
		database.Drop(ctx)
		client.Disconnect(ctx)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &model.User{
			Email:        "benchmark@example.com",
			PasswordHash: "hashed_password",
		}
		repo.CreateUser(ctx, user)
	}
}
