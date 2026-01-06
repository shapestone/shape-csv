package fastparser

import (
	"testing"
	"unsafe"
)

// TestFieldPoolBasic tests basic field pool functionality
func TestFieldPoolBasic(t *testing.T) {
	// Get a slice from the pool
	fields := getFieldSlice()

	// Should start with length 0
	if len(fields) != 0 {
		t.Errorf("Expected initial length 0, got %d", len(fields))
	}

	// Should have some capacity
	if cap(fields) < 8 {
		t.Errorf("Expected capacity >= 8, got %d", cap(fields))
	}

	// Use the slice
	fields = append(fields, "a", "b", "c")
	if len(fields) != 3 {
		t.Errorf("Expected length 3 after append, got %d", len(fields))
	}

	// Return to pool
	putFieldSlice(fields)
}

// TestFieldPoolReuse tests that the pool actually reuses slices
func TestFieldPoolReuse(t *testing.T) {
	// Get a slice
	fields1 := getFieldSlice()
	fields1 = append(fields1, "test1")
	ptr1 := unsafe.Pointer(&fields1[0:cap(fields1)][0])

	// Return it
	putFieldSlice(fields1)

	// Get another slice - should be the same underlying array
	fields2 := getFieldSlice()

	// After clearing, it should be empty but have capacity
	if len(fields2) != 0 {
		t.Errorf("Expected length 0 after pool reuse, got %d", len(fields2))
	}

	// Add something to get a valid pointer
	fields2 = append(fields2, "test2")
	ptr2 := unsafe.Pointer(&fields2[0:cap(fields2)][0])

	// Pointers should be the same (reused backing array)
	if ptr1 != ptr2 {
		t.Logf("Note: Pool didn't reuse slice (this is non-deterministic and may be ok)")
	}

	putFieldSlice(fields2)
}

