package model

// CollectionMetadata holds metadata for a Firestore collection in the new architecture.
type CollectionMetadata struct {
	ProjectID      string
	DatabaseID     string
	CollectionPath string
	PhysicalName   string // MongoDB collection name
	CreatedAt      int64
	UpdatedAt      int64
}
