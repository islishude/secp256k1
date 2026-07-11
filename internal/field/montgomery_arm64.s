// Copyright (c) 2026 islishude.
// The Comba multiplication layout is informed by Go's P-256 ARM64 backend.
// Portions Copyright 2018 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by a BSD-style license; see
// LICENSES/BSD-3-Clause-Go.txt.

//go:build arm64 && secp256k1_asm

#include "textflag.h"

#define res_ptr R0
#define a_ptr R1
#define b_ptr R2

#define acc0 R3
#define acc1 R4
#define acc2 R5
#define acc3 R6
#define acc4 R7
#define acc5 R8
#define acc6 R9
#define acc7 R10

#define c0 R11
#define c1 R12
#define productLo R13
#define productHi R14
#define montK0 R15
#define reductionC R16

#define x0 R19
#define x1 R20
#define x2 R21
#define x3 R22
#define y0 R23
#define y1 R24
#define y2 R25
#define y3 R26

// p = 2^256 - c and montK0 = c^-1 mod 2^64. Montgomery reduction can
// therefore replace each four-limb q*p multiplication with q*2^256 - q*c.
DATA baseFieldK0<>+0x00(SB)/8, $0xd838091dd2253531
DATA baseFieldC<>+0x00(SB)/8, $0x00000001000003d1
GLOBL baseFieldK0<>(SB), RODATA|NOPTR, $8
GLOBL baseFieldC<>(SB), RODATA|NOPTR, $8

// func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement)
//
// The inputs are loaded before the output is written, so all aliasing
// combinations are supported. The first half computes the full 512-bit Comba
// product. The second half applies the same secp256k1-specific Montgomery
// reduction used by squareMontgomery.
TEXT ·mulMontgomery(SB), NOSPLIT, $0-24
	MOVD x+8(FP), a_ptr
	MOVD y+16(FP), b_ptr
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)
	LDP 0*16(b_ptr), (y0, y1)
	LDP 1*16(b_ptr), (y2, y3)

	// y0 * x.
	MUL y0, x0, acc0
	UMULH y0, x0, acc1
	MUL y0, x1, productLo
	ADDS productLo, acc1
	UMULH y0, x1, acc2
	MUL y0, x2, productLo
	ADCS productLo, acc2
	UMULH y0, x2, acc3
	MUL y0, x3, productLo
	ADCS productLo, acc3
	UMULH y0, x3, acc4
	ADC $0, acc4

	// y1 * x.
	MUL y1, x0, productLo
	ADDS productLo, acc1
	UMULH y1, x0, c0
	MUL y1, x1, productLo
	ADCS productLo, acc2
	UMULH y1, x1, c1
	MUL y1, x2, productLo
	ADCS productLo, acc3
	UMULH y1, x2, productHi
	MUL y1, x3, productLo
	ADCS productLo, acc4
	UMULH y1, x3, montK0
	ADC $0, ZR, acc5
	ADDS c0, acc2
	ADCS c1, acc3
	ADCS productHi, acc4
	ADC montK0, acc5

	// y2 * x.
	MUL y2, x0, productLo
	ADDS productLo, acc2
	UMULH y2, x0, c0
	MUL y2, x1, productLo
	ADCS productLo, acc3
	UMULH y2, x1, c1
	MUL y2, x2, productLo
	ADCS productLo, acc4
	UMULH y2, x2, productHi
	MUL y2, x3, productLo
	ADCS productLo, acc5
	UMULH y2, x3, montK0
	ADC $0, ZR, acc6
	ADDS c0, acc3
	ADCS c1, acc4
	ADCS productHi, acc5
	ADC montK0, acc6

	// y3 * x.
	MUL y3, x0, productLo
	ADDS productLo, acc3
	UMULH y3, x0, c0
	MUL y3, x1, productLo
	ADCS productLo, acc4
	UMULH y3, x1, c1
	MUL y3, x2, productLo
	ADCS productLo, acc5
	UMULH y3, x2, productHi
	MUL y3, x3, productLo
	ADCS productLo, acc6
	UMULH y3, x3, montK0
	ADC $0, ZR, acc7
	ADDS c0, acc4
	ADCS c1, acc5
	ADCS productHi, acc6
	ADC montK0, acc7

	MOVD baseFieldK0<>(SB), montK0
	MOVD baseFieldC<>(SB), reductionC
	MOVD ZR, x0

	// Montgomery reduction step 0.
	MUL acc0, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc4, acc4
	ADCS ZR, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc1, acc1
	SBCS ZR, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Montgomery reduction step 1.
	MUL acc1, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Montgomery reduction step 2.
	MUL acc2, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Montgomery reduction step 3.
	MUL acc3, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MOVD $0xfffffffefffffc2f, x1
	MOVD $-1, x2
	SUBS x1, acc4, productLo
	SBCS x2, acc5, productHi
	SBCS x2, acc6, c0
	SBCS x2, acc7, c1
	SBCS ZR, x0, x0
	CSEL CS, productLo, acc4, acc0
	CSEL CS, productHi, acc5, acc1
	CSEL CS, c0, acc6, acc2
	CSEL CS, c1, acc7, acc3

	MOVD out+0(FP), res_ptr
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET

