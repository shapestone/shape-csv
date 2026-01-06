package fastparser

import (
	"testing"
)

// Benchmark data sets
var (
	// Small CSV: 3 rows x 3 columns of simple unquoted data
	smallCSV = []byte("a,b,c\nd,e,f\ng,h,i")

	// Medium CSV: 100 rows x 10 columns of unquoted data
	mediumCSV = generateCSV(100, 10, false)

	// Large CSV: 1000 rows x 10 columns of unquoted data
	largeCSV = generateCSV(1000, 10, false)

	// Quoted CSV: 100 rows x 10 columns with quoted fields
	quotedCSV = generateCSV(100, 10, true)

	// Mixed CSV: 100 rows x 10 columns with mix of quoted and unquoted
	mixedCSV = generateMixedCSV(100, 10)
)

// generateCSV creates a CSV with specified dimensions
func generateCSV(rows, cols int, quoted bool) []byte {
	var data []byte
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c > 0 {
				data = append(data, ',')
			}
			if quoted {
				data = append(data, '"')
			}
			data = append(data, 'f')
			data = append(data, 'i')
			data = append(data, 'e')
			data = append(data, 'l')
			data = append(data, 'd')
			if quoted {
				data = append(data, '"')
			}
		}
		data = append(data, '\n')
	}
	return data
}

// generateLongFieldCSV creates a CSV with longer field values
func generateLongFieldCSV(rows, cols int, fieldLen int) []byte {
	var data []byte
	fieldData := make([]byte, fieldLen)
	for i := 0; i < fieldLen; i++ {
		fieldData[i] = 'a' + byte(i%26)
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c > 0 {
				data = append(data, ',')
			}
			data = append(data, fieldData...)
		}
		data = append(data, '\n')
	}
	return data
}

// generateMixedCSV creates CSV with alternating quoted/unquoted fields
func generateMixedCSV(rows, cols int) []byte {
	var data []byte
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c > 0 {
				data = append(data, ',')
			}
			// Alternate between quoted and unquoted
			if c%2 == 0 {
				data = append(data, '"')
				data = append(data, []byte("quoted field")...)
				data = append(data, '"')
			} else {
				data = append(data, []byte("unquoted")...)
			}
		}
		data = append(data, '\n')
	}
	return data
}

// BenchmarkParse_Small benchmarks parsing small CSV
func BenchmarkParse_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Medium benchmarks parsing medium CSV
func BenchmarkParse_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Large benchmarks parsing large CSV
func BenchmarkParse_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Quoted benchmarks parsing CSV with quoted fields
func BenchmarkParse_Quoted(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Mixed benchmarks parsing CSV with mixed quoted/unquoted
func BenchmarkParse_Mixed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(mixedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_VariableFieldCount benchmarks CSV with varying field counts
func BenchmarkParse_VariableFieldCount(b *testing.B) {
	data := []byte("a\na,b\na,b,c\na,b,c,d\na,b,c,d,e\n")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_QuotedWithEscapes benchmarks quoted fields with escaped quotes
func BenchmarkParse_QuotedWithEscapes(b *testing.B) {
	data := []byte(`"a""b","c""d""e","f""g""h""i"` + "\n")
	// Repeat 100 times to make it meaningful
	fullData := make([]byte, 0, len(data)*100)
	for i := 0; i < 100; i++ {
		fullData = append(fullData, data...)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(fullData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFieldPool tests the field pool performance
func BenchmarkFieldPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fields := getFieldSlice()
		fields = append(fields, "a", "b", "c")
		putFieldSlice(fields)
	}
}

// BenchmarkBufferPool tests the buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := getBuffer()
		buf = append(buf, []byte("test data")...)
		putBuffer(buf)
	}
}

// BenchmarkUnsafeString tests unsafe string conversion
func BenchmarkUnsafeString(b *testing.B) {
	data := []byte("benchmark test string")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = unsafeString(data)
	}
}

// BenchmarkSafeString tests regular string conversion for comparison
func BenchmarkSafeString(b *testing.B) {
	data := []byte("benchmark test string")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = string(data)
	}
}

