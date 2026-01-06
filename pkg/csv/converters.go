// Package csv provides type converters for CSV field values.
package csv

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Converter is the interface for type converters.
// Converters transform string field values into typed Go values.
type Converter interface {
	// Convert transforms a string value into the target type.
	// Returns the converted value and any error encountered.
	Convert(value string) (interface{}, error)
}

// ConverterFunc is a function adapter for the Converter interface.
type ConverterFunc func(string) (interface{}, error)

// Convert implements Converter.
func (f ConverterFunc) Convert(value string) (interface{}, error) {
	return f(value)
}

// IntConverter converts string values to int64.
type IntConverter struct {
	// Base is the numeric base for parsing (default: 10)
	Base int
}

// Convert implements Converter for IntConverter.
func (c IntConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return int64(0), nil
	}
	base := c.Base
	if base == 0 {
		base = 10
	}
	return strconv.ParseInt(strings.TrimSpace(value), base, 64)
}

// FloatConverter converts string values to float64.
type FloatConverter struct{}

// Convert implements Converter for FloatConverter.
func (c FloatConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return float64(0), nil
	}
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

// BoolConverter converts string values to bool.
// Recognizes: true/false, 1/0, yes/no, y/n, on/off, t/f (case-insensitive)
type BoolConverter struct{}

// Convert implements Converter for BoolConverter.
func (c BoolConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return false, nil
	}
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "true", "1", "yes", "y", "on", "t":
		return true, nil
	case "false", "0", "no", "n", "off", "f":
		return false, nil
	default:
		return false, fmt.Errorf("cannot convert %q to bool", value)
	}
}

// DateConverter converts string values to time.Time.
type DateConverter struct {
	// Format is the date format string (default: "2006-01-02")
	Format string
	// Location is the timezone for parsing (default: UTC)
	Location *time.Location
}

// Convert implements Converter for DateConverter.
func (c DateConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return time.Time{}, nil
	}
	format := c.Format
	if format == "" {
		format = "2006-01-02"
	}
	loc := c.Location
	if loc == nil {
		loc = time.UTC
	}
	return time.ParseInLocation(format, strings.TrimSpace(value), loc)
}

// TimeConverter converts string values to time.Time with time component.
type TimeConverter struct {
	// Format is the time format string (default: "15:04:05")
	Format string
	// Location is the timezone for parsing (default: UTC)
	Location *time.Location
}

// Convert implements Converter for TimeConverter.
func (c TimeConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return time.Time{}, nil
	}
	format := c.Format
	if format == "" {
		format = "15:04:05"
	}
	loc := c.Location
	if loc == nil {
		loc = time.UTC
	}
	return time.ParseInLocation(format, strings.TrimSpace(value), loc)
}

// DateTimeConverter converts string values to time.Time with date and time.
type DateTimeConverter struct {
	// Format is the datetime format string (default: "2006-01-02 15:04:05")
	Format string
	// Location is the timezone for parsing (default: UTC)
	Location *time.Location
}

// Convert implements Converter for DateTimeConverter.
func (c DateTimeConverter) Convert(value string) (interface{}, error) {
	if value == "" {
		return time.Time{}, nil
	}
	format := c.Format
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	loc := c.Location
	if loc == nil {
		loc = time.UTC
	}
	return time.ParseInLocation(format, strings.TrimSpace(value), loc)
}

// ConverterRegistry manages named converters.
type ConverterRegistry struct {
	converters map[string]Converter
}

// NewConverterRegistry creates a new converter registry with built-in converters.
func NewConverterRegistry() *ConverterRegistry {
	r := &ConverterRegistry{
		converters: make(map[string]Converter),
	}
	// Register built-in converters
	r.Register("int", IntConverter{})
	r.Register("float", FloatConverter{})
	r.Register("bool", BoolConverter{})
	r.Register("date", DateConverter{})
	r.Register("time", TimeConverter{})
	r.Register("datetime", DateTimeConverter{})
	return r
}

// Register adds a converter to the registry.
func (r *ConverterRegistry) Register(name string, conv Converter) {
	r.converters[name] = conv
}

// Get retrieves a converter by name.
func (r *ConverterRegistry) Get(name string) (Converter, bool) {
	conv, ok := r.converters[name]
	return conv, ok
}

// InferType attempts to infer the type of a string value.
// Returns the inferred type name and converted value.
func InferType(value string) (string, interface{}) {
	if value == "" {
		return "string", value
	}

	v := strings.TrimSpace(value)

	// Try bool
	lower := strings.ToLower(v)
	if lower == "true" || lower == "false" {
		return "bool", lower == "true"
	}

	// Try int
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return "int", i
	}

	// Try float
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return "float", f
	}

	// Try date (common formats)
	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"02-Jan-2006",
	}
	for _, fmt := range dateFormats {
		if t, err := time.Parse(fmt, v); err == nil {
			return "date", t
		}
	}

	// Default to string
	return "string", value
}

// NullValues is a list of values that should be treated as null/nil.
var DefaultNullValues = []string{"", "NULL", "null", "nil", "N/A", "n/a", "NA", "na", "-"}

// IsNullValue checks if a value should be treated as null.
func IsNullValue(value string, nullValues []string) bool {
	for _, nv := range nullValues {
		if value == nv {
			return true
		}
	}
	return false
}

// TypeInferenceOptions configures type inference behavior.
type TypeInferenceOptions struct {
	// InferTypes enables automatic type detection.
	InferTypes bool
	// NullValues is a list of values to treat as null.
	NullValues []string
	// Converters is a map of column names to converter names.
	Converters map[string]string
	// Registry is the converter registry to use.
	Registry *ConverterRegistry
}

// DefaultTypeInferenceOptions returns default type inference options.
func DefaultTypeInferenceOptions() TypeInferenceOptions {
	return TypeInferenceOptions{
		InferTypes: false,
		NullValues: DefaultNullValues,
		Converters: make(map[string]string),
		Registry:   NewConverterRegistry(),
	}
}
