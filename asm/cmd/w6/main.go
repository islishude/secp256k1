// Command w6 generates the fixed-length SSE2 selector for the W6 base table.
package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm,!secp256k1_amd64_w5_bench")
	TEXT("selectGeneratorW6", NOSPLIT, "func(out *[8]uint64, table *[32][8]uint64, magnitude uint64)")
	Pragma("noescape")
	Doc("selectGeneratorW6 scans all 32 packed W6 points with a fixed SSE2 instruction sequence.")

	Load(Param("out"), reg.R8)
	Load(Param("table"), reg.R9)
	Load(Param("magnitude"), reg.R10)
	selected := [4]reg.VecPhysical{reg.X0, reg.X1, reg.X2, reg.X3}
	candidate := [4]reg.VecPhysical{reg.X5, reg.X6, reg.X7, reg.X8}
	for i := range selected {
		MOVOU(Mem{Base: reg.R9}.Offset(i*16), selected[i])
	}

	for digit := 2; digit <= 32; digit++ {
		ADDQ(U8(64), reg.R9)
		CMPQ(reg.R10, U8(digit))
		SETEQ(reg.AL)
		MOVBQZX(reg.AL, reg.RAX)
		NEGQ(reg.RAX)
		MOVQ(reg.RAX, reg.X4)
		PUNPCKLQDQ(reg.X4, reg.X4)
		for i := range selected {
			MOVOU(Mem{Base: reg.R9}.Offset(i*16), candidate[i])
			PXOR(selected[i], candidate[i])
			PAND(reg.X4, candidate[i])
			PXOR(candidate[i], selected[i])
		}
	}

	for i := range selected {
		MOVOU(selected[i], Mem{Base: reg.R8}.Offset(i*16))
	}
	RET()
	Generate()
}
