package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/domain"
)

// RulesDeployer implementa despliegue seguro y eficiente de reglas
type RulesDeployer struct {
	securityEngine repository.SecurityRulesEngine
	validator      RulesValidator
	history        DeployHistoryStore
	config         *DeployerConfig
}

type DeployerConfig struct {
	EnableValidation   bool          `json:"enable_validation"`
	EnableRollback     bool          `json:"enable_rollback"`
	ValidationTimeout  time.Duration `json:"validation_timeout"`
	DeployTimeout      time.Duration `json:"deploy_timeout"`
	MaxHistoryEntries  int           `json:"max_history_entries"`
	EnableDryRun       bool          `json:"enable_dry_run"`
	BackupBeforeDeploy bool          `json:"backup_before_deploy"`
}

type RulesValidator interface {
	ValidateRules(ctx context.Context, rules []*repository.SecurityRule) error
	ValidateAgainstCurrent(ctx context.Context, newRules, currentRules []*repository.SecurityRule) error
}

type DeployHistoryStore interface {
	SaveDeployment(ctx context.Context, projectID, databaseID string, deployment *domain.DeployHistory) error
	GetHistory(ctx context.Context, projectID, databaseID string, limit int) ([]*domain.DeployHistory, error)
	GetLastDeployment(ctx context.Context, projectID, databaseID string) (*domain.DeployHistory, error)
}

// NewRulesDeployer crea una nueva instancia del deployer
func NewRulesDeployer(
	securityEngine repository.SecurityRulesEngine,
	validator RulesValidator,
	history DeployHistoryStore,
	config *DeployerConfig,
) *RulesDeployer {
	if config == nil {
		config = DefaultDeployerConfig()
	}

	return &RulesDeployer{
		securityEngine: securityEngine,
		validator:      validator,
		history:        history,
		config:         config,
	}
}

// DefaultDeployerConfig configuración por defecto
func DefaultDeployerConfig() *DeployerConfig {
	return &DeployerConfig{
		EnableValidation:   true,
		EnableRollback:     true,
		ValidationTimeout:  time.Second * 30,
		DeployTimeout:      time.Second * 60,
		MaxHistoryEntries:  100,
		EnableDryRun:       true,
		BackupBeforeDeploy: true,
	}
}

// Deploy despliega reglas al motor de seguridad
func (d *RulesDeployer) Deploy(ctx context.Context, projectID, databaseID string, rules interface{}) error {
	result, err := d.DeployWithValidation(ctx, projectID, databaseID, rules)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("deployment failed: %v", result.Errors)
	}

	return nil
}

// DeployWithValidation despliega con validación completa
func (d *RulesDeployer) DeployWithValidation(ctx context.Context, projectID, databaseID string, rules interface{}) (*domain.DeployResult, error) {
	startTime := time.Now()

	securityRules, ok := rules.([]*repository.SecurityRule)
	if !ok {
		return nil, fmt.Errorf("invalid rules type")
	}

	result := &domain.DeployResult{
		Version:       generateDeployVersion(),
		RulesDeployed: len(securityRules),
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	// 1. Validación pre-despliegue
	if d.config.EnableValidation && d.validator != nil {
		if err := d.validateBeforeDeploy(ctx, projectID, databaseID, securityRules, result); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("validation failed: %v", err))
			return result, nil
		}
	}

	// 2. Backup de reglas actuales si está habilitado
	var backupRules []*repository.SecurityRule
	if d.config.BackupBeforeDeploy {
		if backup, err := d.backupCurrentRules(ctx, projectID, databaseID); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("backup failed: %v", err))
		} else {
			backupRules = backup
		}
	}

	// 3. Despliegue actual con timeout
	deployCtx, cancel := context.WithTimeout(ctx, d.config.DeployTimeout)
	defer cancel()

	if err := d.securityEngine.SaveRules(deployCtx, projectID, databaseID, securityRules); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("deploy failed: %v", err))

		// Intentar rollback automático si el backup existe
		if d.config.EnableRollback && backupRules != nil {
			if rollbackErr := d.performRollback(ctx, projectID, databaseID, backupRules); rollbackErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("rollback failed: %v", rollbackErr))
			} else {
				result.Warnings = append(result.Warnings, "automatically rolled back to previous version")
			}
		}

		return result, nil
	}

	// 4. Validación post-despliegue
	if d.config.EnableValidation && d.validator != nil {
		if err := d.validateAfterDeploy(ctx, projectID, databaseID, securityRules); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("post-deploy validation warning: %v", err))
		}
	}

	// 5. Guardar en historial
	if d.history != nil {
		deployment := &domain.DeployHistory{
			Version:    result.Version,
			DeployedAt: time.Now(),
			DeployedBy: "rules-importer", // TODO: obtener del contexto
			RulesCount: len(securityRules),
			Status:     "success",
		}

		if err := d.history.SaveDeployment(ctx, projectID, databaseID, deployment); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to save deployment history: %v", err))
		}
	}

	result.Success = true
	result.DeployTime = time.Since(startTime)

	return result, nil
}

