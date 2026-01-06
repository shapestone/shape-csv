// Package csv provides schema definition and validation for CSV data.
package csv

import (
	"fmt"
	"reflect"
	"strings"
)

// ColumnType represents the expected type of a column.
type ColumnType string

const (
	ColumnTypeString   ColumnType = "string"
	ColumnTypeInt      ColumnType = "int"
	ColumnTypeFloat    ColumnType = "float"
	ColumnTypeBool     ColumnType = "bool"
	ColumnTypeDate     ColumnType = "date"
	ColumnTypeTime     ColumnType = "time"
	ColumnTypeDateTime ColumnType = "datetime"
	ColumnTypeAny      ColumnType = "any"
)

// ColumnDefinition defines the schema for a single column.
type ColumnDefinition struct {
	// Name is the column header name.
	Name string
	// Type is the expected data type.
	Type ColumnType
	// Required indicates if the column must have a value.
	Required bool
	// Default is the default value for empty fields.
	Default string
	// Validator is an optional custom validation function.
	Validator func(value string) error
	// AllowedValues restricts values to a specific set.
	AllowedValues []string
	// MinLength is the minimum string length (0 = no minimum).
	MinLength int
	// MaxLength is the maximum string length (0 = no maximum).
	MaxLength int
}

// Schema defines the expected structure of CSV data.
type Schema struct {
	// Columns defines the expected columns in order.
	Columns []ColumnDefinition
	// AllowExtraColumns permits columns not defined in schema.
	AllowExtraColumns bool
	// AllowMissingColumns permits missing columns from schema.
	AllowMissingColumns bool
	// HeaderRequired indicates if CSV must have a header row.
	HeaderRequired bool
}

// NewSchema creates a new empty schema.
func NewSchema() *Schema {
	return &Schema{
		Columns:         make([]ColumnDefinition, 0),
		HeaderRequired:  true,
		AllowExtraColumns: false,
		AllowMissingColumns: false,
	}
}

// AddColumn adds a column definition to the schema.
func (s *Schema) AddColumn(col ColumnDefinition) *Schema {
	s.Columns = append(s.Columns, col)
	return s
}

// AddSimpleColumn adds a column with just name and type.
func (s *Schema) AddSimpleColumn(name string, colType ColumnType) *Schema {
	return s.AddColumn(ColumnDefinition{
		Name: name,
		Type: colType,
	})
}

// AddRequiredColumn adds a required column with name and type.
func (s *Schema) AddRequiredColumn(name string, colType ColumnType) *Schema {
	return s.AddColumn(ColumnDefinition{
		Name:     name,
		Type:     colType,
		Required: true,
	})
}

// ValidationError represents a schema validation error.
type ValidationError struct {
	// Row is the row number (0-indexed, -1 for header).
	Row int
	// Column is the column name or index.
	Column string
	// Value is the invalid value.
	Value string
	// Message describes the validation failure.
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Row < 0 {
		return fmt.Sprintf("header validation error for column %q: %s", e.Column, e.Message)
	}
	return fmt.Sprintf("row %d, column %q: %s (value: %q)", e.Row, e.Column, e.Message, e.Value)
}

// ValidationResult contains all validation errors.
type ValidationResult struct {
	// Valid indicates if validation passed.
	Valid bool
	// Errors contains all validation errors.
	Errors []ValidationError
}

// AddError adds an error to the result.
func (r *ValidationResult) AddError(err ValidationError) {
	r.Errors = append(r.Errors, err)
	r.Valid = false
}

// Error returns the first error message or empty string if valid.
func (r *ValidationResult) Error() string {
	if r.Valid || len(r.Errors) == 0 {
		return ""
	}
	return r.Errors[0].Error()
}

// AllErrors returns all error messages joined by newlines.
func (r *ValidationResult) AllErrors() string {
	if r.Valid || len(r.Errors) == 0 {
		return ""
	}
	msgs := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "\n")
}