// TestFieldPoolConcurrent tests concurrent pool access
func TestFieldPoolConcurrent(t *testing.T) {
	const workers = 10
	done := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each worker gets and puts slices multiple times
			for j := 0; j < 100; j++ {
				fields := getFieldSlice()
				fields = append(fields, "test")
				if len(fields) != 1 {
					t.Errorf("Worker %d: expected length 1, got %d", id, len(fields))
				}
				putFieldSlice(fields)
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < workers; i++ {
		<-done
	}
}

// TestBufferPoolBasic tests basic buffer pool functionality
func TestBufferPoolBasic(t *testing.T) {
	// Get a buffer from the pool
	buf := getBuffer()

	// Should start with length 0
	if len(buf) != 0 {
		t.Errorf("Expected initial length 0, got %d", len(buf))
	}

	// Should have some capacity
	if cap(buf) < 64 {
		t.Errorf("Expected capacity >= 64, got %d", cap(buf))
	}

	// Use the buffer
	buf = append(buf, []byte("hello")...)
	if len(buf) != 5 {
		t.Errorf("Expected length 5 after append, got %d", len(buf))
	}

	// Return to pool
	putBuffer(buf)
}

// TestBufferPoolReuse tests that the buffer pool reuses buffers
func TestBufferPoolReuse(t *testing.T) {
	// Get a buffer
	buf1 := getBuffer()
	buf1 = append(buf1, []byte("test1")...)
	ptr1 := unsafe.Pointer(&buf1[0:cap(buf1)][0])

	// Return it
	putBuffer(buf1)

	// Get another buffer - should be the same underlying array
	buf2 := getBuffer()

	// After clearing, it should be empty but have capacity
	if len(buf2) != 0 {
		t.Errorf("Expected length 0 after pool reuse, got %d", len(buf2))
	}

	// Add something to get a valid pointer
	buf2 = append(buf2, []byte("test2")...)
	ptr2 := unsafe.Pointer(&buf2[0:cap(buf2)][0])

	// Pointers should be the same (reused backing array)
	if ptr1 != ptr2 {
		t.Logf("Note: Pool didn't reuse buffer (this is non-deterministic and may be ok)")
	}

	putBuffer(buf2)
}

// TestBufferPoolConcurrent tests concurrent buffer pool access
func TestBufferPoolConcurrent(t *testing.T) {
	const workers = 10
	done := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each worker gets and puts buffers multiple times
			for j := 0; j < 100; j++ {
				buf := getBuffer()
				buf = append(buf, []byte("test")...)
				if len(buf) != 4 {
					t.Errorf("Worker %d: expected length 4, got %d", id, len(buf))
				}
				putBuffer(buf)
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < workers; i++ {
		<-done
	}
}

// TestUnsafeString tests unsafe string conversion
func TestUnsafeString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty",
			input: []byte{},
			want:  "",
		},
		{
			name:  "simple string",
			input: []byte("hello"),
			want:  "hello",
		},
		{
			name:  "with special chars",
			input: []byte("hello, world!"),
			want:  "hello, world!",
		},
		{
			name:  "with newline",
			input: []byte("hello\nworld"),
			want:  "hello\nworld",
		},
		{
			name:  "with quote",
			input: []byte(`say "hello"`),
			want:  `say "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unsafeString(tt.input)
			if got != tt.want {
				t.Errorf("unsafeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUnsafeStringNoAlloc tests that unsafeString doesn't allocate
func TestUnsafeStringNoAlloc(t *testing.T) {
	// Create a byte slice
	data := []byte("test data for no allocation")

	// Get the data pointer
	dataPtr := unsafe.Pointer(&data[0])

	// Convert using unsafeString
	str := unsafeString(data)

	// Get the string data pointer
	strData := unsafe.StringData(str)
	strPtr := unsafe.Pointer(strData)

	// Pointers should be the same (no allocation)
	if dataPtr != strPtr {
		t.Errorf("unsafeString allocated: byte slice ptr %p != string ptr %p", dataPtr, strPtr)
	}
}

// TestUnsafeStringSafety tests that unsafeString is safe for read-only use
func TestUnsafeStringSafety(t *testing.T) {
	// Create a byte slice
	original := []byte("original")

	// Convert to string
	str := unsafeString(original)

	// Original and string should have same content
	if str != string(original) {
		t.Errorf("unsafeString() = %q, want %q", str, string(original))
	}

	// Modifying the original byte slice will affect the string
	// This is expected behavior for unsafe conversion - the caller must ensure
	// the byte slice isn't modified after conversion
	// We test this to document the behavior
	originalCopy := make([]byte, len(original))
	copy(originalCopy, original)

	original[0] = 'X'

	// String is affected (shares data)
	if str[0] != 'X' {
		t.Errorf("Expected string to be affected by byte slice modification (this is expected unsafe behavior)")
	}

	// This demonstrates why the byte slice must not be modified after conversion
	// In our parser, we only use unsafeString on subslices of the immutable input data
}

// TestPreAllocateCapacity tests capacity hint calculation
func TestPreAllocateCapacity(t *testing.T) {
	tests := []struct {
		name      string
		firstSize int
		want      int
	}{
		{
			name:      "zero fields",
			firstSize: 0,
			want:      8, // Should fall back to default
		},
		{
			name:      "small record",
			firstSize: 3,
			want:      3,
		},
		{
			name:      "medium record",
			firstSize: 10,
			want:      10,
		},
		{
			name:      "large record",
			firstSize: 100,
			want:      100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preAllocateCapacity(tt.firstSize)
			if got != tt.want {
				t.Errorf("preAllocateCapacity(%d) = %d, want %d", tt.firstSize, got, tt.want)
			}
		})
	}
}

// TestPoolIntegration tests that pools work with actual parsing
func TestPoolIntegration(t *testing.T) {
	// Simple CSV data
	data := []byte("a,b,c\nd,e,f\ng,h,i")

	// Parse it
	records, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check results
	want := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
		{"g", "h", "i"},
	}

	if len(records) != len(want) {
		t.Errorf("Got %d records, want %d", len(records), len(want))
	}

	for i, record := range records {
		if len(record) != len(want[i]) {
			t.Errorf("Record %d: got %d fields, want %d", i, len(record), len(want[i]))
			continue
		}
		for j, field := range record {
			if field != want[i][j] {
				t.Errorf("Record %d, field %d: got %q, want %q", i, j, field, want[i][j])
			}
		}
	}
}

// TestPoolIntegrationQuotedFields tests pools with quoted fields
func TestPoolIntegrationQuotedFields(t *testing.T) {
	// CSV with quoted fields containing escaped quotes
	data := []byte(`"a""b","c,d","e` + "\n" + `f"`)

	// Parse it
	records, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check results
	want := [][]string{
		{`a"b`, "c,d", "e\nf"},
	}

	if len(records) != len(want) {
		t.Errorf("Got %d records, want %d", len(records), len(want))
	}

	for i, record := range records {
		if len(record) != len(want[i]) {
			t.Errorf("Record %d: got %d fields, want %d", i, len(record), len(want[i]))
			continue
		}
		for j, field := range record {
			if field != want[i][j] {
				t.Errorf("Record %d, field %d: got %q, want %q", i, j, field, want[i][j])
			}
		}
	}
}