// Rollback revierte al conjunto de reglas anterior
func (d *RulesDeployer) Rollback(ctx context.Context, projectID, databaseID string) error {
	if !d.config.EnableRollback || d.history == nil {
		return fmt.Errorf("rollback not enabled")
	}

	// Obtener último despliegue exitoso
	lastDeploy, err := d.history.GetLastDeployment(ctx, projectID, databaseID)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	if lastDeploy == nil {
		return fmt.Errorf("no previous deployment found")
	}

	// TODO: Implementar lógica para obtener reglas del historial
	// Por ahora retorna error indicando que se necesita implementación adicional
	return fmt.Errorf("rollback implementation requires rules storage in history")
}

// GetCurrentVersion obtiene la versión actual de reglas
func (d *RulesDeployer) GetCurrentVersion(ctx context.Context, projectID, databaseID string) (string, error) {
	if d.history == nil {
		return "", fmt.Errorf("history store not available")
	}

	lastDeploy, err := d.history.GetLastDeployment(ctx, projectID, databaseID)
	if err != nil {
		return "", err
	}

	if lastDeploy == nil {
		return "", fmt.Errorf("no deployments found")
	}

	return lastDeploy.Version, nil
}

// GetDeployHistory obtiene historial de despliegues
func (d *RulesDeployer) GetDeployHistory(ctx context.Context, projectID, databaseID string, limit int) ([]*domain.DeployHistory, error) {
	if d.history == nil {
		return nil, fmt.Errorf("history store not available")
	}

	if limit <= 0 {
		limit = d.config.MaxHistoryEntries
	}

	return d.history.GetHistory(ctx, projectID, databaseID, limit)
}

// Helper methods

func (d *RulesDeployer) validateBeforeDeploy(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule, result *domain.DeployResult) error {
	validationCtx, cancel := context.WithTimeout(ctx, d.config.ValidationTimeout)
	defer cancel()

	// Validación básica de reglas
	if err := d.validator.ValidateRules(validationCtx, rules); err != nil {
		return fmt.Errorf("rules validation failed: %w", err)
	}

	// Validación contra reglas actuales
	currentRules, err := d.getCurrentRules(validationCtx, projectID, databaseID)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not get current rules for comparison: %v", err))
		return nil // No fallar por esto
	}

	if err := d.validator.ValidateAgainstCurrent(validationCtx, rules, currentRules); err != nil {
		return fmt.Errorf("validation against current rules failed: %w", err)
	}

	return nil
}

func (d *RulesDeployer) validateAfterDeploy(ctx context.Context, projectID, databaseID string, deployedRules []*repository.SecurityRule) error {
	// Verificar que las reglas se guardaron correctamente
	currentRules, err := d.getCurrentRules(ctx, projectID, databaseID)
	if err != nil {
		return fmt.Errorf("could not verify deployed rules: %w", err)
	}

	if len(currentRules) != len(deployedRules) {
		return fmt.Errorf("rule count mismatch: expected %d, got %d", len(deployedRules), len(currentRules))
	}

	// TODO: Implementar comparación más detallada de reglas

	return nil
}

func (d *RulesDeployer) backupCurrentRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return d.getCurrentRules(ctx, projectID, databaseID)
}

