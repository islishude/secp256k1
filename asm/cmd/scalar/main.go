// Command scalar generates the ADX/BMI2 public scalar-inversion kernel.
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

var (
	invU   = [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	invV   = [4]reg.GPPhysical{reg.R12, reg.R13, reg.R14, reg.R15}
	invX1  = [4]reg.VecPhysical{reg.X0, reg.X1, reg.X2, reg.X3}
	invX2  = [4]reg.VecPhysical{reg.X4, reg.X5, reg.X6, reg.X7}
	invSum = [5]reg.VecPhysical{reg.X8, reg.X9, reg.X10, reg.X11, reg.X12}
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm")
	emitInvVartime()
	Generate()
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

func emitCopyLimbs(src, dst [4]reg.VecPhysical) {
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
