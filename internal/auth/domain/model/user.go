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
	ProjectID    string             `json:"projectId" bson:"projectId"`   // Firestore project ID
	DatabaseID   string             `json:"databaseId" bson:"databaseId"` // Firestore database ID
	PasswordHash string             `json:"-" bson:"password_hash"`
	FirstName    string             `json:"firstName,omitempty" bson:"firstName,omitempty"`
	LastName     string             `json:"lastName,omitempty" bson:"lastName,omitempty"`
	AvatarURL    string             `json:"avatarUrl,omitempty" bson:"avatarUrl,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

// ValidateFields validates user fields for creation/update
func (u *User) ValidateFields() error {
	if u.Email == "" {
		return NewValidationError("email is required")
	}
	if u.ProjectID == "" {
		return NewValidationError("projectId is required")
	}
	if u.DatabaseID == "" {
		return NewValidationError("databaseId is required")
	}
	if u.FirstName == "" {
		return NewValidationError("firstName is required")
	}
	if u.LastName == "" {
		return NewValidationError("lastName is required")
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}
