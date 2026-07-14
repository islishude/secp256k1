// Command field generates the ADX/BMI2 base-field Montgomery kernels.
package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

const (
	fieldK0 = uint64(0xd838091dd2253531)
	fieldC  = uint64(0x00000001000003d1)
	fieldP0 = uint64(0xfffffffefffffc2f)
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm")
	emitMul()
	emitSquare()
	Generate()
}

func emitMul() {
	TEXT("mulMontgomeryADXAsm", NOSPLIT, "func(out, x, y *[4]uint64)")
	Pragma("noescape")
	Doc("mulMontgomeryADXAsm multiplies two canonical Montgomery field elements.")

	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	// Stage the complete second operand in SSE2 registers before arithmetic.
	// This preserves all output/input alias forms without the stack traffic that
	// disproportionately penalizes the Intel ADX implementation.
	y := [2]reg.VecPhysical{reg.X0, reg.X1}
	Load(Param("y"), reg.RAX)
	for i := range y {
		MOVOU(Mem{Base: reg.RAX}.Offset(i*16), y[i])
	}
	result := emitMontgomeryProduct(x, func(i int) {
		pair := y[i/2]
		if i&1 != 0 {
			PSRLDQ(U8(8), pair)
		}
		MOVQ(pair, reg.RDX)
	})
	emitStoreResult("out", result)
	RET()
}

func emitSquare() {
	TEXT("squareMontgomeryADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryADXAsm squares a canonical Montgomery field element.")

	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	result := emitMontgomeryProduct(x, func(i int) {
		MOVQ(x[i], reg.RDX)
	})
	emitStoreResult("out", result)
	RET()
}

func emitMontgomeryProduct(x [4]reg.GPPhysical, loadMultiplier func(int)) [4]reg.GPPhysical {
	// t is a five-limb accumulator. The top limb is required because the
	// secp256k1 modulus is only 33 bits below 2^256, so intermediate CIOS
	// results can cross the 256-bit boundary.
	t := [5]reg.GPPhysical{reg.R12, reg.R13, reg.R14, reg.R15, reg.RBX}
	XORQ(t[0], t[0])
	for i := 1; i < len(t); i++ {
		MOVQ(t[0], t[i])
	}

	for row := 0; row < 4; row++ {
		loadMultiplier(row)
		emitMulAddRow(x, t)
		emitMontgomeryShift(t)
	}

	result := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	for i := range result {
		MOVQ(t[i], result[i])
	}
	MOVQ(U64(fieldP0), reg.RAX)
	SUBQ(reg.RAX, result[0])
	SBBQ(I8(-1), result[1])
	SBBQ(I8(-1), result[2])
	SBBQ(I8(-1), result[3])
	MOVQ(t[4], reg.RDI)
	SBBQ(U8(0), reg.RDI)
	for i := range result {
		CMOVQCS(t[i], result[i])
	}
	return result
}

func emitMulAddRow(x [4]reg.GPPhysical, t [5]reg.GPPhysical) {
	// ADCX carries the multiplication chain while ADOX carries the existing
	// accumulator. RDI is zero until it captures the top carry.
	XORQ(reg.RDI, reg.RDI)
	previousHigh := reg.GPPhysical(reg.RDI)
	high := [2]reg.GPPhysical{reg.RCX, reg.RSI}
	for i := 0; i < 4; i++ {
		nextHigh := high[i&1]
		MULXQ(x[i], reg.RAX, nextHigh)
		ADCXQ(previousHigh, reg.RAX)
		ADOXQ(t[i], reg.RAX)
		MOVQ(reg.RAX, t[i])
		previousHigh = nextHigh
	}
	ADCXQ(reg.RDI, previousHigh)
	ADOXQ(reg.RDI, previousHigh)
	ADDQ(t[4], previousHigh)
	ADCQ(U8(0), reg.RDI)
	MOVQ(previousHigh, t[4])
}

func emitMontgomeryShift(t [5]reg.GPPhysical) {
	// q = t0*c^-1 mod 2^64. Since p = 2^256-c, q*p adds q at
	// limb four and subtracts q*c from the low limbs. The low product is
	// exactly t0 and is discarded by the Montgomery word shift.
	MOVQ(t[0], reg.RDX)
	MOVQ(U64(fieldK0), reg.RAX)
	IMULQ(reg.RAX, reg.RDX)
	MOVQ(U64(fieldC), reg.RAX)
	MULXQ(reg.RAX, t[0], reg.RCX)

	ADDQ(reg.RDX, t[4])
	ADCQ(U8(0), reg.RDI)
	SUBQ(reg.RCX, t[1])
	SBBQ(U8(0), t[2])
	SBBQ(U8(0), t[3])
	SBBQ(U8(0), t[4])
	SBBQ(U8(0), reg.RDI)

	MOVQ(t[1], t[0])
	MOVQ(t[2], t[1])
	MOVQ(t[3], t[2])
	MOVQ(t[4], t[3])
	MOVQ(reg.RDI, t[4])
}

func emitLoadElement(name string, dst [4]reg.GPPhysical) {
	Load(Param(name), reg.RSI)
	for i := range dst {
		MOVQ(Mem{Base: reg.RSI}.Offset(i*8), dst[i])
	}
}

func emitStoreResult(name string, src [4]reg.GPPhysical) {
	Load(Param(name), reg.RSI)
	for i := range src {
		MOVQ(src[i], Mem{Base: reg.RSI}.Offset(i*8))
	}
}
