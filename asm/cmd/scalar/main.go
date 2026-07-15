// Command scalar generates the ADX/BMI2 scalar-field Montgomery kernels.
package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

const scalarK0 = uint64(0x4b0dff665588b13f)

var scalarModulus = [4]uint64{
	0xbfd25e8cd0364141,
	0xbaaedce6af48a03b,
	0xfffffffffffffffe,
	0xffffffffffffffff,
}

// scalarComplement is c = 2^256-n. Montgomery reduction adds q*n, which is
// q*2^256-q*c. The three-limb complement saves two MULX instructions per row
// compared with multiplying all four limbs of n.
var scalarComplement = [3]uint64{
	0x402da1732fc9bebf,
	0x4551231950b75fc4,
	0x0000000000000001,
}

var (
	xlimbs        = [4]reg.VecPhysical{reg.X0, reg.X1, reg.X2, reg.X3}
	ylimbs        = [4]reg.VecPhysical{reg.X4, reg.X5, reg.X6, reg.X7}
	squareProduct = [8]reg.VecPhysical{reg.X8, reg.X9, reg.X10, reg.X11, reg.X12, reg.X13, reg.X14, reg.X15}
	invU          = [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	invV          = [4]reg.GPPhysical{reg.R12, reg.R13, reg.R14, reg.R15}
	invX1         = [4]reg.VecPhysical{reg.X0, reg.X1, reg.X2, reg.X3}
	invX2         = [4]reg.VecPhysical{reg.X4, reg.X5, reg.X6, reg.X7}
	invSum        = [5]reg.VecPhysical{reg.X8, reg.X9, reg.X10, reg.X11, reg.X12}
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm")
	emitMul()
	emitSquare()
	emitSquareN()
	emitInvVartime()
	Generate()
}

func emitMul() {
	TEXT("mulMontgomeryADXAsm", NOSPLIT, "func(out, x, y *[4]uint64)")
	Pragma("noescape")
	Doc("mulMontgomeryADXAsm multiplies two canonical Montgomery scalar elements.")

	emitLoadElement("x", xlimbs)
	emitLoadElement("y", ylimbs)
	result := emitMontgomeryProduct(xlimbs, ylimbs)
	emitStoreResult("out", result)
	RET()
}

func emitSquare() {
	TEXT("squareMontgomeryADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryADXAsm squares a canonical Montgomery scalar element.")

	emitLoadElement("x", xlimbs)
	result := emitMontgomerySquare(xlimbs)
	emitStoreResult("out", result)
	RET()
}

func emitSquareN() {
	TEXT("squareMontgomeryNADXAsm", NOSPLIT, "func(out, x *[4]uint64, n uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryNADXAsm performs a public number of Montgomery scalar squarings.")

	emitLoadElement("x", xlimbs)
	Load(Param("n"), reg.RDI)
	TESTQ(reg.RDI, reg.RDI)
	JEQ(LabelRef("square_n_store_input"))
	MOVQ(reg.RDI, reg.X4)

	Label("square_n_loop")
	result := emitMontgomerySquare(xlimbs)
	MOVQ(reg.X4, reg.RDI)
	DECQ(reg.RDI)
	JEQ(LabelRef("square_n_store_result"))
	MOVQ(reg.RDI, reg.X4)
	emitMoveResultToLimbs(result, xlimbs)
	JMP(LabelRef("square_n_loop"))

	Label("square_n_store_input")
	emitStoreLimbs("out", xlimbs)
	RET()

	Label("square_n_store_result")
	emitStoreResult("out", result)
	RET()
}

// emitMontgomerySquare computes x^2*R^-1 mod n. A seven-column Comba square
// evaluates each off-diagonal product once and adds it twice, reducing the
// product phase from 16 MULX instructions to 10. The full product is staged in
// SSE2 registers before a fixed four-step Montgomery reduction runs.
func emitMontgomerySquare(input [4]reg.VecPhysical) [4]reg.GPPhysical {
	carry := [3]reg.GPPhysical{reg.R8, reg.R9, reg.R10}
	for _, r := range carry {
		XORQ(r, r)
	}
	lo, hi := reg.R11, reg.R12
	zero := reg.R13
	XORQ(zero, zero)

	for column := 0; column <= 6; column++ {
		for i := range input {
			j := column - i
			if j < i || j < 0 || j >= len(input) {
				continue
			}

			MOVQ(input[i], reg.RDX)
			MOVQ(input[j], reg.RSI)
			MULXQ(reg.RSI, lo, hi)
			ADCXQ(lo, carry[0])
			ADCXQ(hi, carry[1])
			ADCXQ(zero, carry[2])
			if i != j {
				ADOXQ(lo, carry[0])
				ADOXQ(hi, carry[1])
				ADOXQ(zero, carry[2])
			}
		}

		MOVQ(carry[0], squareProduct[column])
		carry = [3]reg.GPPhysical{carry[1], carry[2], carry[0]}
		XORQ(carry[2], carry[2])
	}
	MOVQ(carry[0], squareProduct[7])

	return emitMontgomeryReduceFullSquare(squareProduct)
}

// emitMontgomeryReduceFullSquare reduces an eight-limb square product. For
// each low limb q = t[k]*c^-1, subtracting q*c and adding q at limb k+4 is
// exactly addition of q*n because n=2^256-c. The ninth accumulator limb keeps
// every carry until the final branch-free canonical subtraction.
func emitMontgomeryReduceFullSquare(product [8]reg.VecPhysical) [4]reg.GPPhysical {
	acc := [8]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11, reg.R12, reg.R13, reg.R14, reg.R15}
	for i := range acc {
		MOVQ(product[i], acc[i])
	}
	high := reg.RDI
	XORQ(high, high)

	for k := range 4 {
		MOVQ(acc[k], reg.RDX)
		MOVQ(U64(scalarK0), reg.RAX)
		IMULQ(reg.RAX, reg.RDX)

		// q*c occupies four limbs. The source and high destination of the
		// second MULX intentionally overlap; MULX reads its source first.
		MOVQ(U64(scalarComplement[0]), reg.RSI)
		MULXQ(reg.RSI, reg.RAX, reg.RBX)
		MOVQ(U64(scalarComplement[1]), reg.RSI)
		MULXQ(reg.RSI, reg.RCX, reg.RSI)
		ADDQ(reg.RBX, reg.RCX)
		ADCQ(U8(0), reg.RSI)
		MOVQ(reg.RDX, reg.RBX)
		ADDQ(reg.RSI, reg.RBX)
		MOVQ(U64(0), reg.RSI)
		ADCQ(U8(0), reg.RSI)

		SUBQ(reg.RAX, acc[k])
		SBBQ(reg.RCX, acc[k+1])
		SBBQ(reg.RBX, acc[k+2])
		SBBQ(reg.RSI, acc[k+3])
		for i := k + 4; i < len(acc); i++ {
			SBBQ(U8(0), acc[i])
		}
		SBBQ(U8(0), high)

		ADDQ(reg.RDX, acc[k+4])
		for i := k + 5; i < len(acc); i++ {
			ADCQ(U8(0), acc[i])
		}
		ADCQ(U8(0), high)
	}

	value := [4]reg.GPPhysical{acc[4], acc[5], acc[6], acc[7]}
	result := [4]reg.GPPhysical{acc[0], acc[1], acc[2], acc[3]}
	for i := range value {
		MOVQ(value[i], result[i])
	}
	MOVQ(U64(scalarModulus[0]), reg.RSI)
	SUBQ(reg.RSI, result[0])
	MOVQ(U64(scalarModulus[1]), reg.RSI)
	SBBQ(reg.RSI, result[1])
	SBBQ(I8(-2), result[2])
	SBBQ(I8(-1), result[3])
	MOVQ(high, reg.RDX)
	SBBQ(U8(0), reg.RDX)
	for i := range result {
		CMOVQCS(value[i], result[i])
	}
	return result
}

