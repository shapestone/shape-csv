package csv_test

import (
	"testing"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestSnifferDetectDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		sample   string
		expected rune
	}{
		{
			name:     "comma delimited",
			sample:   "a,b,c\n1,2,3\n4,5,6",
			expected: ',',
		},
		{
			name:     "tab delimited",
			sample:   "a\tb\tc\n1\t2\t3\n4\t5\t6",
			expected: '\t',
		},
		{
			name:     "semicolon delimited",
			sample:   "a;b;c\n1;2;3\n4;5;6",
			expected: ';',
		},
		{
			name:     "pipe delimited",
			sample:   "a|b|c\n1|2|3\n4|5|6",
			expected: '|',
		},
		{
			name:     "empty sample defaults to comma",
			sample:   "",
			expected: ',',
		},
		{
			name:     "single line comma",
			sample:   "a,b,c",
			expected: ',',
		},
		{
			name:     "mixed but more commas",
			sample:   "a,b,c\n1,2,3\n4;5;6",
			expected: ',',
		},
		{
			name:     "quoted commas ignored",
			sample:   "\"a,b\",c,d\n1,2,3",
			expected: ',',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sniffer := csv.NewSniffer(tt.sample)
			got := sniffer.DetectDelimiter()
			if got != tt.expected {
				t.Errorf("DetectDelimiter() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSnifferHasHeader(t *testing.T) {
	tests := []struct {
		name     string
		sample   string
		expected bool
	}{
		{
			name:     "clear header with identifiers",
			sample:   "name,age,email\nJohn,30,john@example.com",
			expected: true,
		},
		{
			name:     "numeric header looks like data",
			sample:   "123,456,789\n111,222,333",
			expected: false,
		},
		{
			name:     "snake_case header",
			sample:   "first_name,last_name,email_address\nJohn,Doe,john@example.com",
			expected: true,
		},
		{
			name:     "camelCase header",
			sample:   "firstName,lastName,emailAddress\nJohn,Doe,john@example.com",
			expected: true,
		},
		{
			name:     "single line",
			sample:   "a,b,c",
			expected: false,
		},
		{
			name:     "Title Case header",
			sample:   "First Name,Last Name,Email\nJohn,Doe,john@example.com",
			expected: true,
		},
		{
			name:     "data with dates",
			sample:   "2024-01-15,John,30\n2024-01-16,Jane,25",
			expected: false,
		},
		{
			name:     "mixed header and data indicators",
			sample:   "id,name,date\n1,John,2024-01-15",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sniffer := csv.NewSniffer(tt.sample)
			got := sniffer.HasHeader()
			if got != tt.expected {
				t.Errorf("HasHeader() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHeaderConverters(t *testing.T) {
	tests := []struct {
		name      string
		converter csv.HeaderConverter
		input     string
		expected  string
	}{
		{
			name:      "lowercase simple",
			converter: csv.LowercaseHeader,
			input:     "FirstName",
			expected:  "firstname",
		},
		{
			name:      "uppercase simple",
			converter: csv.UppercaseHeader,
			input:     "firstName",
			expected:  "FIRSTNAME",
		},
		{
			name:      "snake_case from camelCase",
			converter: csv.SnakeCaseHeader,
			input:     "firstName",
			expected:  "first_name",
		},
		{
			name:      "snake_case from PascalCase",
			converter: csv.SnakeCaseHeader,
			input:     "FirstName",
			expected:  "first_name",
		},
		{
			name:      "snake_case with spaces",
			converter: csv.SnakeCaseHeader,
			input:     "First Name",
			expected:  "first_name",
		},
		{
			name:      "snake_case already snake",
			converter: csv.SnakeCaseHeader,
			input:     "first_name",
			expected:  "first_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.converter(tt.input)
			if got != tt.expected {
				t.Errorf("converter(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestColumnSelector(t *testing.T) {
	t.Run("empty selector includes all", func(t *testing.T) {
		selector := csv.ColumnSelector{}
		if !selector.ShouldInclude("any", 0) {
			t.Error("empty selector should include all columns")
		}
		if !selector.ShouldInclude("other", 5) {
			t.Error("empty selector should include all columns")
		}
	})

	t.Run("select by name", func(t *testing.T) {
		selector := csv.ColumnSelector{
			UseCols: []string{"name", "email"},
		}
		if !selector.ShouldInclude("name", 0) {
			t.Error("should include 'name'")
		}
		if !selector.ShouldInclude("email", 2) {
			t.Error("should include 'email'")
		}
		if selector.ShouldInclude("age", 1) {
			t.Error("should not include 'age'")
		}
	})

	t.Run("select by index", func(t *testing.T) {
		selector := csv.ColumnSelector{
			UseColIndexes: []int{0, 2},
		}
		if !selector.ShouldInclude("any", 0) {
			t.Error("should include index 0")
		}
		if !selector.ShouldInclude("other", 2) {
			t.Error("should include index 2")
		}
		if selector.ShouldInclude("middle", 1) {
			t.Error("should not include index 1")
		}
	})

	t.Run("select by name or index", func(t *testing.T) {
		selector := csv.ColumnSelector{
			UseCols:       []string{"name"},
			UseColIndexes: []int{2},
		}
		if !selector.ShouldInclude("name", 0) {
			t.Error("should include 'name' by name")
		}
		if !selector.ShouldInclude("other", 2) {
			t.Error("should include index 2")
		}
		if selector.ShouldInclude("excluded", 5) {
			t.Error("should not include column not in name or index")
		}
	})
}

func TestSnifferAnalyzeCaching(t *testing.T) {
	sample := "a,b,c\n1,2,3"
	sniffer := csv.NewSniffer(sample)

	// First call should analyze
	delim1 := sniffer.DetectDelimiter()
	header1 := sniffer.HasHeader()

	// Second calls should return cached results
	delim2 := sniffer.DetectDelimiter()
	header2 := sniffer.HasHeader()

	if delim1 != delim2 {
		t.Error("delimiter results should be consistent")
	}
	if header1 != header2 {
		t.Error("header results should be consistent")
	}
}