// func addMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement)
TEXT ·addMontgomery(SB), NOSPLIT, $0-24
	MOVD x+8(FP), a_ptr
	MOVD y+16(FP), b_ptr
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)
	LDP 0*16(b_ptr), (y0, y1)
	LDP 1*16(b_ptr), (y2, y3)

	ADDS y0, x0, acc0
	ADCS y1, x1, acc1
	ADCS y2, x2, acc2
	ADCS y3, x3, acc3
	ADC ZR, ZR, acc4

	MOVD $0xfffffffefffffc2f, x0
	MOVD $-1, x1
	SUBS x0, acc0, productLo
	SBCS x1, acc1, productHi
	SBCS x1, acc2, c0
	SBCS x1, acc3, c1
	SBCS ZR, acc4, acc4
	CSEL CS, productLo, acc0, acc0
	CSEL CS, productHi, acc1, acc1
	CSEL CS, c0, acc2, acc2
	CSEL CS, c1, acc3, acc3

	MOVD out+0(FP), res_ptr
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET

// func mulByB3Montgomery(out, x *fiat.MontgomeryDomainFieldElement)
//
// The input is already in Montgomery form, so multiplying its canonical limbs
// by the ordinary constant 21 preserves the representation. Two fixed folds
// use 2^256 = 2^32 + 977 (mod p).
TEXT ·mulByB3Montgomery(SB), NOSPLIT, $0-16
	MOVD x+8(FP), a_ptr
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)
	MOVD $21, b_ptr

	MUL x0, b_ptr, acc0
	UMULH x0, b_ptr, acc4
	MUL x1, b_ptr, acc1
	ADDS acc4, acc1, acc1
	UMULH x1, b_ptr, acc4
	ADC ZR, acc4, acc4
	MUL x2, b_ptr, acc2
	ADDS acc4, acc2, acc2
	UMULH x2, b_ptr, acc4
	ADC ZR, acc4, acc4
	MUL x3, b_ptr, acc3
	ADDS acc4, acc3, acc3
	UMULH x3, b_ptr, acc4
	ADC ZR, acc4, acc4

	MOVD $0x1000003d1, reductionC
	MUL acc4, reductionC, productLo
	ADDS productLo, acc0, acc0
	ADCS ZR, acc1, acc1
	ADCS ZR, acc2, acc2
	ADCS ZR, acc3, acc3
	ADC ZR, ZR, acc4

	MUL acc4, reductionC, productLo
	ADDS productLo, acc0, acc0
	ADCS ZR, acc1, acc1
	ADCS ZR, acc2, acc2
	ADC ZR, acc3, acc3

	MOVD $0xfffffffefffffc2f, x0
	MOVD $-1, x1
	SUBS x0, acc0, productLo
	SBCS x1, acc1, productHi
	SBCS x1, acc2, c0
	SBCS x1, acc3, c1
	CSEL CS, productLo, acc0, acc0
	CSEL CS, productHi, acc1, acc1
	CSEL CS, c0, acc2, acc2
	CSEL CS, c1, acc3, acc3

	MOVD out+0(FP), res_ptr
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET
// func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement)
//
// Squaring computes the ten unique limb products, doubles the cross terms,
// adds the four diagonal products, and applies a secp256k1-specific Montgomery
// reduction.
TEXT ·squareMontgomery(SB), NOSPLIT, $0-16
	MOVD x+8(FP), a_ptr
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)

	// Cross products.
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

	// Double the cross terms.
	ADDS acc1, acc1
	ADCS acc2, acc2
	ADCS acc3, acc3
	ADCS acc4, acc4
	ADCS acc5, acc5
	ADCS acc6, acc6
	ADC $0, acc7

	// Add diagonal products.
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

	MOVD baseFieldK0<>(SB), montK0
	MOVD baseFieldC<>(SB), reductionC
	MOVD ZR, x0

	// Step 0.
	MUL acc0, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc4, acc4
	ADCS ZR, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc1, acc1
	SBCS ZR, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Step 1.
	MUL acc1, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Step 2.
	MUL acc2, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	// Step 3.
	MUL acc3, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MOVD $0xfffffffefffffc2f, x1
	MOVD $-1, x2
	SUBS x1, acc4, productLo
	SBCS x2, acc5, productHi
	SBCS x2, acc6, c0
	SBCS x2, acc7, c1
	SBCS ZR, x0, x0
	CSEL CS, productLo, acc4, acc0
	CSEL CS, productHi, acc5, acc1
	CSEL CS, c0, acc6, acc2
	CSEL CS, c1, acc7, acc3

	MOVD out+0(FP), res_ptr
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET

