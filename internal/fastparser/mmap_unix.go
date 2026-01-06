//go:build unix

package fastparser

import (
	"fmt"
	"os"
	"syscall"
)

// MmapFile memory-maps a file for reading.
// Returns the mapped byte slice and a cleanup function that must be called to unmap the file.
//
// This is useful for processing large CSV files efficiently:
//   - The file is mapped into memory without loading it entirely
//   - The OS handles paging data in/out as needed
//   - Combined with zero-copy parsing, this enables processing huge files with minimal memory
//
// Example usage:
//
//	data, cleanup, err := MmapFile("large.csv")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//
//	records, err := ParseZeroCopy(data)
//	// Process records...
//
// IMPORTANT: Do not use the data slice after calling cleanup().
func MmapFile(filename string) ([]byte, func(), error) {
	// Open the file
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	size := stat.Size()
	if size == 0 {
		// Empty file - return empty slice and cleanup that just closes the file
		return []byte{}, func() { f.Close() }, nil
	}

	// Memory-map the file
	data, err := syscall.Mmap(
		int(f.Fd()),
		0,
		int(size),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	// Create cleanup function that unmaps and closes
	cleanup := func() {
		_ = syscall.Munmap(data)
		f.Close()
	}

	return data, cleanup, nil
}
