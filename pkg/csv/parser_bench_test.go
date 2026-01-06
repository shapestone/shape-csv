package csv_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	shapecsv "github.com/shapestone/shape-csv/pkg/csv"
)

// Benchmark test data is loaded once and reused across all benchmarks
var (
	smallCSV  string
	mediumCSV string
	largeCSV  string
)

// loadBenchmarkData loads test data files once during test initialization
func loadBenchmarkData() error {
	if smallCSV != "" {
		return nil // already loaded
	}

	testdataDir := filepath.Join("..", "..", "testdata", "benchmarks")

	// Load small.csv
	smallBytes, err := os.ReadFile(filepath.Join(testdataDir, "small.csv"))
	if err != nil {
		return err
	}
	smallCSV = string(smallBytes)

	// Load medium.csv
	mediumBytes, err := os.ReadFile(filepath.Join(testdataDir, "medium.csv"))
	if err != nil {
		return err
	}
	mediumCSV = string(mediumBytes)

	// Load large.csv
	largeBytes, err := os.ReadFile(filepath.Join(testdataDir, "large.csv"))
	if err != nil {
		return err
	}
	largeCSV = string(largeBytes)

	return nil
}

// ================================
// Unmarshal to [][]string Benchmarks
// This is the apples-to-apples comparison with encoding/csv
// ================================

