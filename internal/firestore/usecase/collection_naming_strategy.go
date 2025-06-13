package usecase

// FirestorePathCollectionStrategy implements direct mapping from Firestore path to MongoDB collection name.
type FirestorePathCollectionStrategy struct{}

func (s *FirestorePathCollectionStrategy) CollectionName(projectID, databaseID, collectionPath string) string {
	return "docs_" + projectID + "_" + databaseID + "_" + collectionPath
}

// OptimizedCollectionStrategy implements optimized naming for high-volume collections.
type OptimizedCollectionStrategy struct{}

func (s *OptimizedCollectionStrategy) CollectionName(projectID, databaseID, collectionPath string) string {
	// Example: hash or partitioning logic can be added here
	return "opt_" + projectID + "_" + databaseID + "_" + collectionPath
}
