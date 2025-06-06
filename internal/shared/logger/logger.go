package logger

import (
	"fmt"
	"log"
	"os"
)

// Logger defines the interface for logging operations
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
}

// DefaultLogger implements the Logger interface using standard log package
type DefaultLogger struct {
	logger *log.Logger
	fields map[string]interface{}
}

// NewDefaultLogger creates a new default logger instance
func NewDefaultLogger() Logger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		fields: make(map[string]interface{}),
	}
}

func (l *DefaultLogger) Debug(args ...interface{}) {
	l.log("DEBUG", fmt.Sprint(args...))
}

func (l *DefaultLogger) Info(args ...interface{}) {
	l.log("INFO", fmt.Sprint(args...))
}

func (l *DefaultLogger) Warn(args ...interface{}) {
	l.log("WARN", fmt.Sprint(args...))
}

func (l *DefaultLogger) Error(args ...interface{}) {
	l.log("ERROR", fmt.Sprint(args...))
}

func (l *DefaultLogger) Fatal(args ...interface{}) {
	l.log("FATAL", fmt.Sprint(args...))
	os.Exit(1)
}

func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	l.log("DEBUG", fmt.Sprintf(format, args...))
}

func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	l.log("INFO", fmt.Sprintf(format, args...))
}

func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	l.log("WARN", fmt.Sprintf(format, args...))
}

func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	l.log("ERROR", fmt.Sprintf(format, args...))
}

func (l *DefaultLogger) Fatalf(format string, args ...interface{}) {
	l.log("FATAL", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &DefaultLogger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *DefaultLogger) log(level string, message string) {
	fieldsStr := ""
	if len(l.fields) > 0 {
		fieldsStr = " ["
		first := true
		for k, v := range l.fields {
			if !first {
				fieldsStr += ", "
			}
			fieldsStr += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
		fieldsStr += "]"
	}

	l.logger.Printf("[%s]%s %s", level, fieldsStr, message)
}
