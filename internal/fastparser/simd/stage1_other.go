//go:build !amd64

package simd

// detectStructuralChars detects structural characters using pure Go fallback.
// On non-x86-64 platforms, SIMD is not available.
func detectStructuralChars(data []byte, delimiter byte) Bitmasks {
	return detectStructuralCharsFallback(data, delimiter)
}
