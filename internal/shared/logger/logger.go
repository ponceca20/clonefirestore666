package logger

import (
	"context"
	"os"

	"firestore-clone/internal/shared/contextkeys"

	"github.com/sirupsen/logrus"
)

// Constants for configuration
const (
	// Log levels
	logLevelDebug = "DEBUG"
	logLevelInfo  = "INFO"
	logLevelWarn  = "WARN"
	logLevelError = "ERROR"
	logLevelFatal = "FATAL"

	// Log formats
	logFormatJSON = "json"
	logFormatText = "text"

	// Environment types
	envProduction = "production"
	envProd       = "prod"

	// Timestamp format
	timestampFormat = "2006-01-02T15:04:05.000Z07:00"
	textTimestamp   = "2006-01-02 15:04:05"
)

// Logger defines the interface for structured logging operations
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
	WithComponent(component string) Logger
}

// LogrusLogger implements the Logger interface using logrus
type LogrusLogger struct {
	entry *logrus.Entry
}

// NewLogger creates a new logger instance with default configuration
func NewLogger() Logger {
	logger := logrus.New()

	// Set default configuration
	logger.SetLevel(getLogLevel())
	logger.SetFormatter(getLogFormatter())
	logger.SetOutput(os.Stdout)

	return &LogrusLogger{
		entry: logrus.NewEntry(logger),
	}
}

// NewLoggerWithConfig creates a logger with custom configuration
func NewLoggerWithConfig(level string, format string) Logger {
	logger := logrus.New()

	// Set level
	if parsedLevel, err := logrus.ParseLevel(level); err == nil {
		logger.SetLevel(parsedLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	switch format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	logger.SetOutput(os.Stdout)

	return &LogrusLogger{
		entry: logrus.NewEntry(logger),
	}
}

// Debug logs a debug message
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info logs an info message
func (l *LogrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Warn logs a warning message
func (l *LogrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error logs an error message
func (l *LogrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Fatal logs a fatal message and exits
func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Debugf logs a formatted debug message
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// WithFields adds structured fields to the logger
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		entry: l.entry.WithFields(logrus.Fields(fields)),
	}
}

// WithContext adds context information to the logger using proper context keys
func (l *LogrusLogger) WithContext(ctx context.Context) Logger {
	fields := logrus.Fields{}

	// Extract context values using proper context keys
	l.addContextField(ctx, contextkeys.UserIDKey, "user_id", fields)
	l.addContextField(ctx, contextkeys.TenantIDKey, "tenant_id", fields)
	l.addContextField(ctx, contextkeys.ProjectIDKey, "project_id", fields)
	l.addContextField(ctx, contextkeys.DatabaseIDKey, "database_id", fields)
	l.addContextField(ctx, contextkeys.RequestIDKey, "request_id", fields)
	l.addContextField(ctx, contextkeys.ComponentKey, "component", fields)
	l.addContextField(ctx, contextkeys.OperationKey, "operation", fields)

	return &LogrusLogger{
		entry: l.entry.WithFields(fields),
	}
}

// addContextField extracts a value from context and adds it to fields if present
func (l *LogrusLogger) addContextField(ctx context.Context, key interface{}, fieldName string, fields logrus.Fields) {
	if val := ctx.Value(key); val != nil {
		if strVal, ok := val.(string); ok && strVal != "" {
			fields[fieldName] = strVal
		}
	}
}

// WithComponent adds component name to the logger
func (l *LogrusLogger) WithComponent(component string) Logger {
	return &LogrusLogger{
		entry: l.entry.WithField("component", component),
	}
}

// Helper functions

// getLogLevel determines the log level from environment
func getLogLevel() logrus.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case logLevelDebug, "debug":
		return logrus.DebugLevel
	case logLevelInfo, "info":
		return logrus.InfoLevel
	case logLevelWarn, "warn", "WARNING", "warning":
		return logrus.WarnLevel
	case logLevelError, "error":
		return logrus.ErrorLevel
	case logLevelFatal, "fatal":
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}

// getLogFormatter determines the log formatter from environment
func getLogFormatter() logrus.Formatter {
	env := os.Getenv("ENVIRONMENT")
	format := os.Getenv("LOG_FORMAT")

	if format == logFormatJSON || env == envProduction || env == envProd {
		return &logrus.JSONFormatter{
			TimestampFormat: timestampFormat,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		}
	}

	// Text formatter for development
	return &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: textTimestamp,
		ForceColors:     true,
	}
}

// Global logger instance
var defaultLogger Logger

// Initialize default logger
func init() {
	defaultLogger = NewLogger()
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Info logs an info message using the default logger
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Warn logs a warning message using the default logger
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Error logs an error message using the default logger
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Fatal logs a fatal message using the default logger
func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

// Debugf logs a formatted debug message using the default logger
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Infof logs a formatted info message using the default logger
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warnf logs a formatted warning message using the default logger
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Errorf logs a formatted error message using the default logger
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message using the default logger
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// WithContext creates a logger with context information
func WithContext(ctx context.Context) Logger {
	return defaultLogger.WithContext(ctx)
}

// WithComponent creates a logger with component information
func WithComponent(component string) Logger {
	return defaultLogger.WithComponent(component)
}

// WithFields creates a logger with custom fields
func WithFields(fields map[string]interface{}) Logger {
	return defaultLogger.WithFields(fields)
}
