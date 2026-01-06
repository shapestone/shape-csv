package csv_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestSchemaCreation(t *testing.T) {
	t.Run("new schema", func(t *testing.T) {
		schema := csv.NewSchema()
		if schema == nil {
			t.Fatal("NewSchema returned nil")
		}
		if len(schema.Columns) != 0 {
			t.Error("new schema should have no columns")
		}
	})

	t.Run("add column", func(t *testing.T) {
		schema := csv.NewSchema().
			AddColumn(csv.ColumnDefinition{Name: "id", Type: csv.ColumnTypeInt}).
			AddColumn(csv.ColumnDefinition{Name: "name", Type: csv.ColumnTypeString})

		if len(schema.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(schema.Columns))
		}
	})

	t.Run("add simple column", func(t *testing.T) {
		schema := csv.NewSchema().
			AddSimpleColumn("name", csv.ColumnTypeString)

		if len(schema.Columns) != 1 {
			t.Fatal("expected 1 column")
		}
		if schema.Columns[0].Name != "name" {
			t.Error("column name mismatch")
		}
		if schema.Columns[0].Required {
			t.Error("simple column should not be required")
		}
	})

	t.Run("add required column", func(t *testing.T) {
		schema := csv.NewSchema().
			AddRequiredColumn("id", csv.ColumnTypeInt)

		if !schema.Columns[0].Required {
			t.Error("column should be required")
		}
	})
}

func TestValidateSchema(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		schema := csv.NewSchema().
			AddSimpleColumn("name", csv.ColumnTypeString).
			AddSimpleColumn("age", csv.ColumnTypeInt)

		data := [][]string{
			{"name", "age"},
			{"John", "30"},
			{"Jane", "25"},
		}

		result := csv.ValidateSchema(data, schema)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %s", result.AllErrors())
		}
	})

	t.Run("missing required column in header", func(t *testing.T) {
		schema := csv.NewSchema().
			AddRequiredColumn("name", csv.ColumnTypeString).
			AddRequiredColumn("email", csv.ColumnTypeString)

		data := [][]string{
			{"name", "age"},
			{"John", "30"},
		}

		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to missing column")
		}
		if len(result.Errors) == 0 {
			t.Error("expected error for missing column")
		}
	})

	t.Run("required field empty", func(t *testing.T) {
		schema := csv.NewSchema().
			AddRequiredColumn("name", csv.ColumnTypeString).
			AddSimpleColumn("age", csv.ColumnTypeInt)

		data := [][]string{
			{"name", "age"},
			{"", "30"},
		}

		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to empty required field")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		schema := csv.NewSchema().
			AddSimpleColumn("age", csv.ColumnTypeInt)

		data := [][]string{
			{"age"},
			{"not a number"},
		}

		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to type mismatch")
		}
	})

	t.Run("extra columns not allowed", func(t *testing.T) {
		schema := csv.NewSchema().
			AddSimpleColumn("name", csv.ColumnTypeString)
		schema.AllowExtraColumns = false

		data := [][]string{
			{"name", "extra"},
			{"John", "value"},
		}

		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to extra column")
		}
	})

	t.Run("extra columns allowed", func(t *testing.T) {
		schema := csv.NewSchema().
			AddSimpleColumn("name", csv.ColumnTypeString)
		schema.AllowExtraColumns = true

		data := [][]string{
			{"name", "extra"},
			{"John", "value"},
		}

		result := csv.ValidateSchema(data, schema)
		if !result.Valid {
			t.Errorf("expected valid with extra columns allowed: %s", result.AllErrors())
		}
	})

	t.Run("allowed values", func(t *testing.T) {
		schema := csv.NewSchema().
			AddColumn(csv.ColumnDefinition{
				Name:          "status",
				Type:          csv.ColumnTypeString,
				AllowedValues: []string{"active", "inactive"},
			})

		data := [][]string{
			{"status"},
			{"active"},
			{"invalid"},
		}

		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to disallowed value")
		}
	})

	t.Run("length validation", func(t *testing.T) {
		schema := csv.NewSchema().
			AddColumn(csv.ColumnDefinition{
				Name:      "code",
				Type:      csv.ColumnTypeString,
				MinLength: 3,
				MaxLength: 5,
			})

		// Test min length
		data := [][]string{
			{"code"},
			{"ab"},
		}
		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to min length")
		}

		// Test max length
		data = [][]string{
			{"code"},
			{"abcdef"},
		}
		result = csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to max length")
		}

		// Test valid length
		data = [][]string{
			{"code"},
			{"abcd"},
		}
		result = csv.ValidateSchema(data, schema)
		if !result.Valid {
			t.Errorf("expected valid: %s", result.AllErrors())
		}
	})

	t.Run("custom validator", func(t *testing.T) {
		schema := csv.NewSchema().
			AddColumn(csv.ColumnDefinition{
				Name: "email",
				Type: csv.ColumnTypeString,
				Validator: func(value string) error {
					if value != "" && !containsAt(value) {
						return errors.New("must contain @")
					}
					return nil
				},
			})

		data := [][]string{
			{"email"},
			{"invalid"},
		}
		result := csv.ValidateSchema(data, schema)
		if result.Valid {
			t.Error("expected invalid due to custom validator")
		}

		data = [][]string{
			{"email"},
			{"test@example.com"},
		}
		result = csv.ValidateSchema(data, schema)
		if !result.Valid {
			t.Errorf("expected valid: %s", result.AllErrors())
		}
	})

	t.Run("default value", func(t *testing.T) {
		schema := csv.NewSchema().
			AddColumn(csv.ColumnDefinition{
				Name:      "status",
				Type:      csv.ColumnTypeString,
				Required:  true,
				Default:   "pending",
				AllowedValues: []string{"pending", "active"},
			})

		// Empty value with default should pass required check
		data := [][]string{
			{"status"},
			{""},
		}
		result := csv.ValidateSchema(data, schema)
		if !result.Valid {
			t.Errorf("expected valid with default value: %s", result.AllErrors())
		}
	})

	t.Run("empty data with header required", func(t *testing.T) {
		schema := csv.NewSchema()
		schema.HeaderRequired = true

		result := csv.ValidateSchema([][]string{}, schema)
		if result.Valid {
			t.Error("expected invalid for empty data with header required")
		}
	})
}

