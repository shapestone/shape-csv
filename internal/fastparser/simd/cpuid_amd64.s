// Copyright 2025 Shape Software, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build amd64

#include "textflag.h"

// func cpuid(eaxIn, ecxIn uint32) (eax, ebx, ecx, edx uint32)
//
// Executes the CPUID instruction and returns the register values.
// This is used to detect CPU features like AVX2 and SSE4.2.
TEXT ·cpuid(SB), NOSPLIT, $0-24
    MOVL eaxIn+0(FP), AX
    MOVL ecxIn+4(FP), CX
    CPUID
    MOVL AX, eax+8(FP)
    MOVL BX, ebx+12(FP)
    MOVL CX, ecx+16(FP)
    MOVL DX, edx+20(FP)
    RET
