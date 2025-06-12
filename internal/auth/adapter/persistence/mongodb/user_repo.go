package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Collection names following Firestore naming conventions
	usersCollectionName    = "users"
	sessionsCollectionName = "sessions"

	// Index creation timeout
	indexCreationTimeout = 10 * time.Second

	// Error messages
	errMsgUserNil              = "user cannot be nil"
	errMsgUserIDEmpty          = "user ID cannot be empty"
	errMsgEmailEmpty           = "email cannot be empty"
	errMsgSessionNotFound      = "session not found"
	errMsgCreateUserIndexes    = "failed to create user indexes"
	errMsgCreateSessionIndexes = "failed to create session indexes"
)

// MongoAuthRepository implements AuthRepository using MongoDB following Firestore patterns
// This adapter translates domain operations to MongoDB-specific operations
type MongoAuthRepository struct {
	db                 *mongo.Database
	usersCollection    *mongo.Collection
	sessionsCollection *mongo.Collection
}

// NewMongoAuthRepository creates a new MongoDB auth repository with proper indexing
// following Firestore collection and document structure patterns
func NewMongoAuthRepository(db *mongo.Database) (repository.AuthRepository, error) {
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}

	userCollection := db.Collection(usersCollectionName)
	sessionCollection := db.Collection(sessionsCollectionName)

	repo := &MongoAuthRepository{
		db:                 db,
		usersCollection:    userCollection,
		sessionsCollection: sessionCollection,
	}

	// Create indexes with proper error handling
	if err := repo.createIndexes(); err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	return repo, nil
}

// createIndexes creates all necessary indexes for optimal query performance
// following Firestore indexing patterns and best practices
func (r *MongoAuthRepository) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), indexCreationTimeout)
	defer cancel()

	if err := r.createUserIndexes(ctx); err != nil {
		return fmt.Errorf("%s: %w", errMsgCreateUserIndexes, err)
	}

	if err := r.createSessionIndexes(ctx); err != nil {
		return fmt.Errorf("%s: %w", errMsgCreateSessionIndexes, err)
	}

	return nil
}

// createUserIndexes creates indexes for the users collection
func (r *MongoAuthRepository) createUserIndexes(ctx context.Context) error {
	userIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_users_email_unique"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_users_user_id_unique"),
		},
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_users_tenant_id"),
		},
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
			},
			Options: options.Index().SetName("idx_users_organization_id"),
		},
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_users_tenant_email_unique"),
		},
		{
			Keys: bson.D{
				{Key: "deleted_at", Value: 1},
			},
			Options: options.Index().SetName("idx_users_deleted_at"),
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_users_created_at_desc"),
		},
	}

	_, err := r.usersCollection.Indexes().CreateMany(ctx, userIndexes)
	return err
}

// createSessionIndexes creates indexes for the sessions collection
func (r *MongoAuthRepository) createSessionIndexes(ctx context.Context) error {
	sessionIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetName("idx_sessions_user_id"),
		},
		{
			Keys: bson.D{
				{Key: "expires_at", Value: 1},
			},
			Options: options.Index().
				SetExpireAfterSeconds(0).
				SetName("idx_sessions_expires_at_ttl"),
		},
		{
			Keys: bson.D{
				{Key: "token", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_sessions_token_unique"),
		},
	}

	_, err := r.sessionsCollection.Indexes().CreateMany(ctx, sessionIndexes)
	return err
}

// User operations

