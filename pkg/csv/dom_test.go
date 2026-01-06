package csv_test

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-csv/pkg/csv"
)

// TestNewDocument tests the NewDocument constructor (5.2.1)
func TestNewDocument(t *testing.T) {
	doc := csv.NewDocument()
	if doc == nil {
		t.Fatal("NewDocument() returned nil")
	}

	// Should start empty
	if doc.RecordCount() != 0 {
		t.Errorf("NewDocument().RecordCount() = %d, want 0", doc.RecordCount())
	}

	if len(doc.Headers()) != 0 {
		t.Errorf("NewDocument().Headers() = %v, want []", doc.Headers())
	}
}

// TestDocumentSetHeaders tests setting headers (5.2.2)
func TestDocumentSetHeaders(t *testing.T) {
	doc := csv.NewDocument()
	headers := []string{"name", "age", "email"}

	result := doc.SetHeaders(headers)

	// Should support method chaining
	if result != doc {
		t.Error("SetHeaders() should return the document for chaining")
	}

	gotHeaders := doc.Headers()
	if len(gotHeaders) != len(headers) {
		t.Errorf("Headers() = %v, want %v", gotHeaders, headers)
	}

	for i, h := range headers {
		if gotHeaders[i] != h {
			t.Errorf("Headers()[%d] = %s, want %s", i, gotHeaders[i], h)
		}
	}
}

// TestDocumentAddRecord tests adding records (5.2.2)
func TestDocumentAddRecord(t *testing.T) {
	doc := csv.NewDocument()
	fields1 := []string{"Alice", "30", "alice@example.com"}
	fields2 := []string{"Bob", "25", "bob@example.com"}

	// Add first record
	result := doc.AddRecord(fields1)

	// Should support method chaining
	if result != doc {
		t.Error("AddRecord() should return the document for chaining")
	}

	if doc.RecordCount() != 1 {
		t.Errorf("RecordCount() = %d, want 1", doc.RecordCount())
	}

	// Add second record
	doc.AddRecord(fields2)

	if doc.RecordCount() != 2 {
		t.Errorf("RecordCount() = %d, want 2", doc.RecordCount())
	}

	// Verify first record
	record1, ok := doc.GetRecord(0)
	if !ok {
		t.Fatal("GetRecord(0) failed")
	}
	if record1.Len() != len(fields1) {
		t.Errorf("record1.Len() = %d, want %d", record1.Len(), len(fields1))
	}

	// Verify second record
	record2, ok := doc.GetRecord(1)
	if !ok {
		t.Fatal("GetRecord(1) failed")
	}
	if record2.Len() != len(fields2) {
		t.Errorf("record2.Len() = %d, want %d", record2.Len(), len(fields2))
	}
}

// TestParseDocument tests parsing CSV into a Document (5.2.3)
func TestParseDocument(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantRows int
	}{
		{
			name:     "simple csv - all rows as records",
			input:    "name,age\nAlice,30\nBob,25",
			wantErr:  false,
			wantRows: 3, // All rows are treated as records by default
		},
		{
			name:     "csv data only",
			input:    "Alice,30\nBob,25",
			wantErr:  false,
			wantRows: 2,
		},
		{
			name:     "empty input",
			input:    "",
			wantErr:  false,
			wantRows: 0,
		},
		{
			name:    "invalid csv",
			input:   `"unclosed quote`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := csv.ParseDocument(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if doc == nil {
				t.Fatal("ParseDocument() returned nil document for valid input")
			}

			if doc.RecordCount() != tt.wantRows {
				t.Errorf("RecordCount() = %d, want %d", doc.RecordCount(), tt.wantRows)
			}
		})
	}
}

// TestDocumentCSV tests rendering Document back to CSV (5.2.2)
func TestDocumentCSV(t *testing.T) {
	doc := csv.NewDocument()
	doc.SetHeaders([]string{"name", "age"})
	doc.AddRecord([]string{"Alice", "30"})
	doc.AddRecord([]string{"Bob", "25"})

	csvStr, err := doc.CSV()
	if err != nil {
		t.Fatalf("CSV() error = %v", err)
	}

	if csvStr == "" {
		t.Error("CSV() returned empty string")
	}

	// Verify the CSV string contains expected content
	expectedLines := 3 // header + 2 records
	actualLines := len(strings.Split(strings.TrimSpace(csvStr), "\n"))
	if actualLines != expectedLines {
		t.Errorf("CSV() produced %d lines, want %d", actualLines, expectedLines)
	}

	// Verify round-trip: ParseDocument will treat all rows as records (header + data)
	doc2, err := csv.ParseDocument(csvStr)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	// doc2 has 3 records (header row + 2 data rows)
	// Original doc has 2 records (data only, headers separate)
	var expectedRecords int
	if len(doc.Headers()) > 0 {
		expectedRecords = 1 + doc.RecordCount() // 1 header row + data rows
	} else {
		expectedRecords = doc.RecordCount()
	}

	if doc2.RecordCount() != expectedRecords {
		t.Errorf("Round-trip failed: RecordCount() = %d, want %d", doc2.RecordCount(), expectedRecords)
	}
}

