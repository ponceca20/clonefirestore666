package firestore

import ( // Added imports
	httpadapter "firestore-clone/internal/firestore/adapter/http"
	redispersistence "firestore-clone/internal/firestore/adapter/persistence"
	mongodbpersistence "firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/firestore/config"
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/domain/repository" // May keep for interfaces
	"firestore-clone/internal/firestore/domain/service"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/logger"

	authhttp "firestore-clone/internal/auth/adapter/http" // For auth middleware

	"github.com/gofiber/fiber/v2" // For RegisterRoutes parameter
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// FirestoreModule represents the core Firestore module with multi-tenant support.
type FirestoreModule struct {
	Config           *config.FirestoreConfig
	AuthClient       client.AuthClient                 // Client to interact with Auth module
	TenantAwareRepo  repository.FirestoreRepository    // Multi-tenant repository
	QueryEngine      repository.QueryEngine            // MongoDB query engine implementation
	SecurityRules    repository.SecurityRulesEngine    // MongoDB security rules engine implementation
	FirestoreUsecase usecase.FirestoreUsecaseInterface // Interface type
	RealtimeUsecase  usecase.RealtimeUsecase           // Enhanced real-time usecase (100% Firestore compatible)
	SecurityUsecase  usecase.SecurityUsecase           // Interface type
	Logger           logger.Logger

	// Multi-tenant components
	TenantManager       *database.TenantManager
	OrganizationRepo    *mongodbpersistence.OrganizationRepository
	OrganizationHandler *httpadapter.OrganizationHandler

	// Redis components for distributed event storage
	RedisClient     *redis.Client
	RedisEventStore usecase.EventStore
}

// NewFirestoreModule creates and initializes a new Firestore module with multi-tenant support.
func NewFirestoreModule(
	authClient client.AuthClient,
	log logger.Logger,
	mongoClient *mongo.Client, // MongoDB client for multi-tenant
	masterDB *mongo.Database, // Master database for organization metadata
	redisClient *redis.Client, // Redis client for distributed event storage
) (*FirestoreModule, error) {
	log.Info("Initializing Firestore Module with Multi-Tenant Support...")

	// Load Firestore configuration, fallback to defaults if environment variables are not set
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Warn("Failed to load Firestore config from environment, using defaults", "error", err)
		cfg = config.DefaultFirestoreConfig()
	}
	log.Info("Firestore configuration loaded successfully.")

	// Initialize EventBus
	eventBus := eventbus.NewEventBus(log)

	// Initialize TenantManager for multi-tenant support
	tenantConfig := &database.TenantConfig{
		DatabasePrefix:     "firestore_org_",
		MaxConnections:     100,
		AutoCreateDatabase: true,
	}
	tenantManager := database.NewTenantManager(mongoClient, tenantConfig, log)
	log.Info("TenantManager initialized successfully.")

	// Initialize OrganizationRepository
	orgRepo := mongodbpersistence.NewOrganizationRepository(mongoClient, masterDB, tenantManager, log)
	log.Info("OrganizationRepository initialized successfully.")

	// Initialize TenantAwareDocumentRepository
	tenantAwareRepo := mongodbpersistence.NewTenantAwareDocumentRepository(mongoClient, tenantManager, eventBus, log)
	log.Info("TenantAwareDocumentRepository initialized successfully.")

	// Initialize query engine with tenant-aware MongoDB implementation
	queryEngine := mongodbpersistence.NewTenantAwareQueryEngine(mongoClient, tenantManager, log)
	log.Info("TenantAwareQueryEngine initialized successfully.") // Initialize security rules engine
	securityRulesEngine := mongodbpersistence.NewSecurityRulesEngine(masterDB, log)
	log.Info("SecurityRulesEngine initialized successfully.") // Initialize projection service
	projectionService := service.NewProjectionService()
	log.Info("ProjectionService initialized successfully.")

	// Initialize Redis Event Store for distributed realtime events
	redisEventStore := redispersistence.NewRedisEventStore(redisClient, log)
	log.Info("RedisEventStore initialized successfully.")

	// Initialize use cases with enhanced real-time capabilities using Redis
	realtimeUC := usecase.NewRealtimeUsecaseWithEventStore(log, redisEventStore) // Enhanced with Redis persistence
	securityUC := usecase.NewSecurityUsecase(securityRulesEngine, log)

	// Initialize FirestoreUsecase with tenant-aware repository and projection service
	firestoreUC := usecase.NewFirestoreUsecase(tenantAwareRepo, securityRulesEngine, queryEngine, projectionService, log)

	// Initialize OrganizationHandler
	orgHandler := httpadapter.NewOrganizationHandler(orgRepo)
	log.Info("OrganizationHandler initialized successfully.")
	return &FirestoreModule{
		Config:           cfg,
		AuthClient:       authClient,
		Logger:           log,
		TenantAwareRepo:  tenantAwareRepo,
		QueryEngine:      queryEngine,
		SecurityRules:    securityRulesEngine,
		FirestoreUsecase: firestoreUC,
		RealtimeUsecase:  realtimeUC,
		SecurityUsecase:  securityUC, TenantManager: tenantManager,
		OrganizationRepo:    orgRepo,
		OrganizationHandler: orgHandler,
		RedisClient:         redisClient,
		RedisEventStore:     redisEventStore,
	}, nil
}

