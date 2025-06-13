package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error types for different domains
type ErrorType string

const (
	// Domain errors
	ErrorTypeDomain         ErrorType = "DOMAIN_ERROR"
	ErrorTypeValidation     ErrorType = "VALIDATION_ERROR"
	ErrorTypeInfrastructure ErrorType = "INFRASTRUCTURE_ERROR"
	ErrorTypeAuthentication ErrorType = "AUTHENTICATION_ERROR"
	ErrorTypeAuthorization  ErrorType = "AUTHORIZATION_ERROR"
	ErrorTypeNotFound       ErrorType = "NOT_FOUND_ERROR"
	ErrorTypeConflict       ErrorType = "CONFLICT_ERROR"
	ErrorTypeInternal       ErrorType = "INTERNAL_ERROR"
)

// Common application errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrBadRequest         = errors.New("bad request")
	ErrInternalServer     = errors.New("internal server error")
	ErrConflict           = errors.New("resource conflict")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Firestore-specific errors
var (
	ErrInvalidPath           = errors.New("invalid firestore path")
	ErrInvalidProjectID      = errors.New("invalid project ID")
	ErrInvalidDatabaseID     = errors.New("invalid database ID")
	ErrInvalidCollectionID   = errors.New("invalid collection ID")
	ErrInvalidDocumentID     = errors.New("invalid document ID")
	ErrDocumentNotFound      = errors.New("document not found")
	ErrCollectionNotFound    = errors.New("collection not found")
	ErrInvalidQuery          = errors.New("invalid query")
	ErrInvalidTransaction    = errors.New("invalid transaction")
	ErrSecurityRuleViolation = errors.New("security rule violation")
)

// AppError represents a custom application error with context
type AppError struct {
	Type      ErrorType              `json:"type"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	HTTPCode  int                    `json:"-"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Component string                 `json:"component,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, message string, httpCode int) *AppError {
	return &AppError{
		Type:     errorType,
		Message:  message,
		HTTPCode: httpCode,
		Details:  make(map[string]interface{}),
	}
}

// WithCode adds an error code
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithCause adds the underlying cause
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithComponent adds the component name
func (e *AppError) WithComponent(component string) *AppError {
	e.Component = component
	return e
}

// WithDetail adds a detail field
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Common error constructors

// NewDomainError creates a domain-specific error
func NewDomainError(message string) *AppError {
	return NewAppError(ErrorTypeDomain, message, http.StatusBadRequest)
}

// NewValidationError creates a validation error
func NewValidationError(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, http.StatusBadRequest)
}

// NewInfrastructureError creates an infrastructure error
func NewInfrastructureError(message string) *AppError {
	return NewAppError(ErrorTypeInfrastructure, message, http.StatusInternalServerError)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *AppError {
	return NewAppError(ErrorTypeAuthentication, message, http.StatusUnauthorized)
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(message string) *AppError {
	return NewAppError(ErrorTypeAuthorization, message, http.StatusForbidden)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *AppError {
	return NewAppError(ErrorTypeConflict, message, http.StatusConflict)
}

// NewInternalError creates an internal server error
func NewInternalError(message string) *AppError {
	return NewAppError(ErrorTypeInternal, message, http.StatusInternalServerError)
}

// ValidationError represents validation errors for multiple fields
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", ve.Errors[0].Message)
}

// NewValidationErrors creates a new validation errors instance
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, message string, value interface{}) *ValidationErrors {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
	return ve
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// ToAppError converts validation errors to an AppError
func (ve *ValidationErrors) ToAppError() *AppError {
	if !ve.HasErrors() {
		return nil
	}

	appErr := NewValidationError("validation failed")
	appErr.Details["validation_errors"] = ve.Errors
	return appErr
}

// Helper functions for common error scenarios

// WrapError wraps an error with context
func WrapError(err error, message string) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return NewInternalError(message).WithCause(err)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeNotFound
	}
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrDocumentNotFound) || errors.Is(err, ErrUserNotFound)
}

// IsValidation checks if an error is a validation error
func IsValidation(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeValidation
	}
	return false
}

// IsAuthentication checks if an error is an authentication error
func IsAuthentication(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeAuthentication
	}
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenExpired)
}

// IsAuthorization checks if an error is an authorization error
func IsAuthorization(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeAuthorization
	}
	return errors.Is(err, ErrForbidden) || errors.Is(err, ErrSecurityRuleViolation)
}

// IsConflict checks if an error is a conflict error
func IsConflict(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeConflict
	}
	return errors.Is(err, ErrConflict)
}
