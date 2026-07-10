package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/field"
)

func TestCurveYFromXMatchesCheckedInvariant(t *testing.T) {
	state := uint64(0x452821e638d01377)
	for range 2_000 {
		var words [4]uint64
		for i := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[i] = state
		}
		words[3] %= field.ModuleLimb0
		if !field.LessThanModulusWords(words) {
			words[0] = 0
		}
		var x field.Element
		x.SetNonMontgomeryWords(words)
		for _, odd := range []bool{false, true} {
			got, gotOK := curveYFromX(&x, odd)
			want, wantOK := curveYFromXCheckedForTest(&x, odd)
			if gotOK != wantOK || gotOK && !got.Equal(&want) {
				t.Fatalf("curveYFromX mismatch for x=%x odd=%v", x.Bytes(), odd)
			}
		}
	}
}

func curveYFromXCheckedForTest(x *field.Element, wantOdd bool) (field.Element, bool) {
	var y, rhs, x2 field.Element
	x2.Square(x)
	rhs.Mul(&x2, x)
	rhs.Add(&rhs, &secp256k1BElement)
	if !y.Sqrt(&rhs) {
		return field.Element{}, false
	}
	if y.IsOdd() != wantOdd {
		y.Neg(&y)
	}
	if !isOnCurve(x, &y) {
		return field.Element{}, false
	}
	return y, true
}
