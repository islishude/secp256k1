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

var (
	xlimbs = [4]reg.VecPhysical{reg.X0, reg.X1, reg.X2, reg.X3}
	ylimbs = [4]reg.VecPhysical{reg.X4, reg.X5, reg.X6, reg.X7}
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm")
	emitMul()
	emitSquare()
	emitSquareN()
	Generate()
}

func emitMul() {
	TEXT("mulMontgomeryADXAsm", NOSPLIT, "func(out, x, y *[4]uint64)")
	Pragma("noescape")
	Doc("mulMontgomeryADXAsm multiplies two canonical Montgomery scalar elements.")

	emitLoadElement("x", xlimbs)
	emitLoadElement("y", ylimbs)
	result := emitMontgomeryProduct()
	emitStoreResult("out", result)
	RET()
}

func emitSquare() {
	TEXT("squareMontgomeryADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryADXAsm squares a canonical Montgomery scalar element.")

	emitLoadElement("x", xlimbs)
	emitCopyLimbs(xlimbs, ylimbs)
	result := emitMontgomeryProduct()
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

	Label("square_n_loop")
	emitCopyLimbs(xlimbs, ylimbs)
	result := emitMontgomeryProduct()
	DECQ(reg.RDI)
	JEQ(LabelRef("square_n_store_result"))
	emitMoveResultToLimbs(result, xlimbs)
	JMP(LabelRef("square_n_loop"))

	Label("square_n_store_input")
	emitStoreLimbs("out", xlimbs)
	RET()

	Label("square_n_store_result")
	emitStoreResult("out", result)
	RET()
}

// emitMontgomeryProduct computes x*y*R^-1 mod n. Both complete inputs are
// already resident in SSE2 registers. The six-limb accumulator covers the
// strict bound below 2*n*R throughout CIOS reduction.
func emitMontgomeryProduct() [4]reg.GPPhysical {
	t := [6]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11, reg.R12, reg.R13}
	XORQ(t[0], t[0])
	for i := 1; i < len(t); i++ {
		MOVQ(t[0], t[i])
	}

	for row := range 4 {
		MOVQ(ylimbs[row], reg.RDX)
		emitMulAddVector(t, xlimbs)

		// q = t[0] * -n^-1 mod 2^64.
		MOVQ(t[0], reg.RDX)
		MOVQ(U64(scalarK0), reg.RAX)
		IMULQ(reg.RAX, reg.RDX)
		emitMulAddModulus(t)

		// The low limb is zero by construction and is discarded.
		for i := 0; i < len(t)-1; i++ {
			MOVQ(t[i+1], t[i])
		}
		XORQ(t[len(t)-1], t[len(t)-1])
	}

	return emitCanonicalReduction(t)
}

func emitMulAddVector(t [6]reg.GPPhysical, multiplicand [4]reg.VecPhysical) {
	emitMulAdd(t, func(i int, dst reg.GPPhysical) {
		MOVQ(multiplicand[i], dst)
	})
}

func emitMulAddModulus(t [6]reg.GPPhysical) {
	emitMulAdd(t, func(i int, dst reg.GPPhysical) {
		MOVQ(U64(scalarModulus[i]), dst)
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