// TestRecord tests the Record type (5.2.4-5.2.5)
func TestRecord(t *testing.T) {
	doc := csv.NewDocument()
	doc.SetHeaders([]string{"name", "age", "email"})
	doc.AddRecord([]string{"Alice", "30", "alice@example.com"})

	record, ok := doc.GetRecord(0)
	if !ok {
		t.Fatal("GetRecord(0) failed")
	}

	// Test Len()
	if record.Len() != 3 {
		t.Errorf("Len() = %d, want 3", record.Len())
	}

	// Test Get() by index
	val0, ok := record.Get(0)
	if !ok || val0 != "Alice" {
		t.Errorf("Get(0) = %s, %v, want 'Alice', true", val0, ok)
	}

	val1, ok := record.Get(1)
	if !ok || val1 != "30" {
		t.Errorf("Get(1) = %s, %v, want '30', true", val1, ok)
	}

	val2, ok := record.Get(2)
	if !ok || val2 != "alice@example.com" {
		t.Errorf("Get(2) = %s, %v, want 'alice@example.com', true", val2, ok)
	}

	// Test Get() out of bounds
	_, ok = record.Get(3)
	if ok {
		t.Error("Get(3) should return false for out of bounds index")
	}

	_, ok = record.Get(-1)
	if ok {
		t.Error("Get(-1) should return false for negative index")
	}

	// Test GetByName()
	name, ok := record.GetByName("name")
	if !ok || name != "Alice" {
		t.Errorf("GetByName('name') = %s, %v, want 'Alice', true", name, ok)
	}

	age, ok := record.GetByName("age")
	if !ok || age != "30" {
		t.Errorf("GetByName('age') = %s, %v, want '30', true", age, ok)
	}

	email, ok := record.GetByName("email")
	if !ok || email != "alice@example.com" {
		t.Errorf("GetByName('email') = %s, %v, want 'alice@example.com', true", email, ok)
	}

	// Test GetByName() with non-existent field
	_, ok = record.GetByName("nonexistent")
	if ok {
		t.Error("GetByName('nonexistent') should return false")
	}

	// Test Fields()
	fields := record.Fields()
	expectedFields := []string{"Alice", "30", "alice@example.com"}
	if len(fields) != len(expectedFields) {
		t.Errorf("Fields() = %v, want %v", fields, expectedFields)
	}

	for i, f := range expectedFields {
		if fields[i] != f {
			t.Errorf("Fields()[%d] = %s, want %s", i, fields[i], f)
		}
	}
}

// TestRecordWithoutHeaders tests Record when no headers are set
func TestRecordWithoutHeaders(t *testing.T) {
	doc := csv.NewDocument()
	doc.AddRecord([]string{"Alice", "30"})

	record, ok := doc.GetRecord(0)
	if !ok {
		t.Fatal("GetRecord(0) failed")
	}

	// GetByName should return false when no headers
	_, ok = record.GetByName("name")
	if ok {
		t.Error("GetByName() should return false when no headers are set")
	}

	// Get by index should still work
	val, ok := record.Get(0)
	if !ok || val != "Alice" {
		t.Errorf("Get(0) = %s, %v, want 'Alice', true", val, ok)
	}
}

// TestDocumentRecords tests the Records() method
func TestDocumentRecords(t *testing.T) {
	doc := csv.NewDocument()
	doc.SetHeaders([]string{"name", "age"})
	doc.AddRecord([]string{"Alice", "30"})
	doc.AddRecord([]string{"Bob", "25"})

	records := doc.Records()
	if len(records) != 2 {
		t.Errorf("Records() returned %d records, want 2", len(records))
	}

	// Verify first record
	if records[0].Len() != 2 {
		t.Errorf("records[0].Len() = %d, want 2", records[0].Len())
	}

	name, _ := records[0].Get(0)
	if name != "Alice" {
		t.Errorf("records[0].Get(0) = %s, want 'Alice'", name)
	}

	// Verify second record
	if records[1].Len() != 2 {
		t.Errorf("records[1].Len() = %d, want 2", records[1].Len())
	}

	name, _ = records[1].Get(0)
	if name != "Bob" {
		t.Errorf("records[1].Get(0) = %s, want 'Bob'", name)
	}
}

