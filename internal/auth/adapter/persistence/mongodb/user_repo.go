package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"
	"firestore-clone/internal/auth/usecase"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoAuthRepository implements the AuthRepository interface using MongoDB
type MongoAuthRepository struct {
	db                 *mongo.Database
	usersCollection    *mongo.Collection
	sessionsCollection *mongo.Collection
}

// NewMongoAuthRepository creates a new MongoDB auth repository
func NewMongoAuthRepository(db *mongo.Database) (*MongoAuthRepository, error) {
	repo := &MongoAuthRepository{
		db:                 db,
		usersCollection:    db.Collection("users"),
		sessionsCollection: db.Collection("sessions"),
	}

	// Create indexes for efficient querying with Firestore project isolation
	if err := repo.createIndexes(); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return repo, nil
}

// createIndexes creates necessary indexes for performance and uniqueness
func (r *MongoAuthRepository) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Compound index for email uniqueness per project (Firestore requirement)
	emailProjectIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
			{Key: "projectId", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("email_project_unique"),
	}

	// Index for projectId queries
	projectIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "projectId", Value: 1}},
		Options: options.Index().SetName("project_id_idx"),
	}

	// Index for user ID and project combination
	userProjectIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "id", Value: 1},
			{Key: "projectId", Value: 1},
		},
		Options: options.Index().SetName("user_project_idx"),
	}

	// Create indexes
	_, err := r.usersCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		emailProjectIndex,
		projectIndex,
		userProjectIndex,
	})
	if err != nil {
		return err
	}

	// Session index for cleanup
	sessionIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetName("session_user_idx"),
	}

	_, err = r.sessionsCollection.Indexes().CreateOne(ctx, sessionIndex)
	return err
}

// CreateUser creates a new user with Firestore project context
func (r *MongoAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	// Set timestamps and object ID
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.ObjectID = primitive.NewObjectID()

	// Insert user with project isolation
	_, err := r.usersCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return usecase.ErrEmailTaken
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email within a specific Firestore project
func (r *MongoAuthRepository) GetUserByEmail(ctx context.Context, email, projectID string) (*model.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty")
	}

	var user model.User
	filter := bson.M{
		"email":     email,
		"projectId": projectID,
	}

	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID within a specific Firestore project
func (r *MongoAuthRepository) GetUserByID(ctx context.Context, id, projectID string) (*model.User, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}
	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty")
	}

	var user model.User
	filter := bson.M{
		"id":        id,
		"projectId": projectID,
	}

	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecase.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetUsersByProject retrieves all users for a specific Firestore project
func (r *MongoAuthRepository) GetUsersByProject(ctx context.Context, projectID string) ([]*model.User, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty")
	}

	filter := bson.M{"projectId": projectID}
	cursor, err := r.usersCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by project: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*model.User
	for cursor.Next(ctx) {
		var user model.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}

// CreateSession creates a new user session
func (r *MongoAuthRepository) CreateSession(ctx context.Context, session *model.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}
	session.CreatedAt = time.Now()
	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	_, err := r.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByID retrieves a session by ID
func (r *MongoAuthRepository) GetSessionByID(ctx context.Context, id string) (*model.Session, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	var session model.Session
	filter := bson.M{"id": id}

	err := r.sessionsCollection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a session by ID
func (r *MongoAuthRepository) DeleteSession(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	filter := bson.M{"id": id}
	result, err := r.sessionsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return usecase.ErrSessionNotFound
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (r *MongoAuthRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}

	filter := bson.M{"user_id": userID}
	_, err := r.sessionsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// Ensure MongoAuthRepository implements AuthRepository
var _ repository.AuthRepository = (*MongoAuthRepository)(nil)
