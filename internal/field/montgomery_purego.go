//go:build !arm64 || !secp256k1_asm

package field

import fiat "github.com/islishude/secp256k1/internal/fiat/basefield"

func addMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Add(out, x, y)
}

func subMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Sub(out, x, y)
}

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Mul(out, x, y)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	fiat.Square(out, x)
}