// func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64)
//
// The loop count is public and comes from a static exponentiation chain. Field
// values remain in registers between iterations.
TEXT ·squareMontgomeryN(SB), NOSPLIT, $0-24
	MOVD out+0(FP), res_ptr
	MOVD x+8(FP), a_ptr
	MOVD n+16(FP), b_ptr
	LDP 0*16(a_ptr), (x0, x1)
	LDP 1*16(a_ptr), (x2, x3)
	CBZ b_ptr, square_n_store_input
	MOVD baseFieldK0<>(SB), montK0
	MOVD baseFieldC<>(SB), reductionC

square_n_body:
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

	MOVD ZR, x0
	MUL acc0, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc4, acc4
	ADCS ZR, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc1, acc1
	SBCS ZR, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MUL acc1, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc5, acc5
	ADCS ZR, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc2, acc2
	SBCS ZR, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MUL acc2, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc6, acc6
	ADCS ZR, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc3, acc3
	SBCS ZR, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MUL acc3, montK0, productLo
	UMULH productLo, reductionC, productHi
	ADDS productLo, acc7, acc7
	ADC ZR, x0, x0
	SUBS productHi, acc4, acc4
	SBCS ZR, acc5, acc5
	SBCS ZR, acc6, acc6
	SBCS ZR, acc7, acc7
	SBC ZR, x0, x0

	MOVD $0xfffffffefffffc2f, x1
	MOVD $-1, x2
	SUBS x1, acc4, productLo
	SBCS x2, acc5, productHi
	SBCS x2, acc6, c0
	SBCS x2, acc7, c1
	SBCS ZR, x0, x0
	CSEL CS, productLo, acc4, acc0
	CSEL CS, productHi, acc5, acc1
	CSEL CS, c0, acc6, acc2
	CSEL CS, c1, acc7, acc3

	SUB $1, b_ptr, b_ptr
	CBZ b_ptr, square_n_store_result
	MOVD acc0, x0
	MOVD acc1, x1
	MOVD acc2, x2
	MOVD acc3, x3
	B square_n_body

square_n_store_input:
	STP (x0, x1), 0*16(res_ptr)
	STP (x2, x3), 1*16(res_ptr)
	RET

square_n_store_result:
	STP (acc0, acc1), 0*16(res_ptr)
	STP (acc2, acc3), 1*16(res_ptr)
	RET