// TestGetRecordOutOfBounds tests GetRecord with invalid indices
func TestGetRecordOutOfBounds(t *testing.T) {
	doc := csv.NewDocument()
	doc.AddRecord([]string{"Alice", "30"})

	// Test out of bounds
	_, ok := doc.GetRecord(1)
	if ok {
		t.Error("GetRecord(1) should return false for out of bounds index")
	}

	// Test negative index
	_, ok = doc.GetRecord(-1)
	if ok {
		t.Error("GetRecord(-1) should return false for negative index")
	}
}

// TestDocumentToAST tests converting Document to AST
func TestDocumentToAST(t *testing.T) {
	doc := csv.NewDocument()
	doc.SetHeaders([]string{"name", "age"})
	doc.AddRecord([]string{"Alice", "30"})
	doc.AddRecord([]string{"Bob", "25"})

	astNode, err := doc.ToAST()
	if err != nil {
		t.Fatalf("ToAST() error = %v", err)
	}

	if astNode == nil {
		t.Fatal("ToAST() returned nil")
	}

	// Verify it's an ArrayDataNode
	if astNode.Len() != 3 { // 1 header + 2 data records
		t.Errorf("ToAST() returned array with %d elements, want 3", astNode.Len())
	}
}

// TestDocumentToASTNoHeaders tests converting Document without headers to AST
func TestDocumentToASTNoHeaders(t *testing.T) {
	doc := csv.NewDocument()
	doc.AddRecord([]string{"Alice", "30"})
	doc.AddRecord([]string{"Bob", "25"})

	astNode, err := doc.ToAST()
	if err != nil {
		t.Fatalf("ToAST() error = %v", err)
	}

	if astNode == nil {
		t.Fatal("ToAST() returned nil")
	}

	// Should have only data records (no header)
	if astNode.Len() != 2 {
		t.Errorf("ToAST() returned array with %d elements, want 2", astNode.Len())
	}
}

// TestDocumentToASTEmpty tests converting empty Document to AST
func TestDocumentToASTEmpty(t *testing.T) {
	doc := csv.NewDocument()

	astNode, err := doc.ToAST()
	if err != nil {
		t.Fatalf("ToAST() error = %v", err)
	}

	if astNode == nil {
		t.Fatal("ToAST() returned nil")
	}

	// Empty document
	if astNode.Len() != 0 {
		t.Errorf("ToAST() returned array with %d elements, want 0", astNode.Len())
	}
}

// TestFromAST tests creating Document from AST
func TestFromAST(t *testing.T) {
	// Create a simple AST structure
	csvStr := "name,age\nAlice,30\nBob,25"
	astNode, err := csv.Parse(csvStr)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	doc, err := csv.FromAST(astNode)
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	if doc == nil {
		t.Fatal("FromAST() returned nil")
	}

	// Verify all rows are treated as records (no headers set automatically)
	if doc.RecordCount() != 3 {
		t.Errorf("FromAST() RecordCount() = %d, want 3", doc.RecordCount())
	}
}

// TestFromASTInvalidNode tests FromAST with invalid node types
func TestFromASTInvalidNode(t *testing.T) {
	// Test with LiteralNode (not ArrayDataNode)
	literalNode := ast.NewLiteralNode("test", ast.ZeroPosition())
	_, err := csv.FromAST(literalNode)
	if err == nil {
		t.Error("FromAST(LiteralNode) should return error")
	}
}

// TestWriteRecordWithQuoting tests writeRecord with fields requiring quoting
func TestWriteRecordWithQuoting(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{
			name:   "field with comma",
			fields: []string{"Hello, World", "test"},
			want:   "\"Hello, World\",test\n",
		},
		{
			name:   "field with quotes",
			fields: []string{"Say \"Hi\"", "test"},
			want:   "\"Say \"\"Hi\"\"\",test\n",
		},
		{
			name:   "field with newline",
			fields: []string{"Line1\nLine2", "test"},
			want:   "\"Line1\nLine2\",test\n",
		},
		{
			name:   "field with carriage return",
			fields: []string{"Line1\rLine2", "test"},
			want:   "\"Line1\rLine2\",test\n",
		},
		{
			name:   "simple fields",
			fields: []string{"simple", "test"},
			want:   "simple,test\n",
		},
		{
			name:   "empty field",
			fields: []string{"", "test"},
			want:   ",test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := csv.NewDocument()
			doc.AddRecord(tt.fields)

			csvStr, err := doc.CSV()
			if err != nil {
				t.Fatalf("CSV() error = %v", err)
			}

			if csvStr != tt.want {
				t.Errorf("CSV() = %q, want %q", csvStr, tt.want)
			}
		})
	}
}
