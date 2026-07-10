//go:build arm64 && secp256k1_asm

package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

func scalarBaseMultProjective(k *scalar.Element) projectivePoint {
	words := k.Words()
	defer clear(words[:])
	var r projectivePoint
	r.setInfinity()
	var carry uint64
	for i := range generatorAffineTableW5Words {
		value := uint64(fixedWindowDigit(&words, uint(i), baseWindow)) + carry
		negative := (value + (1 << (baseWindow - 1))) >> baseWindow
		negativeMask := uint64(0) - negative
		digitBits := value - negative*(1<<baseWindow)
		magnitude := (digitBits ^ negativeMask) + negative

		var selected [8]uint64
		selectGeneratorW5(&selected, &generatorAffineTableW5Words[i], magnitude)
		var affine affinePoint
		affine.x.SetMontgomeryWords([4]uint64{selected[0], selected[1], selected[2], selected[3]})
		affine.y.SetMontgomeryWords([4]uint64{selected[4], selected[5], selected[6], selected[7]})
		var negY field.Element
		negY.Neg(&affine.y)
		affine.y.Select(&affine.y, &negY, negative)
		var sum projectivePoint
		if i == 0 {
			sum.setAffine(&affine)
		} else {
			sum.addCompleteMixed(&r, &affine)
		}
		r.selectPoint(&r, &sum, equalByte(byte(magnitude), 0)^1)
		carry = negative
	}
	return r
}

// selectGeneratorW5 scans all sixteen packed points with a fixed NEON
// instruction sequence. magnitude is secret and is never used as an address.
//
//go:noescape
func selectGeneratorW5(out *[8]uint64, table *[baseTableSize][8]uint64, magnitude uint64)
