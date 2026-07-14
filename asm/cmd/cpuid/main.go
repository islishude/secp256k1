// Command cpuid generates the AMD64 CPUID helper used by the opt-in backend.
package main

import (
	. "github.com/mmcloughlin/avo/build"
	r "github.com/mmcloughlin/avo/reg"
)

func main() {
	ConstraintExpr("amd64,secp256k1_asm")

	TEXT("cpuid", NOSPLIT, "func(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)")
	Pragma("noescape")
	Doc("cpuid executes CPUID for the requested leaf and subleaf.")
	Load(Param("leaf"), r.EAX)
	Load(Param("subleaf"), r.ECX)
	CPUID()
	Store(r.EAX, Return("eax"))
	Store(r.EBX, Return("ebx"))
	Store(r.ECX, Return("ecx"))
	Store(r.EDX, Return("edx"))
	RET()

	Generate()
}