// BenchmarkShapeCSV_Unmarshal_Records_Small benchmarks unmarshaling to [][]string.
func BenchmarkShapeCSV_Unmarshal_Records_Small(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	data := []byte(smallCSV)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records [][]string
		err := shapecsv.Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkShapeCSV_Unmarshal_Records_Medium benchmarks unmarshaling to [][]string.
func BenchmarkShapeCSV_Unmarshal_Records_Medium(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	data := []byte(mediumCSV)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records [][]string
		err := shapecsv.Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkShapeCSV_Unmarshal_Records_Large benchmarks unmarshaling to [][]string.
func BenchmarkShapeCSV_Unmarshal_Records_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	data := []byte(largeCSV)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records [][]string
		err := shapecsv.Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkEncodingCSV_ReadAll_Small benchmarks encoding/csv.
func BenchmarkEncodingCSV_ReadAll_Small(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(smallCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := csv.NewReader(strings.NewReader(smallCSV))
		records, err := reader.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkEncodingCSV_ReadAll_Medium benchmarks encoding/csv.
func BenchmarkEncodingCSV_ReadAll_Medium(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(mediumCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := csv.NewReader(strings.NewReader(mediumCSV))
		records, err := reader.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkEncodingCSV_ReadAll_Large benchmarks encoding/csv.
func BenchmarkEncodingCSV_ReadAll_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(largeCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := csv.NewReader(strings.NewReader(largeCSV))
		records, err := reader.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// ================================
// AST Parser Benchmarks (for position tracking, manipulation)
// These are slower because they build full AST with source positions
// ================================

// BenchmarkShapeCSV_ASTParse_Small benchmarks parsing to AST.
func BenchmarkShapeCSV_ASTParse_Small(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(smallCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node, err := shapecsv.Parse(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
		_ = node
	}
}

// BenchmarkShapeCSV_ASTParse_Medium benchmarks parsing to AST.
func BenchmarkShapeCSV_ASTParse_Medium(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(mediumCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node, err := shapecsv.Parse(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
		_ = node
	}
}

// BenchmarkShapeCSV_ASTParse_Large benchmarks parsing to AST.
func BenchmarkShapeCSV_ASTParse_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(largeCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node, err := shapecsv.Parse(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
		_ = node
	}
}

// ================================
// Streaming Benchmarks
// ================================

// BenchmarkShapeCSV_Scanner_Large benchmarks streaming parsing.
func BenchmarkShapeCSV_Scanner_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(largeCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := shapecsv.NewScanner(strings.NewReader(largeCSV))
		scanner.SetHasHeaders(true)
		count := 0
		for scanner.Scan() {
			_ = scanner.Record()
			count++
		}
		if err := scanner.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncodingCSV_Reader_Large benchmarks encoding/csv streaming.
func BenchmarkEncodingCSV_Reader_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	b.SetBytes(int64(len(largeCSV)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := csv.NewReader(strings.NewReader(largeCSV))
		for {
			record, err := reader.Read()
			if err != nil {
				break
			}
			_ = record
		}
	}
}

// ================================
// Unmarshal Benchmarks (to structs)
// ================================

type BenchmarkRecord struct {
	ID         int     `csv:"id"`
	FirstName  string  `csv:"first_name"`
	LastName   string  `csv:"last_name"`
	Email      string  `csv:"email"`
	Department string  `csv:"department"`
	Salary     float64 `csv:"salary"`
	Active     bool    `csv:"active"`
	HireDate   string  `csv:"hire_date"`
	ManagerID  int     `csv:"manager_id"`
	Notes      string  `csv:"notes"`
}

// BenchmarkShapeCSV_Unmarshal_Medium benchmarks unmarshaling to structs.
func BenchmarkShapeCSV_Unmarshal_Medium(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	data := []byte(mediumCSV)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records []BenchmarkRecord
		err := shapecsv.Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkShapeCSV_Unmarshal_Large benchmarks unmarshaling large CSV.
func BenchmarkShapeCSV_Unmarshal_Large(b *testing.B) {
	if err := loadBenchmarkData(); err != nil {
		b.Fatalf("Failed to load benchmark data: %v", err)
	}

	// Use a subset for large unmarshal (full struct doesn't match large.csv)
	type LargeRecord struct {
		ID         int     `csv:"id"`
		FirstName  string  `csv:"first_name"`
		LastName   string  `csv:"last_name"`
		Email      string  `csv:"email"`
		Department string  `csv:"department"`
		Salary     float64 `csv:"salary"`
		Active     bool    `csv:"active"`
	}

	data := []byte(largeCSV)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records []LargeRecord
		err := shapecsv.Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// ================================
// Marshal Benchmarks
// ================================

// BenchmarkShapeCSV_Marshal_Medium benchmarks marshaling structs to CSV.
func BenchmarkShapeCSV_Marshal_Medium(b *testing.B) {
	records := make([]BenchmarkRecord, 100)
	for i := 0; i < 100; i++ {
		records[i] = BenchmarkRecord{
			ID:         i + 1,
			FirstName:  fmt.Sprintf("FirstName%d", i),
			LastName:   fmt.Sprintf("LastName%d", i),
			Email:      fmt.Sprintf("user%d@example.com", i),
			Department: fmt.Sprintf("Department%d", i%10),
			Salary:     50000.0 + float64(i)*100,
			Active:     i%2 == 0,
			HireDate:   "2024-01-15",
			ManagerID:  (i % 10) + 1,
			Notes:      fmt.Sprintf("Notes for user %d", i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := shapecsv.Marshal(records)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkEncodingCSV_Write_Medium benchmarks encoding/csv writing.
func BenchmarkEncodingCSV_Write_Medium(b *testing.B) {
	records := make([][]string, 101)
	records[0] = []string{"id", "first_name", "last_name", "email", "department", "salary", "active", "hire_date", "manager_id", "notes"}
	for i := 0; i < 100; i++ {
		records[i+1] = []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("FirstName%d", i),
			fmt.Sprintf("LastName%d", i),
			fmt.Sprintf("user%d@example.com", i),
			fmt.Sprintf("Department%d", i%10),
			fmt.Sprintf("%.2f", 50000.0+float64(i)*100),
			fmt.Sprintf("%t", i%2 == 0),
			"2024-01-15",
			fmt.Sprintf("%d", (i%10)+1),
			fmt.Sprintf("Notes for user %d", i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		err := writer.WriteAll(records)
		if err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
	}
}

// BenchmarkShapeCSV_Marshal_Large benchmarks marshaling large datasets.
func BenchmarkShapeCSV_Marshal_Large(b *testing.B) {
	type SimpleRecord struct {
		ID    int    `csv:"id"`
		Name  string `csv:"name"`
		Email string `csv:"email"`
	}

	records := make([]SimpleRecord, 10000)
	for i := 0; i < 10000; i++ {
		records[i] = SimpleRecord{
			ID:    i + 1,
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := shapecsv.Marshal(records)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkEncodingCSV_Write_Large benchmarks encoding/csv writing large data.
func BenchmarkEncodingCSV_Write_Large(b *testing.B) {
	records := make([][]string, 10001)
	records[0] = []string{"id", "name", "email"}
	for i := 0; i < 10000; i++ {
		records[i+1] = []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("User%d", i),
			fmt.Sprintf("user%d@example.com", i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		err := writer.WriteAll(records)
		if err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
	}
}

// ================================
// Quoted Fields Benchmarks
// ================================

// BenchmarkShapeCSV_Unmarshal_QuotedFields benchmarks parsing with quoted fields.
func BenchmarkShapeCSV_Unmarshal_QuotedFields(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("name,description,notes\n")
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("\"User %d\",\"Description with, comma and \"\"quotes\"\"\",\"Multi\nline\nnotes\"\n", i))
	}
	input := []byte(sb.String())

	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var records [][]string
		err := shapecsv.Unmarshal(input, &records)
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}

// BenchmarkEncodingCSV_QuotedFields benchmarks encoding/csv with quoted fields.
func BenchmarkEncodingCSV_QuotedFields(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("name,description,notes\n")
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("\"User %d\",\"Description with, comma and \"\"quotes\"\"\",\"Multi\nline\nnotes\"\n", i))
	}
	input := sb.String()

	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := csv.NewReader(strings.NewReader(input))
		records, err := reader.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
		_ = records
	}
}
