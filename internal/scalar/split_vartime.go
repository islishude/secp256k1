package scalar

import "math/bits"

var (
	endoA1Words = [3]uint64{
		0xe86c90e49284eb15,
		0x3086d221a7d46bcd,
	}
	endoA2Words = [3]uint64{
		0x57c1108d9d44cfd8,
		0x14ca50f7a8e2f3f6,
		0x0000000000000001,
	}
	endoB1AbsWords = [3]uint64{
		0x6f547fa90abfe4c3,
		0xe4437ed6010e8828,
	}
	endoB2Words = endoA1Words
)

// SplitScalar is the signed magnitude of one half of a GLV scalar split.
// Words are little-endian and Bits is the significant bit length of Words.
type SplitScalar struct {
	Words [3]uint64
	Neg   bool
	Bits  uint8
}

// SplitEndomorphismSignedVartime converts the existing GLV split to compact
// signed magnitudes. It is intended only for public-input scalar multiplication.
func SplitEndomorphismSignedVartime(k *Element) (SplitScalar, SplitScalar) {
	k1, k2 := SplitEndomorphism(k)
	return splitScalarFromElementVartime(&k1), splitScalarFromElementVartime(&k2)
}

// SplitEndomorphismVartimeWords returns compact signed k1 and k2 such that
// k = k1 + k2*lambda mod n. It performs the split directly on canonical,
// non-Montgomery limbs and is only suitable for public inputs.
func SplitEndomorphismVartimeWords(k [4]uint64) (SplitScalar, SplitScalar) {
	if !LessThanOrderWords(k) {
		panic("scalar: non-canonical GLV input")
	}

	c1 := mul512Rsh320RoundWords(k, endoZ1Words)
	c2 := mul512Rsh320RoundWords(k, endoZ2Words)

	// k1 = k - c1*a1 - c2*a2.
	c1a1 := mul4x3Words(c1, endoA1Words)
	c2a2 := mul4x3Words(c2, endoA2Words)
	k1Wide := [5]uint64{k[0], k[1], k[2], k[3]}
	subWideWords(&k1Wide, c1a1)
	subWideWords(&k1Wide, c2a2)

	// k2 = c1*(-b1) - c2*b2, where b1 is negative.
	c1B1 := mul4x3Words(c1, endoB1AbsWords)
	c2B2 := mul4x3Words(c2, endoB2Words)
	k2Wide := [5]uint64{c1B1[0], c1B1[1], c1B1[2], c1B1[3], c1B1[4]}
	if c1B1[5]|c1B1[6] != 0 {
		panic("scalar: GLV product overflow")
	}
	subWideWords(&k2Wide, c2B2)

	return splitScalarFromTwosComplement(k1Wide), splitScalarFromTwosComplement(k2Wide)
}

func splitScalarFromElementVartime(x *Element) SplitScalar {
	words := x.Words()
	neg := IsHighWords(words)
	if neg {
		words = NegWords(words)
	}
	if words[3] != 0 {
		panic("scalar: GLV half exceeds compact representation")
	}
	return newSplitScalar([3]uint64{words[0], words[1], words[2]}, neg)
}

func splitScalarFromTwosComplement(words [5]uint64) SplitScalar {
	neg := words[4]>>63 == 1
	if neg {
		var carry uint64 = 1
		for i := range words {
			words[i], carry = bits.Add64(^words[i], 0, carry)
		}
	}
	if words[3]|words[4] != 0 {
		panic("scalar: GLV half exceeds compact representation")
	}
	return newSplitScalar([3]uint64{words[0], words[1], words[2]}, neg)
}

func newSplitScalar(words [3]uint64, neg bool) SplitScalar {
	bitLen := 0
	for i := len(words) - 1; i >= 0; i-- {
		if words[i] != 0 {
			bitLen = i*64 + bits.Len64(words[i])
			break
		}
	}
	if bitLen > 130 {
		panic("scalar: GLV half exceeds 130 bits")
	}
	if bitLen == 0 {
		neg = false
	}
	return SplitScalar{Words: words, Neg: neg, Bits: uint8(bitLen)}
}

func mul4x3Words(x [4]uint64, y [3]uint64) [7]uint64 {
	var out [7]uint64
	for i := range x {
		var carry uint64
		for j := range y {
			k := i + j
			hi, lo := bits.Mul64(x[i], y[j])
			var c uint64
			lo, c = bits.Add64(lo, out[k], 0)
			hi, _ = bits.Add64(hi, 0, c)
			lo, c = bits.Add64(lo, carry, 0)
			hi, _ = bits.Add64(hi, 0, c)
			out[k] = lo
			carry = hi
		}
		k := i + len(y)
		var c uint64
		out[k], c = bits.Add64(out[k], carry, 0)
		for c != 0 {
			k++
			out[k], c = bits.Add64(out[k], 0, c)
		}
	}
	return out
}

func subWideWords(x *[5]uint64, y [7]uint64) {
	if y[5]|y[6] != 0 {
		panic("scalar: GLV product overflow")
	}
	var borrow uint64
	for i := range x {
		x[i], borrow = bits.Sub64(x[i], y[i], borrow)
	}
}
