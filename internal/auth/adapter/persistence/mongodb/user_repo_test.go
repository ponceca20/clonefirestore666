package mongodb_test

import (
	"context"
	"firestore-clone/internal/auth/adapter/persistence/mongodb"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepoTestSuite struct {
	suite.Suite
	client     *mongo.Client
	database   *mongo.Database
	repository *mongodb.MongoAuthRepository
}

func (suite *MongoRepoTestSuite) SetupSuite() {
	// Setup code for MongoDB connection and repository initialization
}

func (suite *MongoRepoTestSuite) TearDownSuite() {
	// Teardown code for closing MongoDB connection
}

func (suite *MongoRepoTestSuite) TestCreateUser_NilUser() {
	err := suite.repository.CreateUser(context.Background(), nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "user cannot be nil")
}

func (suite *MongoRepoTestSuite) TestGetUserByEmail_EmptyEmail() {
	user, err := suite.repository.GetUserByEmail(context.Background(), "", "project1")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.Contains(suite.T(), err.Error(), "email cannot be empty")
}

func (suite *MongoRepoTestSuite) TestGetUserByID_EmptyID() {
	user, err := suite.repository.GetUserByID(context.Background(), "", "project1")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.Contains(suite.T(), err.Error(), "id cannot be empty")
}

func TestMongoRepoTestSuite(t *testing.T) {
	suite.Run(t, new(MongoRepoTestSuite))
}
