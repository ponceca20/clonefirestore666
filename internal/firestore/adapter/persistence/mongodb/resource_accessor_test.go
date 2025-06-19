package mongodb_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestResourceAccessor_GetDocument(t *testing.T) {
	ctx := context.Background()

	// Connect to test MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_resource_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	accessor := mongodb.NewResourceAccessor(testDB, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Simple Document Retrieval", func(t *testing.T) {
		// Insert test document
		collectionName := "test-project_test-db_users"
		collection := testDB.Collection(collectionName)

		testDoc := bson.M{
			"_id":         "user123",
			"project_id":  projectID,
			"database_id": databaseID,
			"name":        "John Doe",
			"email":       "john@example.com",
			"role":        "admin",
		}

		_, err := collection.InsertOne(ctx, testDoc)
		require.NoError(t, err)

		// Test document retrieval
		doc, err := accessor.GetDocument(ctx, projectID, databaseID, "/users/user123")
		require.NoError(t, err)
		require.NotNil(t, doc)

		// Check that document contains expected fields (without MongoDB internal fields)
		assert.Equal(t, "John Doe", doc["name"])
		assert.Equal(t, "john@example.com", doc["email"])
		assert.Equal(t, "admin", doc["role"])

		// Check that MongoDB internal fields are filtered out
		assert.NotContains(t, doc, "_id")
		assert.NotContains(t, doc, "project_id")
		assert.NotContains(t, doc, "database_id")
	})

	t.Run("Subcollection Document Retrieval", func(t *testing.T) {
		// Insert test document in subcollection
		collectionName := "test-project_test-db_users.posts"
		collection := testDB.Collection(collectionName)

		testDoc := bson.M{
			"_id":         "post456",
			"project_id":  projectID,
			"database_id": databaseID,
			"title":       "Test Post",
			"content":     "This is a test post",
			"authorId":    "user123",
		}

		_, err := collection.InsertOne(ctx, testDoc)
		require.NoError(t, err)

		// Test subcollection document retrieval
		doc, err := accessor.GetDocument(ctx, projectID, databaseID, "/users/user123/posts/post456")
		require.NoError(t, err)
		require.NotNil(t, doc)

		assert.Equal(t, "Test Post", doc["title"])
		assert.Equal(t, "This is a test post", doc["content"])
		assert.Equal(t, "user123", doc["authorId"])
	})

	t.Run("Nonexistent Document", func(t *testing.T) {
		doc, err := accessor.GetDocument(ctx, projectID, databaseID, "/users/nonexistent")
		require.NoError(t, err)
		assert.Nil(t, doc)
	})

	t.Run("Invalid Path Format", func(t *testing.T) {
		_, err := accessor.GetDocument(ctx, projectID, databaseID, "/users") // Missing document ID
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document path")
	})
}

func TestResourceAccessor_ExistsDocument(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	testDB := client.Database("firestore_exists_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	accessor := mongodb.NewResourceAccessor(testDB, log)

	projectID := "test-project"
	databaseID := "test-db"

	t.Run("Existing Document", func(t *testing.T) {
		// Insert test document
		collectionName := "test-project_test-db_users"
		collection := testDB.Collection(collectionName)

		testDoc := bson.M{
			"_id":         "user123",
			"project_id":  projectID,
			"database_id": databaseID,
			"name":        "John Doe",
		}

		_, err := collection.InsertOne(ctx, testDoc)
		require.NoError(t, err)

		// Test existence check
		exists, err := accessor.ExistsDocument(ctx, projectID, databaseID, "/users/user123")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Nonexistent Document", func(t *testing.T) {
		exists, err := accessor.ExistsDocument(ctx, projectID, databaseID, "/users/nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Invalid Path Format", func(t *testing.T) {
		_, err := accessor.ExistsDocument(ctx, projectID, databaseID, "/users")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document path")
	})
}

func TestResourceAccessor_PathParsing(t *testing.T) {
	log := logger.NewTestLogger()
	accessor := mongodb.NewResourceAccessor(nil, log).(*mongodb.ResourceAccessor)

	testCases := []struct {
		name               string
		path               string
		expectedCollection string
		expectedDocID      string
		expectError        bool
	}{
		{
			name:               "Simple document path",
			path:               "/users/user123",
			expectedCollection: "users",
			expectedDocID:      "user123",
			expectError:        false,
		},
		{
			name:               "Subcollection document path",
			path:               "/users/user123/posts/post456",
			expectedCollection: "users.posts",
			expectedDocID:      "post456",
			expectError:        false,
		},
		{
			name:               "Deep nested path",
			path:               "/users/user123/posts/post456/comments/comment789",
			expectedCollection: "users.posts.comments",
			expectedDocID:      "comment789",
			expectError:        false,
		},
		{
			name:               "Path without leading slash",
			path:               "users/user123",
			expectedCollection: "users",
			expectedDocID:      "user123",
			expectError:        false,
		},
		{
			name:        "Invalid path - collection only",
			path:        "/users",
			expectError: true,
		},
		{
			name:        "Invalid path - empty",
			path:        "",
			expectError: true,
		},
		{
			name:        "Invalid path - single segment",
			path:        "/users/",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collection, docID, err := accessor.ParsePath(tc.path)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedCollection, collection)
				assert.Equal(t, tc.expectedDocID, docID)
			}
		})
	}
}
