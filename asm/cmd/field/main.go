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
	emitSquareN()
	emitMulByB3()
	Generate()
}

func emitMul() {
	TEXT("mulMontgomeryADXAsm", NOSPLIT, "func(out, x, y *[4]uint64)")
	Pragma("noescape")
	Doc("mulMontgomeryADXAsm multiplies two canonical Montgomery field elements.")

	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	result := emitMontgomeryProduct(x, func(i int) {
		Load(Param("y"), reg.RAX)
		MOVQ(Mem{Base: reg.RAX}.Offset(i*8), reg.RDX)
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

func emitSquareN() {
	TEXT("squareMontgomeryNADXAsm", NOSPLIT, "func(out, x *[4]uint64, n uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryNADXAsm performs a public number of repeated squarings.")

	counter := AllocLocal(8)
	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	Load(Param("n"), reg.RAX)
	MOVQ(reg.RAX, counter)
	TESTQ(reg.RAX, reg.RAX)
	JE(LabelRef("square_n_store"))

	Label("square_n_loop")
	x = emitMontgomeryProduct(x, func(i int) {
		MOVQ(x[i], reg.RDX)
	})
	DECQ(counter)
	JNE(LabelRef("square_n_loop"))

	Label("square_n_store")
	emitStoreResult("out", x)
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

func emitMulByB3() {
	TEXT("mulByB3MontgomeryADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("mulByB3MontgomeryADXAsm multiplies a Montgomery field element by 21.")

	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	result := [4]reg.GPPhysical{reg.RAX, reg.RBX, reg.RCX, reg.RDI}
	emitLoadElement("x", x)
	MOVQ(U32(21), reg.RDX)
	MULXQ(x[0], result[0], reg.R12)
	for i := 1; i < 4; i++ {
		MULXQ(x[i], reg.RSI, reg.R13)
		ADDQ(reg.R12, reg.RSI)
		ADCQ(U8(0), reg.R13)
		MOVQ(reg.RSI, result[i])
		MOVQ(reg.R13, reg.R12)
	}

	MOVQ(U64(fieldC), reg.R13)
	IMULQ(reg.R13, reg.R12)
	ADDQ(reg.R12, result[0])
	ADCQ(U8(0), result[1])
	ADCQ(U8(0), result[2])
	ADCQ(U8(0), result[3])
	MOVQ(U32(0), reg.R12)
	ADCQ(U8(0), reg.R12)
	IMULQ(reg.R13, reg.R12)
	ADDQ(reg.R12, result[0])
	ADCQ(U8(0), result[1])
	ADCQ(U8(0), result[2])
	ADCQ(U8(0), result[3])

	diff := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	for i := range diff {
		MOVQ(result[i], diff[i])
	}
	MOVQ(U64(fieldP0), reg.R12)
	SUBQ(reg.R12, diff[0])
	SBBQ(I8(-1), diff[1])
	SBBQ(I8(-1), diff[2])
	SBBQ(I8(-1), diff[3])
	for i := range diff {
		CMOVQCS(result[i], diff[i])
	}
	emitStoreResult("out", diff)
	RET()
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
