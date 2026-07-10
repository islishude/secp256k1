// Copyright (c) 2026 islishude.
// The Comba multiplication layout is informed by Go's P-256 ARM64 backend.
// Portions Copyright 2018 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by a BSD-style license; see
// LICENSES/BSD-3-Clause-Go.txt.

//go:build arm64 && secp256k1_asm

#include "textflag.h"

#define res_ptr R0
#define a_ptr R1

#define acc0 R3
#define acc1 R4
#define acc2 R5
#define acc3 R6
#define acc4 R7
#define acc5 R8
#define acc6 R9
#define acc7 R10

#define p0 R11
#define p1 R12
#define p2 R13
#define p3 R14
#define p4 R15
#define q R16

#define x0 R19
#define x1 R20
#define x2 R21
#define x3 R22
#define montK0 R23
#define productLo R24
#define productHi R25
#define acc8 R26

DATA scalarConstants<>+0x00(SB)/8, $0x4b0dff665588b13f
DATA scalarConstants<>+0x08(SB)/8, $0x402da1732fc9bebf
DATA scalarConstants<>+0x10(SB)/8, $0x4551231950b75fc4
GLOBL scalarConstants<>(SB), RODATA|NOPTR, $24

// func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64)
//
// The loop count is public and comes from a static exponentiation chain. The
// Montgomery value remains in registers between iterations.
TEXT ·squareMontgomeryN(SB), NOSPLIT, $0-24
	MOVD out+0(FP), res_ptr
	MOVD x+8(FP), a_ptr
	MOVD n+16(FP), R2
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)
	CBZ R2, scalar_square_n_store_input
	MOVD $scalarConstants<>(SB), a_ptr
	MOVD 0(a_ptr), montK0

scalar_square_n_body:
	MUL x0, x1, acc1
	UMULH x0, x1, acc2
	MUL x0, x2, productLo
	ADDS productLo, acc2, acc2
	UMULH x0, x2, acc3
	MUL x0, x3, productLo
	ADCS productLo, acc3, acc3
	UMULH x0, x3, acc4
	ADC $0, acc4, acc4

	MUL x1, x2, productLo
	ADDS productLo, acc3
	UMULH x1, x2, productHi
	ADCS productHi, acc4
	ADC $0, ZR, acc5
	MUL x1, x3, productLo
	ADDS productLo, acc4
	UMULH x1, x3, productHi
	ADC productHi, acc5
	MUL x2, x3, productLo
	ADDS productLo, acc5
	UMULH x2, x3, acc6
	ADC $0, acc6
	MOVD ZR, acc7

	ADDS acc1, acc1
	ADCS acc2, acc2
	ADCS acc3, acc3
	ADCS acc4, acc4
	ADCS acc5, acc5
	ADCS acc6, acc6
	ADC $0, acc7

	MUL x0, x0, acc0
	UMULH x0, x0, productLo
	ADDS productLo, acc1, acc1
	MUL x1, x1, productLo
	ADCS productLo, acc2, acc2
	UMULH x1, x1, productHi
	ADCS productHi, acc3, acc3
	MUL x2, x2, productLo
	ADCS productLo, acc4, acc4
	UMULH x2, x2, productHi
	ADCS productHi, acc5, acc5
	MUL x3, x3, productLo
	ADCS productLo, acc6, acc6
	UMULH x3, x3, productHi
	ADCS productHi, acc7, acc7

	LDP 8(a_ptr), (x0, x1)
	MOVD ZR, acc8

	MUL acc0, montK0, q
	UMULH q, x0, p1
	MUL q, x1, productLo
	ADDS productLo, p1, p1
	UMULH q, x1, p2
	ADCS q, p2, p2
	ADC ZR, ZR, p3
	ADDS q, acc4, acc4
	ADCS ZR, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, acc8, acc8
	SUBS p1, acc1, acc1
	SBCS p2, acc2, acc2
	SBCS p3, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, acc8, acc8

	MUL acc1, montK0, q
	UMULH q, x0, p1
	MUL q, x1, productLo
	ADDS productLo, p1, p1
	UMULH q, x1, p2
	ADCS q, p2, p2
	ADC ZR, ZR, p3
	ADDS q, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, acc8, acc8
	SUBS p1, acc2, acc2
	SBCS p2, acc3, acc3
	SBCS p3, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, acc8, acc8

	MUL acc2, montK0, q
	UMULH q, x0, p1
	MUL q, x1, productLo
	ADDS productLo, p1, p1
	UMULH q, x1, p2
	ADCS q, p2, p2
	ADC ZR, ZR, p3
	ADDS q, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, acc8, acc8
	SUBS p1, acc3, acc3
	SBCS p2, acc4, acc4
	SBCS p3, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, acc8, acc8

	MUL acc3, montK0, q
	UMULH q, x0, p1
	MUL q, x1, productLo
	ADDS productLo, p1, p1
	UMULH q, x1, p2
	ADCS q, p2, p2
	ADC ZR, ZR, p3
	ADDS q, acc7, acc7
	ADC ZR, acc8, acc8
	SUBS p1, acc4, acc4
	SBCS p2, acc5, acc5
	SBCS p3, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, acc8, acc8

	NEG x0, x0
	MVN x1, x1
	MOVD $-2, x2
	MOVD $-1, x3
	SUBS x0, acc4, p0
	SBCS x1, acc5, p1
	SBCS x2, acc6, p2
	SBCS x3, acc7, p3
	SBCS ZR, acc8, acc8
	CSEL CS, p0, acc4, acc0
	CSEL CS, p1, acc5, acc1
	CSEL CS, p2, acc6, acc2
	CSEL CS, p3, acc7, acc3

	SUB $1, R2, R2
	CBZ R2, scalar_square_n_store_result
	MOVD acc0, x0
	MOVD acc1, x1
	MOVD acc2, x2
	MOVD acc3, x3
	B scalar_square_n_body

scalar_square_n_store_input:
	STP (x0, x1), 0*16(res_ptr)
	STP (x2, x3), 1*16(res_ptr)
	RET

scalar_square_n_store_result:
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET

