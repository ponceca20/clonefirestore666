package mongodb

import (
	"context"
	"fmt"
	"log"

	"firestore-clone/internal/firestore/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/contextkeys"
	"firestore-clone/internal/shared/database"
	"firestore-clone/internal/shared/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

// TenantAwareQueryEngine implements repository.QueryEngine with multi-tenant support
type TenantAwareQueryEngine struct {
	mongoClient   *mongo.Client
	tenantManager *database.TenantManager
	logger        logger.Logger
}

// NewTenantAwareQueryEngine creates a new tenant-aware query engine
func NewTenantAwareQueryEngine(
	mongoClient *mongo.Client,
	tenantManager *database.TenantManager,
	logger logger.Logger,
) *TenantAwareQueryEngine {
	return &TenantAwareQueryEngine{
		mongoClient:   mongoClient,
		tenantManager: tenantManager,
		logger:        logger,
	}
}

// ExecuteQuery ejecuta una consulta sobre una colección MongoDB específica del tenant
func (qe *TenantAwareQueryEngine) ExecuteQuery(ctx context.Context, collectionPath string, query model.Query) ([]*model.Document, error) {
	log.Printf("[TenantAwareQueryEngine] === INICIANDO EJECUCIÓN DE CONSULTA ===")
	log.Printf("[TenantAwareQueryEngine] CollectionPath: %s", collectionPath)
	log.Printf("[TenantAwareQueryEngine] Query completo: %+v", query)
	log.Printf("[TenantAwareQueryEngine] Filtros del query: %+v", query.Filters)

	// Extraer organizationID del contexto
	organizationID, err := qe.extractOrganizationID(ctx)
	if err != nil {
		log.Printf("[TenantAwareQueryEngine] ERROR: No se pudo extraer organization ID: %v", err)
		return nil, fmt.Errorf("failed to extract organization ID: %w", err)
	}

	log.Printf("[TenantAwareQueryEngine] Organization ID extraído: %s", organizationID)
	// Obtener la base de datos específica del tenant
	tenantDB, err := qe.tenantManager.GetDatabaseForOrganization(ctx, organizationID)
	if err != nil {
		log.Printf("[TenantAwareQueryEngine] ERROR: No se pudo obtener la base de datos del tenant: %v", err)
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	log.Printf("[TenantAwareQueryEngine] Base de datos del tenant obtenida correctamente")

	// Crear un MongoQueryEngine específico para este tenant
	mongoQueryEngine := NewMongoQueryEngine(tenantDB)

	log.Printf("[TenantAwareQueryEngine] Ejecutando consulta para tenant %s en colección %s", organizationID, collectionPath)
	// Ejecutar la consulta usando el MongoQueryEngine mejorado
	documents, err := mongoQueryEngine.ExecuteQuery(ctx, collectionPath, query)
	if err != nil {
		log.Printf("[TenantAwareQueryEngine] ERROR: Falló la ejecución de la consulta: %v", err)
		return nil, err
	}

	log.Printf("[TenantAwareQueryEngine] === CONSULTA COMPLETADA EXITOSAMENTE ===")
	log.Printf("[TenantAwareQueryEngine] Documentos retornados: %d", len(documents))

	return documents, nil
}

// extractOrganizationID extrae el ID de la organización del contexto
func (qe *TenantAwareQueryEngine) extractOrganizationID(ctx context.Context) (string, error) {
	// Usar la key correcta definida en contextkeys
	if orgID := ctx.Value(contextkeys.OrganizationIDKey); orgID != nil {
		if orgIDStr, ok := orgID.(string); ok && orgIDStr != "" {
			return orgIDStr, nil
		}
	}

	// Fallback: intentar obtenerlo desde otras keys posibles para backward compatibility
	if orgID := ctx.Value("organization_id"); orgID != nil {
		if orgIDStr, ok := orgID.(string); ok && orgIDStr != "" {
			return orgIDStr, nil
		}
	}

	if orgID := ctx.Value("organizationId"); orgID != nil {
		if orgIDStr, ok := orgID.(string); ok && orgIDStr != "" {
			return orgIDStr, nil
		}
	}

	if orgID := ctx.Value("org_id"); orgID != nil {
		if orgIDStr, ok := orgID.(string); ok && orgIDStr != "" {
			return orgIDStr, nil
		}
	}

	return "", fmt.Errorf("organization ID not found in context")
}

// ExecuteQueryWithProjection executes a query with field projection for a specific tenant
func (qe *TenantAwareQueryEngine) ExecuteQueryWithProjection(ctx context.Context, collectionPath string, query model.Query, projection []string) ([]*model.Document, error) {
	qe.logger.Info("Executing query with projection for tenant", "collectionPath", collectionPath, "projection", projection)

	// Extract organization ID from context
	organizationID, err := qe.extractOrganizationID(ctx)
	if err != nil {
		qe.logger.Error("Failed to extract organization ID", "error", err)
		return nil, fmt.Errorf("failed to extract organization ID: %w", err)
	}

	// Get tenant-specific database
	tenantDB, err := qe.tenantManager.GetDatabaseForOrganization(ctx, organizationID)
	if err != nil {
		qe.logger.Error("Failed to get tenant database", "error", err, "organizationID", organizationID)
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	// Create enhanced MongoDB query engine for this tenant
	enhancedEngine := NewEnhancedMongoQueryEngine(tenantDB)

	// Execute query with projection
	return enhancedEngine.ExecuteQueryWithProjection(ctx, collectionPath, query, projection)
}

// CountDocuments returns the count of documents matching the query for a specific tenant
func (qe *TenantAwareQueryEngine) CountDocuments(ctx context.Context, collectionPath string, query model.Query) (int64, error) {
	qe.logger.Info("Counting documents for tenant", "collectionPath", collectionPath)

	// Extract organization ID from context
	organizationID, err := qe.extractOrganizationID(ctx)
	if err != nil {
		qe.logger.Error("Failed to extract organization ID", "error", err)
		return 0, fmt.Errorf("failed to extract organization ID: %w", err)
	}

	// Get tenant-specific database
	tenantDB, err := qe.tenantManager.GetDatabaseForOrganization(ctx, organizationID)
	if err != nil {
		qe.logger.Error("Failed to get tenant database", "error", err, "organizationID", organizationID)
		return 0, fmt.Errorf("failed to get tenant database: %w", err)
	}

	// Create enhanced MongoDB query engine for this tenant
	enhancedEngine := NewEnhancedMongoQueryEngine(tenantDB)

	// Count documents
	return enhancedEngine.CountDocuments(ctx, collectionPath, query)
}

// ValidateQuery validates if a query is supported by the engine for multi-tenant environment
func (qe *TenantAwareQueryEngine) ValidateQuery(query model.Query) error {
	qe.logger.Debug("Validating query", "query", query)

	// Use enhanced MongoDB query engine for validation
	// Since validation doesn't require tenant context, we can use a temporary engine
	tempDB := qe.mongoClient.Database("temp_validation")
	enhancedEngine := NewEnhancedMongoQueryEngine(tempDB)

	return enhancedEngine.ValidateQuery(query)
}

// GetQueryCapabilities returns the capabilities of this tenant-aware query engine
func (qe *TenantAwareQueryEngine) GetQueryCapabilities() repository.QueryCapabilities {
	// Return capabilities that combine multi-tenant support with MongoDB enhanced features
	return repository.QueryCapabilities{
		SupportsNestedFields:     true,
		SupportsArrayContains:    true,
		SupportsArrayContainsAny: true,
		SupportsCompositeFilters: true,
		SupportsOrderBy:          true,
		SupportsCursorPagination: true,
		SupportsOffsetPagination: true,
		SupportsProjection:       true,
		MaxFilterCount:           100, // MongoDB limit
		MaxOrderByCount:          32,  // MongoDB sort limit
		MaxNestingDepth:          100, // Firestore/MongoDB support deep nesting
	}
}

// Asegurar que implementa la interfaz
var _ repository.QueryEngine = (*TenantAwareQueryEngine)(nil)
