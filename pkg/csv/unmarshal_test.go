package csv

import (
	"testing"
)

// TestUnmarshal tests the basic Unmarshal function with slice of structs
func TestUnmarshal(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		City string
	}

	tests := []struct {
		name    string
		input   string
		want    []Person
		wantErr bool
	}{
		{
			name:  "simple CSV with headers",
			input: "Name,Age,City\nAlice,30,NYC\nBob,25,LA\n",
			want: []Person{
				{Name: "Alice", Age: 30, City: "NYC"},
				{Name: "Bob", Age: 25, City: "LA"},
			},
		},
		{
			name:  "headers in different order",
			input: "City,Name,Age\nNYC,Alice,30\nLA,Bob,25\n",
			want: []Person{
				{Name: "Alice", Age: 30, City: "NYC"},
				{Name: "Bob", Age: 25, City: "LA"},
			},
		},
		{
			name:  "empty data",
			input: "Name,Age,City\n",
			want:  []Person{},
		},
		{
			name:  "single record",
			input: "Name,Age,City\nCharlie,35,SF\n",
			want: []Person{
				{Name: "Charlie", Age: 35, City: "SF"},
			},
		},
		{
			name:    "unclosed quote",
			input:   "Name,Age\nAlice,\"30\nBob,25", // Unclosed quote
			wantErr: true,
		},
		{
			name:  "missing fields filled with zero values",
			input: "Name,Age\nAlice,30\nBob,25\n",
			want: []Person{
				{Name: "Alice", Age: 30, City: ""},
				{Name: "Bob", Age: 25, City: ""},
			},
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
			if err != nil {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Unmarshal() got %d records, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unmarshal() record %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestUnmarshalWithTags tests Unmarshal with custom CSV struct tags
func TestUnmarshalWithTags(t *testing.T) {
	type Product struct {
		ID          int     `csv:"id"`
		ProductName string  `csv:"name"`
		Price       float64 `csv:"price"`
		InStock     bool    `csv:"in_stock"`
		Internal    string  `csv:"-"` // Should be ignored
	}

	tests := []struct {
		name    string
		input   string
		want    []Product
		wantErr bool
	}{
		{
			name:  "with csv tags",
			input: "id,name,price,in_stock\n1,Widget,9.99,true\n2,Gadget,19.99,false\n",
			want: []Product{
				{ID: 1, ProductName: "Widget", Price: 9.99, InStock: true},
				{ID: 2, ProductName: "Gadget", Price: 19.99, InStock: false},
			},
		},
		{
			name:  "case insensitive header matching",
			input: "ID,NAME,PRICE,IN_STOCK\n1,Widget,9.99,true\n",
			want: []Product{
				{ID: 1, ProductName: "Widget", Price: 9.99, InStock: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []Product
			err := Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Unmarshal() got %d records, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unmarshal() record %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestUnmarshalTypes tests Unmarshal with different field types
func TestUnmarshalTypes(t *testing.T) {
	type TypeTest struct {
		String  string  `csv:"str"`
		Int     int     `csv:"int"`
		Int64   int64   `csv:"int64"`
		Float32 float32 `csv:"float32"`
		Float64 float64 `csv:"float64"`
		Bool    bool    `csv:"bool"`
		Uint    uint    `csv:"uint"`
	}

	input := "str,int,int64,float32,float64,bool,uint\ntest,42,9223372036854775807,3.14,2.718281828,true,100\nanother,-1,-9223372036854775808,-1.5,0.0,false,0\n"

	var got []TypeTest
	err := Unmarshal([]byte(input), &got)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("Unmarshal() got %d records, want 2", len(got))
	}

	// Check first record
	if got[0].String != "test" {
		t.Errorf("got[0].String = %v, want test", got[0].String)
	}
	if got[0].Int != 42 {
		t.Errorf("got[0].Int = %v, want 42", got[0].Int)
	}
	if got[0].Int64 != 9223372036854775807 {
		t.Errorf("got[0].Int64 = %v, want 9223372036854775807", got[0].Int64)
	}
	if got[0].Bool != true {
		t.Errorf("got[0].Bool = %v, want true", got[0].Bool)
	}

	// Check second record
	if got[1].String != "another" {
		t.Errorf("got[1].String = %v, want another", got[1].String)
	}
	if got[1].Int != -1 {
		t.Errorf("got[1].Int = %v, want -1", got[1].Int)
	}
	if got[1].Bool != false {
		t.Errorf("got[1].Bool = %v, want false", got[1].Bool)
	}
}

// TestUnmarshalErrors tests Unmarshal error cases
func TestUnmarshalErrors(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	tests := []struct {
		name    string
		input   string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "nil target",
			input:   "Name,Age\nAlice,30\n",
			target:  nil,
			wantErr: true,
		},
		{
			name:    "non-pointer target",
			input:   "Name,Age\nAlice,30\n",
			target:  []Person{},
			wantErr: true,
		},
		{
			name:    "nil pointer",
			input:   "Name,Age\nAlice,30\n",
			target:  (*[]Person)(nil),
			wantErr: true,
		},
		{
			name:    "pointer to non-slice",
			input:   "Name,Age\nAlice,30\n",
			target:  new(Person),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.input), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUnmarshalQuotedFields tests Unmarshal with quoted fields containing special characters
func TestUnmarshalQuotedFields(t *testing.T) {
	type Record struct {
		Name        string `csv:"name"`
		Description string `csv:"description"`
	}

	tests := []struct {
		name    string
		input   string
		want    []Record
		wantErr bool
	}{
		{
			name:  "fields with commas",
			input: "name,description\nItem1,\"Has, comma\"\nItem2,Normal\n",
			want: []Record{
				{Name: "Item1", Description: "Has, comma"},
				{Name: "Item2", Description: "Normal"},
			},
		},
		{
			name:  "fields with quotes",
			input: "name,description\nItem1,\"Has \"\"quotes\"\"\"\nItem2,Normal\n",
			want: []Record{
				{Name: "Item1", Description: "Has \"quotes\""},
				{Name: "Item2", Description: "Normal"},
			},
		},
		{
			name:  "fields with newlines",
			input: "name,description\nItem1,\"Has\nnewline\"\nItem2,Normal\n",
			want: []Record{
				{Name: "Item1", Description: "Has\nnewline"},
				{Name: "Item2", Description: "Normal"},
			},
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
			if err != nil {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Unmarshal() got %d records, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unmarshal() record %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestUnmarshalRoundTrip tests that Marshal and Unmarshal are inverse operations
func TestUnmarshalRoundTrip(t *testing.T) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
		City string `csv:"city"`
	}

	original := []Person{
		{Name: "Alice", Age: 30, City: "NYC"},
		{Name: "Bob", Age: 25, City: "LA"},
		{Name: "Charlie", Age: 35, City: "SF"},
	}

	// Marshal to CSV
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal back
	var result []Person
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if len(result) != len(original) {
		t.Fatalf("Round trip: got %d records, want %d", len(result), len(original))
	}

	for i := range result {
		if result[i] != original[i] {
			t.Errorf("Round trip: record %d = %+v, want %+v", i, result[i], original[i])
		}
	}
}