func emitInvVartime() {
	TEXT("invVartimeWordsADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("invVartimeWordsADXAsm computes a scalar inverse for public canonical input words.")

	Load(Param("x"), reg.RSI)
	for i := range invU {
		MOVQ(Mem{Base: reg.RSI}.Offset(i*8), invU[i])
		MOVQ(U64(scalarModulus[i]), invV[i])
	}

	MOVQ(U64(1), reg.RAX)
	MOVQ(reg.RAX, invX1[0])
	XORQ(reg.RAX, reg.RAX)
	for i := 1; i < len(invX1); i++ {
		MOVQ(reg.RAX, invX1[i])
	}
	for i := range invX2 {
		MOVQ(reg.RAX, invX2[i])
	}

	MOVQ(invU[0], reg.RAX)
	for i := 1; i < len(invU); i++ {
		ORQ(invU[i], reg.RAX)
	}
	JEQ(LabelRef("inv_return_zero"))

	Label("inv_loop")
	emitJumpIfOne("inv_u", invU, "inv_return_x1")
	emitJumpIfOne("inv_v", invV, "inv_return_x2")

	Label("inv_reduce_u")
	TESTQ(U32(1), invU[0])
	JNZ(LabelRef("inv_reduce_v"))
	emitShiftAndHalf("inv_u", invU, invX1)
	JMP(LabelRef("inv_reduce_u"))

	Label("inv_reduce_v")
	TESTQ(U32(1), invV[0])
	JNZ(LabelRef("inv_reduced"))
	emitShiftAndHalf("inv_v", invV, invX2)
	JMP(LabelRef("inv_reduce_v"))

	Label("inv_reduced")
	emitJumpIfOne("inv_reduced_u", invU, "inv_return_x1")
	emitJumpIfOne("inv_reduced_v", invV, "inv_return_x2")

	// Compare u and v by retaining only the final borrow from u-v.
	for i := range invU {
		MOVQ(invU[i], reg.RAX)
		if i == 0 {
			SUBQ(invV[i], reg.RAX)
		} else {
			SBBQ(invV[i], reg.RAX)
		}
	}
	JCS(LabelRef("inv_v_greater"))

	for i := range invU {
		if i == 0 {
			SUBQ(invV[i], invU[i])
		} else {
			SBBQ(invV[i], invU[i])
		}
	}
	emitSubModOrder("inv_x1_sub_x2", invX1, invX2)
	JMP(LabelRef("inv_loop"))

	Label("inv_v_greater")
	for i := range invV {
		if i == 0 {
			SUBQ(invU[i], invV[i])
		} else {
			SBBQ(invU[i], invV[i])
		}
	}
	emitSubModOrder("inv_x2_sub_x1", invX2, invX1)
	JMP(LabelRef("inv_loop"))

	Label("inv_return_x1")
	emitStoreLimbs("out", invX1)
	RET()

	Label("inv_return_x2")
	emitStoreLimbs("out", invX2)
	RET()

	Label("inv_return_zero")
	Load(Param("out"), reg.RSI)
	XORQ(reg.RAX, reg.RAX)
	for i := range 4 {
		MOVQ(reg.RAX, Mem{Base: reg.RSI}.Offset(i*8))
	}
	RET()
}

func emitJumpIfOne(prefix string, value [4]reg.GPPhysical, target string) {
	MOVQ(value[1], reg.RAX)
	ORQ(value[2], reg.RAX)
	ORQ(value[3], reg.RAX)
	JNZ(LabelRef(prefix + "_not_one"))
	CMPQ(value[0], U8(1))
	JEQ(LabelRef(target))
	Label(prefix + "_not_one")
}

// emitShiftAndHalf removes up to 64 trailing zero bits from value and divides
// coefficient by the same power of two modulo the scalar order. BSF handles a
// nonzero low limb in one step; an all-zero low limb is shifted by 64 bits.
func emitShiftAndHalf(prefix string, value [4]reg.GPPhysical, coefficient [4]reg.VecPhysical) {
	TESTQ(value[0], value[0])
	JEQ(LabelRef(prefix + "_shift_64"))
	BSFQ(value[0], reg.RCX)
	MOVQ(U64(64), reg.RSI)
	SUBQ(reg.RCX, reg.RSI)
	for i := 0; i < len(value)-1; i++ {
		SHRXQ(reg.RCX, value[i], reg.RAX)
		SHLXQ(reg.RSI, value[i+1], reg.RBX)
		ORQ(reg.RBX, reg.RAX)
		MOVQ(reg.RAX, value[i])
	}
	SHRXQ(reg.RCX, value[len(value)-1], reg.RAX)
	MOVQ(reg.RAX, value[len(value)-1])
	JMP(LabelRef(prefix + "_half"))

	Label(prefix + "_shift_64")
	for i := 0; i < len(value)-1; i++ {
		MOVQ(value[i+1], value[i])
	}
	XORQ(value[len(value)-1], value[len(value)-1])
	MOVQ(U64(64), reg.RCX)

	Label(prefix + "_half")
	emitBatchHalfModOrder(prefix, coefficient)
}

// emitBatchHalfModOrder computes coefficient/2^k mod n for 1 <= k <= 64.
// q = -coefficient*n^-1 mod 2^k makes coefficient+q*n divisible by 2^k;
// the five-limb sum is then shifted once instead of halving k times.
func emitBatchHalfModOrder(prefix string, coefficient [4]reg.VecPhysical) {
	MOVQ(coefficient[0], reg.RDX)
	MOVQ(U64(scalarK0), reg.RAX)
	IMULQ(reg.RAX, reg.RDX)
	CMPQ(reg.RCX, U8(64))
	JEQ(LabelRef(prefix + "_q_ready"))
	MOVQ(U64(1), reg.RAX)
	SHLXQ(reg.RCX, reg.RAX, reg.RAX)
	DECQ(reg.RAX)
	ANDQ(reg.RAX, reg.RDX)

	Label(prefix + "_q_ready")
	MOVQ(reg.RCX, reg.X13)
	zero := reg.RSI
	XORQ(zero, zero)
	previousHigh := reg.GPPhysical(zero)
	high := [2]reg.GPPhysical{reg.RDI, reg.RBX}
	for i := range scalarModulus {
		nextHigh := high[i&1]
		MOVQ(U64(scalarModulus[i]), reg.RCX)
		MULXQ(reg.RCX, reg.RAX, nextHigh)
		ADCXQ(previousHigh, reg.RAX)
		MOVQ(coefficient[i], reg.RCX)
		ADOXQ(reg.RCX, reg.RAX)
		MOVQ(reg.RAX, invSum[i])
		previousHigh = nextHigh
	}
	ADCXQ(zero, previousHigh)
	ADOXQ(zero, previousHigh)
	MOVQ(previousHigh, invSum[4])
	MOVQ(reg.X13, reg.RCX)

	CMPQ(reg.RCX, U8(64))
	JEQ(LabelRef(prefix + "_half_64"))
	MOVQ(U64(64), reg.RSI)
	SUBQ(reg.RCX, reg.RSI)
	for i := range coefficient {
		MOVQ(invSum[i], reg.RAX)
		MOVQ(invSum[i+1], reg.RBX)
		SHRXQ(reg.RCX, reg.RAX, reg.RAX)
		SHLXQ(reg.RSI, reg.RBX, reg.RBX)
		ORQ(reg.RBX, reg.RAX)
		MOVQ(reg.RAX, coefficient[i])
	}
	JMP(LabelRef(prefix + "_half_done"))

	Label(prefix + "_half_64")
	for i := range coefficient {
		MOVQ(invSum[i+1], coefficient[i])
	}

	Label(prefix + "_half_done")
}

func emitSubModOrder(prefix string, dst, src [4]reg.VecPhysical) {
	result := [4]reg.VecPhysical{reg.X8, reg.X9, reg.X10, reg.X11}
	for i := range dst {
		MOVQ(dst[i], reg.RAX)
		MOVQ(src[i], reg.RBX)
		if i == 0 {
			SUBQ(reg.RBX, reg.RAX)
		} else {
			SBBQ(reg.RBX, reg.RAX)
		}
		MOVQ(reg.RAX, result[i])
	}
	JCC(LabelRef(prefix + "_no_borrow"))

	for i := range dst {
		MOVQ(result[i], reg.RAX)
		MOVQ(U64(scalarModulus[i]), reg.RBX)
		if i == 0 {
			ADDQ(reg.RBX, reg.RAX)
		} else {
			ADCQ(reg.RBX, reg.RAX)
		}
		MOVQ(reg.RAX, dst[i])
	}
	JMP(LabelRef(prefix + "_done"))

	Label(prefix + "_no_borrow")
	emitCopyLimbs(result, dst)
	Label(prefix + "_done")
}

// emitMontgomeryProduct computes x*y*R^-1 mod n. Both complete inputs are
// already resident in SSE2 registers. The six-limb accumulator covers the
// strict bound below 2*n*R throughout CIOS reduction.
func emitMontgomeryProduct(multiplicand, multiplier [4]reg.VecPhysical) [4]reg.GPPhysical {
	t := [6]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11, reg.R12, reg.R13}
	XORQ(t[0], t[0])
	for i := 1; i < len(t); i++ {
		MOVQ(t[0], t[i])
	}

	for row := range 4 {
		MOVQ(multiplier[row], reg.RDX)
		emitMulAddVector(t, multiplicand)

		// q = t[0] * -n^-1 mod 2^64.
		MOVQ(t[0], reg.RDX)
		MOVQ(U64(scalarK0), reg.RAX)
		IMULQ(reg.RAX, reg.RDX)
		emitMontgomeryReduceScalar(t)

		// The low limb is zero by construction and is discarded. Rotate the
		// register mapping at generation time instead of emitting five MOVQ
		// instructions for the Montgomery word shift.
		t = [6]reg.GPPhysical{t[1], t[2], t[3], t[4], t[5], t[0]}
		XORQ(t[len(t)-1], t[len(t)-1])
	}

	return emitCanonicalReduction(t)
}

