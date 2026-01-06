package fastparser

import (
	"reflect"
	"testing"
)

// TestDFA_BasicParsing tests basic DFA parsing functionality
func TestDFA_BasicParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  [][]string{},
		},
		{
			name:  "single field",
			input: "a",
			want:  [][]string{{"a"}},
		},
		{
			name:  "simple record",
			input: "a,b,c",
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "two records",
			input: "a,b\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "two records with CRLF",
			input: "a,b\r\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "empty fields",
			input: "a,,c",
			want:  [][]string{{"a", "", "c"}},
		},
		{
			name:  "all empty fields",
			input: ",,",
			want:  [][]string{{"", "", ""}},
		},
		{
			name:  "quoted field with comma",
			input: `"hello,world"`,
			want:  [][]string{{"hello,world"}},
		},
		{
			name:  "quoted field with escaped quote",
			input: `"say ""hello"""`,
			want:  [][]string{{`say "hello"`}},
		},
		{
			name:  "quoted field with newline",
			input: "\"hello\nworld\"",
			want:  [][]string{{"hello\nworld"}},
		},
		{
			name:  "quoted field with CRLF",
			input: "\"hello\r\nworld\"",
			want:  [][]string{{"hello\r\nworld"}},
		},
		{
			name:  "mixed quoted and unquoted",
			input: `a,"b,c",d`,
			want:  [][]string{{"a", "b,c", "d"}},
		},
		{
			name:  "trailing newline",
			input: "a,b\n",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "trailing CRLF",
			input: "a,b\r\n",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "empty lines skipped",
			input: "a,b\n\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "quoted empty field",
			input: `""`,
			want:  [][]string{{""}},
		},
		{
			name:  "multiple quoted empty fields",
			input: `"","",""`,
			want:  [][]string{{"", "", ""}},
		},
		{
			name:    "unclosed quoted field",
			input:   `"hello`,
			wantErr: true,
		},
		{
			name:    "quote in middle of unquoted field",
			input:   `hel"lo`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDFA([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDFA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDFA() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDFA_RFC4180Examples tests RFC 4180 compliance
func TestDFA_RFC4180Examples(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name: "RFC 4180 Example 1",
			input: `aaa,bbb,ccc
zzz,yyy,xxx`,
			want: [][]string{
				{"aaa", "bbb", "ccc"},
				{"zzz", "yyy", "xxx"},
			},
		},
		{
			name:  "RFC 4180 Example 5 - quoted fields",
			input: `"aaa","bbb","ccc"` + "\n" + `zzz,yyy,xxx`,
			want: [][]string{
				{"aaa", "bbb", "ccc"},
				{"zzz", "yyy", "xxx"},
			},
		},
		{
			name:  "RFC 4180 Example 6 - embedded comma",
			input: `"aaa","b,bb","ccc"`,
			want:  [][]string{{"aaa", "b,bb", "ccc"}},
		},
		{
			name:  "RFC 4180 Example 7 - embedded newline",
			input: "\"aaa\",\"b\nbb\",\"ccc\"",
			want:  [][]string{{"aaa", "b\nbb", "ccc"}},
		},
		{
			name:  "RFC 4180 Example 8 - embedded double-quote",
			input: `"aaa","b""bb","ccc"`,
			want:  [][]string{{"aaa", `b"bb`, "ccc"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDFA([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseDFA() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDFA() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDFA_EdgeCases tests edge cases
func TestDFA_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
		wantErr bool
	}{
		{
			name:  "single comma",
			input: ",",
			want:  [][]string{{"", ""}},
		},
		{
			name:  "newline only",
			input: "\n",
			want:  [][]string{},
		},
		{
			name:  "CRLF only",
			input: "\r\n",
			want:  [][]string{},
		},
		{
			name:  "multiple empty lines",
			input: "\n\n\n",
			want:  [][]string{},
		},
		{
			name:  "quoted field spanning multiple lines",
			input: "\"line1\nline2\nline3\"",
			want:  [][]string{{"line1\nline2\nline3"}},
		},
		{
			name:  "consecutive escaped quotes",
			input: `""""`,
			want:  [][]string{{`"`}},
		},
		{
			name:  "four quotes (two escaped)",
			input: `""""""`,
			want:  [][]string{{`""`}},
		},
		{
			name:  "quoted field with only comma",
			input: `","`,
			want:  [][]string{{","}},
		},
		{
			name:  "CR only (treated as newline)",
			input: "a\rb",
			want:  [][]string{{"a"}, {"b"}},
		},
		{
			name:  "complex quoted fields",
			input: `"a,b","c""d""e","f` + "\n" + `g"`,
			want:  [][]string{{"a,b", `c"d"e`, "f\ng"}},
		},
		{
			name:  "spaces in unquoted fields",
			input: "hello world,foo bar",
			want:  [][]string{{"hello world", "foo bar"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDFA([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDFA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDFA() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDFA_EquivalenceWithOriginal ensures DFA produces same results as original parser
func TestDFA_EquivalenceWithOriginal(t *testing.T) {
	testCases := [][]byte{
		smallCSV,
		mediumCSV,
		largeCSV,
		quotedCSV,
		mixedCSV,
		[]byte("a,b,c\nd,e,f"),
		[]byte(`"a","b","c"`),
		[]byte(`"a,b","c""d""e"`),
		[]byte("a,b\r\nc,d"),
		[]byte(",,\n,,,"),
		[]byte("\n\na,b\n\n"),
		[]byte(`""""`),
	}

	for i, tc := range testCases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			original, err1 := Parse(tc)
			dfa, err2 := ParseDFA(tc)

			if (err1 != nil) != (err2 != nil) {
				t.Errorf("Error mismatch: Parse=%v, ParseDFA=%v", err1, err2)
				return
			}

			if err1 == nil && !reflect.DeepEqual(original, dfa) {
				t.Errorf("Results differ:\nOriginal: %v\nDFA:      %v", original, dfa)
			}
		})
	}
}

// TestCharClass tests character classification
func TestCharClass(t *testing.T) {
	tests := []struct {
		char byte
		want charClass
	}{
		{'"', classQuote},
		{',', classComma},
		{'\r', classCR},
		{'\n', classLF},
		{'a', classOther},
		{'0', classOther},
		{' ', classOther},
		{'\t', classOther},
		{255, classOther},
		{0, classOther},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			got := charClassTable[tt.char]
			if got != tt.want {
				t.Errorf("charClassTable[%q] = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

// TestDFATransitionTable validates the DFA transition table is complete
func TestDFATransitionTable(t *testing.T) {
	// Ensure all state/charClass combinations have a valid transition
	for s := dfaState(0); s < numStates; s++ {
		for c := charClass(0); c < numCharClasses; c++ {
			trans := dfaTransitions[s][c]
			// Validate nextState is within bounds
			if trans.nextState >= numStates && trans.nextState != stateError {
				t.Errorf("Invalid nextState for (%v, %v): %v", s, c, trans.nextState)
			}
			// Validate action is within bounds
			if trans.action >= numActions {
				t.Errorf("Invalid action for (%v, %v): %v", s, c, trans.action)
			}
		}
	}
}

// TestDFA_LargeInput tests DFA with large input
func TestDFA_LargeInput(t *testing.T) {
	// Test with a large CSV to ensure no stack overflow or other issues
	var data []byte
	for i := 0; i < 10000; i++ {
		data = append(data, []byte("a,b,c,d,e,f,g,h,i,j\n")...)
	}

	result, err := ParseDFA(data)
	if err != nil {
		t.Fatalf("ParseDFA() error = %v", err)
	}

	if len(result) != 10000 {
		t.Errorf("Expected 10000 records, got %d", len(result))
	}

	for i, record := range result {
		if len(record) != 10 {
			t.Errorf("Record %d: expected 10 fields, got %d", i, len(record))
		}
	}
}

// TestDFA_QuotedFieldsWithEscapes tests escaped quotes in quoted fields
func TestDFA_QuotedFieldsWithEscapes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "single escaped quote",
			input: `"a""b"`,
			want:  [][]string{{`a"b`}},
		},
		{
			name:  "multiple escaped quotes",
			input: `"a""b""c""d"`,
			want:  [][]string{{`a"b"c"d`}},
		},
		{
			name:  "escaped quote at start",
			input: `"""hello"`,
			want:  [][]string{{`"hello`}},
		},
		{
			name:  "escaped quote at end",
			input: `"hello"""`,
			want:  [][]string{{`hello"`}},
		},
		{
			name:  "only escaped quotes",
			input: `""""""""`,
			want:  [][]string{{`"""`}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDFA([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseDFA() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDFA() = %v, want %v", got, tt.want)
			}
		})
	}
}
