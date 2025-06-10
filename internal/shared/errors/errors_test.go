package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_Compile(t *testing.T) {
	// Placeholder: Add real error utility tests here
}

func TestAppError_Behavior(t *testing.T) {
	err := NewValidationError("invalid input").WithCode("VAL001").WithDetail("field", "name").WithComponent("test-component")
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "invalid input", err.Message)
	assert.Equal(t, "VAL001", err.Code)
	assert.Equal(t, "test-component", err.Component)
	assert.Equal(t, "name", err.Details["field"])
	assert.Equal(t, "invalid input", err.Error())
}

func TestAppError_WithCause_Unwrap(t *testing.T) {
	cause := ErrNotFound
	err := NewNotFoundError("resource").WithCause(cause)
	assert.Equal(t, cause, err.Unwrap())
}

func TestValidationErrors(t *testing.T) {
	ve := NewValidationErrors()
	ve.Add("field1", "must be set", "")
	assert.True(t, ve.HasErrors())
	appErr := ve.ToAppError()
	assert.NotNil(t, appErr)
	assert.Equal(t, ErrorTypeValidation, appErr.Type)
}

func TestIsNotFound_IsValidation_IsAuthentication_IsAuthorization(t *testing.T) {
	nf := NewNotFoundError("doc")
	assert.True(t, IsNotFound(nf))
	assert.False(t, IsValidation(nf))
	assert.False(t, IsAuthentication(nf))
	assert.False(t, IsAuthorization(nf))

	val := NewValidationError("bad")
	assert.True(t, IsValidation(val))
	auth := NewAuthenticationError("bad")
	assert.True(t, IsAuthentication(auth))
	authz := NewAuthorizationError("bad")
	assert.True(t, IsAuthorization(authz))
}