func (d *RulesDeployer) getCurrentRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	// TODO: Implementar método para obtener reglas actuales del SecurityRulesEngine
	// Por ahora retorna slice vacío
	return make([]*repository.SecurityRule, 0), nil
}

func (d *RulesDeployer) performRollback(ctx context.Context, projectID, databaseID string, backupRules []*repository.SecurityRule) error {
	return d.securityEngine.SaveRules(ctx, projectID, databaseID, backupRules)
}

func generateDeployVersion() string {
	return fmt.Sprintf("deploy-%d", time.Now().Unix())
}

// SimpleValidator implementación básica de RulesValidator
type SimpleValidator struct{}

// NewSimpleValidator crea un validador básico
func NewSimpleValidator() *SimpleValidator {
	return &SimpleValidator{}
}

// ValidateRules valida reglas básicamente
func (v *SimpleValidator) ValidateRules(ctx context.Context, rules []*repository.SecurityRule) error {
	for i, rule := range rules {
		if rule.Match == "" {
			return fmt.Errorf("rule %d has empty match pattern", i)
		}

		if len(rule.Allow) == 0 && len(rule.Deny) == 0 {
			return fmt.Errorf("rule %d has no allow or deny statements", i)
		}

		// Validar condiciones básicas
		for op, condition := range rule.Allow {
			if condition == "" {
				return fmt.Errorf("rule %d has empty condition for operation %v", i, op)
			}
		}

		for op, condition := range rule.Deny {
			if condition == "" {
				return fmt.Errorf("rule %d has empty condition for deny operation %v", i, op)
			}
		}
	}

	return nil
}

// ValidateAgainstCurrent valida contra reglas actuales
func (v *SimpleValidator) ValidateAgainstCurrent(ctx context.Context, newRules, currentRules []*repository.SecurityRule) error {
	// Validación básica: verificar que no se están eliminando todas las reglas accidentalmente
	if len(currentRules) > 0 && len(newRules) == 0 {
		return fmt.Errorf("attempting to deploy empty ruleset - this would remove all security rules")
	}

	// TODO: Implementar validaciones más sofisticadas
	// - Verificar que no se están rompiendo accesos existentes
	// - Verificar que nuevas reglas no son demasiado permisivas
	// - Verificar compatibilidad con aplicaciones existentes

	return nil
}

// MemoryHistoryStore implementación en memoria de DeployHistoryStore
type MemoryHistoryStore struct {
	deployments map[string][]*domain.DeployHistory
	mutex       sync.RWMutex
}

// NewMemoryHistoryStore crea un store de historial en memoria
func NewMemoryHistoryStore() *MemoryHistoryStore {
	return &MemoryHistoryStore{
		deployments: make(map[string][]*domain.DeployHistory),
	}
}

// SaveDeployment guarda un despliegue en el historial
func (h *MemoryHistoryStore) SaveDeployment(ctx context.Context, projectID, databaseID string, deployment *domain.DeployHistory) error {
	key := fmt.Sprintf("%s:%s", projectID, databaseID)

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.deployments[key] == nil {
		h.deployments[key] = make([]*domain.DeployHistory, 0)
	}

	h.deployments[key] = append(h.deployments[key], deployment)

	return nil
}

// GetHistory obtiene historial de despliegues
func (h *MemoryHistoryStore) GetHistory(ctx context.Context, projectID, databaseID string, limit int) ([]*domain.DeployHistory, error) {
	key := fmt.Sprintf("%s:%s", projectID, databaseID)

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	deployments := h.deployments[key]
	if len(deployments) == 0 {
		return make([]*domain.DeployHistory, 0), nil
	}

	// Retornar los últimos 'limit' despliegues
	start := len(deployments) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*domain.DeployHistory, len(deployments)-start)
	copy(result, deployments[start:])

	return result, nil
}

// GetLastDeployment obtiene el último despliegue
func (h *MemoryHistoryStore) GetLastDeployment(ctx context.Context, projectID, databaseID string) (*domain.DeployHistory, error) {
	key := fmt.Sprintf("%s:%s", projectID, databaseID)

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	deployments := h.deployments[key]
	if len(deployments) == 0 {
		return nil, nil
	}

	return deployments[len(deployments)-1], nil
}
