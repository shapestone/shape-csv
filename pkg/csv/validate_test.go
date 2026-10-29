package csv

import (
	stdcsv "encoding/csv"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// validateCases is the Validate verdict table. old is the verdict of shape
// v0.4.0 csv.Validate (recorded by running that module from the module cache);
// std is encoding/csv with FieldsPerRecord = 0.
var validateCases = []struct {
	name  string
	in    string
	want  bool
	old   bool
	notes string
}{
	{"simple", "a,b\n1,2", true, true, ""},
	{"CRLF", "a,b\r\n1,2\r\n", true, true, ""},
	{"blank lines skipped", "a,b\n\n1,2\n", true, true, ""},
	{"trailing empty field", "a,b\n1,", true, true, ""},
	{"quoted comma", "\"a,b\",c\n1,2", true, true, ""},
	{"single column", "a\n", true, true, ""},
	{"empty input", "", true, false, "encoding/csv accepts empty input; v0.4.0 rejected it"},

	// Column counts: checked again.
	{"too many fields", "a,b\n1,2,3", false, false, ""},
	{"too few fields", "a,b\n1", false, false, ""},

	// Quote strictness stays.
	{"unclosed quote", "a,\"b\n1,2", false, true, "v0.4.0 split lines first, so it missed this"},
	{"text after closing quote", "a,\"b\"c", false, true, "stricter than v0.4.0, matches encoding/csv"},
	{"bare quote in unquoted field", "a,b\"c", false, true, "stricter than v0.4.0, matches encoding/csv"},

	// Multi-line quoted fields are valid records; v0.4.0 split on every newline.
	{"quoted newline", "\"a\nb\",c\n1,2", true, false, "v0.4.0 rejected valid multi-line fields"},
	{"quoted newline in row 2", "a,b\n\"x\ny\",2", true, false, "v0.4.0 rejected valid multi-line fields"},
}

func stdValid(s string) bool {
	r := stdcsv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = 0
	_, err := r.ReadAll()
	return err == nil
}

func TestValidate_Table(t *testing.T) {
	for _, tc := range validateCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.in); (err == nil) != tc.want {
				t.Fatalf("Validate(%q) = %v, want valid=%v", tc.in, err, tc.want)
			}
			if err := ValidateReader(strings.NewReader(tc.in)); (err == nil) != tc.want {
				t.Fatalf("ValidateReader(%q) = %v, want valid=%v", tc.in, err, tc.want)
			}
			if got := stdValid(tc.in); got != tc.want {
				t.Errorf("encoding/csv says %v for %q, table says %v", got, tc.in, tc.want)
			}
			if tc.want != tc.old && tc.notes == "" {
				t.Errorf("verdict differs from v0.4.0 for %q without an explanation", tc.in)
			}
		})
	}
}

func TestValidate_FieldCountError(t *testing.T) {
	err := Validate("a,b\n1,2,3")
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %T: %v", err, err)
	}
	if !errors.Is(err, ErrFieldCount) {
		t.Errorf("error does not wrap ErrFieldCount: %v", err)
	}
	if pe.Line != 2 || pe.StartLine != 2 {
		t.Errorf("line = %d (start %d), want 2", pe.Line, pe.StartLine)
	}
	for _, want := range []string{"line 2", "got 3", "expected 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

// The tokenizer-based parser used to report "line 1" for every field-count error.
func TestParseWithOptions_FieldCountLine(t *testing.T) {
	opts := DefaultReaderOptions()
	opts.FieldsPerRecord = 0
	_, err := ParseWithOptions("a,b\n1,2\n1,2,3\n", opts)
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("want field-count error on line 3, got %v", err)
	}
	if err := ValidateWithOptions("a,b\n1,2\n1,2\n", opts); err != nil {
		t.Errorf("consistent records rejected: %v", err)
	}
}

func TestValidate_RaggedStillAllowedWithOptions(t *testing.T) {
	if err := ValidateWithOptions("a,b\n1", DefaultReaderOptions()); err != nil {
		t.Errorf("default options allow ragged records, got %v", err)
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
		err  bool
	}{
		{"plain", "a,b,c", []string{"a", "b", "c"}, false},
		{"quoted comma (v0.4.0 example)", `"Smith, John",30,NYC`, []string{"Smith, John", "30", "NYC"}, false},
		{"escaped quotes", `"he said ""hi""",x`, []string{`he said "hi"`, "x"}, false},
		{"empty line", "", []string{""}, false},
		{"trailing comma", "a,", []string{"a", ""}, false},
		{"empty middle", "a,,c", []string{"a", "", "c"}, false},
		{"spaces kept", `" x ",y`, []string{" x ", "y"}, false},
		{"trailing LF", "a,b\n", []string{"a", "b"}, false},
		{"embedded newline in quotes", "\"a\nb\",c", []string{"a\nb", "c"}, false},
		{"unclosed quote", `"abc`, nil, true},
		{"text after closing quote", `"a"b,c`, nil, true},
		{"bare quote", `a"b,c`, nil, true},
		{"two records", "a,b\nc,d", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLine(tc.in)
			if (err != nil) != tc.err {
				t.Fatalf("ParseLine(%q) error = %v, wantErr %v", tc.in, err, tc.err)
			}
			if !tc.err && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ParseLine must agree with encoding/csv on every well-formed single record.
func TestParseLine_MatchesEncodingCSV(t *testing.T) {
	for _, in := range []string{
		"a,b,c", `"x,y",z`, `"q""q",""`, ",,", `"",`, "héllo,日本,\"tab\there\"",
	} {
		r := stdcsv.NewReader(strings.NewReader(in))
		r.FieldsPerRecord = -1
		want, err := r.Read()
		if err != nil {
			t.Fatalf("encoding/csv rejects %q: %v", in, err)
		}
		got, err := ParseLine(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("ParseLine(%q) = %q, %v; encoding/csv = %q", in, got, err, want)
		}
	}
}

func TestValidate_QuoteErrorIsParseError(t *testing.T) {
	tests := []struct {
		in        string
		line, col int
	}{
		{"a,b\n1,\"2", 2, 4},
		{"a,\"b\"c", 1, 6},
		{"a,b\"c", 1, 4},
	}
	for _, tc := range tests {
		err := Validate(tc.in)
		var pe *ParseError
		if !errors.As(err, &pe) || !errors.Is(err, ErrQuote) {
			t.Errorf("Validate(%q) = %v, want *ParseError wrapping ErrQuote", tc.in, err)
			continue
		}
		if pe.Line != tc.line || pe.Column != tc.col {
			t.Errorf("Validate(%q): line %d col %d, want line %d col %d (%v)", tc.in, pe.Line, pe.Column, tc.line, tc.col, err)
		}
	}
}
