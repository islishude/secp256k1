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

	t := AllocLocal(9 * 8)
	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	y := [4]reg.GPPhysical{reg.RDX, reg.R12, reg.R13, reg.R14}

	Load(Param("x"), reg.RSI)
	for i := range x {
		MOVQ(Mem{Base: reg.RSI}.Offset(i*8), x[i])
	}
	Load(Param("y"), reg.RSI)
	for i := range y {
		MOVQ(Mem{Base: reg.RSI}.Offset(i*8), y[i])
	}

	XORQ(reg.R15, reg.R15)
	for i := 0; i < 9; i++ {
		MOVQ(reg.R15, t.Offset(i*8))
	}

	for row := 0; row < 4; row++ {
		if row != 0 {
			MOVQ(y[row], reg.RDX)
		}
		emitProductRow(t, row, x)
	}

	emitReduction(t, [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11})
	emitStoreResult("out", [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11})
	RET()
}

func emitProductRow(t Mem, row int, x [4]reg.GPPhysical) {
	// ADCX carries the multiplication chain while ADOX carries the accumulated
	// product already present in the destination limbs.
	XORQ(reg.R15, reg.R15)
	previousHigh := reg.GPPhysical(reg.R15)
	high := [2]reg.GPPhysical{reg.RBX, reg.RCX}
	for i := 0; i < 4; i++ {
		nextHigh := high[i&1]
		MULXQ(x[i], reg.RAX, nextHigh)
		ADCXQ(previousHigh, reg.RAX)
		ADOXQ(t.Offset((row+i)*8), reg.RAX)
		MOVQ(reg.RAX, t.Offset((row+i)*8))
		previousHigh = nextHigh
	}
	ADCXQ(reg.R15, previousHigh)
	ADOXQ(reg.R15, previousHigh)
	MOVQ(previousHigh, t.Offset((row+4)*8))
}

func emitSquare() {
	TEXT("squareMontgomeryADXAsm", NOSPLIT, "func(out, x *[4]uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryADXAsm squares a canonical Montgomery field element.")

	t := AllocLocal(9 * 8)
	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	emitSquareProduct(t, x)
	emitReduction(t, x)
	emitStoreResult("out", x)
	RET()
}

func emitSquareN() {
	TEXT("squareMontgomeryNADXAsm", NOSPLIT, "func(out, x *[4]uint64, n uint64)")
	Pragma("noescape")
	Doc("squareMontgomeryNADXAsm performs a public number of repeated squarings.")

	t := AllocLocal(9 * 8)
	x := [4]reg.GPPhysical{reg.R8, reg.R9, reg.R10, reg.R11}
	emitLoadElement("x", x)
	Load(Param("n"), reg.R14)
	TESTQ(reg.R14, reg.R14)
	JE(LabelRef("square_n_store"))

	Label("square_n_loop")
	emitSquareProduct(t, x)
	emitReduction(t, x)
	DECQ(reg.R14)
	JNE(LabelRef("square_n_loop"))

	Label("square_n_store")
	emitStoreResult("out", x)
	RET()
}

func emitSquareProduct(t Mem, x [4]reg.GPPhysical) {
	zero := reg.R12
	acc := [3]reg.GPPhysical{reg.RAX, reg.RBX, reg.RCX}
	lo, hi, top := reg.RSI, reg.RDI, reg.R13

	XORQ(zero, zero)
	for _, r := range acc {
		MOVQ(zero, r)
	}

	for column := 0; column <= 6; column++ {
		for i := 0; i < 4; i++ {
			j := column - i
			if j < i || j < 0 || j >= 4 {
				continue
			}
			MOVQ(x[i], reg.RDX)
			MULXQ(x[j], lo, hi)
			MOVQ(zero, top)
			if i != j {
				ADDQ(lo, lo)
				ADCQ(hi, hi)
				ADCQ(zero, top)
			}
			CLC()
			ADCXQ(lo, acc[0])
			ADCXQ(hi, acc[1])
			ADCXQ(top, acc[2])
		}
		MOVQ(acc[0], t.Offset(column*8))
		MOVQ(acc[1], acc[0])
		MOVQ(acc[2], acc[1])
		MOVQ(zero, acc[2])
	}
	MOVQ(acc[0], t.Offset(7*8))
	MOVQ(zero, t.Offset(8*8))
}

func emitReduction(t Mem, result [4]reg.GPPhysical) {
	// For p = 2^256-c and k0 = c^-1 mod 2^64, each Montgomery
	// cancellation adds q at limb i+4 and subtracts high(q*c) at i+1.
	for i := 0; i < 4; i++ {
		MOVQ(t.Offset(i*8), reg.RDX)
		MOVQ(U64(fieldK0), reg.RAX)
		IMULQ(reg.RAX, reg.RDX)
		MOVQ(U64(fieldC), reg.RAX)
		MULXQ(reg.RAX, reg.RBX, reg.RCX)

		ADDQ(reg.RDX, t.Offset((i+4)*8))
		for j := i + 5; j <= 8; j++ {
			ADCQ(U8(0), t.Offset(j*8))
		}
		SUBQ(reg.RCX, t.Offset((i+1)*8))
		for j := i + 2; j <= 8; j++ {
			SBBQ(U8(0), t.Offset(j*8))
		}
	}

	for i := range result {
		MOVQ(t.Offset((i+4)*8), result[i])
	}
	diff := [4]reg.GPPhysical{reg.RAX, reg.RBX, reg.RCX, reg.RDI}
	for i := range diff {
		MOVQ(result[i], diff[i])
	}
	MOVQ(U64(fieldP0), reg.R12)
	SUBQ(reg.R12, diff[0])
	SBBQ(I8(-1), diff[1])
	SBBQ(I8(-1), diff[2])
	SBBQ(I8(-1), diff[3])
	MOVQ(t.Offset(8*8), reg.RSI)
	SBBQ(U8(0), reg.RSI)
	for i := range diff {
		CMOVQCS(result[i], diff[i])
		MOVQ(diff[i], result[i])
	}
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
