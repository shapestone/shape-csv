package fastparser

import (
	"reflect"
	"testing"
)

// TestGetStructInfo tests basic struct info retrieval
func TestGetStructInfo(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	headers := []string{"name", "age"}
	structType := reflect.TypeOf(Person{})

	// First call should compute and cache
	info1 := getStructInfo(structType, headers)
	if info1 == nil {
		t.Fatal("getStructInfo() returned nil")
	}

	// Second call should return cached version
	info2 := getStructInfo(structType, headers)
	if info2 == nil {
		t.Fatal("getStructInfo() returned nil on second call")
	}

	// Should be the exact same instance (pointer equality)
	if info1 != info2 {
		t.Error("getStructInfo() did not return cached instance")
	}

	// Verify field map
	if len(info1.fieldMap) != 2 {
		t.Errorf("fieldMap has %d entries, want 2", len(info1.fieldMap))
	}

	// Check that column 0 maps to field index 0 (Name)
	if fieldIdx, ok := info1.fieldMap[0]; !ok || fieldIdx != 0 {
		t.Errorf("fieldMap[0] = %v, want 0", fieldIdx)
	}

	// Check that column 1 maps to field index 1 (Age)
	if fieldIdx, ok := info1.fieldMap[1]; !ok || fieldIdx != 1 {
		t.Errorf("fieldMap[1] = %v, want 1", fieldIdx)
	}

	// Verify setters
	if len(info1.setters) != 2 {
		t.Errorf("setters has %d entries, want 2", len(info1.setters))
	}
}

// TestStructInfoSetters tests that setters work correctly
func TestStructInfoSetters(t *testing.T) {
	type Record struct {
		Name   string  `csv:"name"`
		Age    int     `csv:"age"`
		Score  float64 `csv:"score"`
		Active bool    `csv:"active"`
	}

	headers := []string{"name", "age", "score", "active"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	// Create a record to test setters on
	val := reflect.New(structType).Elem()

	// Test string setter (column 0 -> Name field)
	if setter, ok := info.setters[0]; ok {
		err := setter(val.Field(0), "John", 0, 0)
		if err != nil {
			t.Errorf("setter for string failed: %v", err)
		}
		if val.Field(0).String() != "John" {
			t.Errorf("string setter set value to %q, want %q", val.Field(0).String(), "John")
		}
	} else {
		t.Error("no setter for column 0")
	}

	// Test int setter (column 1 -> Age field)
	if setter, ok := info.setters[1]; ok {
		err := setter(val.Field(1), "42", 0, 1)
		if err != nil {
			t.Errorf("setter for int failed: %v", err)
		}
		if val.Field(1).Int() != 42 {
			t.Errorf("int setter set value to %d, want 42", val.Field(1).Int())
		}
	} else {
		t.Error("no setter for column 1")
	}

	// Test float setter (column 2 -> Score field)
	if setter, ok := info.setters[2]; ok {
		err := setter(val.Field(2), "95.5", 0, 2)
		if err != nil {
			t.Errorf("setter for float failed: %v", err)
		}
		if val.Field(2).Float() != 95.5 {
			t.Errorf("float setter set value to %f, want 95.5", val.Field(2).Float())
		}
	} else {
		t.Error("no setter for column 2")
	}

	// Test bool setter (column 3 -> Active field)
	if setter, ok := info.setters[3]; ok {
		err := setter(val.Field(3), "true", 0, 3)
		if err != nil {
			t.Errorf("setter for bool failed: %v", err)
		}
		if !val.Field(3).Bool() {
			t.Error("bool setter set value to false, want true")
		}
	} else {
		t.Error("no setter for column 3")
	}
}

// TestStructInfoCaseInsensitive tests case-insensitive header matching
func TestStructInfoCaseInsensitive(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	// Headers in different case
	headers := []string{"NAME", "AGE"}
	structType := reflect.TypeOf(Person{})

	info := getStructInfo(structType, headers)

	// Should still map correctly despite case difference
	if len(info.fieldMap) != 2 {
		t.Errorf("fieldMap has %d entries, want 2", len(info.fieldMap))
	}

	if fieldIdx, ok := info.fieldMap[0]; !ok || fieldIdx != 0 {
		t.Errorf("fieldMap[0] = %v, want 0 (case-insensitive match failed)", fieldIdx)
	}
}

// TestStructInfoUnexportedFields tests that unexported fields are skipped
func TestStructInfoUnexportedFields(t *testing.T) {
	type Record struct {
		Name       string `csv:"name"`
		unexported int    `csv:"age"` //nolint:unused // This unexported field is intentionally here to test that it's ignored
	}

	headers := []string{"name", "age"}
	structType := reflect.TypeOf(Record{})

	info := getStructInfo(structType, headers)

	// Only exported field should be mapped
	if len(info.fieldMap) != 1 {
		t.Errorf("fieldMap has %d entries, want 1 (unexported should be skipped)", len(info.fieldMap))
	}

	// Column 0 (name) should map
	if _, ok := info.fieldMap[0]; !ok {
		t.Error("fieldMap[0] should exist for exported field")
	}

	// Column 1 (age) should NOT map to unexported field
	if fieldIdx, ok := info.fieldMap[1]; ok {
		t.Errorf("fieldMap[1] = %d, want no mapping for unexported field", fieldIdx)
	}
}

// TestStructInfoMissingColumns tests handling of missing columns
func TestStructInfoMissingColumns(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
		City string `csv:"city"`
	}

	// Only name and age in headers, city is missing
	headers := []string{"name", "age"}
	structType := reflect.TypeOf(Person{})

	info := getStructInfo(structType, headers)

	// Should only map the two present columns
	if len(info.fieldMap) != 2 {
		t.Errorf("fieldMap has %d entries, want 2", len(info.fieldMap))
	}

	// City field should not be in the map
	for colIdx, fieldIdx := range info.fieldMap {
		if fieldIdx == 2 { // Field index 2 is City
			t.Errorf("fieldMap[%d] maps to City field, should not be mapped", colIdx)
		}
	}
}

