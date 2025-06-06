package model

// Collection represents a collection in Firestore.
type Collection struct {
	ID   string `json:"id"`
	Path string `json:"path"` // Full path to the collection, e.g., "users/userId/posts"
}