func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

func TestValidationError(t *testing.T) {
	t.Run("header error message", func(t *testing.T) {
		err := csv.ValidationError{
			Row:     -1,
			Column:  "email",
			Message: "required column not found",
		}
		msg := err.Error()
		if msg == "" {
			t.Error("error message should not be empty")
		}
		if !containsSubstring(msg, "header") {
			t.Error("header error should mention header")
		}
	})

	t.Run("row error message", func(t *testing.T) {
		err := csv.ValidationError{
			Row:     5,
			Column:  "age",
			Value:   "abc",
			Message: "invalid integer",
		}
		msg := err.Error()
		if !containsSubstring(msg, "5") {
			t.Error("error should include row number")
		}
		if !containsSubstring(msg, "age") {
			t.Error("error should include column name")
		}
		if !containsSubstring(msg, "abc") {
			t.Error("error should include value")
		}
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := &csv.ValidationResult{Valid: true}
		if result.Error() != "" {
			t.Error("valid result should have empty error")
		}
		if result.AllErrors() != "" {
			t.Error("valid result should have empty all errors")
		}
	})

	t.Run("invalid result with multiple errors", func(t *testing.T) {
		result := &csv.ValidationResult{Valid: true}
		result.AddError(csv.ValidationError{Row: 1, Column: "a", Message: "error 1"})
		result.AddError(csv.ValidationError{Row: 2, Column: "b", Message: "error 2"})

		if result.Valid {
			t.Error("result should be invalid after adding errors")
		}
		if len(result.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result.Errors))
		}

		// Error() returns first error
		firstErr := result.Error()
		if !containsSubstring(firstErr, "error 1") {
			t.Error("Error() should return first error")
		}

		// AllErrors() returns all
		allErrs := result.AllErrors()
		if !containsSubstring(allErrs, "error 1") || !containsSubstring(allErrs, "error 2") {
			t.Error("AllErrors() should contain all errors")
		}
	})
}

