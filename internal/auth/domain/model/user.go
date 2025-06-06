package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID           string             `json:"id" bson:"id,omitempty"`
	ObjectID     primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	TenantID     string             `json:"tenantID,omitempty" bson:"tenantID,omitempty"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	FirstName    string             `json:"firstName,omitempty" bson:"firstName,omitempty"`
	LastName     string             `json:"lastName,omitempty" bson:"lastName,omitempty"`
	AvatarURL    string             `json:"avatarUrl,omitempty" bson:"avatarUrl,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}
