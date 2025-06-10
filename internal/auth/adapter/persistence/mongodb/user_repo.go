package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/auth/domain/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoAuthRepository implements AuthRepository using MongoDB
type MongoAuthRepository struct {
	db                 *mongo.Database
	usersCollection    *mongo.Collection
	sessionsCollection *mongo.Collection
}

// NewMongoAuthRepository creates a new MongoDB auth repository
func NewMongoAuthRepository(db *mongo.Database) (repository.AuthRepository, error) {
	userCollection := db.Collection("users")
	sessionCollection := db.Collection("sessions")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// User indexes
	_, err := userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"email", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"user_id", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"tenant_id", 1}}},
		{Keys: bson.D{{"organization_id", 1}}},
		{Keys: bson.D{{"tenant_id", 1}, {"email", 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user indexes: %w", err)
	}
	// Session indexes
	_, err = sessionCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"user_id", 1}}},
		{Keys: bson.D{{"expires_at", 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{"token", 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session indexes: %w", err)
	}

	return &MongoAuthRepository{
		db:                 db,
		usersCollection:    userCollection,
		sessionsCollection: sessionCollection,
	}, nil
}

// User operations

func (r *MongoAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	if user.UserID == "" {
		user.UserID = primitive.NewObjectID().Hex()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.usersCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *MongoAuthRepository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	var user model.User
	err := r.usersCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

func (r *MongoAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	var user model.User
	err := r.usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

func (r *MongoAuthRepository) UpdateUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()

	result, err := r.usersCollection.ReplaceOne(ctx, bson.M{"user_id": user.UserID}, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

func (r *MongoAuthRepository) DeleteUser(ctx context.Context, userID string) error {
	now := time.Now()
	result, err := r.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"deleted_at": now}},
	)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

func (r *MongoAuthRepository) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*model.User, error) {
	filter := bson.M{"tenant_id": tenantID, "deleted_at": nil}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1})

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

// Password operations

func (r *MongoAuthRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	result, err := r.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"password": hashedPassword, "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

func (r *MongoAuthRepository) VerifyPassword(ctx context.Context, userID, hashedPassword string) (bool, error) {
	var user model.User
	err := r.usersCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, model.ErrUserNotFound
		}
		return false, fmt.Errorf("failed to get user for password verification: %w", err)
	}

	return user.Password == hashedPassword, nil
}

// Session operations

func (r *MongoAuthRepository) CreateSession(ctx context.Context, session *model.Session) error {
	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	session.CreatedAt = time.Now()

	_, err := r.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *MongoAuthRepository) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := r.sessionsCollection.FindOne(ctx, bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *MongoAuthRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	filter := bson.M{"user_id": userID}

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

func (r *MongoAuthRepository) DeleteSession(ctx context.Context, sessionID string) error {
	result, err := r.sessionsCollection.DeleteOne(ctx, bson.M{"_id": sessionID})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (r *MongoAuthRepository) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	_, err := r.sessionsCollection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete sessions for user: %w", err)
	}

	return nil
}

func (r *MongoAuthRepository) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now()
	_, err := r.sessionsCollection.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lt": now}})
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// Tenant operations

func (r *MongoAuthRepository) GetUsersByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	filter := bson.M{"tenant_id": tenantID, "deleted_at": nil}

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

func (r *MongoAuthRepository) CheckUserTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	count, err := r.usersCollection.CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"deleted_at": nil,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check user tenant access: %w", err)
	}

	return count > 0, nil
}

func (r *MongoAuthRepository) AddUserToTenant(ctx context.Context, userID, tenantID string) error {
	result, err := r.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"tenant_id": tenantID, "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to add user to tenant: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

func (r *MongoAuthRepository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	result, err := r.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID, "tenant_id": tenantID},
		bson.M{"$unset": bson.M{"tenant_id": ""}, "$set": bson.M{"updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove user from tenant: %w", err)
	}

	if result.MatchedCount == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// Health check

func (r *MongoAuthRepository) HealthCheck(ctx context.Context) error {
	return r.db.Client().Ping(ctx, nil)
}
