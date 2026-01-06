//go:build amd64

package simd

// detectStructuralCharsASM is implemented in assembly (stage1_amd64.s).
// It detects structural characters using AVX2 instructions.
func detectStructuralCharsASM(data *byte, delimiter byte, masks *Bitmasks)

// detectStructuralChars detects structural characters in a 64-byte chunk.
// On x86-64 systems with AVX2, this uses the assembly implementation.
func detectStructuralChars(data []byte, delimiter byte) Bitmasks {
	if len(data) != ChunkSize {
		// Should not happen - caller should ensure chunk size
		return detectStructuralCharsFallback(data, delimiter)
	}

	var masks Bitmasks
	detectStructuralCharsASM(&data[0], delimiter, &masks)
	return masks
}