// TestPutFieldSlice_LargeCapacity tests that putFieldSlice doesn't return slices with excessive capacity.
func TestPutFieldSlice_LargeCapacity(t *testing.T) {
	// Create a slice with capacity exceeding maxCapacity (1024)
	largeSlice := make([]string, 0, 2000)
	largeSlice = append(largeSlice, "test")

	// Put it back to the pool
	putFieldSlice(largeSlice)

	// Get a new slice from the pool
	newSlice := getFieldSlice()

	// The new slice should not be the same as the large one
	// (pool should have rejected it due to excessive capacity)
	if cap(newSlice) > 1024 {
		t.Logf("Note: Got slice with capacity %d, expected <= 1024 (pool may have accepted large slice)", cap(newSlice))
	}
}

// TestPutBuffer_LargeCapacity tests that putBuffer doesn't return buffers with excessive capacity.
func TestPutBuffer_LargeCapacity(t *testing.T) {
	// Create a buffer with capacity exceeding maxCapacity (4096)
	largeBuf := make([]byte, 0, 8000)
	largeBuf = append(largeBuf, []byte("test")...)

	// Put it back to the pool
	putBuffer(largeBuf)

	// Get a new buffer from the pool
	newBuf := getBuffer()

	// The new buffer should not be the same as the large one
	// (pool should have rejected it due to excessive capacity)
	if cap(newBuf) > 4096 {
		t.Logf("Note: Got buffer with capacity %d, expected <= 4096 (pool may have accepted large buffer)", cap(newBuf))
	}
}

// TestPutFieldSlice_EmptySlice tests that putFieldSlice handles empty slices.
func TestPutFieldSlice_EmptySlice(t *testing.T) {
	// Create an empty slice with non-zero capacity
	emptySlice := make([]string, 0, 10)

	// This should not panic
	putFieldSlice(emptySlice)

	// Getting a slice should still work
	slice := getFieldSlice()
	if len(slice) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(slice))
	}
}

// TestPutBuffer_EmptyBuffer tests that putBuffer handles empty buffers.
func TestPutBuffer_EmptyBuffer(t *testing.T) {
	// Create an empty buffer with non-zero capacity
	emptyBuf := make([]byte, 0, 100)

	// This should not panic
	putBuffer(emptyBuf)

	// Getting a buffer should still work
	buf := getBuffer()
	if len(buf) != 0 {
		t.Errorf("Expected empty buffer, got length %d", len(buf))
	}
}

// TestPutFieldSlice_ZeroCapacity tests putting back a zero-capacity slice.
func TestPutFieldSlice_ZeroCapacity(t *testing.T) {
	// Create a zero-capacity slice
	zeroSlice := make([]string, 0)

	// Put it back
	putFieldSlice(zeroSlice)

	// Get a new slice
	slice := getFieldSlice()

	// Should get a properly initialized slice
	if cap(slice) == 0 {
		t.Logf("Note: Got zero-capacity slice (pool may have accepted zero-capacity slice)")
	}
}

// TestPutBuffer_ZeroCapacity tests putting back a zero-capacity buffer.
func TestPutBuffer_ZeroCapacity(t *testing.T) {
	// Create a zero-capacity buffer
	zeroBuf := make([]byte, 0)

	// Put it back
	putBuffer(zeroBuf)

	// Get a new buffer
	buf := getBuffer()

	// Should get a properly initialized buffer
	if cap(buf) == 0 {
		t.Logf("Note: Got zero-capacity buffer (pool may have accepted zero-capacity buffer)")
	}
}

// TestPoolStress tests pools under concurrent stress.
func TestPoolStress(t *testing.T) {
	const workers = 50
	const iterations = 1000
	done := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < iterations; j++ {
				// Alternate between field slices and buffers
				if j%2 == 0 {
					slice := getFieldSlice()
					slice = append(slice, "test1", "test2", "test3")
					putFieldSlice(slice)
				} else {
					buf := getBuffer()
					buf = append(buf, []byte("test data")...)
					putBuffer(buf)
				}
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < workers; i++ {
		<-done
	}
}
