// Package csv provides error types and recovery modes for CSV parsing.
package csv

import (
	"errors"
	"fmt"
)

// BadLineMode specifies how the parser handles malformed CSV lines.
type BadLineMode int

const (
	// BadLineModeError returns an error on malformed lines (default).
	BadLineModeError BadLineMode = iota
	// BadLineModeWarn logs a warning but continues parsing.
	BadLineModeWarn
	// BadLineModeSkip silently skips malformed lines.
	BadLineModeSkip
)

// String returns the string representation of BadLineMode.
func (m BadLineMode) String() string {
	switch m {
	case BadLineModeError:
		return "error"
	case BadLineModeWarn:
		return "warn"
	case BadLineModeSkip:
		return "skip"
	default:
		return fmt.Sprintf("BadLineMode(%d)", m)
	}
}

// ParseError represents a parsing error with position information.
// It provides detailed context about where the error occurred in the CSV data.
type ParseError struct {
	// StartLine is the line where parsing started for this record (1-indexed).
	StartLine int
	// Line is the current line where the error occurred (1-indexed).
	Line int
	// Column is the column where the error occurred (1-indexed).
	Column int
	// Err is the underlying error.
	Err error
}

// Error returns a formatted error message with position information.
func (e *ParseError) Error() string {
	if e.StartLine == e.Line {
		return fmt.Sprintf("parse error on line %d, column %d: %v", e.Line, e.Column, e.Err)
	}
	return fmt.Sprintf("parse error on line %d (started line %d), column %d: %v",
		e.Line, e.StartLine, e.Column, e.Err)
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Common parsing errors
var (
	// ErrQuote indicates a quote-related parsing error.
	ErrQuote = errors.New("bare \" in non-quoted-field")

	// ErrFieldCount indicates a record has the wrong number of fields.
	ErrFieldCount = errors.New("wrong number of fields")

	// ErrFieldTooLarge indicates a field exceeded MaxFieldSize.
	ErrFieldTooLarge = errors.New("field exceeds maximum size")

	// ErrRecordTooLarge indicates a record exceeded MaxRecordSize.
	ErrRecordTooLarge = errors.New("record exceeds maximum size")
)

// BadLineHandler is a callback function invoked when a bad line is encountered.
// It receives the line number, the raw line content, and the error.
// Return true to continue parsing, false to stop.
type BadLineHandler func(line int, content string, err error) bool

// WarningHandler is a callback function for logging warnings.
type WarningHandler func(line int, message string)

// ErrorRecoveryOptions configures error handling behavior.
type ErrorRecoveryOptions struct {
	// OnBadLine specifies how to handle malformed lines.
	// Default: BadLineModeError
	OnBadLine BadLineMode

	// BadLineCallback is invoked when a bad line is encountered.
	// Only called when OnBadLine is not BadLineModeError.
	// If nil, a default handler is used.
	BadLineCallback BadLineHandler

	// WarningCallback is invoked for warnings (when OnBadLine is BadLineModeWarn).
	// If nil, warnings are silently ignored.
	WarningCallback WarningHandler

	// MaxFieldSize is the maximum allowed size for a single field in bytes.
	// 0 means no limit.
	MaxFieldSize int

	// MaxRecordSize is the maximum allowed size for a single record in bytes.
	// 0 means no limit.
	MaxRecordSize int
}

// DefaultErrorRecoveryOptions returns the default error recovery configuration.
func DefaultErrorRecoveryOptions() ErrorRecoveryOptions {
	return ErrorRecoveryOptions{
		OnBadLine:       BadLineModeError,
		BadLineCallback: nil,
		WarningCallback: nil,
		MaxFieldSize:    0,
		MaxRecordSize:   0,
	}
}
