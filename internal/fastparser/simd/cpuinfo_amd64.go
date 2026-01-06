//go:build amd64

package simd

// getCPUFeatures detects AVX2 and SSE4.2 support on x86-64 platforms.
// Uses CPUID instruction to query CPU capabilities.
func getCPUFeatures() cpuFeatures {
	return cpuFeatures{
		hasAVX2:   hasAVX2Support(),
		hasSSE4_2: hasSSE42Support(),
	}
}

// hasAVX2Support checks if the CPU supports AVX2 instructions.
// AVX2 is indicated by bit 5 in EBX from CPUID with EAX=7, ECX=0.
func hasAVX2Support() bool {
	// Check CPUID.(EAX=07H, ECX=0H):EBX.AVX2[bit 5]
	_, ebx, _, _ := cpuid(7, 0)
	return (ebx & (1 << 5)) != 0
}

// hasSSE42Support checks if the CPU supports SSE4.2 instructions.
// SSE4.2 is indicated by bit 20 in ECX from CPUID with EAX=1.
func hasSSE42Support() bool {
	// Check CPUID.(EAX=01H):ECX.SSE4_2[bit 20]
	_, _, ecx, _ := cpuid(1, 0)
	return (ecx & (1 << 20)) != 0
}

// cpuid executes the CPUID instruction and returns the results.
// This is implemented in cpuid_amd64.s.
func cpuid(eaxIn, ecxIn uint32) (eax, ebx, ecx, edx uint32)