// BenchmarkParseField_Unquoted benchmarks unquoted field parsing
func BenchmarkParseField_Unquoted(b *testing.B) {
	data := []byte("field1,field2,field3")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := &parser{
			data:   data,
			pos:    0,
			length: len(data),
		}
		for p.pos < p.length {
			_, err := p.parseField()
			if err != nil {
				b.Fatal(err)
			}
			if p.pos < p.length && p.data[p.pos] == ',' {
				p.pos++
			}
		}
	}
}

// BenchmarkParseField_Quoted benchmarks quoted field parsing
func BenchmarkParseField_Quoted(b *testing.B) {
	data := []byte(`"field1","field2","field3"`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := &parser{
			data:   data,
			pos:    0,
			length: len(data),
		}
		for p.pos < p.length {
			_, err := p.parseField()
			if err != nil {
				b.Fatal(err)
			}
			if p.pos < p.length && p.data[p.pos] == ',' {
				p.pos++
			}
		}
	}
}

// BenchmarkUnmarshal_Simple benchmarks unmarshaling to structs
func BenchmarkUnmarshal_Simple(b *testing.B) {
	type Record struct {
		Name  string
		Age   int
		Email string
	}

	data := []byte("Name,Age,Email\nJohn,30,john@example.com\nJane,25,jane@example.com\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshal_Large benchmarks unmarshaling large datasets
func BenchmarkUnmarshal_Large(b *testing.B) {
	type Record struct {
		F1 string
		F2 string
		F3 string
		F4 string
		F5 string
	}

	// Generate large dataset
	var data []byte
	data = append(data, []byte("F1,F2,F3,F4,F5\n")...)
	for i := 0; i < 1000; i++ {
		data = append(data, []byte("a,b,c,d,e\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshal_TypeCacheHit benchmarks the performance when type cache is hit
func BenchmarkUnmarshal_TypeCacheHit(b *testing.B) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	data := []byte("name,age\nAlice,30\nBob,25\n")

	// First call to warm up cache
	var warmup []Person
	_ = Unmarshal(data, &warmup)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Person
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshal_TypeCacheMiss benchmarks the performance when cache is cleared each time
func BenchmarkUnmarshal_TypeCacheMiss(b *testing.B) {
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}

	data := []byte("name,age\nAlice,30\nBob,25\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		clearStructCache() // Force cache miss
		var records []Person
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshal_MixedTypes benchmarks unmarshaling with various field types
func BenchmarkUnmarshal_MixedTypes(b *testing.B) {
	type Record struct {
		Name   string  `csv:"name"`
		Age    int     `csv:"age"`
		Score  float64 `csv:"score"`
		Active bool    `csv:"active"`
	}

	// Generate dataset
	var data []byte
	data = append(data, []byte("name,age,score,active\n")...)
	for i := 0; i < 1000; i++ {
		data = append(data, []byte("John,30,95.5,true\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshal_ComplexStruct benchmarks unmarshaling with many fields
func BenchmarkUnmarshal_ComplexStruct(b *testing.B) {
	type Record struct {
		F1  string  `csv:"f1"`
		F2  string  `csv:"f2"`
		F3  string  `csv:"f3"`
		F4  int     `csv:"f4"`
		F5  int     `csv:"f5"`
		F6  int     `csv:"f6"`
		F7  float64 `csv:"f7"`
		F8  float64 `csv:"f8"`
		F9  bool    `csv:"f9"`
		F10 bool    `csv:"f10"`
	}

	// Generate dataset
	var data []byte
	data = append(data, []byte("f1,f2,f3,f4,f5,f6,f7,f8,f9,f10\n")...)
	for i := 0; i < 500; i++ {
		data = append(data, []byte("a,b,c,1,2,3,1.5,2.5,true,false\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================
// ByteRecord Benchmarks (BurntSushi Pattern)
// ============================================

// BenchmarkParseByteRecords_Small benchmarks ByteRecord parsing on small CSV
func BenchmarkParseByteRecords_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseByteRecords(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseByteRecords_Medium benchmarks ByteRecord parsing on medium CSV
func BenchmarkParseByteRecords_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseByteRecords(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseByteRecords_Large benchmarks ByteRecord parsing on large CSV
func BenchmarkParseByteRecords_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseByteRecords(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseByteRecords_Quoted benchmarks ByteRecord parsing with quoted fields
func BenchmarkParseByteRecords_Quoted(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseByteRecords(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkByteRecord_FieldAccess benchmarks accessing fields as strings
func BenchmarkByteRecord_FieldAccess(b *testing.B) {
	records, err := ParseByteRecords(mediumCSV)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, record := range records {
			for j := 0; j < record.NumFields(); j++ {
				_ = record.Field(j)
			}
		}
	}
}

// BenchmarkByteRecord_FieldBytesAccess benchmarks zero-copy field access
func BenchmarkByteRecord_FieldBytesAccess(b *testing.B) {
	records, err := ParseByteRecords(mediumCSV)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, record := range records {
			for j := 0; j < record.NumFields(); j++ {
				_ = record.FieldBytes(j)
			}
		}
	}
}

// BenchmarkUnmarshalBytes_ToStringSlice benchmarks UnmarshalBytes to [][]string
func BenchmarkUnmarshalBytes_ToStringSlice(b *testing.B) {
	data := mediumCSV

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records [][]string
		err := UnmarshalBytes(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalBytes_ToStruct benchmarks UnmarshalBytes to structs
func BenchmarkUnmarshalBytes_ToStruct(b *testing.B) {
	type Record struct {
		F1  string
		F2  string
		F3  string
		F4  string
		F5  string
		F6  string
		F7  string
		F8  string
		F9  string
		F10 string
	}

	// Generate dataset with header
	var data []byte
	data = append(data, []byte("F1,F2,F3,F4,F5,F6,F7,F8,F9,F10\n")...)
	for i := 0; i < 100; i++ {
		data = append(data, []byte("a,b,c,d,e,f,g,h,i,j\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := UnmarshalBytes(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalBytes_Large benchmarks UnmarshalBytes on large dataset
func BenchmarkUnmarshalBytes_Large(b *testing.B) {
	type Record struct {
		F1 string
		F2 string
		F3 string
		F4 string
		F5 string
	}

	// Generate large dataset
	var data []byte
	data = append(data, []byte("F1,F2,F3,F4,F5\n")...)
	for i := 0; i < 1000; i++ {
		data = append(data, []byte("a,b,c,d,e\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := UnmarshalBytes(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Comparison benchmarks: Parse vs ParseByteRecords

// BenchmarkComparison_Parse_vs_ParseByteRecords runs side-by-side comparison
func BenchmarkComparison_Parse_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseByteRecords_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseByteRecords(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Unmarshal runs side-by-side comparison for Unmarshal
func BenchmarkComparison_Unmarshal_Structs(b *testing.B) {
	type Record struct {
		F1 string
		F2 string
		F3 string
		F4 string
		F5 string
	}

	var data []byte
	data = append(data, []byte("F1,F2,F3,F4,F5\n")...)
	for i := 0; i < 100; i++ {
		data = append(data, []byte("a,b,c,d,e\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := Unmarshal(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_UnmarshalBytes_Structs(b *testing.B) {
	type Record struct {
		F1 string
		F2 string
		F3 string
		F4 string
		F5 string
	}

	var data []byte
	data = append(data, []byte("F1,F2,F3,F4,F5\n")...)
	for i := 0; i < 100; i++ {
		data = append(data, []byte("a,b,c,d,e\n")...)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var records []Record
		err := UnmarshalBytes(data, &records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================
// Chunked Parser Benchmarks (Phase 14.5)
// ============================================

// BenchmarkParseChunked_Small benchmarks chunked parsing on small CSV
func BenchmarkParseChunked_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseChunked_Medium benchmarks chunked parsing on medium CSV
func BenchmarkParseChunked_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseChunked_Large benchmarks chunked parsing on large CSV
func BenchmarkParseChunked_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseChunked_VeryLarge benchmarks chunked parsing on very large CSV
func BenchmarkParseChunked_VeryLarge(b *testing.B) {
	// Generate 10000 rows x 10 columns
	veryLargeCSV := generateCSV(10000, 10, false)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(veryLargeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseChunked_Quoted benchmarks chunked parsing with quoted fields
func BenchmarkParseChunked_Quoted(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseChunked_Mixed benchmarks chunked parsing with mixed quoted/unquoted
func BenchmarkParseChunked_Mixed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(mixedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSWAR_HasDelimiter benchmarks SWAR delimiter detection
func BenchmarkSWAR_HasDelimiter(b *testing.B) {
	data := uint64(0x6c65696665646362) // "bcdefiel" in little-endian
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hasDelimiter(data, ',')
	}
}

// BenchmarkSWAR_FindDelimiterPos benchmarks SWAR delimiter position finding
func BenchmarkSWAR_FindDelimiterPos(b *testing.B) {
	data := uint64(0x6768666564632c62) // "b,cdefgh" in little-endian
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = findDelimiterPos(data, ',')
	}
}

// Comparison benchmarks: Parse vs ParseChunked

// BenchmarkComparison_Parse_vs_ParseChunked_Small compares byte-by-byte vs chunked on small CSV
func BenchmarkComparison_Parse_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseChunked_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_vs_ParseChunked_Large compares on large CSV
func BenchmarkComparison_Parse_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseChunked_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_vs_ParseChunked_VeryLarge compares on very large CSV
func BenchmarkComparison_Parse_VeryLarge(b *testing.B) {
	veryLargeCSV := generateCSV(10000, 10, false)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(veryLargeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseChunked_VeryLarge(b *testing.B) {
	veryLargeCSV := generateCSV(10000, 10, false)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(veryLargeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison with long fields - this is where SWAR shines

// BenchmarkComparison_Parse_LongFields benchmarks standard parser with 50-char fields
func BenchmarkComparison_Parse_LongFields(b *testing.B) {
	longFieldCSV := generateLongFieldCSV(1000, 10, 50)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(longFieldCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseChunked_LongFields(b *testing.B) {
	longFieldCSV := generateLongFieldCSV(1000, 10, 50)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(longFieldCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_VeryLongFields benchmarks with 200-char fields
func BenchmarkComparison_Parse_VeryLongFields(b *testing.B) {
	longFieldCSV := generateLongFieldCSV(1000, 10, 200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(longFieldCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_ParseChunked_VeryLongFields(b *testing.B) {
	longFieldCSV := generateLongFieldCSV(1000, 10, 200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseChunked(longFieldCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================
// DFA Parser Benchmarks (Phase 14.4)
// ============================================

// BenchmarkParseDFA_Small benchmarks DFA parsing small CSV
func BenchmarkParseDFA_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDFA_Medium benchmarks DFA parsing medium CSV
func BenchmarkParseDFA_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDFA_Large benchmarks DFA parsing large CSV
func BenchmarkParseDFA_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDFA_Quoted benchmarks DFA parsing CSV with quoted fields
func BenchmarkParseDFA_Quoted(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDFA_Mixed benchmarks DFA parsing CSV with mixed quoted/unquoted
func BenchmarkParseDFA_Mixed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(mixedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDFA_QuotedWithEscapes benchmarks DFA quoted fields with escaped quotes
func BenchmarkParseDFA_QuotedWithEscapes(b *testing.B) {
	data := []byte(`"a""b","c""d""e","f""g""h""i"` + "\n")
	// Repeat 100 times to make it meaningful
	fullData := make([]byte, 0, len(data)*100)
	for i := 0; i < 100; i++ {
		fullData = append(fullData, data...)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(fullData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================
// Comparison Benchmarks: Parse vs ParseDFA
// ============================================

// BenchmarkComparison_Parse_vs_DFA_Small compares Parse and ParseDFA on small input
func BenchmarkComparison_Parse_Small_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_DFA_Small_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(smallCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_vs_DFA_Medium compares Parse and ParseDFA on medium input
func BenchmarkComparison_Parse_Medium_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_DFA_Medium_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(mediumCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_vs_DFA_Large compares Parse and ParseDFA on large input
func BenchmarkComparison_Parse_Large_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_DFA_Large_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(largeCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison_Parse_vs_DFA_Quoted compares Parse and ParseDFA on quoted fields
func BenchmarkComparison_Parse_Quoted_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComparison_DFA_Quoted_DFA(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseDFA(quotedCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}
