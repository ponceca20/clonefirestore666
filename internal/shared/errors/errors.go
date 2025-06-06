package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Common application errors
var (
	ErrNotFound        = errors.New("resource not found")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrForbidden       = errors.New("forbidden access")
	ErrConflict        = errors.New("resource conflict")
	ErrBadRequest      = errors.New("bad request")
	ErrInternalError   = errors.New("internal server error")
	ErrTimeout         = errors.New("request timeout")
	ErrTooManyRequests = errors.New("too many requests")
)

// AppError represents a custom application error
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %d, message: %s, error: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors
func NewBadRequestError(message string, err error) *AppError {
	return NewAppError(http.StatusBadRequest, message, err)
}

func NewUnauthorizedError(message string, err error) *AppError {
	return NewAppError(http.StatusUnauthorized, message, err)
}

func NewForbiddenError(message string, err error) *AppError {
	return NewAppError(http.StatusForbidden, message, err)
}

func NewNotFoundError(message string, err error) *AppError {
	return NewAppError(http.StatusNotFound, message, err)
}

func NewConflictError(message string, err error) *AppError {
	return NewAppError(http.StatusConflict, message, err)
}

func NewInternalServerError(message string, err error) *AppError {
	return NewAppError(http.StatusInternalServerError, message, err)
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
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
func NewValidationErrors(errors []ValidationError) *ValidationErrors {
	return &ValidationErrors{Errors: errors}
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, message string) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}
