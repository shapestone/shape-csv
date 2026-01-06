package simd

// detectStructuralCharsFallback detects structural characters using pure Go.
// This is used when SIMD is not available or for partial chunks.
func detectStructuralCharsFallback(data []byte, delimiter byte) Bitmasks {
	var masks Bitmasks

	// Scan through the chunk and set bits for each character class
	for i := 0; i < len(data) && i < ChunkSize; i++ {
		c := data[i]

		// Check for quote
		if c == '"' {
			masks.Quotes |= 1 << uint(i)
		}

		// Check for delimiter
		if c == delimiter {
			masks.Delimiters |= 1 << uint(i)
		}

		// Check for newline (CR or LF)
		if c == '\r' || c == '\n' {
			masks.Newlines |= 1 << uint(i)
		}
	}

	return masks
}
