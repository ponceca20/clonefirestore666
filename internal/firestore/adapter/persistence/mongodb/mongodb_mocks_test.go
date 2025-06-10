package mongodb

import (
	"context"
	"firestore-clone/internal/shared/logger" // Ensure logger is imported

	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of the logger.Logger interface.
type MockLogger struct {
	mock.Mock
}

// Debug mocks the Debug method.
func (m *MockLogger) Debug(args ...interface{}) { m.Called(args...) }

// Info mocks the Info method.
func (m *MockLogger) Info(args ...interface{}) { m.Called(args...) }

// Warn mocks the Warn method.
func (m *MockLogger) Warn(args ...interface{}) { m.Called(args...) }

// Error mocks the Error method.
func (m *MockLogger) Error(args ...interface{}) { m.Called(args...) }

// Fatal mocks the Fatal method.
func (m *MockLogger) Fatal(args ...interface{}) { m.Called(args...) }

// Debugf mocks the Debugf method.
func (m *MockLogger) Debugf(format string, args ...interface{}) { m.Called(format, args) }

// Infof mocks the Infof method.
func (m *MockLogger) Infof(format string, args ...interface{}) { m.Called(format, args) }

// Warnf mocks the Warnf method.
func (m *MockLogger) Warnf(format string, args ...interface{}) { m.Called(format, args) }

// Errorf mocks the Errorf method.
func (m *MockLogger) Errorf(format string, args ...interface{}) { m.Called(format, args) }

// Fatalf mocks the Fatalf method.
func (m *MockLogger) Fatalf(format string, args ...interface{}) { m.Called(format, args) }

// WithError mocks the WithError method.
func (m *MockLogger) WithError(err error) logger.Logger {
	callArgs := m.Called(err)
	if callArgs.Get(0) == nil {
		return m
	}
	return callArgs.Get(0).(logger.Logger)
}

// WithField mocks the WithField method.
func (m *MockLogger) WithField(key string, value interface{}) logger.Logger {
	callArgs := m.Called(key, value)
	if callArgs.Get(0) == nil {
		return m
	}
	return callArgs.Get(0).(logger.Logger)
}

// WithFields mocks the WithFields method.
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	callArgs := m.Called(fields)
	if callArgs.Get(0) == nil {
		return m
	}
	return callArgs.Get(0).(logger.Logger)
}

// WithContext mocks the WithContext method.
func (m *MockLogger) WithContext(ctx context.Context) logger.Logger {
	callArgs := m.Called(ctx)
	if callArgs.Get(0) == nil {
		return m
	}
	return callArgs.Get(0).(logger.Logger)
}

// WithComponent mocks the WithComponent method.
func (m *MockLogger) WithComponent(component string) logger.Logger {
	callArgs := m.Called(component)
	if callArgs.Get(0) == nil {
		return m
	}
	return callArgs.Get(0).(logger.Logger)
}
