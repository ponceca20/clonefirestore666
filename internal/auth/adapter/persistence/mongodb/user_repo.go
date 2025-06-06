package mongodb

import (
	"context"
	"time"

	"firestore-clone/internal/auth/domain/model"
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

	// Create indexes
	ctx := context.Background()

	// Email index for users (unique)
	emailIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := repo.usersCollection.Indexes().CreateOne(ctx, emailIndex)
	if err != nil {
		return nil, err
	}

	// ID index for users (for UUID lookups)
	idIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetSparse(true), // Sparse because not all documents may have this field
	}

	_, err = repo.usersCollection.Indexes().CreateOne(ctx, idIndex)
	if err != nil {
		return nil, err
	}

	// Token index for sessions
	tokenIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "token", Value: 1}},
	}

	_, err = repo.sessionsCollection.Indexes().CreateOne(ctx, tokenIndex)
	if err != nil {
		return nil, err
	}

	// TTL index for sessions
	expiresAtIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err = repo.sessionsCollection.Indexes().CreateOne(ctx, expiresAtIndex)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// CreateUser creates a new user in the database
func (r *MongoAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Generate an ID if not provided
	if user.ID == "" {
		user.ID = primitive.NewObjectID().Hex()
	}

	// Use the provided user ID instead of generating a new ObjectID
	// Convert the user ID to ObjectID for MongoDB storage if it's a valid ObjectID hex
	// Otherwise, store it as is
	var doc bson.M
	if objectID, err := primitive.ObjectIDFromHex(user.ID); err == nil {
		// If the ID is a valid ObjectID hex, use it as ObjectID
		doc = bson.M{
			"_id":           objectID,
			"email":         user.Email,
			"password_hash": user.PasswordHash,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
			// tenantID will be added below
		}
	} else {
		// If the ID is not a valid ObjectID (like UUID), store it as a string in id field
		doc = bson.M{
			"id":            user.ID,
			"email":         user.Email,
			"password_hash": user.PasswordHash,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
			// tenantID will be added below
		}
	}
	// Add tenantID if it's not empty
	if user.TenantID != "" {
		doc["tenantID"] = user.TenantID
	}

	_, err := r.usersCollection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return usecase.ErrEmailTaken
		}
		return err
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *MongoAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}

	// Ensure ID field is populated
	if user.ID == "" && !user.ObjectID.IsZero() {
		user.ID = user.ObjectID.Hex()
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *MongoAuthRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	var err error

	// Try to find by ObjectID first (if the ID is a valid ObjectID hex)
	if objectID, objErr := primitive.ObjectIDFromHex(id); objErr == nil {
		err = r.usersCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	} else {
		// Try to find by string ID field (for UUIDs)
		err = r.usersCollection.FindOne(ctx, bson.M{"id": id}).Decode(&user)
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, usecase.ErrUserNotFound
		}
		return nil, err
	}

	// Ensure ID field is populated
	if user.ID == "" && !user.ObjectID.IsZero() {
		user.ID = user.ObjectID.Hex()
	}

	return &user, nil
}

// CreateSession creates a new session
func (r *MongoAuthRepository) CreateSession(ctx context.Context, session *model.Session) error {
	now := time.Now()
	session.CreatedAt = now

	result, err := r.sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		return err
	}

	// Set the generated ID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		session.ID = oid.Hex()
	}

	return nil
}

// GetSessionByID retrieves a session by ID
func (r *MongoAuthRepository) GetSessionByID(ctx context.Context, id string) (*model.Session, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, usecase.ErrSessionNotFound
	}

	var session model.Session
	err = r.sessionsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, err
	}

	return &session, nil
}

// DeleteSession deletes a session by ID
func (r *MongoAuthRepository) DeleteSession(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return usecase.ErrSessionNotFound
	}

	result, err := r.sessionsCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return usecase.ErrSessionNotFound
	}

	return nil
}