func emitMulAddVector(t [6]reg.GPPhysical, multiplicand [4]reg.VecPhysical) {
	emitMulAdd(t, func(i int, dst reg.GPPhysical) {
		MOVQ(multiplicand[i], dst)
	})
}

// emitMulAdd adds multiplier*loadMultiplicand() to the six-limb accumulator.
// RDX contains the multiplier. ADCX carries the product chain while ADOX
// carries the existing accumulator.
func emitMulAdd(t [6]reg.GPPhysical, loadMultiplicand func(int, reg.GPPhysical)) {
	zero := reg.RBX
	XORQ(zero, zero)
	previousHigh := reg.GPPhysical(zero)
	high := [2]reg.GPPhysical{reg.R14, reg.R15}
	for i := range 4 {
		nextHigh := high[i&1]
		loadMultiplicand(i, reg.RSI)
		MULXQ(reg.RSI, reg.RAX, nextHigh)
		ADCXQ(previousHigh, reg.RAX)
		ADOXQ(t[i], reg.RAX)
		MOVQ(reg.RAX, t[i])
		previousHigh = nextHigh
	}
	ADCXQ(zero, previousHigh)
	ADOXQ(zero, previousHigh)
	ADDQ(t[4], previousHigh)
	ADCQ(U8(0), t[5])
	MOVQ(previousHigh, t[4])
}

