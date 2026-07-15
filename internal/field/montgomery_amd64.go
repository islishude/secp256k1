//go:build amd64 && secp256k1_asm

package field

import (
	"github.com/islishude/secp256k1/internal/cpufeat"
	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

type amd64KernelSet struct {
	add, sub, mul, square bool
}

// amd64Kernels is fixed from public CPU state during package initialization.
// It remains a variable so package tests can exercise fallback and benchmark
// each retained kernel independently without changing production routing.
var amd64Kernels = amd64KernelSet{
	add:    cpufeat.HasADXAndBMI2,
	sub:    cpufeat.HasADXAndBMI2,
	mul:    cpufeat.HasADXAndBMI2,
	square: cpufeat.HasADXAndBMI2,
}

func addMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	if amd64Kernels.add {
		addMontgomeryADXAsm(fieldWords(out), fieldWords(x), fieldWords(y))
		return
	}
	fiat.Add(out, x, y)
}

func subMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	if amd64Kernels.sub {
		subMontgomeryADXAsm(fieldWords(out), fieldWords(x), fieldWords(y))
		return
	}
	fiat.Sub(out, x, y)
}

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	if amd64Kernels.mul {
		mulMontgomeryADXAsm(fieldWords(out), fieldWords(x), fieldWords(y))
		return
	}
	fiat.Mul(out, x, y)
}

func mulByB3Montgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	fiat.Mul(out, x, &b3Montgomery)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	if amd64Kernels.square {
		squareMontgomeryADXAsm(fieldWords(out), fieldWords(x))
		return
	}
	fiat.Square(out, x)
}

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}

func fieldWords(x *fiat.MontgomeryDomainFieldElement) *[4]uint64 {
	return (*[4]uint64)(x)
}