func TestSchemaFromStruct(t *testing.T) {
	type Person struct {
		Name  string `csv:"name"`
		Age   int    `csv:"age"`
		Email string `csv:"email,required"`
	}

	schema, err := csv.SchemaFromStruct(Person{})
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	if len(schema.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(schema.Columns))
	}

	// Check name column
	if schema.Columns[0].Name != "name" {
		t.Errorf("expected column name 'name', got %q", schema.Columns[0].Name)
	}
	if schema.Columns[0].Type != csv.ColumnTypeString {
		t.Errorf("expected string type for name")
	}

	// Check age column
	if schema.Columns[1].Name != "age" {
		t.Errorf("expected column name 'age', got %q", schema.Columns[1].Name)
	}
	if schema.Columns[1].Type != csv.ColumnTypeInt {
		t.Errorf("expected int type for age")
	}

	// Check email column with required
	if schema.Columns[2].Name != "email" {
		t.Errorf("expected column name 'email', got %q", schema.Columns[2].Name)
	}
	if !schema.Columns[2].Required {
		t.Error("email should be required")
	}
}

func TestSchemaFromStructPointer(t *testing.T) {
	type Item struct {
		ID   int     `csv:"id"`
		Name string  `csv:"name"`
		Price float64 `csv:"price"`
	}

	schema, err := csv.SchemaFromStruct(&Item{})
	if err != nil {
		t.Fatalf("SchemaFromStruct with pointer failed: %v", err)
	}

	if len(schema.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(schema.Columns))
	}

	if schema.Columns[2].Type != csv.ColumnTypeFloat {
		t.Error("expected float type for price")
	}
}

func TestSchemaFromStructNonStruct(t *testing.T) {
	_, err := csv.SchemaFromStruct("not a struct")
	if err == nil {
		t.Error("expected error for non-struct type")
	}
}

func TestSchemaFromStructSkipField(t *testing.T) {
	type Record struct {
		Public  string `csv:"public"`
		Private string `csv:"-"`
	}

	schema, err := csv.SchemaFromStruct(Record{})
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	if len(schema.Columns) != 1 {
		t.Errorf("expected 1 column (skipping '-'), got %d", len(schema.Columns))
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestValidateType tests the validateType function indirectly through schema validation
func TestValidateTypeWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name    string
		colType csv.ColumnType
		value   string
		wantErr bool
	}{
		// Valid cases
		{"valid int", csv.ColumnTypeInt, "42", false},
		{"valid float", csv.ColumnTypeFloat, "3.14", false},
		{"valid bool true", csv.ColumnTypeBool, "true", false},
		{"valid bool false", csv.ColumnTypeBool, "false", false},
		{"valid string", csv.ColumnTypeString, "hello", false},
		{"valid any", csv.ColumnTypeAny, "anything", false},
		{"valid date", csv.ColumnTypeDate, "2023-01-15", false},
		{"valid time", csv.ColumnTypeTime, "14:30:00", false},
		{"valid datetime", csv.ColumnTypeDateTime, "2023-01-15 14:30:00", false},

		// Invalid cases
		{"invalid int", csv.ColumnTypeInt, "not-a-number", true},
		{"invalid float", csv.ColumnTypeFloat, "not-a-float", true},
		{"invalid bool", csv.ColumnTypeBool, "maybe", true},
		{"invalid date", csv.ColumnTypeDate, "not-a-date", true},
		{"invalid time", csv.ColumnTypeTime, "not-a-time", true},
		{"invalid datetime", csv.ColumnTypeDateTime, "not-a-datetime", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a schema with a single column of the specified type
			schema := csv.NewSchema().AddColumn(csv.ColumnDefinition{
				Name:     "test_col",
				Type:     tt.colType,
				Required: true,
			})

			// Create data with the test value
			data := [][]string{
				{"test_col"},
				{tt.value},
			}

			result := csv.ValidateSchema(data, schema)
			if (result != nil && !result.Valid) != tt.wantErr {
				if tt.wantErr {
					t.Errorf("ValidateSchema() expected errors but got valid result")
				} else {
					t.Errorf("ValidateSchema() got errors: %v, wantErr %v", result.Errors, tt.wantErr)
				}
			}
		})
	}
}