// emitMontgomeryReduceScalar adds q*n by computing q*2^256-q*c, where
// c=2^256-n has only three limbs. RDX contains q and remains unchanged by
// MULX. The four-limb q*c product stays in general-purpose registers before
// its fixed borrow chain is applied to the accumulator. This avoids the eight
// GPR/SSE2 transfers per reduction row used by the first complement schedule.
func emitMontgomeryReduceScalar(t [6]reg.GPPhysical) {
	MOVQ(U64(scalarComplement[0]), reg.RSI)
	MULXQ(reg.RSI, reg.R14, reg.RBX)

	MOVQ(U64(scalarComplement[1]), reg.RSI)
	MULXQ(reg.RSI, reg.R15, reg.RCX)
	ADDQ(reg.RBX, reg.R15)
	ADCQ(U8(0), reg.RCX)

	MOVQ(reg.RDX, reg.RAX)
	ADDQ(reg.RCX, reg.RAX)
	MOVQ(U64(0), reg.RSI)
	ADCQ(U8(0), reg.RSI)

	SUBQ(reg.R14, t[0])
	SBBQ(reg.R15, t[1])
	SBBQ(reg.RAX, t[2])
	SBBQ(reg.RSI, t[3])
	SBBQ(U8(0), t[4])
	SBBQ(U8(0), t[5])
	ADDQ(reg.RDX, t[4])
	ADCQ(U8(0), t[5])
}

