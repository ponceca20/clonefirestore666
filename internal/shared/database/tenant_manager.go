package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

// TenantManager manages database connections per tenant/organization
// Following Firestore's multi-tenant architecture
type TenantManager struct {
	client      *mongo.Client
	connections map[string]*mongo.Database // organizationID -> database
	mu          sync.RWMutex
	logger      logger.Logger
	config      *TenantConfig
}

// TenantConfig holds configuration for tenant database management
type TenantConfig struct {
	// Database naming strategy
	DatabasePrefix    string        `env:"DB_PREFIX" envDefault:"firestore_org_"`
	MaxConnections    int           `env:"MAX_TENANT_CONNECTIONS" envDefault:"100"`
	ConnectionTimeout time.Duration `env:"CONNECTION_TIMEOUT" envDefault:"30s"`

	// Auto-creation settings (like Firestore)
	AutoCreateDatabase bool `env:"AUTO_CREATE_DB" envDefault:"true"`

	// Connection pooling per tenant
	MaxPoolSize uint64 `env:"MAX_POOL_SIZE" envDefault:"10"`
	MinPoolSize uint64 `env:"MIN_POOL_SIZE" envDefault:"2"`
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(client *mongo.Client, config *TenantConfig, logger logger.Logger) *TenantManager {
	if config == nil {
		config = &TenantConfig{
			DatabasePrefix:     "firestore_org_",
			MaxConnections:     100,
			ConnectionTimeout:  30 * time.Second,
			AutoCreateDatabase: true,
			MaxPoolSize:        10,
			MinPoolSize:        2,
		}
	}

	return &TenantManager{
		client:      client,
		connections: make(map[string]*mongo.Database),
		logger:      logger,
		config:      config,
	}
}

// GetDatabaseForOrganization returns the MongoDB database for a specific organization
// This follows Firestore's pattern where each organization has isolated data
func (tm *TenantManager) GetDatabaseForOrganization(ctx context.Context, organizationID string) (*mongo.Database, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	// Sanitize organization ID for database name
	dbName := tm.getDatabaseName(organizationID)

	tm.mu.RLock()
	if db, exists := tm.connections[organizationID]; exists {
		tm.mu.RUnlock()
		return db, nil
	}
	tm.mu.RUnlock()

	// Double-check locking pattern
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if db, exists := tm.connections[organizationID]; exists {
		return db, nil
	}

	// Create new database connection
	db := tm.client.Database(dbName)

	// Auto-create database if enabled (like Firestore auto-creates projects)
	if tm.config.AutoCreateDatabase {
		if err := tm.ensureDatabaseExists(ctx, db); err != nil {
			return nil, fmt.Errorf("failed to ensure database exists: %w", err)
		}
	}

	tm.connections[organizationID] = db

	tm.logger.WithFields(map[string]interface{}{
		"organization_id": organizationID,
		"database_name":   dbName,
	}).Info("Created new database connection for organization")

	return db, nil
}

// GetDatabaseForProject returns the database for a specific project within an organization
// Following Firestore hierarchy: Organization → Project → Database
func (tm *TenantManager) GetDatabaseForProject(ctx context.Context, organizationID, projectID string) (*mongo.Database, error) {
	// In Firestore, projects belong to organizations, so we get the org database
	return tm.GetDatabaseForOrganization(ctx, organizationID)
}

// ListOrganizationDatabases lists all databases for organizations
func (tm *TenantManager) ListOrganizationDatabases(ctx context.Context) ([]string, error) {
	databaseNames, err := tm.client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	var orgDatabases []string
	for _, name := range databaseNames {
		if strings.HasPrefix(name, tm.config.DatabasePrefix) {
			orgDatabases = append(orgDatabases, name)
		}
	}

	return orgDatabases, nil
}

// CreateOrganizationDatabase explicitly creates a database for an organization
// This mimics Firestore's project creation
func (tm *TenantManager) CreateOrganizationDatabase(ctx context.Context, organizationID string) error {
	dbName := tm.getDatabaseName(organizationID)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if already exists
	if _, exists := tm.connections[organizationID]; exists {
		return fmt.Errorf("database for organization %s already exists", organizationID)
	}

	db := tm.client.Database(dbName)

	// Create a dummy collection to ensure database creation (MongoDB requirement)
	if err := tm.ensureDatabaseExists(ctx, db); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	tm.connections[organizationID] = db

	tm.logger.WithFields(map[string]interface{}{
		"organization_id": organizationID,
		"database_name":   dbName,
	}).Info("Created database for organization")

	return nil
}

// DeleteOrganizationDatabase deletes a database for an organization
func (tm *TenantManager) DeleteOrganizationDatabase(ctx context.Context, organizationID string) error {
	dbName := tm.getDatabaseName(organizationID)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Remove from connections
	delete(tm.connections, organizationID)

	// Drop the database
	if err := tm.client.Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	tm.logger.WithFields(map[string]interface{}{
		"organization_id": organizationID,
		"database_name":   dbName,
	}).Info("Deleted database for organization")

	return nil
}

// Close closes all database connections
func (tm *TenantManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Clear connections map
	tm.connections = make(map[string]*mongo.Database)

	tm.logger.Info("Closed all tenant database connections")
	return nil
}

// GetConnectionCount returns the number of active connections
func (tm *TenantManager) GetConnectionCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.connections)
}

// Private methods

// getDatabaseName generates a database name for an organization
func (tm *TenantManager) getDatabaseName(organizationID string) string {
	// Sanitize the organization ID for use as database name
	sanitized := strings.ToLower(organizationID)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	return tm.config.DatabasePrefix + sanitized
}

// ensureDatabaseExists creates the database by creating a metadata collection
func (tm *TenantManager) ensureDatabaseExists(ctx context.Context, db *mongo.Database) error {
	// Create a metadata collection to ensure database exists
	collection := db.Collection("_metadata")

	_, err := collection.InsertOne(ctx, map[string]interface{}{
		"type":       "database_metadata",
		"created_at": time.Now(),
		"version":    "1.0",
	})

	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	return nil
}

// ValidateOrganizationID validates an organization ID
func ValidateOrganizationID(organizationID string) error {
	if organizationID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}

	if len(organizationID) > 100 {
		return fmt.Errorf("organization ID too long (max 100 characters)")
	}

	// Check for valid characters (alphanumeric, hyphens, underscores, dots)
	for _, char := range organizationID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.') {
			return fmt.Errorf("organization ID contains invalid characters")
		}
	}

	return nil
}
