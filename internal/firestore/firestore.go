package firestore

import (
	"fmt"
	// Added imports
	httpadapter "firestore-clone/internal/firestore/adapter/http"
	"firestore-clone/internal/firestore/config" // Assuming config might be used later
	"firestore-clone/internal/firestore/domain/client"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/firestore/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/gofiber/fiber/v2" // For RegisterRoutes parameter
	// "go.uber.org/zap" // Example logger, use passed logger
)

// FirestoreModule represents the core Firestore module.
type FirestoreModule struct {
	Config           *config.FirestoreConfig // To be defined in internal/firestore/config/config.go
	AuthClient       client.AuthClient       // Client to interact with Auth module
	DocumentRepo     repository.FirestoreRepository
	QueryEngine      repository.QueryEngine
	SecurityRules    repository.SecurityRulesEngine
	FirestoreUsecase usecase.FirestoreUsecase
	RealtimeUsecase  usecase.RealtimeUsecase // Added
	SecurityUsecase  usecase.SecurityUsecase
	Logger           logger.Logger
	// wsHandler *httpadapter.WebSocketHandler // Optional: if needed beyond RegisterRoutes
}

// NewFirestoreModule creates and initializes a new Firestore module.
func NewFirestoreModule(
	authClient client.AuthClient, // Added
	log logger.Logger, // Added
	// Other dependencies like db connections
) (*FirestoreModule, error) {
	fmt.Println("Initializing Firestore Module...")
	log.Info("Initializing Firestore Module...")

	// TODO: Initialize config
	// TODO: Initialize repositories (e.g., MongoDB implementation)
	// TODO: Initialize query engine
	// TODO: Initialize security rules engine

	// Initialize use cases
	realtimeUC := usecase.NewRealtimeUsecase(log)
	securityUC := usecase.NewSecurityUsecase(log)

	// TODO: Initialize FirestoreUsecase with all dependencies
	// firestoreUC := usecase.NewFirestoreUsecase(documentRepo, log, realtimeUC)

	return &FirestoreModule{
		AuthClient:      authClient,
		Logger:          log,
		RealtimeUsecase: realtimeUC,
		SecurityUsecase: securityUC,
		// FirestoreUsecase: firestoreUC, // Uncomment when repositories are ready
		// Assign other initialized components here
	}, nil
}

// RegisterRoutes registers the HTTP routes for the Firestore module.
func (m *FirestoreModule) RegisterRoutes(router *fiber.App) {
	// httpAdapter := httpadapter.NewFirestoreRouter(m.FirestoreUsecase, m.SecurityUsecase, m.Logger)
	// httpAdapter.RegisterRoutes(router)

	// Ensure RealtimeUsecase and AuthClient are not nil if they are critical for WS
	if m.RealtimeUsecase == nil {
		m.Logger.Error("RealtimeUsecase is not initialized in FirestoreModule")
		// Potentially panic or return an error if this setup is critical
		return
	}
	if m.AuthClient == nil {
		m.Logger.Error("AuthClient is not initialized in FirestoreModule for WebSocketHandler")
		// Potentially panic or return an error
		return
	}

	wsHandler := httpadapter.NewWebSocketHandler(m.RealtimeUsecase, m.AuthClient, m.Logger)
	wsHandler.RegisterRoutes(router)

	m.Logger.Info("Firestore routes and WebSocket handler registered.")
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
	// TODO: Gracefully close database connections, stop background services, etc.
	// If RealtimeUsecase had resources to clean up (e.g. closing all client channels), it would be done here.
	return nil
}
