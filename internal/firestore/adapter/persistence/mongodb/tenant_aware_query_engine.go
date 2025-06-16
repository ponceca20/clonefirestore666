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

// Asegurar que implementa la interfaz
var _ repository.QueryEngine = (*TenantAwareQueryEngine)(nil)