// CreateUser creates a new user in the database following Firestore document creation patterns
func (r *MongoAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New(errMsgUserNil)
	}

	// Generate user ID if not provided (similar to Firestore auto-generated IDs)
	if user.UserID == "" {
		user.UserID = primitive.NewObjectID().Hex()
	}

	// Set timestamps following Firestore timestamp patterns
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Ensure required fields are present
	if err := r.validateUserForCreation(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	_, err := r.usersCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID following Firestore document retrieval patterns
func (r *MongoAuthRepository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	if userID == "" {
		return nil, errors.New(errMsgUserIDEmpty)
	}

	var user model.User
	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil, // Only return non-deleted users
	}

	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by their email following Firestore query patterns
func (r *MongoAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, errors.New(errMsgEmailEmpty)
	}

	var user model.User
	filter := bson.M{
		"email":      email,
		"deleted_at": nil, // Only return non-deleted users
	}

	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user following Firestore document update patterns
func (r *MongoAuthRepository) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New(errMsgUserNil)
	}

	if user.UserID == "" {
		return errors.New(errMsgUserIDEmpty)
	}

	// Update timestamp
	user.UpdatedAt = time.Now()

	// Validate user data before update
	if err := r.validateUserForUpdate(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	filter := bson.M{
		"user_id":    user.UserID,
		"deleted_at": nil, // Only update non-deleted users
	}

	result, err := r.usersCollection.ReplaceOne(ctx, filter, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// DeleteUser performs soft delete on a user following Firestore soft delete patterns
func (r *MongoAuthRepository) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New(errMsgUserIDEmpty)
	}

	now := time.Now()
	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil, // Only delete non-deleted users
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
		},
	}

	result, err := r.usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// ListUsers retrieves a paginated list of users for a tenant following Firestore query patterns
func (r *MongoAuthRepository) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 100 // Default limit similar to Firestore default
	}
	if offset < 0 {
		offset = 0
	}

	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": nil,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1}) // Most recent first

	cursor, err := r.usersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
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
	return users, cursor.Err()
}

// validateUserForCreation validates user data before creation
func (r *MongoAuthRepository) validateUserForCreation(user *model.User) error {
	if user.Email == "" {
		return errors.New("email is required")
	}
	if user.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// validateUserForUpdate validates user data before update
func (r *MongoAuthRepository) validateUserForUpdate(user *model.User) error {
	if user.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// Password operations

// UpdatePassword updates a user's password hash following Firestore security patterns
func (r *MongoAuthRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	if userID == "" {
		return errors.New(errMsgUserIDEmpty)
	}
	if hashedPassword == "" {
		return errors.New("hashed password cannot be empty")
	}

	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil, // Only update non-deleted users
	}

	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		},
	}

	result, err := r.usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// VerifyPassword verifies if the provided password hash matches the stored one
func (r *MongoAuthRepository) VerifyPassword(ctx context.Context, userID, hashedPassword string) (bool, error) {
	if userID == "" {
		return false, errors.New(errMsgUserIDEmpty)
	}

	var user model.User
	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil, // Only verify for non-deleted users
	}

	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, model.ErrUserNotFound
		}
		return false, fmt.Errorf("failed to get user for password verification: %w", err)
	}

	return user.Password == hashedPassword, nil
}

// Session operations

// CreateSession creates a new user session following Firestore document patterns
func (r *MongoAuthRepository) CreateSession(ctx context.Context, session *model.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	// Generate session ID if not provided
	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	// Set creation timestamp
	session.CreatedAt = time.Now()

	// Validate session data
	if err := r.validateSessionForCreation(session); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	_, err := r.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by its ID
func (r *MongoAuthRepository) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if sessionID == "" {
		return nil, errors.New("session ID cannot be empty")
	}

	var session model.Session
	filter := bson.M{"_id": sessionID}

	err := r.sessionsCollection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(errMsgSessionNotFound)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session has expired")
	}

	return &session, nil
}

// GetSessionsByUserID retrieves all active sessions for a user
func (r *MongoAuthRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	if userID == "" {
		return nil, errors.New(errMsgUserIDEmpty)
	}

	filter := bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$gt": time.Now()}, // Only active sessions
	}

	cursor, err := r.sessionsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for user: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*model.Session
	for cursor.Next(ctx) {
		var session model.Session
		if err := cursor.Decode(&session); err != nil {
			return nil, fmt.Errorf("failed to decode session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, cursor.Err()
}

// DeleteSession removes a specific session
func (r *MongoAuthRepository) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return errors.New("session ID cannot be empty")
	}

	filter := bson.M{"_id": sessionID}
	result, err := r.sessionsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New(errMsgSessionNotFound)
	}

	return nil
}

