//go:build amd64 && secp256k1_asm

package field

import (
	"github.com/islishude/secp256k1/internal/cpufeat"
	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

// useADXAndBMI2 is fixed from public CPU state during package initialization.
// It remains a variable so the fallback route can be exercised in package
// tests without pretending GOAMD64 implies ADX support.
var useADXAndBMI2 = cpufeat.HasADXAndBMI2

func addMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Add(out, x, y)
}

func subMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Sub(out, x, y)
}

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	if useADXAndBMI2 {
		mulMontgomeryADXAsm(fieldWords(out), fieldWords(x), fieldWords(y))
		return
	}
	fiat.Mul(out, x, y)
}

func mulByB3Montgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	if useADXAndBMI2 {
		mulByB3MontgomeryADXAsm(fieldWords(out), fieldWords(x))
		return
	}
	fiat.Mul(out, x, &b3Montgomery)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	if useADXAndBMI2 {
		squareMontgomeryADXAsm(fieldWords(out), fieldWords(x))
		return
	}
	fiat.Square(out, x)
}

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	if useADXAndBMI2 {
		squareMontgomeryNADXAsm(fieldWords(out), fieldWords(x), n)
		return
	}
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}

func fieldWords(x *fiat.MontgomeryDomainFieldElement) *[4]uint64 {
	return (*[4]uint64)(x)
}
