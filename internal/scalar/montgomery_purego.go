//go:build (!arm64 && !amd64) || !secp256k1_asm

package scalar

import fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"

func squareMontgomeryN(out, x *fiat.MontgomeryDomainFieldElement, n uint64) {
	*out = *x
	for range n {
		fiat.Square(out, out)
	}
}

func invVartimeWords(out, x *[4]uint64) {
	*out = invVartimeWordsGo(*x)
}
