package di

import (
	"fmt"

	authmodule "firestore-clone/internal/auth"
	authmongodb "firestore-clone/internal/auth/adapter/persistence/mongodb"
	authsecurity "firestore-clone/internal/auth/adapter/security"
	authconfig "firestore-clone/internal/auth/config"
	authrepo "firestore-clone/internal/auth/domain/repository"
	firestoremodule "firestore-clone/internal/firestore"
	firestoreauthclient "firestore-clone/internal/firestore/adapter/auth_client"
	firestoremongodb "firestore-clone/internal/firestore/adapter/persistence/mongodb"
	firestoreconfig "firestore-clone/internal/firestore/config"
	"firestore-clone/internal/firestore/domain/client"
	firestorerepo "firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

// MultitenantContainer provides dependency injection for multitenant architecture
type MultitenantContainer struct {
	// Database connections
	mongoClient   *mongo.Client
	masterDB      *mongo.Database
	tenantManager *database.TenantManager

	// Configuration
	authConfig      *authconfig.Config
	firestoreConfig *firestoreconfig.FirestoreConfig

	// Shared components
	logger   logger.Logger
	eventBus *eventbus.EventBus

	// Auth components
	authRepository authrepo.AuthRepository
	tokenService   authrepo.TokenService
	authModule     *authmodule.AuthModule

	// Firestore components
	firestoreRepository firestorerepo.FirestoreRepository
	organizationRepo    *firestoremongodb.OrganizationRepository
	securityRulesEngine firestorerepo.SecurityRulesEngine
	queryEngine         firestorerepo.QueryEngine
	authClient          client.AuthClient
	firestoreModule     *firestoremodule.FirestoreModule
}

// NewMultitenantContainer creates a new multitenant DI container
func NewMultitenantContainer(
	mongoClient *mongo.Client,
	tenantManager *database.TenantManager,
	authCfg *authconfig.Config,
	firestoreCfg *firestoreconfig.FirestoreConfig,
	log logger.Logger,
) (*MultitenantContainer, error) {
	container := &MultitenantContainer{
		mongoClient:     mongoClient,
		tenantManager:   tenantManager,
		authConfig:      authCfg,
		firestoreConfig: firestoreCfg,
		logger:          log,
	}

	// Initialize master database
	container.masterDB = mongoClient.Database(authCfg.DatabaseName)

	// Initialize event bus
	container.eventBus = eventbus.NewEventBus(log)

	// Initialize components
	if err := container.initializeAuthComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth components: %w", err)
	}

	if err := container.initializeFirestoreComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize firestore components: %w", err)
	}

	return container, nil
}

// initializeAuthComponents initializes authentication-related components
func (c *MultitenantContainer) initializeAuthComponents() error {
	// Initialize auth repository
	authRepo, err := authmongodb.NewMongoAuthRepository(c.masterDB)
	if err != nil {
		return fmt.Errorf("failed to create auth repository: %w", err)
	}
	c.authRepository = authRepo

	// Initialize token service
	tokenSvc, err := authsecurity.NewJWTokenService(c.authConfig)
	if err != nil {
		return fmt.Errorf("failed to create token service: %w", err)
	}
	c.tokenService = tokenSvc

	// Initialize auth module
	authMod, err := authmodule.NewAuthModule(c.masterDB, c.authConfig)
	if err != nil {
		return fmt.Errorf("failed to create auth module: %w", err)
	}
	c.authModule = authMod

	return nil
}

// initializeFirestoreComponents initializes Firestore-related components
func (c *MultitenantContainer) initializeFirestoreComponents() error {
	// Initialize organization repository
	c.organizationRepo = firestoremongodb.NewOrganizationRepository(
		c.mongoClient,
		c.masterDB,
		c.tenantManager,
		c.logger,
	)

	// Initialize tenant-aware Firestore repository
	c.firestoreRepository = firestoremongodb.NewTenantAwareDocumentRepository(
		c.mongoClient,
		c.tenantManager,
		c.eventBus,
		c.logger,
	)

	// Initialize security rules engine
	c.securityRulesEngine = firestoremongodb.NewSecurityRulesEngine(c.masterDB, c.logger)

	// Initialize query engine
	c.queryEngine = firestoremongodb.NewMongoQueryEngine(c.masterDB)

	// Initialize auth client
	c.authClient = firestoreauthclient.NewSimpleAuthClient()

	// Initialize Firestore module
	firestoreMod, err := firestoremodule.NewFirestoreModule(
		c.authClient,
		c.logger,
		c.mongoClient,
		c.masterDB,
	)
	if err != nil {
		return fmt.Errorf("failed to create firestore module: %w", err)
	}
	c.firestoreModule = firestoreMod

	return nil
}

// GetAuthRepository returns the auth repository
func (c *MultitenantContainer) GetAuthRepository() authrepo.AuthRepository {
	return c.authRepository
}

// GetTokenService returns the token service
func (c *MultitenantContainer) GetTokenService() authrepo.TokenService {
	return c.tokenService
}

// GetAuthModule returns the auth module
func (c *MultitenantContainer) GetAuthModule() *authmodule.AuthModule {
	return c.authModule
}

// GetFirestoreRepository returns the Firestore repository
func (c *MultitenantContainer) GetFirestoreRepository() firestorerepo.FirestoreRepository {
	return c.firestoreRepository
}

// GetOrganizationRepository returns the organization repository
func (c *MultitenantContainer) GetOrganizationRepository() *firestoremongodb.OrganizationRepository {
	return c.organizationRepo
}

// GetSecurityRulesEngine returns the security rules engine
func (c *MultitenantContainer) GetSecurityRulesEngine() firestorerepo.SecurityRulesEngine {
	return c.securityRulesEngine
}

// GetQueryEngine returns the query engine
func (c *MultitenantContainer) GetQueryEngine() firestorerepo.QueryEngine {
	return c.queryEngine
}

// GetAuthClient returns the auth client
func (c *MultitenantContainer) GetAuthClient() client.AuthClient {
	return c.authClient
}

// GetFirestoreModule returns the Firestore module
func (c *MultitenantContainer) GetFirestoreModule() *firestoremodule.FirestoreModule {
	return c.firestoreModule
}

// GetTenantManager returns the tenant manager
func (c *MultitenantContainer) GetTenantManager() *database.TenantManager {
	return c.tenantManager
}

// GetEventBus returns the event bus
func (c *MultitenantContainer) GetEventBus() *eventbus.EventBus {
	return c.eventBus
}

// GetLogger returns the logger
func (c *MultitenantContainer) GetLogger() logger.Logger {
	return c.logger
}

// Close closes all resources
func (c *MultitenantContainer) Close() error {
	// Close modules
	if c.authModule != nil {
		c.authModule.Stop()
	}

	if c.firestoreModule != nil {
		c.firestoreModule.Stop()
	}

	// Close tenant manager
	if c.tenantManager != nil {
		c.tenantManager.Close()
	}

	return nil
}
