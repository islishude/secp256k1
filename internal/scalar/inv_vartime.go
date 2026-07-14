package scalar

import "math/bits"

var orderWords = [4]uint64{orderLimb3, orderLimb2, orderLimb1, orderLimb0}

// InvVartime computes z = 1/x mod n with a variable-time binary extended GCD.
// For x = 0, InvVartime returns 0, matching Inv.
//
// This function is only suitable for public inputs such as an ECDSA signature
// scalar during verification or recovery. Secret scalars must use Inv.
func (z *Element) InvVartime(x *Element) *Element {
	words := x.Words()
	var inverse [4]uint64
	invVartimeWords(&inverse, &words)
	return z.SetWords(inverse)
}

func invVartimeWordsGo(input [4]uint64) [4]uint64 {
	u := input
	if wordsZeroVartime(u) {
		return [4]uint64{}
	}
	v := orderWords
	x1 := [4]uint64{1}
	var x2 [4]uint64

	for !wordsOneVartime(u) && !wordsOneVartime(v) {
		for u[0]&1 == 0 {
			shrWordsOneVartime(&u)
			halfModOrderVartime(&x1)
		}
		for v[0]&1 == 0 {
			shrWordsOneVartime(&v)
			halfModOrderVartime(&x2)
		}

		if cmpWordsVartime(u, v) >= 0 {
			subWordsVartime(&u, v)
			x1 = subModOrderVartime(x1, x2)
		} else {
			subWordsVartime(&v, u)
			x2 = subModOrderVartime(x2, x1)
		}
	}

	if wordsOneVartime(u) {
		return x1
	}
	return x2
}

func wordsZeroVartime(x [4]uint64) bool {
	return x[0]|x[1]|x[2]|x[3] == 0
}

func wordsOneVartime(x [4]uint64) bool {
	return x[0] == 1 && x[1]|x[2]|x[3] == 0
}

func cmpWordsVartime(x, y [4]uint64) int {
	for i := len(x) - 1; i >= 0; i-- {
		if x[i] < y[i] {
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
	}
	return 0
}

func shrWordsOneVartime(x *[4]uint64) {
	x[0] = x[0]>>1 | x[1]<<63
	x[1] = x[1]>>1 | x[2]<<63
	x[2] = x[2]>>1 | x[3]<<63
	x[3] >>= 1
}

func halfModOrderVartime(x *[4]uint64) {
	if x[0]&1 == 0 {
		shrWordsOneVartime(x)
		return
	}

	var carry uint64
	x[0], carry = bits.Add64(x[0], orderWords[0], 0)
	x[1], carry = bits.Add64(x[1], orderWords[1], carry)
	x[2], carry = bits.Add64(x[2], orderWords[2], carry)
	x[3], carry = bits.Add64(x[3], orderWords[3], carry)
	x[0] = x[0]>>1 | x[1]<<63
	x[1] = x[1]>>1 | x[2]<<63
	x[2] = x[2]>>1 | x[3]<<63
	x[3] = x[3]>>1 | carry<<63
}

func subWordsVartime(x *[4]uint64, y [4]uint64) {
	var borrow uint64
	x[0], borrow = bits.Sub64(x[0], y[0], 0)
	x[1], borrow = bits.Sub64(x[1], y[1], borrow)
	x[2], borrow = bits.Sub64(x[2], y[2], borrow)
	x[3], _ = bits.Sub64(x[3], y[3], borrow)
}

func subModOrderVartime(x, y [4]uint64) [4]uint64 {
	var out [4]uint64
	var borrow uint64
	out[0], borrow = bits.Sub64(x[0], y[0], 0)
	out[1], borrow = bits.Sub64(x[1], y[1], borrow)
	out[2], borrow = bits.Sub64(x[2], y[2], borrow)
	out[3], borrow = bits.Sub64(x[3], y[3], borrow)
	if borrow == 0 {
		return out
	}

	var carry uint64
	out[0], carry = bits.Add64(out[0], orderWords[0], 0)
	out[1], carry = bits.Add64(out[1], orderWords[1], carry)
	out[2], carry = bits.Add64(out[2], orderWords[2], carry)
	out[3], _ = bits.Add64(out[3], orderWords[3], carry)
	return out
}
