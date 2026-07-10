package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	varWNAFWindow       = 8
	varWNAFTableSize    = 1 << (varWNAFWindow - 2)
	generatorWNAFWindow = 13
	generatorWNAFSize   = 1 << (generatorWNAFWindow - 2)
	halfWNAFSize        = 131
)

func signedWNAFVartime(k *scalar.Element, window int) ([257]int16, int, int16) {
	sign := int16(1)
	words := k.Words()
	if scalar.IsHighWords(words) {
		words = scalar.NegWords(words)
		sign = -1
	}
	naf, length := wnafVartime(&words, window)
	return naf, length, sign
}

func signedWNAFHalfVartime(out *[halfWNAFSize]int16, k scalar.SplitScalar, window int) int {
	words := k.Words
	length := 0
	for i := 0; i < len(out) && !isZeroHalfWords(&words); i++ {
		if words[0]&1 == 1 {
			digit := int(words[0] & ((1 << window) - 1))
			if digit > 1<<(window-1) {
				digit -= 1 << window
			}
			if k.Neg {
				(*out)[i] = -int16(digit)
			} else {
				(*out)[i] = int16(digit)
			}
			if digit > 0 {
				subSmallHalf(&words, uint64(digit))
			} else {
				addSmallHalf(&words, uint64(-digit))
			}
		}
		shr1Half(&words)
		length = i + 1
	}
	if !isZeroHalfWords(&words) {
		panic("secp256k1: half-width wNAF overflow")
	}
	return length
}

func addVariableWNAFPointVartime(r *point, table *[varWNAFTableSize]affinePoint, digit int16) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffineWNAFVartime(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffineWNAFVartime(r, &entry.x, &y)
}

func addGeneratorWNAFPointVartime(r *point, table *[generatorWNAFSize]affinePoint, digit int16) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffineWNAFVartime(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffineWNAFVartime(r, &entry.x, &y)
}

func wnafVartime(words *[4]uint64, window int) ([257]int16, int) {
	var out [257]int16
	length := 0
	for i := 0; i < len(out) && !isZeroWords(words); i++ {
		if (*words)[0]&1 == 1 {
			digit := int((*words)[0] & ((1 << window) - 1))
			if digit > 1<<(window-1) {
				digit -= 1 << window
			}
			out[i] = int16(digit)
			if digit > 0 {
				subSmall(words, uint64(digit))
			} else {
				addSmall(words, uint64(-digit))
			}
		}
		shr1(words)
		length = i + 1
	}
	return out, length
}

func isZeroWords(words *[4]uint64) bool {
	return words[0]|words[1]|words[2]|words[3] == 0
}

func isZeroHalfWords(words *[3]uint64) bool {
	return words[0]|words[1]|words[2] == 0
}

func addSmallHalf(words *[3]uint64, v uint64) {
	words[0] += v
	if words[0] >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]++
		if words[i] != 0 {
			return
		}
	}
}

func subSmallHalf(words *[3]uint64, v uint64) {
	old := words[0]
	words[0] -= v
	if old >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]--
		if words[i] != ^uint64(0) {
			return
		}
	}
}

func shr1Half(words *[3]uint64) {
	words[0] = words[0]>>1 | words[1]<<63
	words[1] = words[1]>>1 | words[2]<<63
	words[2] >>= 1
}

func addSmall(words *[4]uint64, v uint64) {
	words[0] += v
	if words[0] >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]++
		if words[i] != 0 {
			return
		}
	}
}

func subSmall(words *[4]uint64, v uint64) {
	old := words[0]
	words[0] -= v
	if old >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]--
		if words[i] != ^uint64(0) {
			return
		}
	}
}

func shr1(words *[4]uint64) {
	for i := range len(words) - 1 {
		words[i] = (words[i] >> 1) | (words[i+1] << 63)
	}
	words[len(words)-1] >>= 1
}

func bitAt(k *[32]byte, i int) byte {
	return (k[i/8] >> uint(7-i%8)) & 1
}

func nibbleAt(k *[32]byte, i int) byte {
	b := k[31-i/2]
	if i%2 == 0 {
		return b & 0x0f
	}
	return b >> 4
}

func equalByte(x, y byte) uint64 {
	v := uint64(x ^ y)
	v |= v >> 4
	v |= v >> 2
	v |= v >> 1
	return (v ^ 1) & 1
}
