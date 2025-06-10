package usecase

import (
	"context"
	"time"

	"firestore-clone/internal/auth/domain/model"
	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/shared/errors"
	"firestore-clone/internal/shared/firestore"
	"firestore-clone/internal/shared/logger"

	"go.uber.org/zap"
)

// SecurityUsecase defines the interface for security-related operations.
type SecurityUsecase interface {
	// ValidateRead checks if a user can read from a specific Firestore path
	ValidateRead(ctx context.Context, user *model.User, firestorePath string) error

	// ValidateWrite checks if a user can write to a specific Firestore path
	ValidateWrite(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}) error

	// ValidateDelete checks if a user can delete from a specific Firestore path
	ValidateDelete(ctx context.Context, user *model.User, firestorePath string) error

	// ValidateCreate checks if a user can create a document at a specific Firestore path
	ValidateCreate(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}) error

	// ValidateUpdate checks if a user can update a document at a specific Firestore path
	ValidateUpdate(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}, existingData map[string]interface{}) error
}

type securityUsecaseImpl struct {
	rulesEngine repository.SecurityRulesEngine
	log         logger.Logger
}

// NewSecurityUsecase creates a new instance of SecurityUsecase.
func NewSecurityUsecase(rulesEngine repository.SecurityRulesEngine, log logger.Logger) SecurityUsecase {
	return &securityUsecaseImpl{
		rulesEngine: rulesEngine,
		log:         log,
	}
}

// ValidateRead implements the SecurityUsecase interface.
func (uc *securityUsecaseImpl) ValidateRead(ctx context.Context, user *model.User, firestorePath string) error {
	return uc.validateOperation(ctx, user, firestorePath, repository.OperationRead, nil, nil)
}

// ValidateWrite implements the SecurityUsecase interface.
func (uc *securityUsecaseImpl) ValidateWrite(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}) error {
	return uc.validateOperation(ctx, user, firestorePath, repository.OperationWrite, data, nil)
}

// ValidateDelete implements the SecurityUsecase interface.
func (uc *securityUsecaseImpl) ValidateDelete(ctx context.Context, user *model.User, firestorePath string) error {
	return uc.validateOperation(ctx, user, firestorePath, repository.OperationDelete, nil, nil)
}

// ValidateCreate implements the SecurityUsecase interface.
func (uc *securityUsecaseImpl) ValidateCreate(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}) error {
	return uc.validateOperation(ctx, user, firestorePath, repository.OperationCreate, data, nil)
}

// ValidateUpdate implements the SecurityUsecase interface.
func (uc *securityUsecaseImpl) ValidateUpdate(ctx context.Context, user *model.User, firestorePath string, data map[string]interface{}, existingData map[string]interface{}) error {
	return uc.validateOperation(ctx, user, firestorePath, repository.OperationUpdate, data, existingData)
}

// validateOperation is a common method for validating operations
func (uc *securityUsecaseImpl) validateOperation(ctx context.Context, user *model.User, firestorePath string, operation repository.OperationType, requestData map[string]interface{}, existingData map[string]interface{}) error {
	// Parse Firestore path
	pathInfo, err := firestore.ParseFirestorePath(firestorePath)
	if err != nil {
		uc.log.Error("Invalid Firestore path",
			zap.String("path", firestorePath),
			zap.Error(err))
		return errors.NewValidationError("invalid firestore path")
	}

	// Create security context
	securityContext := &repository.SecurityContext{
		User:       user,
		ProjectID:  pathInfo.ProjectID,
		DatabaseID: pathInfo.DatabaseID,
		Resource:   existingData,
		Request:    requestData,
		Timestamp:  time.Now().Unix(),
		Path:       pathInfo.DocumentPath,
	}

	// Evaluate access using rules engine
	result, err := uc.rulesEngine.EvaluateAccess(ctx, operation, securityContext)
	if err != nil {
		uc.log.Error("Error evaluating security rules",
			zap.String("operation", string(operation)),
			zap.String("path", firestorePath),
			zap.String("userID", getUserID(user)),
			zap.Error(err))
		return errors.NewInternalError("security rules evaluation failed")
	}
	// Check if access is allowed
	if !result.Allowed {
		uc.log.Warn("Access denied by security rules",
			zap.String("operation", string(operation)),
			zap.String("path", firestorePath),
			zap.String("userID", getUserID(user)),
			zap.String("reason", result.Reason),
			zap.String("deniedBy", result.DeniedBy))
		return errors.NewAuthorizationError("access denied by security rules")
	}

	uc.log.Debug("Access granted by security rules",
		zap.String("operation", string(operation)),
		zap.String("path", firestorePath),
		zap.String("userID", getUserID(user)),
		zap.String("allowedBy", result.AllowedBy))

	return nil
}

// getUserID safely extracts user ID from user object
func getUserID(user *model.User) string {
	if user == nil {
		return "anonymous"
	}
	return user.ID.Hex()
}
