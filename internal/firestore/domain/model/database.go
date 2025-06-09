package model

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Database represents a Firestore database within a project
// projects/{PROJECT_ID}/databases/{DATABASE_ID}
type Database struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProjectID  string             `json:"projectId" bson:"project_id"`
	DatabaseID string             `json:"databaseId" bson:"database_id"`

	// Database configuration
	DisplayName     string          `json:"displayName,omitempty" bson:"display_name,omitempty"`
	LocationID      string          `json:"locationId" bson:"location_id"`
	Type            DatabaseType    `json:"type" bson:"type"`
	ConcurrencyMode ConcurrencyMode `json:"concurrencyMode" bson:"concurrency_mode"`

	// Database settings
	AppEngineIntegrationMode AppEngineIntegrationMode `json:"appEngineIntegrationMode" bson:"app_engine_integration_mode"`
	KeyPrefix                string                   `json:"keyPrefix,omitempty" bson:"key_prefix,omitempty"`

	// Metadata
	CreatedAt              time.Time     `json:"createdAt" bson:"created_at"`
	UpdatedAt              time.Time     `json:"updatedAt" bson:"updated_at"`
	VersionRetentionPeriod time.Duration `json:"versionRetentionPeriod" bson:"version_retention_period"`

	// State and lifecycle
	State     DatabaseState `json:"state" bson:"state"`
	DeletedAt *time.Time    `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`

	// Performance and limits
	PointInTimeRecoveryEnablement PITREnablement        `json:"pointInTimeRecoveryEnablement" bson:"point_in_time_recovery_enablement"`
	DeleteProtectionState         DeleteProtectionState `json:"deleteProtectionState" bson:"delete_protection_state"`

	// Stats and usage
	DocumentCount int64 `json:"documentCount" bson:"document_count"`
	StorageSize   int64 `json:"storageSize" bson:"storage_size"`
}

// DatabaseType represents the type of Firestore database
type DatabaseType string

const (
	DatabaseTypeFirestoreNative DatabaseType = "FIRESTORE_NATIVE"
	DatabaseTypeDatastoreMode   DatabaseType = "DATASTORE_MODE"
)

// ConcurrencyMode represents the concurrency mode of the database
type ConcurrencyMode string

const (
	ConcurrencyModeOptimistic  ConcurrencyMode = "OPTIMISTIC"
	ConcurrencyModePessimistic ConcurrencyMode = "PESSIMISTIC"
)

// AppEngineIntegrationMode represents how the database integrates with App Engine
type AppEngineIntegrationMode string

const (
	AppEngineIntegrationEnabled  AppEngineIntegrationMode = "ENABLED"
	AppEngineIntegrationDisabled AppEngineIntegrationMode = "DISABLED"
)

// DatabaseState represents the current state of a database
type DatabaseState string

const (
	DatabaseStateCreating        DatabaseState = "CREATING"
	DatabaseStateActive          DatabaseState = "ACTIVE"
	DatabaseStateDeleting        DatabaseState = "DELETING"
	DatabaseStateDeleteRequested DatabaseState = "DELETE_REQUESTED"
)

// PITREnablement represents Point-in-Time Recovery configuration
type PITREnablement string

const (
	PITREnabled  PITREnablement = "POINT_IN_TIME_RECOVERY_ENABLED"
	PITRDisabled PITREnablement = "POINT_IN_TIME_RECOVERY_DISABLED"
)

// DeleteProtectionState represents delete protection configuration
type DeleteProtectionState string

const (
	DeleteProtectionEnabled  DeleteProtectionState = "DELETE_PROTECTION_ENABLED"
	DeleteProtectionDisabled DeleteProtectionState = "DELETE_PROTECTION_DISABLED"
)

// GetResourceName returns the full resource name for this database
func (d *Database) GetResourceName() string {
	return "projects/" + d.ProjectID + "/databases/" + d.DatabaseID
}

// IsActive returns true if the database is in active state
func (d *Database) IsActive() bool {
	return d.State == DatabaseStateActive
}

// ValidateDatabaseID validates a Firestore database ID format
func ValidateDatabaseID(databaseID string) error {
	if len(databaseID) == 0 {
		return ErrInvalidDatabaseID
	}

	// Special case for default database
	if databaseID == "(default)" {
		return nil
	}

	// Database ID must be 1-63 characters, lowercase letters, numbers, and hyphens
	// Must start and end with alphanumeric character
	if len(databaseID) > 63 {
		return ErrInvalidDatabaseIDLength
	}

	// Check first and last character
	first := rune(databaseID[0])
	last := rune(databaseID[len(databaseID)-1])

	if !isAlphanumeric(first) || !isAlphanumeric(last) {
		return ErrInvalidDatabaseIDFormat
	}

	// Check all characters
	for _, r := range databaseID {
		if !isAlphanumeric(r) && r != '-' {
			return ErrInvalidDatabaseIDFormat
		}
	}

	return nil
}

// NewDefaultDatabase creates a new default database for a project
func NewDefaultDatabase(projectID string) *Database {
	now := time.Now()
	return &Database{
		ProjectID:                     projectID,
		DatabaseID:                    "(default)",
		DisplayName:                   "Default Database",
		LocationID:                    "us-central1", // Default location
		Type:                          DatabaseTypeFirestoreNative,
		ConcurrencyMode:               ConcurrencyModeOptimistic,
		AppEngineIntegrationMode:      AppEngineIntegrationDisabled,
		CreatedAt:                     now,
		UpdatedAt:                     now,
		VersionRetentionPeriod:        24 * time.Hour, // 1 day default
		State:                         DatabaseStateActive,
		PointInTimeRecoveryEnablement: PITRDisabled,
		DeleteProtectionState:         DeleteProtectionDisabled,
		DocumentCount:                 0,
		StorageSize:                   0,
	}
}

// Helper function to check if a rune is alphanumeric
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// Common database-related errors
var (
	ErrInvalidDatabaseID       = errors.New("database ID cannot be empty")
	ErrInvalidDatabaseIDLength = errors.New("database ID must be 1-63 characters")
	ErrInvalidDatabaseIDFormat = errors.New("database ID must contain only lowercase letters, numbers, and hyphens, and start/end with alphanumeric")
	ErrDatabaseNotFound        = errors.New("database not found")
	ErrDatabaseInactive        = errors.New("database is not active")
	ErrDefaultDatabaseRequired = errors.New("project must have a default database")
)
