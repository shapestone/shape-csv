package fastparser

import (
	"sync"
	"unsafe"
)

// fieldPool is a sync.Pool for []string slices used in record parsing.
// This reduces allocations when parsing multiple CSV records.
var fieldPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate with capacity for typical CSV records (8 fields)
		s := make([]string, 0, 8)
		return &s
	},
}

// bufferPool is a sync.Pool for []byte buffers used in quoted field parsing.
// These buffers are used to accumulate data when processing escaped quotes.
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate with capacity for typical quoted field content
		b := make([]byte, 0, 64)
		return &b
	},
}

// getFieldSlice gets a []string slice from the pool.
// The slice is returned with length 0 but may have capacity.
func getFieldSlice() []string {
	p := fieldPool.Get().(*[]string)
	fields := *p
	// Clear the slice but keep the capacity
	fields = fields[:0]
	return fields
}

// putFieldSlice returns a []string slice to the pool.
// The slice will be cleared before reuse.
func putFieldSlice(fields []string) {
	// Only return to pool if capacity is reasonable (avoid keeping huge slices)
	const maxCapacity = 1024
	if cap(fields) > maxCapacity {
		return
	}

	// Clear the slice
	fields = fields[:0]

	// Return to pool
	fieldPool.Put(&fields)
}

// getBuffer gets a []byte buffer from the pool.
// The buffer is returned with length 0 but may have capacity.
func getBuffer() []byte {
	p := bufferPool.Get().(*[]byte)
	buf := *p
	// Clear the buffer but keep the capacity
	buf = buf[:0]
	return buf
}

// putBuffer returns a []byte buffer to the pool.
// The buffer will be cleared before reuse.
func putBuffer(buf []byte) {
	// Only return to pool if capacity is reasonable (avoid keeping huge buffers)
	const maxCapacity = 4096
	if cap(buf) > maxCapacity {
		return
	}

	// Clear the buffer
	buf = buf[:0]

	// Return to pool
	bufferPool.Put(&buf)
}

// unsafeString converts a []byte to a string without allocation.
//
// This uses unsafe.String() which is available in Go 1.20+.
// The conversion creates a string that shares the underlying byte array,
// so the byte slice MUST NOT be modified after conversion.
//
// In our parser, we only use this on subslices of the immutable input data,
// so this is safe.
//
// Performance: This eliminates string allocations for unquoted fields,
// which typically make up the majority of CSV data.
func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// preAllocateCapacity returns the capacity hint for pre-allocating slices
// based on the first record's field count.
//
// This helps reduce allocations for subsequent records by allocating with
// the right capacity upfront.
func preAllocateCapacity(firstRecordFields int) int {
	if firstRecordFields <= 0 {
		return 8 // Default capacity
	}
	return firstRecordFields
}
