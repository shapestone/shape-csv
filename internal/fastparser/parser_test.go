package fastparser

import (
	"reflect"
	"testing"
)

func TestFastParser_BasicParsing(t *testing.T) {
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
			name:  "multiple records with various field counts",
			input: "a\na,b\na,b,c",
			want:  [][]string{{"a"}, {"a", "b"}, {"a", "b", "c"}},
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
			name:  "complex quoted fields",
			input: `"a,b","c""d""e","f` + "\n" + `g"`,
			want:  [][]string{{"a,b", `c"d"e`, "f\ng"}},
		},
		{
			name:  "spaces in unquoted fields",
			input: "hello world,foo bar",
			want:  [][]string{{"hello world", "foo bar"}},
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
		{
			name:    "single quote at start of field after content",
			input:   `hello"world`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFastParser_EdgeCases(t *testing.T) {
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
			name:  "record followed by multiple newlines",
			input: "a,b\n\n\n",
			want:  [][]string{{"a", "b"}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFastParser_RFC4180Examples(t *testing.T) {
	// Examples from RFC 4180
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
			name:  "RFC 4180 Example 2 - with header",
			input: `field1,field2,field3` + "\n" + `aaa,bbb,ccc` + "\n" + `zzz,yyy,xxx`,
			want: [][]string{
				{"field1", "field2", "field3"},
				{"aaa", "bbb", "ccc"},
				{"zzz", "yyy", "xxx"},
			},
		},
		{
			name: "RFC 4180 Example 5 - quoted fields",
			input: `"aaa","bbb","ccc"
zzz,yyy,xxx`,
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
			got, err := Parse([]byte(tt.input))
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
