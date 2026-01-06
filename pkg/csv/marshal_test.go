package csv

import (
	"strings"
	"testing"
)

// TestMarshal tests the basic Marshal function with slice of structs
func TestMarshal(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		City string
	}

	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple slice of structs",
			input: []Person{
				{Name: "Alice", Age: 30, City: "NYC"},
				{Name: "Bob", Age: 25, City: "LA"},
			},
			want: "Age,City,Name\n30,NYC,Alice\n25,LA,Bob\n",
		},
		{
			name:  "empty slice",
			input: []Person{},
			want:  "",
		},
		{
			name: "single item",
			input: []Person{
				{Name: "Charlie", Age: 35, City: "SF"},
			},
			want: "Age,City,Name\n35,SF,Charlie\n",
		},
		{
			name: "fields with special characters",
			input: []struct {
				Name        string
				Description string
			}{
				{Name: "Item1", Description: "Has, comma"},
				{Name: "Item2", Description: "Has \"quotes\""},
				{Name: "Item3", Description: "Has\nnewline"},
			},
			want: "Description,Name\n\"Has, comma\",Item1\n\"Has \"\"quotes\"\"\",Item2\n\"Has\nnewline\",Item3\n",
		},
		{
			name:    "non-slice input",
			input:   Person{Name: "Alice", Age: 30, City: "NYC"},
			wantErr: true,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Marshal() mismatch:\ngot:\n%s\nwant:\n%s", gotStr, tt.want)
			}
		})
	}
}

// TestMarshalWithTags tests Marshal with custom CSV struct tags
func TestMarshalWithTags(t *testing.T) {
	type Product struct {
		ID          int     `csv:"id"`
		ProductName string  `csv:"name"`
		Price       float64 `csv:"price"`
		InStock     bool    `csv:"in_stock"`
		Internal    string  `csv:"-"` // Should be skipped
	}

	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name: "with csv tags",
			input: []Product{
				{ID: 1, ProductName: "Widget", Price: 9.99, InStock: true, Internal: "secret"},
				{ID: 2, ProductName: "Gadget", Price: 19.99, InStock: false, Internal: "hidden"},
			},
			want: "id,in_stock,name,price\n1,true,Widget,9.99\n2,false,Gadget,19.99\n",
		},
		{
			name: "mixed tags and field names",
			input: []struct {
				FirstName string `csv:"first_name"`
				LastName  string `csv:"last_name"`
				Email     string // No tag, should use field name
			}{
				{FirstName: "John", LastName: "Doe", Email: "john@example.com"},
			},
			want: "Email,first_name,last_name\njohn@example.com,John,Doe\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Marshal() mismatch:\ngot:\n%s\nwant:\n%s", gotStr, tt.want)
			}
		})
	}
}

// TestMarshalOmitEmpty tests Marshal with omitempty tag option
func TestMarshalOmitEmpty(t *testing.T) {
	type Record struct {
		Name     string  `csv:"name"`
		Optional string  `csv:"optional,omitempty"`
		Value    int     `csv:"value,omitempty"`
		Price    float64 `csv:"price,omitempty"`
	}

	tests := []struct {
		name    string
		input   []Record
		want    string
		wantErr bool
	}{
		{
			name: "with empty values",
			input: []Record{
				{Name: "A", Optional: "present", Value: 10, Price: 5.5},
				{Name: "B", Optional: "", Value: 0, Price: 0},
				{Name: "C", Optional: "also", Value: 20, Price: 0},
			},
			want: "name,optional,price,value\nA,present,5.5,10\nB,,,\nC,also,,20\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Marshal() mismatch:\ngot:\n%s\nwant:\n%s", gotStr, tt.want)
			}
		})
	}
}

// TestMarshalTypes tests Marshal with different field types
func TestMarshalTypes(t *testing.T) {
	type TypeTest struct {
		String  string  `csv:"str"`
		Int     int     `csv:"int"`
		Int64   int64   `csv:"int64"`
		Float32 float32 `csv:"float32"`
		Float64 float64 `csv:"float64"`
		Bool    bool    `csv:"bool"`
		Uint    uint    `csv:"uint"`
	}

	input := []TypeTest{
		{
			String:  "test",
			Int:     42,
			Int64:   9223372036854775807,
			Float32: 3.14,
			Float64: 2.718281828,
			Bool:    true,
			Uint:    100,
		},
		{
			String:  "another",
			Int:     -1,
			Int64:   -9223372036854775808,
			Float32: -1.5,
			Float64: 0.0,
			Bool:    false,
			Uint:    0,
		},
	}

	got, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	gotStr := string(got)
	// Check that all headers are present
	if !strings.Contains(gotStr, "str") || !strings.Contains(gotStr, "int") {
		t.Errorf("Marshal() missing expected headers in:\n%s", gotStr)
	}

	// Check that values are present
	if !strings.Contains(gotStr, "test") || !strings.Contains(gotStr, "42") {
		t.Errorf("Marshal() missing expected values in:\n%s", gotStr)
	}
}

// TestMarshalPointers tests Marshal with pointer fields
func TestMarshalPointers(t *testing.T) {
	type Record struct {
		Name  string  `csv:"name"`
		Value *int    `csv:"value"`
		Price *float64 `csv:"price"`
	}

	val := 42
	price := 9.99

	tests := []struct {
		name    string
		input   []Record
		want    string
		wantErr bool
	}{
		{
			name: "with pointer fields",
			input: []Record{
				{Name: "A", Value: &val, Price: &price},
				{Name: "B", Value: nil, Price: nil},
			},
			want: "name,price,value\nA,9.99,42\nB,,\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Marshal() mismatch:\ngot:\n%s\nwant:\n%s", gotStr, tt.want)
			}
		})
	}
}

// TestMarshalOmitEmptyAdvanced tests omitempty with various field types
func TestMarshalOmitEmptyAdvanced(t *testing.T) {
	type Record struct {
		Name      string  `csv:"name"`
		OptString string  `csv:"opt_string,omitempty"`
		OptInt    int     `csv:"opt_int,omitempty"`
		OptBool   bool    `csv:"opt_bool,omitempty"`
		OptPtr    *int    `csv:"opt_ptr,omitempty"`
		OptFloat  float64 `csv:"opt_float,omitempty"`
	}

	val := 42
	tests := []struct {
		name    string
		input   []Record
		want    string
		wantErr bool
	}{
		{
			name: "with omitempty - empty values",
			input: []Record{
				{Name: "A", OptString: "", OptInt: 0, OptBool: false, OptPtr: nil, OptFloat: 0.0},
				{Name: "B", OptString: "test", OptInt: 42, OptBool: true, OptPtr: &val, OptFloat: 3.14},
			},
			// omitempty means field is still present but empty (maintains column structure)
			want: "name,opt_bool,opt_float,opt_int,opt_ptr,opt_string\nA,,,,,\nB,true,3.14,42,42,test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			// Check that columns are present (omitempty doesn't remove columns in CSV)
			if !strings.Contains(gotStr, "opt_string") || !strings.Contains(gotStr, "opt_int") {
				t.Errorf("Marshal() should include all columns even with omitempty:\n%s", gotStr)
			}
		})
	}
}
