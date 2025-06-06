package model

import "time"

// Session represents a user session
type Session struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Token     string    `json:"token" bson:"token"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
