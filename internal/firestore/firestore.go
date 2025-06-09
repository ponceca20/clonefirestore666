package firestore

import (
	"fmt"

	// Added imports
	httpadapter "firestore-clone/internal/firestore/adapter/http"
	mongodbpersistence "firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/firestore/config"
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/domain/repository" // May keep for interfaces
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/eventbus"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2" // For RegisterRoutes parameter
	"go.mongodb.org/mongo-driver/mongo"
	// "go.uber.org/zap" // Example logger, use passed logger
)

// FirestoreModule represents the core Firestore module.
type FirestoreModule struct {
	Config           *config.FirestoreConfig
	AuthClient       client.AuthClient                      // Client to interact with Auth module
	DocumentRepo     *mongodbpersistence.DocumentRepository // MongoDB implementation
	QueryEngine      repository.QueryEngine                 // MongoDB query engine implementation
	SecurityRules    repository.SecurityRulesEngine         // MongoDB security rules engine implementation
	FirestoreUsecase usecase.FirestoreUsecaseInterface      // Interface type
	RealtimeUsecase  usecase.RealtimeUsecase                // Interface type
	SecurityUsecase  usecase.SecurityUsecase                // Interface type
	Logger           logger.Logger
	// wsHandler *httpadapter.WebSocketHandler // Optional: if needed beyond RegisterRoutes
}

// NewFirestoreModule creates and initializes a new Firestore module.
func NewFirestoreModule(
	authClient client.AuthClient,
	log logger.Logger,
	mongoDB *mongo.Database, // Se agrega la base de datos como dependencia
) (*FirestoreModule, error) {
	fmt.Println("Initializing Firestore Module...")
	log.Info("Initializing Firestore Module...")

	// Load Firestore configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load Firestore config", "error", err)
		return nil, fmt.Errorf("failed to load firestore config: %w", err)
	}
	log.Info("Firestore configuration loaded successfully.")

	// Inicializar EventBus
	eventBus := eventbus.NewEventBus(log) // Inicializar DocumentRepository (MongoDB implementation)
	docRepo := mongodbpersistence.NewDocumentRepository(mongoDB, eventBus, log)
	log.Info("DocumentRepository initialized successfully.")

	// Initialize query engine with MongoDB implementation
	queryEngine := mongodbpersistence.NewMongoQueryEngine(mongoDB)
	log.Info("QueryEngine initialized successfully.")

	// Initialize security rules engine
	securityRulesEngine := mongodbpersistence.NewSecurityRulesEngine(mongoDB, log)
	log.Info("SecurityRulesEngine initialized successfully.")

	// Initialize use cases
	realtimeUC := usecase.NewRealtimeUsecase(log)
	securityUC := usecase.NewSecurityUsecase(securityRulesEngine, log)
	// Initialize FirestoreUsecase with all dependencies
	firestoreUC := usecase.NewFirestoreUsecase(docRepo, securityRulesEngine, queryEngine, log)

	return &FirestoreModule{
		Config:           cfg,
		AuthClient:       authClient,
		Logger:           log,
		DocumentRepo:     docRepo,
		QueryEngine:      queryEngine,
		SecurityRules:    securityRulesEngine,
		RealtimeUsecase:  realtimeUC,
		SecurityUsecase:  securityUC,
		FirestoreUsecase: firestoreUC,
		// Assign other initialized components here
	}, nil
}

// RegisterRoutes registers the HTTP routes for the Firestore module.
func (m *FirestoreModule) RegisterRoutes(router *fiber.App) {
	// Register HTTP adapter for Firestore REST API
	httpHandler := httpadapter.NewFirestoreHTTPHandler(m.FirestoreUsecase, m.SecurityUsecase, m.RealtimeUsecase, m.AuthClient, m.Logger)
	httpHandler.RegisterRoutes(router)

	// Register WebSocket handler for real-time updates
	wsHandler := httpadapter.NewWebSocketHandler(m.RealtimeUsecase, m.SecurityUsecase, m.AuthClient, m.Logger)
	wsHandler.RegisterRoutes(router)

	m.Logger.Info("Firestore HTTP routes and WebSocket handler registered.")
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
	// No hay método Close en DocumentRepo, así que solo logueamos
	// Si se requiere cerrar la conexión a MongoDB, debe hacerse fuera de este módulo
	m.Logger.Info("Firestore Module stopped.")
	return nil
}
