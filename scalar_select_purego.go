//go:build (!arm64 && !amd64) || !secp256k1_asm || (amd64 && secp256k1_amd64_w5_bench)

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

		selected := generatorAffineTableW5Words[i][0]
		for j := 2; j <= len(generatorAffineTableW5Words[i]); j++ {
			mask := uint64(0) - equalByte(byte(magnitude), byte(j))
			for limb := range selected {
				selected[limb] = (selected[limb] &^ mask) | (generatorAffineTableW5Words[i][j-1][limb] & mask)
			}
		}
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

func selectGeneratorW5(out *[8]uint64, table *[baseTableSize][8]uint64, magnitude uint64) {
	selected := table[0]
	for j := 2; j <= len(table); j++ {
		mask := uint64(0) - equalByte(byte(magnitude), byte(j))
		for limb := range selected {
			selected[limb] = (selected[limb] &^ mask) | (table[j-1][limb] & mask)
		}
	}
	*out = selected
}