// ValidateSchema validates CSV data against a schema.
// data should be a slice of records ([][]string) where each record is a row of fields.
func ValidateSchema(data [][]string, schema *Schema) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if len(data) == 0 {
		if schema.HeaderRequired {
			result.AddError(ValidationError{
				Row:     -1,
				Message: "CSV data is empty, header required",
			})
		}
		return result
	}

	// Build column index map from header
	header := data[0]
	columnIndex := make(map[string]int)
	for i, name := range header {
		columnIndex[name] = i
	}

	// Validate header has required columns
	for _, col := range schema.Columns {
		if _, exists := columnIndex[col.Name]; !exists && !schema.AllowMissingColumns {
			result.AddError(ValidationError{
				Row:     -1,
				Column:  col.Name,
				Message: "required column not found in header",
			})
		}
	}

	// Check for extra columns
	if !schema.AllowExtraColumns {
		schemaColumns := make(map[string]bool)
		for _, col := range schema.Columns {
			schemaColumns[col.Name] = true
		}
		for _, name := range header {
			if !schemaColumns[name] {
				result.AddError(ValidationError{
					Row:     -1,
					Column:  name,
					Message: "unexpected column not in schema",
				})
			}
		}
	}

	// Validate data rows
	for rowIdx := 1; rowIdx < len(data); rowIdx++ {
		row := data[rowIdx]

		for _, col := range schema.Columns {
			colIdx, exists := columnIndex[col.Name]
			if !exists {
				continue // Already reported as missing
			}

			var value string
			if colIdx < len(row) {
				value = row[colIdx]
			}

			// Apply default for empty values
			if value == "" && col.Default != "" {
				value = col.Default
			}

			// Required validation
			if col.Required && value == "" {
				result.AddError(ValidationError{
					Row:     rowIdx,
					Column:  col.Name,
					Value:   value,
					Message: "required field is empty",
				})
				continue
			}

			// Skip further validation for empty optional fields
			if value == "" {
				continue
			}

			// Type validation
			if err := validateType(value, col.Type); err != nil {
				result.AddError(ValidationError{
					Row:     rowIdx,
					Column:  col.Name,
					Value:   value,
					Message: err.Error(),
				})
			}

			// Allowed values validation
			if len(col.AllowedValues) > 0 {
				found := false
				for _, allowed := range col.AllowedValues {
					if value == allowed {
						found = true
						break
					}
				}
				if !found {
					result.AddError(ValidationError{
						Row:     rowIdx,
						Column:  col.Name,
						Value:   value,
						Message: fmt.Sprintf("value not in allowed set: %v", col.AllowedValues),
					})
				}
			}

			// Length validation
			if col.MinLength > 0 && len(value) < col.MinLength {
				result.AddError(ValidationError{
					Row:     rowIdx,
					Column:  col.Name,
					Value:   value,
					Message: fmt.Sprintf("value length %d is less than minimum %d", len(value), col.MinLength),
				})
			}
			if col.MaxLength > 0 && len(value) > col.MaxLength {
				result.AddError(ValidationError{
					Row:     rowIdx,
					Column:  col.Name,
					Value:   value,
					Message: fmt.Sprintf("value length %d exceeds maximum %d", len(value), col.MaxLength),
				})
			}

			// Custom validator
			if col.Validator != nil {
				if err := col.Validator(value); err != nil {
					result.AddError(ValidationError{
						Row:     rowIdx,
						Column:  col.Name,
						Value:   value,
						Message: err.Error(),
					})
				}
			}
		}
	}

	return result
}

// validateType checks if a value matches the expected type.
func validateType(value string, colType ColumnType) error {
	if colType == ColumnTypeAny || colType == ColumnTypeString {
		return nil
	}

	registry := NewConverterRegistry()

	switch colType {
	case ColumnTypeInt:
		conv, _ := registry.Get("int")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
	case ColumnTypeFloat:
		conv, _ := registry.Get("float")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid float: %s", value)
		}
	case ColumnTypeBool:
		conv, _ := registry.Get("bool")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
	case ColumnTypeDate:
		conv, _ := registry.Get("date")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid date: %s", value)
		}
	case ColumnTypeTime:
		conv, _ := registry.Get("time")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid time: %s", value)
		}
	case ColumnTypeDateTime:
		conv, _ := registry.Get("datetime")
		if _, err := conv.Convert(value); err != nil {
			return fmt.Errorf("invalid datetime: %s", value)
		}
	}

	return nil
}

// SchemaFromStruct creates a schema from a struct type using csv tags.
func SchemaFromStruct(v interface{}) (*Schema, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("SchemaFromStruct requires a struct type, got %s", t.Kind())
	}

	schema := NewSchema()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("csv")
		if tag == "-" {
			continue
		}

		name := field.Name
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" {
			name = parts[0]
		}

		col := ColumnDefinition{
			Name: name,
			Type: goTypeToColumnType(field.Type),
		}

		// Parse tag options
		for _, part := range parts[1:] {
			if part == "required" {
				col.Required = true
			}
		}

		schema.AddColumn(col)
	}

	return schema, nil
}

// goTypeToColumnType maps Go types to column types.
func goTypeToColumnType(t reflect.Type) ColumnType {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ColumnTypeInt
	case reflect.Float32, reflect.Float64:
		return ColumnTypeFloat
	case reflect.Bool:
		return ColumnTypeBool
	case reflect.String:
		return ColumnTypeString
	default:
		// Check for time.Time
		if t.String() == "time.Time" {
			return ColumnTypeDateTime
		}
		return ColumnTypeAny
	}
}
