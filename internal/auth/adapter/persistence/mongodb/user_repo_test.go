package mongodb_test

import (
	"context"
	"firestore-clone/internal/auth/adapter/persistence/mongodb"
	"firestore-clone/internal/auth/domain/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepoTestSuite struct {
	suite.Suite
	client     *mongo.Client
	database   *mongo.Database
	repository repository.AuthRepository
}

func (suite *MongoRepoTestSuite) SetupSuite() {
	// For testing purposes, we'll use a mock or skip actual MongoDB connection
	// In a real test environment, you would connect to a test MongoDB instance
	ctx := context.Background()

	// Connect to MongoDB test instance
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		suite.T().Skip("MongoDB not available for testing")
		return
	}

	suite.client = client
	suite.database = client.Database("auth_test_db")

	// Create repository
	repo, err := mongodb.NewMongoAuthRepository(suite.database)
	if err != nil {
		suite.T().Skip("Failed to create repository for testing")
		return
	}
	suite.repository = repo
}

func (suite *MongoRepoTestSuite) TearDownSuite() {
	if suite.client != nil {
		// Clean up test database
		suite.database.Drop(context.Background())
		suite.client.Disconnect(context.Background())
	}
}

func (suite *MongoRepoTestSuite) TestCreateUser_NilUser() {
	err := suite.repository.CreateUser(context.Background(), nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "user cannot be nil")
}

func (suite *MongoRepoTestSuite) TestGetUserByEmail_EmptyEmail() {
	user, err := suite.repository.GetUserByEmail(context.Background(), "")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.Contains(suite.T(), err.Error(), "email cannot be empty")
}

func (suite *MongoRepoTestSuite) TestGetUserByID_EmptyID() {
	user, err := suite.repository.GetUserByID(context.Background(), "")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.Contains(suite.T(), err.Error(), "user ID cannot be empty")
}

func TestMongoRepoTestSuite(t *testing.T) {
	suite.Run(t, new(MongoRepoTestSuite))
}
