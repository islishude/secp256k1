//go:build arm64 && secp256k1_asm

package field

import fiat "github.com/islishude/secp256k1/internal/fiat/basefield"

// The ARM64 backend uses the same saturated 4x64 Montgomery representation as
// the generated fiat backend. Its implementations are fixed instruction
// sequences with no secret-dependent branches or loads.
//
//go:noescape
func addMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement)

func subMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Sub(out, x, y)
}

func mulMontgomery(out, x, y *fiat.MontgomeryDomainFieldElement) {
	fiat.Mul(out, x, y)
}

//go:noescape
func squareMontgomery(out, x *fiat.MontgomeryDomainFieldElement)