// NewFirestoreModuleWithConfig creates and initializes a new Firestore module with provided configuration.
// This constructor is useful for testing or when you want to provide custom configuration.
func NewFirestoreModuleWithConfig(
	authClient client.AuthClient,
	log logger.Logger,
	mongoClient *mongo.Client, // MongoDB client for multi-tenant
	masterDB *mongo.Database, // Master database for organization metadata
	cfg *config.FirestoreConfig, // Provided configuration
	redisClient *redis.Client, // Redis client for caching
) (*FirestoreModule, error) {
	log.Info("Initializing Firestore Module with Multi-Tenant Support...")

	// Use provided configuration
	if cfg == nil {
		cfg = config.DefaultFirestoreConfig()
		log.Info("No configuration provided, using defaults.")
	}
	log.Info("Firestore configuration set successfully.")

	// Initialize EventBus
	eventBus := eventbus.NewEventBus(log)

	// Initialize TenantManager for multi-tenant support
	tenantConfig := &database.TenantConfig{
		DatabasePrefix:     "firestore_org_",
		MaxConnections:     100,
		AutoCreateDatabase: true,
	}
	tenantManager := database.NewTenantManager(mongoClient, tenantConfig, log)
	log.Info("TenantManager initialized successfully.")

	// Initialize OrganizationRepository
	orgRepo := mongodbpersistence.NewOrganizationRepository(mongoClient, masterDB, tenantManager, log)
	log.Info("OrganizationRepository initialized successfully.")

	// Initialize TenantAwareDocumentRepository
	tenantAwareRepo := mongodbpersistence.NewTenantAwareDocumentRepository(mongoClient, tenantManager, eventBus, log)
	log.Info("TenantAwareDocumentRepository initialized successfully.")

	// Initialize query engine with tenant-aware MongoDB implementation
	queryEngine := mongodbpersistence.NewTenantAwareQueryEngine(mongoClient, tenantManager, log) // Initialize security rules engine
	securityRulesEngine := mongodbpersistence.NewSecurityRulesEngine(masterDB, log)
	log.Info("SecurityRulesEngine initialized successfully.")

	// Initialize Redis Event Store for distributed realtime events
	redisEventStore2 := redispersistence.NewRedisEventStore(redisClient, log)
	log.Info("RedisEventStore initialized successfully.")

	// Initialize use cases with enhanced real-time capabilities using Redis
	realtimeUC := usecase.NewRealtimeUsecaseWithEventStore(log, redisEventStore2) // Enhanced with Redis persistence
	securityUC := usecase.NewSecurityUsecase(securityRulesEngine, log)

	// Initialize projection service
	projectionService2 := service.NewProjectionService()
	log.Info("ProjectionService initialized successfully.")

	// Initialize FirestoreUsecase with tenant-aware repository and projection service
	firestoreUC := usecase.NewFirestoreUsecase(tenantAwareRepo, securityRulesEngine, queryEngine, projectionService2, log)

	// Initialize OrganizationHandler
	orgHandler := httpadapter.NewOrganizationHandler(orgRepo)
	log.Info("OrganizationHandler initialized successfully.")
	return &FirestoreModule{
		Config:           cfg,
		AuthClient:       authClient,
		Logger:           log,
		TenantAwareRepo:  tenantAwareRepo,
		QueryEngine:      queryEngine,
		SecurityRules:    securityRulesEngine,
		FirestoreUsecase: firestoreUC,
		RealtimeUsecase:  realtimeUC,
		SecurityUsecase:  securityUC, TenantManager: tenantManager,
		OrganizationRepo:    orgRepo,
		OrganizationHandler: orgHandler,
		RedisClient:         redisClient,
		RedisEventStore:     redisEventStore2,
	}, nil
}

// RegisterRoutes registers the HTTP routes for the Firestore module.
func (m *FirestoreModule) RegisterRoutes(router fiber.Router, authMiddleware *authhttp.AuthMiddleware) { // Register Enhanced WebSocket handler for 100% Firestore-compatible real-time updates
	enhancedWSHandler := httpadapter.NewEnhancedWebSocketHandler(m.RealtimeUsecase, m.SecurityUsecase, m.AuthClient, m.Logger)
	enhancedWSHandler.RegisterRoutes(router, authMiddleware.RequireAuth())

	// Register HTTP adapter for Firestore REST API (now with Enhanced WebSocket handler included)
	httpHandler := httpadapter.NewFirestoreHTTPHandler(m.FirestoreUsecase, m.SecurityUsecase, m.RealtimeUsecase, m.AuthClient, m.Logger, m.OrganizationHandler, enhancedWSHandler)
	httpHandler.RegisterRoutes(router)

	m.Logger.Info("Firestore HTTP routes and Enhanced WebSocket handler registered with 100% compatibility.")
}

// StartRealtimeServices starts any background services for real-time functionality.
func (m *FirestoreModule) StartRealtimeServices() {
	// e.g., connect to message queues, start event listeners for RealtimeUsecase if it had any.
	// For the current in-memory RealtimeUsecase, there might be nothing explicit to start here
	// unless it needs to initialize some internal goroutines or listeners.
	m.Logger.Info("Real-time services (if any) would be started here.")
}

// Stop gracefully shuts down the Firestore module.
func (m *FirestoreModule) Stop() error {
	m.Logger.Info("Stopping Firestore Module...")
	// Any cleanup operations would go here
	m.Logger.Info("Firestore Module stopped.")
	return nil
}