// TestGoTypeToColumnType tests type mapping from Go types to column types
func TestGoTypeToColumnType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected csv.ColumnType
	}{
		{"int", 42, csv.ColumnTypeInt},
		{"int8", int8(42), csv.ColumnTypeInt},
		{"int16", int16(42), csv.ColumnTypeInt},
		{"int32", int32(42), csv.ColumnTypeInt},
		{"int64", int64(42), csv.ColumnTypeInt},
		{"uint", uint(42), csv.ColumnTypeInt},
		{"uint8", uint8(42), csv.ColumnTypeInt},
		{"uint16", uint16(42), csv.ColumnTypeInt},
		{"uint32", uint32(42), csv.ColumnTypeInt},
		{"uint64", uint64(42), csv.ColumnTypeInt},
		{"float32", float32(3.14), csv.ColumnTypeFloat},
		{"float64", float64(3.14), csv.ColumnTypeFloat},
		{"bool", true, csv.ColumnTypeBool},
		{"string", "hello", csv.ColumnTypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a struct with a field of the specified type
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: "Field",
					Type: reflect.TypeOf(tt.value),
					Tag:  reflect.StructTag(`csv:"field"`),
				},
			})

			// Create schema from struct
			instance := reflect.New(structType).Interface()
			schema, err := csv.SchemaFromStruct(instance)
			if err != nil {
				t.Fatalf("SchemaFromStruct() error = %v", err)
			}

			if len(schema.Columns) != 1 {
				t.Fatalf("expected 1 column, got %d", len(schema.Columns))
			}

			if schema.Columns[0].Type != tt.expected {
				t.Errorf("got column type %v, want %v", schema.Columns[0].Type, tt.expected)
			}
		})
	}
}

// TestGoTypeToColumnType_TimeType tests time.Time mapping
func TestGoTypeToColumnType_TimeType(t *testing.T) {
	type Record struct {
		Timestamp time.Time `csv:"timestamp"`
	}

	schema, err := csv.SchemaFromStruct(Record{})
	if err != nil {
		t.Fatalf("SchemaFromStruct() error = %v", err)
	}

	if len(schema.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(schema.Columns))
	}

	if schema.Columns[0].Type != csv.ColumnTypeDateTime {
		t.Errorf("time.Time should map to ColumnTypeDateTime, got %v", schema.Columns[0].Type)
	}
}

// TestGoTypeToColumnType_UnsupportedType tests unsupported type mapping
func TestGoTypeToColumnType_UnsupportedType(t *testing.T) {
	type Record struct {
		Complex complex128 `csv:"complex"`
	}

	schema, err := csv.SchemaFromStruct(Record{})
	if err != nil {
		t.Fatalf("SchemaFromStruct() error = %v", err)
	}

	if len(schema.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(schema.Columns))
	}

	// Unsupported types should map to ColumnTypeAny
	if schema.Columns[0].Type != csv.ColumnTypeAny {
		t.Errorf("unsupported type should map to ColumnTypeAny, got %v", schema.Columns[0].Type)
	}
}