func emitCanonicalReduction(t [6]reg.GPPhysical) [4]reg.GPPhysical {
	result := [4]reg.GPPhysical{reg.R14, reg.R15, reg.RAX, reg.RCX}
	for i := range result {
		MOVQ(t[i], result[i])
	}

	MOVQ(U64(scalarModulus[0]), reg.RSI)
	SUBQ(reg.RSI, result[0])
	MOVQ(U64(scalarModulus[1]), reg.RSI)
	SBBQ(reg.RSI, result[1])
	SBBQ(I8(-2), result[2])
	SBBQ(I8(-1), result[3])
	MOVQ(t[4], reg.RDX)
	SBBQ(U8(0), reg.RDX)
	for i := range result {
		CMOVQCS(t[i], result[i])
	}
	return result
}

func emitLoadElement(name string, dst [4]reg.VecPhysical) {
	Load(Param(name), reg.RSI)
	for i := range dst {
		MOVQ(Mem{Base: reg.RSI}.Offset(i*8), dst[i])
	}
}

func emitCopyLimbs(src, dst [4]reg.VecPhysical) {
	for i := range src {
		MOVQ(src[i], dst[i])
	}
}

func emitMoveResultToLimbs(src [4]reg.GPPhysical, dst [4]reg.VecPhysical) {
	for i := range src {
		MOVQ(src[i], dst[i])
	}
}

func emitStoreLimbs(name string, src [4]reg.VecPhysical) {
	Load(Param(name), reg.RSI)
	for i := range src {
		MOVQ(src[i], Mem{Base: reg.RSI}.Offset(i*8))
	}
}

func emitStoreResult(name string, src [4]reg.GPPhysical) {
	Load(Param(name), reg.RSI)
	for i := range src {
		MOVQ(src[i], Mem{Base: reg.RSI}.Offset(i*8))
	}
}
