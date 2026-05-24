package secp256k1

import (
	"encoding/binary"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	varWNAFWindow       = 8
	varWNAFTableSize    = 1 << (varWNAFWindow - 2)
	generatorWNAFWindow = 8
	generatorWNAFSize   = 1 << (generatorWNAFWindow - 2)
)

func signedWNAF(k *scalar.Element, window int) ([257]int8, int, int8) {
	sign := int8(1)
	kBytes := k.Bytes()
	if scalar.IsHighBytes(&kBytes) {
		var neg scalar.Element
		neg.Neg(k)
		kBytes = neg.Bytes()
		sign = -1
	}
	naf, length := wnaf(&kBytes, window)
	return naf, length, sign
}

func addVariableWNAFPoint(r *point, table *[varWNAFTableSize]affinePoint, digit int8) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffine(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffine(r, &entry.x, &y)
}

func addGeneratorWNAFPoint(r *point, table *[generatorWNAFSize]affinePoint, digit int8) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffine(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffine(r, &entry.x, &y)
}

func wnaf(k *[32]byte, window int) ([257]int8, int) {
	words := scalarWords(k)
	var out [257]int8
	length := 0
	for i := 0; i < len(out) && !isZeroWords(&words); i++ {
		if words[0]&1 == 1 {
			digit := int(words[0] & ((1 << window) - 1))
			if digit > 1<<(window-1) {
				digit -= 1 << window
			}
			out[i] = int8(digit)
			if digit > 0 {
				subSmall(&words, uint64(digit))
			} else {
				addSmall(&words, uint64(-digit))
			}
		}
		shr1(&words)
		length = i + 1
	}
	return out, length
}

func scalarWords(k *[32]byte) [4]uint64 {
	return [4]uint64{
		binary.BigEndian.Uint64(k[24:32]),
		binary.BigEndian.Uint64(k[16:24]),
		binary.BigEndian.Uint64(k[8:16]),
		binary.BigEndian.Uint64(k[0:8]),
	}
}

func isZeroWords(words *[4]uint64) bool {
	return words[0]|words[1]|words[2]|words[3] == 0
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
