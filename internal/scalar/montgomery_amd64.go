//go:build amd64 && secp256k1_asm

package scalar

import (
	"github.com/islishude/secp256k1/internal/cpufeat"
	fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"
)

type amd64ScalarKernelSet struct {
	mul, square, squareN, invVartime bool
}

// amd64ScalarKernels is fixed from public CPU state during package
// initialization. It remains mutable for fallback tests and the benchmark-only
// selector; production builds never consult process environment variables.
var amd64ScalarKernels = amd64ScalarKernelSet{
	mul:        cpufeat.HasADXAndBMI2,
	square:     cpufeat.HasADXAndBMI2,
	squareN:    cpufeat.HasADXAndBMI2,
	invVartime: cpufeat.HasADXAndBMI2,
}

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	if amd64ScalarKernels.mul {
		mulMontgomeryADXAsm(scalarMontgomeryWords(out), scalarMontgomeryWords(x), scalarMontgomeryWords(y))
		return
	}
	fiat.Mul(out, x, y)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	if amd64ScalarKernels.square {
		squareMontgomeryADXAsm(scalarMontgomeryWords(out), scalarMontgomeryWords(x))
		return
	}
	fiat.Square(out, x)
}

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	if amd64ScalarKernels.squareN {
		squareMontgomeryNADXAsm(scalarMontgomeryWords(out), scalarMontgomeryWords(x), n)
		return
	}
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}

func scalarMontgomeryWords(x *fiat.MontgomeryDomainFieldElement) *[4]uint64 {
	return (*[4]uint64)(x)
}

func invVartimeWords(out, x *[4]uint64) {
	if amd64ScalarKernels.invVartime {
		invVartimeWordsADXAsm(out, x)
		return
	}
	*out = invVartimeWordsGo(*x)
}
