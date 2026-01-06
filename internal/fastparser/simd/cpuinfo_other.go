//go:build !amd64

package simd

// getCPUFeatures returns no SIMD capabilities on non-x86-64 platforms.
// Future: Add ARM NEON support for Apple Silicon.
func getCPUFeatures() cpuFeatures {
	return cpuFeatures{
		hasAVX2:   false,
		hasSSE4_2: false,
	}
}
