package model

import (
	"regexp"
	"time"
)

// TimestampParser provides utilities for parsing timestamp strings
type TimestampParser struct{}

// NewTimestampParser creates a new TimestampParser
func NewTimestampParser() *TimestampParser {
	return &TimestampParser{}
}

// Common timestamp formats supported by Firestore (in order of priority)
var supportedTimestampFormats = []string{
	time.RFC3339,                     // "2006-01-02T15:04:05Z07:00" - Most common
	time.RFC3339Nano,                 // "2006-01-02T15:04:05.999999999Z07:00"
	"2006-01-02T15:04:05Z",           // ISO 8601 UTC
	"2006-01-02T15:04:05.000Z",       // ISO 8601 UTC with milliseconds
	"2006-01-02T15:04:05.000000Z",    // ISO 8601 UTC with microseconds
	"2006-01-02T15:04:05.000000000Z", // ISO 8601 UTC with nanoseconds
	"2006-01-02 15:04:05",            // Simple format
	"2006-01-02",                     // Date only
}

// Optimized regex patterns for timestamp detection (compiled once)
var timestampPatterns = []*regexp.Regexp{
	// ISO 8601 formats (most specific first)
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:?\d{2})$`),
	// Simple datetime
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`),
	// Date only (YYYY-MM-DD)
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
}

// IsTimestampString efficiently determines if a string represents a timestamp
func (tp *TimestampParser) IsTimestampString(s string) bool {
	// Quick length checks to avoid regex overhead
	if len(s) < 10 || len(s) > 35 {
		return false
	}

	// Check against optimized regex patterns
	for _, pattern := range timestampPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	return false
}

// ParseTimestamp attempts to parse a string as a timestamp with early exit
func (tp *TimestampParser) ParseTimestamp(s string) (time.Time, error) {
	// Try each supported format in order of likelihood
	for _, format := range supportedTimestampFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	// If no format matches, return error
	return time.Time{}, &TimestampParseError{Input: s}
}

// TimestampParseError represents a timestamp parsing error
type TimestampParseError struct {
	Input string
}

func (e *TimestampParseError) Error() string {
	return "cannot parse '" + e.Input + "' as timestamp"
}

// ParseWithHint parses with explicit type hint from client
func (tp *TimestampParser) ParseWithHint(value interface{}, typeHint string) (time.Time, bool) {
	if typeHint != "timestampValue" {
		return time.Time{}, false
	}

	switch v := value.(type) {
	case string:
		if t, err := tp.ParseTimestamp(v); err == nil {
			return t, true
		}
	case time.Time:
		return v, true
	}

	return time.Time{}, false
}

// TryParseAsTimestamp is the main function - tries to parse any value as timestamp
// This is the ONLY function that should be used for timestamp detection
func (tp *TimestampParser) TryParseAsTimestamp(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		// Only attempt parsing if it looks like a timestamp pattern
		if tp.IsTimestampString(v) {
			if t, err := tp.ParseTimestamp(v); err == nil {
				return t, true
			}
		}
	}

	return time.Time{}, false
}
