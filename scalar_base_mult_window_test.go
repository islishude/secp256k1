package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	baseWindowW6    = 6
	baseWindowsW6   = (256 + baseWindowW6 - 1) / baseWindowW6
	baseTableSizeW6 = 1 << (baseWindowW6 - 1)
)

func TestScalarBaseMultProjectiveW5MatchesW4(t *testing.T) {
	w4Table := newGeneratorAffineTableW4()
	w5Table := newGeneratorAffineTableW5()
	w6Table := newGeneratorAffineTableW6ForTest()
	inputs := [][32]byte{
		must32("00"),
		must32("01"),
		must32("02"),
		must32("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
		must32("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
	}
	state := uint64(0x082efa98ec4e6c89)
	for range 1_000 {
		var words [4]uint64
		for i := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[i] = state
		}
		var k scalar.Element
		k.SetWordsModOrder(words)
		inputs = append(inputs, k.Bytes())
	}

	for _, input := range inputs {
		var k scalar.Element
		if !k.SetBytes(&input) {
			t.Fatalf("SetBytes(%x) failed", input)
		}
		got := scalarBaseMultProjective(&k)
		gotW5Affine := scalarBaseMultProjectiveW5Affine(&k, &w5Table)
		want := scalarBaseMultProjectiveW4(&k, &w4Table)
		gotW6 := scalarBaseMultProjectiveW6ForTest(&k, &w6Table)
		gotX, gotY, gotOK := got.affine()
		wantX, wantY, wantOK := want.affine()
		if gotOK != wantOK || gotOK && (!gotX.Equal(&wantX) || !gotY.Equal(&wantY)) {
			t.Fatalf("W5 mismatch for %x", input)
		}
		gotW5AffineX, gotW5AffineY, gotW5AffineOK := gotW5Affine.affine()
		if gotW5AffineOK != wantOK || gotW5AffineOK && (!gotW5AffineX.Equal(&wantX) || !gotW5AffineY.Equal(&wantY)) {
			t.Fatalf("affine-table W5 mismatch for %x", input)
		}
		gotW6X, gotW6Y, gotW6OK := gotW6.affine()
		if gotW6OK != wantOK || gotW6OK && (!gotW6X.Equal(&wantX) || !gotW6Y.Equal(&wantY)) {
			t.Fatalf("W6 mismatch for %x", input)
		}
	}
}

func TestFixedWindowDigit(t *testing.T) {
	words := [4]uint64{
		0xfedcba9876543210,
		0x0123456789abcdef,
		0x0f1e2d3c4b5a6978,
		0x8877665544332211,
	}
	for window := uint(4); window <= 6; window++ {
		for i := uint(0); i < (256+window-1)/window; i++ {
			got := fixedWindowDigit(&words, i, window)
			var want uint64
			for j := uint(0); j < window && i*window+j < 256; j++ {
				bit := i*window + j
				want |= (words[bit/64] >> (bit % 64)) & 1 << j
			}
			if got != byte(want) {
				t.Fatalf("window %d digit %d = %d, want %d", window, i, got, want)
			}
		}
	}
}

func TestSelectGeneratorW5(t *testing.T) {
	for window := range generatorAffineTableW5Words {
		table := &generatorAffineTableW5Words[window]
		for magnitude := uint64(0); magnitude <= baseTableSize; magnitude++ {
			var got [8]uint64
			selectGeneratorW5(&got, table, magnitude)
			wantIndex := max(magnitude, 1) - 1
			if want := table[wantIndex]; got != want {
				t.Fatalf("window %d magnitude %d selected %x, want %x", window, magnitude, got, want)
			}
		}
	}
}

func newGeneratorAffineTableW6ForTest() [baseWindowsW6][baseTableSizeW6]affinePoint {
	var table [baseWindowsW6][baseTableSizeW6]affinePoint
	base := generator
	for i := range table {
		var points [baseTableSizeW6]point
		points[0].set(&base)
		for j := 1; j < len(points); j++ {
			points[j].add(&points[j-1], &base)
		}
		affine := batchNormalize32ForTest(points)
		for j := range table[i] {
			table[i][j] = affine[j]
		}
		for range baseWindowW6 {
			base.double(&base)
		}
	}
	return table
}

func batchNormalize32ForTest(points [32]point) [32]affinePoint {
	var prefixes [32]field.Element
	var acc field.Element
	acc.SetOne()
	for i := range points {
		prefixes[i].Set(&acc)
		acc.Mul(&acc, &points[i].z)
	}
	var accInv field.Element
	accInv.Inv(&acc)
	var out [32]affinePoint
	for i := len(points) - 1; i >= 0; i-- {
		var zInv, z2, z3 field.Element
		zInv.Mul(&accInv, &prefixes[i])
		accInv.Mul(&accInv, &points[i].z)
		z2.Square(&zInv)
		z3.Mul(&z2, &zInv)
		out[i].x.Mul(&points[i].x, &z2)
		out[i].y.Mul(&points[i].y, &z3)
	}
	return out
}

func scalarBaseMultProjectiveW6ForTest(k *scalar.Element, table *[baseWindowsW6][baseTableSizeW6]affinePoint) projectivePoint {
	words := k.Words()
	defer clear(words[:])
	var r projectivePoint
	r.setInfinity()
	var carry uint64
	for i := range table {
		value := uint64(fixedWindowDigit(&words, uint(i), baseWindowW6)) + carry
		negative := (value + (1 << (baseWindowW6 - 1))) >> baseWindowW6
		negativeMask := uint64(0) - negative
		digitBits := value - negative*(1<<baseWindowW6)
		magnitude := (digitBits ^ negativeMask) + negative
		selected := table[i][0]
		for j := 2; j <= len(table[i]); j++ {
			selected.selectPoint(&selected, &table[i][j-1], equalByte(byte(magnitude), byte(j)))
		}
		var negY field.Element
		negY.Neg(&selected.y)
		selected.y.Select(&selected.y, &negY, negative)
		var sum projectivePoint
		sum.addCompleteMixed(&r, &selected)
		r.selectPoint(&r, &sum, equalByte(byte(magnitude), 0)^1)
		carry = negative
	}
	return r
}
