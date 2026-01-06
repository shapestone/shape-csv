package fastparser

import (
	"reflect"
	"testing"
)

func TestFastUnmarshal(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	tests := []struct {
		name    string
		input   string
		want    []Person
		wantErr bool
	}{
		{
			name:  "simple unmarshal",
			input: "name,age\nAlice,30\nBob,25",
			want: []Person{
				{Name: "Alice", Age: 30},
				{Name: "Bob", Age: 25},
			},
		},
		{
			name:  "with trailing newline",
			input: "name,age\nAlice,30\nBob,25\n",
			want: []Person{
				{Name: "Alice", Age: 30},
				{Name: "Bob", Age: 25},
			},
		},
		{
			name:  "single record",
			input: "name,age\nAlice,30",
			want: []Person{
				{Name: "Alice", Age: 30},
			},
		},
		{
			name:  "empty data rows",
			input: "name,age",
			want:  []Person{},
		},
		{
			name:  "quoted fields",
			input: `name,age` + "\n" + `"Alice Smith",30`,
			want: []Person{
				{Name: "Alice Smith", Age: 30},
			},
		},
		{
			name:    "type mismatch - string to int",
			input:   "name,age\nAlice,abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Person
			err := Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFastUnmarshal_MultipleTypes(t *testing.T) {
	type Record struct {
		Name    string  `csv:"name"`
		Age     int     `csv:"age"`
		Score   float64 `csv:"score"`
		Active  bool    `csv:"active"`
		Comment string  `csv:"comment"`
	}

	tests := []struct {
		name    string
		input   string
		want    []Record
		wantErr bool
	}{
		{
			name:  "all types",
			input: "name,age,score,active,comment\nAlice,30,95.5,true,Good\nBob,25,87.3,false,Fair",
			want: []Record{
				{Name: "Alice", Age: 30, Score: 95.5, Active: true, Comment: "Good"},
				{Name: "Bob", Age: 25, Score: 87.3, Active: false, Comment: "Fair"},
			},
		},
		{
			name:  "with empty fields",
			input: "name,age,score,active,comment\nAlice,30,95.5,true,\nBob,25,87.3,false,Fair",
			want: []Record{
				{Name: "Alice", Age: 30, Score: 95.5, Active: true, Comment: ""},
				{Name: "Bob", Age: 25, Score: 87.3, Active: false, Comment: "Fair"},
			},
		},
		{
			name:    "invalid int",
			input:   "name,age,score,active,comment\nAlice,abc,95.5,true,Good",
			wantErr: true,
		},
		{
			name:    "invalid float",
			input:   "name,age,score,active,comment\nAlice,30,xyz,true,Good",
			wantErr: true,
		},
		{
			name:    "invalid bool",
			input:   "name,age,score,active,comment\nAlice,30,95.5,maybe,Good",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Record
			err := Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFastUnmarshal_FieldMapping(t *testing.T) {
	type Person struct {
		FirstName string `csv:"first_name"`
		LastName  string `csv:"last_name"`
		Age       int    `csv:"age"`
	}

	input := "first_name,last_name,age\nAlice,Smith,30\nBob,Jones,25"

	var got []Person
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Person{
		{FirstName: "Alice", LastName: "Smith", Age: 30},
		{FirstName: "Bob", LastName: "Jones", Age: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_ExtraColumns(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	// Input has extra column that should be ignored
	input := "name,age,city\nAlice,30,NYC\nBob,25,LA"

	var got []Person
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_MissingColumns(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
		City string `csv:"city"`
	}

	// Input is missing city column - should use zero value
	input := "name,age\nAlice,30\nBob,25"

	var got []Person
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30, City: ""},
		{Name: "Bob", Age: 25, City: ""},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_UnevenRows(t *testing.T) {
	type Record struct {
		A string `csv:"a"`
		B string `csv:"b"`
		C string `csv:"c"`
	}

	// Some rows have fewer fields
	input := "a,b,c\n1,2,3\n4,5\n7,8,9"

	var got []Record
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Record{
		{A: "1", B: "2", C: "3"},
		{A: "4", B: "5", C: ""},
		{A: "7", B: "8", C: "9"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_IntegerTypes(t *testing.T) {
	type Record struct {
		I8  int8  `csv:"i8"`
		I16 int16 `csv:"i16"`
		I32 int32 `csv:"i32"`
		I64 int64 `csv:"i64"`
		U8  uint8 `csv:"u8"`
		U16 uint16 `csv:"u16"`
		U32 uint32 `csv:"u32"`
		U64 uint64 `csv:"u64"`
	}

	input := "i8,i16,i32,i64,u8,u16,u32,u64\n127,32767,2147483647,9223372036854775807,255,65535,4294967295,18446744073709551615"

	var got []Record
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Record{
		{
			I8:  127,
			I16: 32767,
			I32: 2147483647,
			I64: 9223372036854775807,
			U8:  255,
			U16: 65535,
			U32: 4294967295,
			U64: 18446744073709551615,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_FloatTypes(t *testing.T) {
	type Record struct {
		F32 float32 `csv:"f32"`
		F64 float64 `csv:"f64"`
	}

	input := "f32,f64\n3.14,2.718281828"

	var got []Record
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Unmarshal() returned %d records, want 1", len(got))
	}

	// Check float32
	if got[0].F32 < 3.13 || got[0].F32 > 3.15 {
		t.Errorf("F32 = %v, want ~3.14", got[0].F32)
	}

	// Check float64
	if got[0].F64 < 2.71 || got[0].F64 > 2.72 {
		t.Errorf("F64 = %v, want ~2.718281828", got[0].F64)
	}
}

func TestFastUnmarshal_BoolValues(t *testing.T) {
	type Record struct {
		Active bool `csv:"active"`
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"true lowercase", "active\ntrue", true},
		{"false lowercase", "active\nfalse", false},
		{"1 as true", "active\n1", true},
		{"0 as false", "active\n0", false},
		{"t as true", "active\nt", true},
		{"f as false", "active\nf", false},
		{"T as true", "active\nT", true},
		{"F as false", "active\nF", false},
		{"TRUE uppercase", "active\nTRUE", true},
		{"FALSE uppercase", "active\nFALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Record
			err := Unmarshal([]byte(tt.input), &got)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if len(got) != 1 {
				t.Fatalf("Unmarshal() returned %d records, want 1", len(got))
			}

			if got[0].Active != tt.want {
				t.Errorf("Active = %v, want %v", got[0].Active, tt.want)
			}
		})
	}
}

func TestFastUnmarshal_Errors(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	tests := []struct {
		name  string
		input string
		dest  interface{}
	}{
		{
			name:  "nil pointer",
			input: "name,age\nAlice,30",
			dest:  nil,
		},
		{
			name:  "non-pointer",
			input: "name,age\nAlice,30",
			dest:  []Person{},
		},
		{
			name:  "nil slice pointer",
			input: "name,age\nAlice,30",
			dest:  (*[]Person)(nil),
		},
		{
			name:  "pointer to non-slice",
			input: "name,age\nAlice,30",
			dest:  new(Person),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.input), tt.dest)
			if err == nil {
				t.Errorf("Unmarshal() expected error for %s", tt.name)
			}
		})
	}
}

func TestFastUnmarshal_EmptyInput(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only header", "name,age"},
		{"only newlines", "\n\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Person
			err := Unmarshal([]byte(tt.input), &got)
			if err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if len(got) != 0 {
				t.Errorf("Unmarshal() returned %d records, want 0", len(got))
			}
		})
	}
}

func TestFastUnmarshal_NoStructTags(t *testing.T) {
	// Test struct without csv tags - should use field names
	type Person struct {
		Name string
		Age  int
	}

	input := "Name,Age\nAlice,30\nBob,25"

	var got []Person
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_CaseInsensitiveHeaders(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	// Headers in different case
	input := "NAME,AGE\nAlice,30\nBob,25"

	var got []Person
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_UintTypes(t *testing.T) {
	type Record struct {
		U   uint   `csv:"u"`
		U8  uint8  `csv:"u8"`
		U16 uint16 `csv:"u16"`
		U32 uint32 `csv:"u32"`
		U64 uint64 `csv:"u64"`
	}

	input := "u,u8,u16,u32,u64\n42,255,65535,4294967295,18446744073709551615"

	var got []Record
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []Record{
		{
			U:   42,
			U8:  255,
			U16: 65535,
			U32: 4294967295,
			U64: 18446744073709551615,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %+v, want %+v", got, want)
	}
}

func TestFastUnmarshal_Float32(t *testing.T) {
	type Record struct {
		F32 float32 `csv:"f32"`
		F64 float64 `csv:"f64"`
	}

	input := "f32,f64\n3.14159,2.718281828459045"

	var got []Record
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(got))
	}

	// Check float32 (will have precision loss)
	if got[0].F32 < 3.14 || got[0].F32 > 3.15 {
		t.Errorf("F32 = %v, want ~3.14159", got[0].F32)
	}

	// Check float64
	if got[0].F64 < 2.71 || got[0].F64 > 2.72 {
		t.Errorf("F64 = %v, want ~2.718281828459045", got[0].F64)
	}
}

func TestFastUnmarshal_UnsupportedSliceElementType(t *testing.T) {
	// Test with slice of int (not struct or []string)
	var result []int
	input := "1,2,3\n4,5,6"

	err := Unmarshal([]byte(input), &result)
	if err == nil {
		t.Error("Unmarshal() should return error for slice of int")
	}
	if err != nil && !stringContains(err.Error(), "slice of") {
		t.Errorf("Error message should mention 'slice of', got: %v", err)
	}
}

func TestFastUnmarshal_ToStringSlice(t *testing.T) {
	// Test unmarshaling to [][]string
	var result [][]string
	input := "name,age\nAlice,30\nBob,25"

	err := Unmarshal([]byte(input), &result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := [][]string{
		{"name", "age"},
		{"Alice", "30"},
		{"Bob", "25"},
	}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("Unmarshal() = %v, want %v", result, want)
	}
}

func TestUnmarshalBytes_WithByteRecords(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	input := "name,age\nAlice,30\nBob,25"

	var got []Person
	err := UnmarshalBytes([]byte(input), &got)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnmarshalBytes() = %+v, want %+v", got, want)
	}
}

func TestUnmarshalBytes_ToStringSliceWithHeaders(t *testing.T) {
	// Test UnmarshalBytes to [][]string - verifying headers are included
	var result [][]string
	input := "name,age\nAlice,30\nBob,25"

	err := UnmarshalBytes([]byte(input), &result)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := [][]string{
		{"name", "age"},
		{"Alice", "30"},
		{"Bob", "25"},
	}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("UnmarshalBytes() = %v, want %v", result, want)
	}
}

func TestUnmarshalBytes_ErrorValidation(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	tests := []struct {
		name  string
		input string
		dest  interface{}
	}{
		{
			name:  "nil pointer",
			input: "name,age\nAlice,30",
			dest:  nil,
		},
		{
			name:  "non-pointer",
			input: "name,age\nAlice,30",
			dest:  []Person{},
		},
		{
			name:  "nil slice pointer",
			input: "name,age\nAlice,30",
			dest:  (*[]Person)(nil),
		},
		{
			name:  "pointer to non-slice",
			input: "name,age\nAlice,30",
			dest:  new(Person),
		},
		{
			name:  "unsupported slice element type",
			input: "1,2,3",
			dest:  new([]int),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalBytes([]byte(tt.input), tt.dest)
			if err == nil {
				t.Errorf("UnmarshalBytes() expected error for %s", tt.name)
			}
		})
	}
}

func TestUnmarshalBytes_EmptyInput(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only header", "name,age"},
		{"only newlines", "\n\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Person
			err := UnmarshalBytes([]byte(tt.input), &got)
			if err != nil {
				t.Errorf("UnmarshalBytes() error = %v", err)
				return
			}
			if len(got) != 0 {
				t.Errorf("UnmarshalBytes() returned %d records, want 0", len(got))
			}
		})
	}
}

func TestUnmarshalBytes_ExtraColumnsInData(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	// Row has more fields than headers
	input := "name,age\nAlice,30,ExtraField"

	var got []Person
	err := UnmarshalBytes([]byte(input), &got)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := []Person{
		{Name: "Alice", Age: 30},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnmarshalBytes() = %+v, want %+v", got, want)
	}
}

// Helper function to check if a string contains a substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
