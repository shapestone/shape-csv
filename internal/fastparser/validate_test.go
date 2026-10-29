package fastparser

import (
	"encoding/csv"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// stdlibValid reports whether encoding/csv (strict quotes, field count fixed by
// the first record) accepts s.
func stdlibValid(s string) bool {
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = 0
	_, err := r.ReadAll()
	return err == nil
}

// TestValidate_DifferentialExhaustive compares Validate with encoding/csv on
// every string up to length 7 over an alphabet that exercises quoting, field
// separators and line breaks.
func TestValidate_DifferentialExhaustive(t *testing.T) {
	alphabet := []byte{'a', ',', '"', '\n', '\r'}
	const maxLen = 7

	mismatches := map[string]bool{}
	var walk func(prefix []byte)
	walk = func(prefix []byte) {
		s := string(prefix)
		got := Validate(prefix) == nil
		if want := stdlibValid(s); got != want {
			mismatches[s] = want
		}
		if len(prefix) == maxLen {
			return
		}
		for _, c := range alphabet {
			walk(append(prefix, c))
		}
	}
	walk(make([]byte, 0, maxLen))

	// The only documented difference: a bare "\r" in the middle of a record is a
	// line break for this parser but ordinary data for encoding/csv.
	for s, want := range mismatches {
		if !hasBareCR(s) {
			t.Errorf("Validate(%q) disagrees with encoding/csv (stdlib valid=%v)", s, want)
		}
	}
}

// hasBareCR reports whether s has a "\r" that is not followed by "\n".
func hasBareCR(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && (i+1 == len(s) || s[i+1] != '\n') {
			return true
		}
	}
	return false
}

func TestValidate_FieldCountLine(t *testing.T) {
	tests := []struct {
		name            string
		in              string
		line, got, want int
	}{
		{"extra field on line 2", "a,b\n1,2,3", 2, 3, 2},
		{"missing field on line 2", "a,b\n1", 2, 1, 2},
		{"blank lines are counted", "a,b\n\n\n1", 4, 1, 2},
		{"after multi-line quoted field", "a,b\n\"x\ny\",2\n1,2,3", 4, 3, 2},
		{"CRLF", "a,b\r\n1,2\r\n1\r\n", 3, 1, 2},
		{"lone CR is a line break", "a,b\r1,2\r1", 3, 1, 2},
		{"empty trailing field counts", "a,b\n1,2\n3,", 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate([]byte(tc.in))
			if tc.line == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var fc *FieldCountError
			if !errors.As(err, &fc) {
				t.Fatalf("want *FieldCountError, got %v", err)
			}
			if fc.Line != tc.line || fc.Got != tc.got || fc.Want != tc.want {
				t.Errorf("got line=%d got=%d want=%d, expected line=%d got=%d want=%d",
					fc.Line, fc.Got, fc.Want, tc.line, tc.got, tc.want)
			}
		})
	}
}

func TestValidate_QuoteErrorsCarryLine(t *testing.T) {
	err := Validate([]byte("a,b\n1,\"2"))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("want unclosed-quote error on line 2, got %v", err)
	}
}

func TestParseLine(t *testing.T) {
	ok := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{`"Smith, John",30,NYC`, []string{"Smith, John", "30", "NYC"}},
		{`"he said ""hi""",x`, []string{`he said "hi"`, "x"}},
		{"", []string{""}},
		{"a,", []string{"a", ""}},
		{"a,,c", []string{"a", "", "c"}},
		{`" x ",y`, []string{" x ", "y"}},
		{`""`, []string{""}},
		{"a,b\n", []string{"a", "b"}},
		{"a,b\r\n", []string{"a", "b"}},
		{"\"a\nb\",c", []string{"a\nb", "c"}},
		{"héllo,日本", []string{"héllo", "日本"}},
	}
	for _, tc := range ok {
		got, err := ParseLine(tc.in)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParseLine(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}

	bad := []string{`"abc`, `"a"b,c`, `a"b,c`, "a,b\nc,d", "a\n\n", "a\r\rb", `"a""`}
	for _, in := range bad {
		if got, err := ParseLine(in); err == nil {
			t.Errorf("ParseLine(%q) = %q, want error", in, got)
		}
	}
}

// TestParseLine_DifferentialExhaustive compares ParseLine with encoding/csv
// reading a single record, over all short strings.
func TestParseLine_DifferentialExhaustive(t *testing.T) {
	alphabet := []byte{'a', ',', '"', '\n'}
	const maxLen = 7

	var walk func(prefix []byte)
	walk = func(prefix []byte) {
		s := string(prefix)
		got, gerr := ParseLine(s)

		r := csv.NewReader(strings.NewReader(s))
		r.FieldsPerRecord = -1
		recs, serr := r.ReadAll()

		// encoding/csv skips empty lines and reads many records; ParseLine wants one.
		switch {
		case s == "" || s == "\n":
			// ParseLine: one empty field; encoding/csv: no records.
			if gerr != nil || !reflect.DeepEqual(got, []string{""}) {
				t.Errorf("ParseLine(%q) = %q, %v", s, got, gerr)
			}
		case serr == nil && len(recs) == 1 && (strings.HasPrefix(s, "\n") || strings.Contains(s, "\n\n")):
			// encoding/csv skips blank lines; ParseLine takes one line and
			// allows a single trailing terminator, so it rejects these.
			if gerr == nil && !strings.Contains(s, "\"") {
				t.Errorf("ParseLine(%q) accepted blank lines", s)
			}
		case serr == nil && len(recs) == 1:
			if gerr != nil || !reflect.DeepEqual(got, recs[0]) {
				t.Errorf("ParseLine(%q) = %q, %v; encoding/csv = %q", s, got, gerr, recs[0])
			}
		case serr != nil:
			if gerr == nil {
				t.Errorf("ParseLine(%q) = %q, want error (encoding/csv: %v)", s, got, serr)
			}
		default: // several records
			if gerr == nil {
				t.Errorf("ParseLine(%q) = %q, want error (%d records)", s, got, len(recs))
			}
		}

		if len(prefix) == maxLen {
			return
		}
		for _, c := range alphabet {
			walk(append(prefix, c))
		}
	}
	walk(make([]byte, 0, maxLen))
}