// TestStructInfoExtraColumns tests handling of extra columns not in struct
func TestStructInfoExtraColumns(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	// Extra column that doesn't match any struct field
	headers := []string{"name", "age", "city"}
	structType := reflect.TypeOf(Person{})

	info := getStructInfo(structType, headers)

	// Should only map name and age
	if len(info.fieldMap) != 2 {
		t.Errorf("fieldMap has %d entries, want 2", len(info.fieldMap))
	}

	// Column 2 (city) should not be mapped
	if _, ok := info.fieldMap[2]; ok {
		t.Error("fieldMap[2] should not exist (no matching struct field)")
	}
}

// TestStructInfoNoTags tests fallback to field names when no tags present
func TestStructInfoNoTags(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	headers := []string{"Name", "Age"}
	structType := reflect.TypeOf(Person{})

	info := getStructInfo(structType, headers)

	if len(info.fieldMap) != 2 {
		t.Errorf("fieldMap has %d entries, want 2", len(info.fieldMap))
	}

	if fieldIdx, ok := info.fieldMap[0]; !ok || fieldIdx != 0 {
		t.Errorf("fieldMap[0] = %v, want 0", fieldIdx)
	}

	if fieldIdx, ok := info.fieldMap[1]; !ok || fieldIdx != 1 {
		t.Errorf("fieldMap[1] = %v, want 1", fieldIdx)
	}
}

