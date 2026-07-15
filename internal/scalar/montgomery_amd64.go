//go:build amd64 && secp256k1_asm

package scalar

import (
	"github.com/islishude/secp256k1/internal/cpufeat"
	fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"
)

type amd64ScalarKernelSet struct {
	invVartime bool
}

// amd64ScalarKernels is fixed from public CPU state during package
// initialization. It remains mutable for fallback tests and the benchmark-only
// selector; production builds never consult process environment variables.
var amd64ScalarKernels = amd64ScalarKernelSet{
	invVartime: cpufeat.HasADXAndBMI2,
}

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}

func invVartimeWords(out, x *[4]uint64) {
	if amd64ScalarKernels.invVartime {
		invVartimeWordsADXAsm(out, x)
		return
	}
	*out = invVartimeWordsGo(*x)
}
