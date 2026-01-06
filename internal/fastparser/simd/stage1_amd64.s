// Copyright 2025 Shape Software, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build amd64

#include "textflag.h"

// func detectStructuralCharsASM(data *byte, delimiter byte, masks *Bitmasks)
//
// Detects structural characters (quotes, delimiters, newlines) in a 64-byte chunk
// using AVX2 instructions. Produces three 64-bit bitmasks.
//
// This function processes two 32-byte YMM registers (total 64 bytes) in parallel:
// - VPCMPEQB: Compare 32 bytes against a target character (produces 0xFF or 0x00)
// - VPMOVMSKB: Extract high bit from each byte to form a 32-bit mask
// - Combine two 32-bit masks into a 64-bit mask
//
// Algorithm inspired by simdjson and minio/simdcsv.
TEXT ·detectStructuralCharsASM(SB), NOSPLIT, $0-24
    MOVQ    data+0(FP), SI          // SI = pointer to data
    MOVQ    masks+16(FP), DI        // DI = pointer to output masks
    MOVB    delimiter+8(FP), AX     // AL = delimiter byte

    // Zero out the output masks
    MOVQ    $0, 0(DI)               // masks.Quotes = 0
    MOVQ    $0, 8(DI)               // masks.Delimiters = 0
    MOVQ    $0, 16(DI)              // masks.Newlines = 0

    // Load data into YMM registers (two 32-byte chunks)
    VMOVDQU 0(SI), Y0               // Y0 = data[0:32]
    VMOVDQU 32(SI), Y1              // Y1 = data[32:64]

    // ========================================
    // Detect quotes (")
    // ========================================
    MOVB    $'"', BX                // BL = quote character
    MOVD    BX, X2
    VPBROADCASTB X2, Y2             // Y2 = 32 copies of '"'

    VPCMPEQB Y0, Y2, Y3             // Y3 = (Y0 == '"') ? 0xFF : 0x00
    VPCMPEQB Y1, Y2, Y4             // Y4 = (Y1 == '"') ? 0xFF : 0x00

    VPMOVMSKB Y3, R8                // R8 = 32-bit mask from Y3
    VPMOVMSKB Y4, R9                // R9 = 32-bit mask from Y4

    // Combine into 64-bit mask
    MOVL    R8, R10                 // R10 = lower 32 bits
    SHLQ    $32, R9                 // R9 = upper 32 bits << 32
    ORQ     R9, R10                 // R10 = 64-bit quote mask
    MOVQ    R10, 0(DI)              // masks.Quotes = R10

    // ========================================
    // Detect delimiters (customizable)
    // ========================================
    MOVD    AX, X2
    VPBROADCASTB X2, Y2             // Y2 = 32 copies of delimiter

    VPCMPEQB Y0, Y2, Y3             // Y3 = (Y0 == delimiter) ? 0xFF : 0x00
    VPCMPEQB Y1, Y2, Y4             // Y4 = (Y1 == delimiter) ? 0xFF : 0x00

    VPMOVMSKB Y3, R8                // R8 = 32-bit mask from Y3
    VPMOVMSKB Y4, R9                // R9 = 32-bit mask from Y4

    // Combine into 64-bit mask
    MOVL    R8, R10                 // R10 = lower 32 bits
    SHLQ    $32, R9                 // R9 = upper 32 bits << 32
    ORQ     R9, R10                 // R10 = 64-bit delimiter mask
    MOVQ    R10, 8(DI)              // masks.Delimiters = R10

    // ========================================
    // Detect newlines (CR or LF)
    // ========================================
    // Strategy: Detect CR and LF separately, then OR them together

    // Detect CR (\r = 0x0D)
    MOVB    $'\r', BX               // BL = CR
    MOVD    BX, X2
    VPBROADCASTB X2, Y2             // Y2 = 32 copies of CR

    VPCMPEQB Y0, Y2, Y3             // Y3 = (Y0 == CR) ? 0xFF : 0x00
    VPCMPEQB Y1, Y2, Y4             // Y4 = (Y1 == CR) ? 0xFF : 0x00

    VPMOVMSKB Y3, R8                // R8 = 32-bit mask from Y3
    VPMOVMSKB Y4, R9                // R9 = 32-bit mask from Y4

    MOVL    R8, R10                 // R10 = lower 32 bits
    SHLQ    $32, R9                 // R9 = upper 32 bits << 32
    ORQ     R9, R10                 // R10 = 64-bit CR mask

    // Detect LF (\n = 0x0A)
    MOVB    $'\n', BX               // BL = LF
    MOVD    BX, X2
    VPBROADCASTB X2, Y2             // Y2 = 32 copies of LF

    VPCMPEQB Y0, Y2, Y3             // Y3 = (Y0 == LF) ? 0xFF : 0x00
    VPCMPEQB Y1, Y2, Y4             // Y4 = (Y1 == LF) ? 0xFF : 0x00

    VPMOVMSKB Y3, R8                // R8 = 32-bit mask from Y3
    VPMOVMSKB Y4, R9                // R9 = 32-bit mask from Y4

    MOVL    R8, R11                 // R11 = lower 32 bits
    SHLQ    $32, R9                 // R9 = upper 32 bits << 32
    ORQ     R9, R11                 // R11 = 64-bit LF mask

    // Combine CR and LF masks
    ORQ     R11, R10                // R10 = CR | LF
    MOVQ    R10, 16(DI)             // masks.Newlines = R10

    // Clean up YMM registers (required after AVX2 usage)
    VZEROUPPER

    RET