// TestStructInfoSetterErrors tests error handling in setters
func TestStructInfoSetterErrors(t *testing.T) {
	type Record struct {
		Age int `csv:"age"`
	}

	headers := []string{"age"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	// Test invalid int value
	if setter, ok := info.setters[0]; ok {
		err := setter(val.Field(0), "not-a-number", 5, 0)
		if err == nil {
			t.Error("setter should return error for invalid int value")
		}
	}
}

// TestStructInfoDifferentHeaders tests that cache keys include headers
func TestStructInfoDifferentHeaders(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	structType := reflect.TypeOf(Person{})

	// Get struct info with different header orderings
	headers1 := []string{"name", "age"}
	info1 := getStructInfo(structType, headers1)

	headers2 := []string{"age", "name"}
	info2 := getStructInfo(structType, headers2)

	// Should be different instances because headers differ
	if info1 == info2 {
		t.Error("getStructInfo() returned same instance for different header orders")
	}

	// Verify the field maps are different
	if info1.fieldMap[0] == info2.fieldMap[0] {
		t.Error("fieldMap[0] should differ for different header orders")
	}
}

// TestStructInfoIntegerTypes tests all integer type setters
func TestStructInfoIntegerTypes(t *testing.T) {
	type Record struct {
		I8  int8   `csv:"i8"`
		I16 int16  `csv:"i16"`
		I32 int32  `csv:"i32"`
		I64 int64  `csv:"i64"`
		U8  uint8  `csv:"u8"`
		U16 uint16 `csv:"u16"`
		U32 uint32 `csv:"u32"`
		U64 uint64 `csv:"u64"`
	}

	headers := []string{"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	tests := []struct {
		colIdx int
		value  string
		check  func(reflect.Value) bool
	}{
		{0, "127", func(v reflect.Value) bool { return v.Int() == 127 }},
		{1, "32767", func(v reflect.Value) bool { return v.Int() == 32767 }},
		{2, "2147483647", func(v reflect.Value) bool { return v.Int() == 2147483647 }},
		{3, "9223372036854775807", func(v reflect.Value) bool { return v.Int() == 9223372036854775807 }},
		{4, "255", func(v reflect.Value) bool { return v.Uint() == 255 }},
		{5, "65535", func(v reflect.Value) bool { return v.Uint() == 65535 }},
		{6, "4294967295", func(v reflect.Value) bool { return v.Uint() == 4294967295 }},
		{7, "18446744073709551615", func(v reflect.Value) bool { return v.Uint() == 18446744073709551615 }},
	}

	for _, tt := range tests {
		setter := info.setters[tt.colIdx]
		fieldIdx := info.fieldMap[tt.colIdx]
		field := val.Field(fieldIdx)

		err := setter(field, tt.value, 0, tt.colIdx)
		if err != nil {
			t.Errorf("setter for column %d failed: %v", tt.colIdx, err)
			continue
		}

		if !tt.check(field) {
			t.Errorf("setter for column %d set incorrect value", tt.colIdx)
		}
	}
}

// TestStructInfoEmptyValues tests handling of empty string values
func TestStructInfoEmptyValues(t *testing.T) {
	type Record struct {
		Age    int     `csv:"age"`
		Score  float64 `csv:"score"`
		Active bool    `csv:"active"`
	}

	headers := []string{"age", "score", "active"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	// Test empty int -> 0
	setter := info.setters[0]
	fieldIdx := info.fieldMap[0]
	err := setter(val.Field(fieldIdx), "", 0, 0)
	if err != nil {
		t.Errorf("setter for empty int failed: %v", err)
	}
	if val.Field(fieldIdx).Int() != 0 {
		t.Errorf("empty int should be 0, got %d", val.Field(fieldIdx).Int())
	}

	// Test empty float -> 0.0
	setter = info.setters[1]
	fieldIdx = info.fieldMap[1]
	err = setter(val.Field(fieldIdx), "", 0, 1)
	if err != nil {
		t.Errorf("setter for empty float failed: %v", err)
	}
	if val.Field(fieldIdx).Float() != 0.0 {
		t.Errorf("empty float should be 0.0, got %f", val.Field(fieldIdx).Float())
	}

	// Test empty bool -> false
	setter = info.setters[2]
	fieldIdx = info.fieldMap[2]
	err = setter(val.Field(fieldIdx), "", 0, 2)
	if err != nil {
		t.Errorf("setter for empty bool failed: %v", err)
	}
	if val.Field(fieldIdx).Bool() != false {
		t.Error("empty bool should be false")
	}
}

// TestClearStructCache tests cache invalidation
func TestClearStructCache(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	headers := []string{"name", "age"}
	structType := reflect.TypeOf(Person{})

	// Get cached info
	info1 := getStructInfo(structType, headers)
	if info1 == nil {
		t.Fatal("getStructInfo() returned nil")
	}

	// Clear cache
	clearStructCache()

	// Get info again - should be a new instance
	info2 := getStructInfo(structType, headers)
	if info2 == nil {
		t.Fatal("getStructInfo() returned nil after cache clear")
	}

	// Should be different instances (new computation)
	if info1 == info2 {
		t.Error("getStructInfo() returned same instance after cache clear")
	}

	// But should have same content
	if !reflect.DeepEqual(info1.fieldMap, info2.fieldMap) {
		t.Error("fieldMap differs after cache clear")
	}
}

// TestStructInfoOverflowInt tests integer overflow detection
func TestStructInfoOverflowInt(t *testing.T) {
	type Record struct {
		I8  int8  `csv:"i8"`
		I16 int16 `csv:"i16"`
		I32 int32 `csv:"i32"`
	}

	headers := []string{"i8", "i16", "i32"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	tests := []struct {
		name   string
		colIdx int
		value  string
		errMsg string
	}{
		{"int8 overflow positive", 0, "128", "overflows"},
		{"int8 overflow negative", 0, "-129", "overflows"},
		{"int16 overflow positive", 1, "32768", "overflows"},
		{"int16 overflow negative", 1, "-32769", "overflows"},
		{"int32 overflow positive", 2, "2147483648", "overflows"},
		{"int32 overflow negative", 2, "-2147483649", "overflows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setter := info.setters[tt.colIdx]
			fieldIdx := info.fieldMap[tt.colIdx]
			field := val.Field(fieldIdx)

			err := setter(field, tt.value, 0, tt.colIdx)
			if err == nil {
				t.Errorf("setter should return overflow error for %s", tt.value)
			}
			if err != nil && !containsStr(err.Error(), tt.errMsg) {
				t.Errorf("error should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestStructInfoOverflowUint tests unsigned integer overflow detection
func TestStructInfoOverflowUint(t *testing.T) {
	type Record struct {
		U8  uint8  `csv:"u8"`
		U16 uint16 `csv:"u16"`
		U32 uint32 `csv:"u32"`
	}

	headers := []string{"u8", "u16", "u32"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	tests := []struct {
		name   string
		colIdx int
		value  string
		errMsg string
	}{
		{"uint8 overflow", 0, "256", "overflows"},
		{"uint16 overflow", 1, "65536", "overflows"},
		{"uint32 overflow", 2, "4294967296", "overflows"},
		{"uint negative", 0, "-1", "invalid syntax"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setter := info.setters[tt.colIdx]
			fieldIdx := info.fieldMap[tt.colIdx]
			field := val.Field(fieldIdx)

			err := setter(field, tt.value, 0, tt.colIdx)
			if err == nil {
				t.Errorf("setter should return error for %s", tt.value)
			}
			if err != nil && !containsStr(err.Error(), tt.errMsg) {
				t.Errorf("error should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestStructInfoOverflowFloat tests float overflow detection
func TestStructInfoOverflowFloat(t *testing.T) {
	type Record struct {
		F32 float32 `csv:"f32"`
	}

	headers := []string{"f32"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	// Test float32 overflow with extremely large value
	setter := info.setters[0]
	fieldIdx := info.fieldMap[0]
	field := val.Field(fieldIdx)

	// This value is too large for float32
	err := setter(field, "3.5e+38", 0, 0)
	if err == nil {
		t.Error("setter should return overflow error for float32 overflow")
	}
	if err != nil && !containsStr(err.Error(), "overflows") {
		t.Errorf("error should contain 'overflows', got: %v", err)
	}
}

// TestStructInfoInvalidValues tests invalid value parsing
func TestStructInfoInvalidValues(t *testing.T) {
	type Record struct {
		I   int     `csv:"i"`
		U   uint    `csv:"u"`
		F   float64 `csv:"f"`
		B   bool    `csv:"b"`
	}

	headers := []string{"i", "u", "f", "b"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	tests := []struct {
		name   string
		colIdx int
		value  string
		errMsg string
	}{
		{"invalid int", 0, "not-a-number", "cannot parse"},
		{"invalid uint", 1, "not-a-number", "cannot parse"},
		{"invalid float", 2, "not-a-number", "cannot parse"},
		{"invalid bool", 3, "maybe", "cannot parse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setter := info.setters[tt.colIdx]
			fieldIdx := info.fieldMap[tt.colIdx]
			field := val.Field(fieldIdx)

			err := setter(field, tt.value, 0, tt.colIdx)
			if err == nil {
				t.Errorf("setter should return error for invalid value: %s", tt.value)
			}
			if err != nil && !containsStr(err.Error(), tt.errMsg) {
				t.Errorf("error should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestStructInfoUnsupportedType tests unsupported field types
func TestStructInfoUnsupportedType(t *testing.T) {
	type Record struct {
		Complex complex128 `csv:"complex"`
	}

	headers := []string{"complex"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	// Get the setter for the complex field
	setter := info.setters[0]
	fieldIdx := info.fieldMap[0]
	field := val.Field(fieldIdx)

	err := setter(field, "1+2i", 0, 0)
	if err == nil {
		t.Error("setter should return error for unsupported type")
	}
	if err != nil && !containsStr(err.Error(), "unsupported") {
		t.Errorf("error should contain 'unsupported', got: %v", err)
	}
}

// TestStructInfoEmptyUint tests empty string to uint
func TestStructInfoEmptyUint(t *testing.T) {
	type Record struct {
		U uint `csv:"u"`
	}

	headers := []string{"u"}
	structType := reflect.TypeOf(Record{})
	info := getStructInfo(structType, headers)

	val := reflect.New(structType).Elem()

	setter := info.setters[0]
	fieldIdx := info.fieldMap[0]
	field := val.Field(fieldIdx)

	err := setter(field, "", 0, 0)
	if err != nil {
		t.Errorf("setter should handle empty string for uint: %v", err)
	}
	if field.Uint() != 0 {
		t.Errorf("empty string should set uint to 0, got %d", field.Uint())
	}
}

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
