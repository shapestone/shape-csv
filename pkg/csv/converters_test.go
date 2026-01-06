package csv_test

import (
	"testing"
	"time"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestIntConverter(t *testing.T) {
	conv := csv.IntConverter{}

	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"123", 123, false},
		{"-456", -456, false},
		{"0", 0, false},
		{"", 0, false},
		{"  42  ", 42, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := conv.Convert(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("IntConverter.Convert(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.(int64) != tt.want {
				t.Errorf("IntConverter.Convert(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIntConverterWithBase(t *testing.T) {
	conv := csv.IntConverter{Base: 16}
	got, err := conv.Convert("ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.(int64) != 255 {
		t.Errorf("IntConverter{Base:16}.Convert(\"ff\") = %v, want 255", got)
	}
}

func TestFloatConverter(t *testing.T) {
	conv := csv.FloatConverter{}

	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"3.14", 3.14, false},
		{"-2.5", -2.5, false},
		{"0", 0, false},
		{"", 0, false},
		{"  1.5  ", 1.5, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := conv.Convert(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FloatConverter.Convert(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.(float64) != tt.want {
				t.Errorf("FloatConverter.Convert(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBoolConverter(t *testing.T) {
	conv := csv.BoolConverter{}

	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"TRUE", true, false},
		{"FALSE", false, false},
		{"1", true, false},
		{"0", false, false},
		{"yes", true, false},
		{"no", false, false},
		{"y", true, false},
		{"n", false, false},
		{"on", true, false},
		{"off", false, false},
		{"t", true, false},
		{"f", false, false},
		{"", false, false},
		{"maybe", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := conv.Convert(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BoolConverter.Convert(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.(bool) != tt.want {
				t.Errorf("BoolConverter.Convert(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDateConverter(t *testing.T) {
	conv := csv.DateConverter{}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2024-01-15", false},
		{"", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := conv.Convert(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateConverter.Convert(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.input != "" {
				tm := got.(time.Time)
				if tm.Year() != 2024 || tm.Month() != 1 || tm.Day() != 15 {
					t.Errorf("DateConverter.Convert(%q) = %v, wrong date", tt.input, tm)
				}
			}
		})
	}
}

func TestDateConverterCustomFormat(t *testing.T) {
	conv := csv.DateConverter{Format: "01/02/2006"}
	got, err := conv.Convert("12/25/2024")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tm := got.(time.Time)
	if tm.Month() != 12 || tm.Day() != 25 || tm.Year() != 2024 {
		t.Errorf("DateConverter with custom format: got %v", tm)
	}
}

func TestTimeConverter(t *testing.T) {
	conv := csv.TimeConverter{}

	got, err := conv.Convert("14:30:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tm := got.(time.Time)
	if tm.Hour() != 14 || tm.Minute() != 30 {
		t.Errorf("TimeConverter.Convert(\"14:30:00\") = %v", tm)
	}
}

func TestDateTimeConverter(t *testing.T) {
	conv := csv.DateTimeConverter{}

	got, err := conv.Convert("2024-01-15 14:30:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tm := got.(time.Time)
	if tm.Year() != 2024 || tm.Month() != 1 || tm.Day() != 15 {
		t.Errorf("DateTimeConverter: wrong date: %v", tm)
	}
	if tm.Hour() != 14 || tm.Minute() != 30 {
		t.Errorf("DateTimeConverter: wrong time: %v", tm)
	}
}

func TestConverterRegistry(t *testing.T) {
	registry := csv.NewConverterRegistry()

	t.Run("built-in converters", func(t *testing.T) {
		names := []string{"int", "float", "bool", "date", "time", "datetime"}
		for _, name := range names {
			if _, ok := registry.Get(name); !ok {
				t.Errorf("built-in converter %q not found", name)
			}
		}
	})

	t.Run("custom converter", func(t *testing.T) {
		registry.Register("upper", csv.ConverterFunc(func(s string) (interface{}, error) {
			return s + "!", nil
		}))

		conv, ok := registry.Get("upper")
		if !ok {
			t.Fatal("custom converter not found")
		}

		got, err := conv.Convert("test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "test!" {
			t.Errorf("custom converter returned %v, want \"test!\"", got)
		}
	})

	t.Run("unknown converter", func(t *testing.T) {
		_, ok := registry.Get("unknown")
		if ok {
			t.Error("unknown converter should not be found")
		}
	})
}

func TestInferType(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
	}{
		{"true", "bool"},
		{"false", "bool"},
		{"123", "int"},
		{"-456", "int"},
		{"3.14", "float"},
		{"hello", "string"},
		{"", "string"},
		{"2024-01-15", "date"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotType, _ := csv.InferType(tt.input)
			if gotType != tt.wantType {
				t.Errorf("InferType(%q) type = %q, want %q", tt.input, gotType, tt.wantType)
			}
		})
	}
}

func TestIsNullValue(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", true},
		{"NULL", true},
		{"null", true},
		{"nil", true},
		{"N/A", true},
		{"n/a", true},
		{"NA", true},
		{"-", true},
		{"valid", false},
		{"123", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := csv.IsNullValue(tt.value, csv.DefaultNullValues)
			if got != tt.want {
				t.Errorf("IsNullValue(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestDefaultTypeInferenceOptions(t *testing.T) {
	opts := csv.DefaultTypeInferenceOptions()

	if opts.InferTypes {
		t.Error("InferTypes should be false by default")
	}
	if len(opts.NullValues) == 0 {
		t.Error("NullValues should have defaults")
	}
	if opts.Registry == nil {
		t.Error("Registry should not be nil")
	}
}
