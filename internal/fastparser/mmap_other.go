//go:build !unix

package fastparser

import (
	"fmt"
	"os"
)

// MmapFile reads a file into memory on non-Unix platforms.
// On platforms without mmap support, this falls back to reading the entire file.
// Returns the file data and a cleanup function.
//
// This provides the same API as the Unix version but without memory mapping.
// The cleanup function is still provided for API compatibility.
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
func MmapFile(filename string) ([]byte, func(), error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Provide a no-op cleanup function for API compatibility
	cleanup := func() {}

	return data, cleanup, nil
}
