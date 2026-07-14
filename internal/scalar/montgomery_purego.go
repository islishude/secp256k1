//go:build (!arm64 && !amd64) || !secp256k1_asm

package scalar

import fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Mul(out, x, y)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	fiat.Square(out, x)
}

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}
