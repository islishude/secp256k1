//go:build arm64 && secp256k1_asm

package scalar

import fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Mul(out, x, y)
}

func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement) {
	fiat.Square(out, x)
}

func invVartimeWords(out, x *[4]uint64) {
	*out = invVartimeWordsGo(*x)
}

// squareMontgomeryN performs a public, fixed number of squarings. Its loop
// count is determined by the static inversion addition chain, not secret data.
// It supports out == x and n == 0.
//
//go:noescape
func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64)