// DeleteSessionsByUserID removes all sessions for a specific user
func (r *MongoAuthRepository) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New(errMsgUserIDEmpty)
	}

	filter := bson.M{"user_id": userID}
	_, err := r.sessionsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete sessions for user: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions from the database
func (r *MongoAuthRepository) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now()
	filter := bson.M{"expires_at": bson.M{"$lt": now}}

	result, err := r.sessionsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	// Log the number of sessions cleaned up (could be useful for monitoring)
	_ = result.DeletedCount

	return nil
}

// validateSessionForCreation validates session data before creation
func (r *MongoAuthRepository) validateSessionForCreation(session *model.Session) error {
	if session.UserID == "" {
		return errors.New("user ID is required")
	}
	if session.Token == "" {
		return errors.New("token is required")
	}
	if session.ExpiresAt.IsZero() {
		return errors.New("expiration time is required")
	}
	if session.ExpiresAt.Before(time.Now()) {
		return errors.New("expiration time must be in the future")
	}
	return nil
}

// Tenant operations

// GetUsersByTenant retrieves all users for a specific tenant following Firestore query patterns
func (r *MongoAuthRepository) GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID cannot be empty")
	}

	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": nil,
	}

	cursor, err := r.usersCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by tenant: %w", err)
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

	return users, cursor.Err()
}

// CheckUserTenantAccess verifies if a user has access to a specific tenant
func (r *MongoAuthRepository) CheckUserTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	if userID == "" {
		return false, errors.New(errMsgUserIDEmpty)
	}
	if tenantID == "" {
		return false, errors.New("tenant ID cannot be empty")
	}

	filter := bson.M{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"deleted_at": nil,
	}

	count, err := r.usersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check user tenant access: %w", err)
	}

	return count > 0, nil
}

// AddUserToTenant assigns a user to a specific tenant
func (r *MongoAuthRepository) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	if userID == "" {
		return errors.New(errMsgUserIDEmpty)
	}
	if tenantID == "" {
		return errors.New("tenant ID cannot be empty")
	}

	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"tenant_id":  tenantID,
			"updated_at": time.Now(),
		},
	}

	result, err := r.usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add user to tenant: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// RemoveUserFromTenant removes a user from a specific tenant
func (r *MongoAuthRepository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	if userID == "" {
		return errors.New(errMsgUserIDEmpty)
	}
	if tenantID == "" {
		return errors.New("tenant ID cannot be empty")
	}

	filter := bson.M{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"deleted_at": nil,
	}

	update := bson.M{
		"$unset": bson.M{"tenant_id": ""},
		"$set":   bson.M{"updated_at": time.Now()},
	}

	result, err := r.usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove user from tenant: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// Health check

// HealthCheck performs a health check on the database connection
// following Firestore health check patterns for monitoring and observability
func (r *MongoAuthRepository) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := r.db.Client().Ping(ctx, nil); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check collections accessibility
	if err := r.checkCollectionHealth(ctx, r.usersCollection); err != nil {
		return fmt.Errorf("users collection health check failed: %w", err)
	}

	if err := r.checkCollectionHealth(ctx, r.sessionsCollection); err != nil {
		return fmt.Errorf("sessions collection health check failed: %w", err)
	}

	return nil
}

// checkCollectionHealth performs a basic health check on a collection
func (r *MongoAuthRepository) checkCollectionHealth(ctx context.Context, collection *mongo.Collection) error {
	// Try to count documents (lightweight operation)
	_, err := collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return fmt.Errorf("collection health check failed: %w", err)
	}
	return nil
}
