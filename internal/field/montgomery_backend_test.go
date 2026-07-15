package field

import (
	"math/rand"
	"testing"

	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

func TestMontgomeryBackendMatchesFiat(t *testing.T) {
	edges := montgomeryBackendEdges()
	for _, x := range edges {
		for _, y := range edges {
			checkMontgomeryBackend(t, x, y)
		}
	}

	rng := rand.New(rand.NewSource(1))
	for range 100_000 {
		x := randomCanonicalFieldWords(rng)
		y := randomCanonicalFieldWords(rng)
		checkMontgomeryBackend(t, x, y)
	}
}

func TestMontgomerySquareNCountsMatchFiat(t *testing.T) {
	counts := []uint64{0, 1, 2, 3, 4, 7, 22, 46, 64, 110}
	for _, xWords := range montgomeryBackendEdges() {
		var x Element
		x.SetNonMontgomeryWords(xWords)
		for _, count := range counts {
			want := x.x
			for range count {
				fiat.Square(&want, &want)
			}

			var got fiat.MontgomeryDomainFieldElement
			squareMontgomeryN(&got, &x.x, count)
			if got != want {
				t.Fatalf("square n mismatch for x=%x n=%d: got %x want %x", xWords, count, got, want)
			}
			assertCanonicalMontgomery(t, &got)

			got = x.x
			squareMontgomeryN(&got, &got, count)
			if got != want {
				t.Fatalf("square n alias mismatch for x=%x n=%d", xWords, count)
			}
		}
	}
}

func montgomeryBackendEdges() [][4]uint64 {
	return [][4]uint64{
		{},
		{1},
		{2},
		{^uint64(0)},
		{0, ^uint64(0)},
		{0, 0, ^uint64(0)},
		{0, 0, 0, ^uint64(0) - 1},
		{0xfffffffefffffc2e, ^uint64(0), ^uint64(0), ^uint64(0)},
		{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	}
}

func checkMontgomeryBackend(t *testing.T, xWords, yWords [4]uint64) {
	t.Helper()
	var x, y Element
	x.SetNonMontgomeryWords(xWords)
	y.SetNonMontgomeryWords(yWords)

	var wantAdd, wantSub, wantMul, wantMulByB3, wantSquare fiat.MontgomeryDomainFieldElement
	fiat.Add(&wantAdd, &x.x, &y.x)
	fiat.Sub(&wantSub, &x.x, &y.x)
	fiat.Mul(&wantMul, &x.x, &y.x)
	fiat.Mul(&wantMulByB3, &x.x, &b3Montgomery)
	fiat.Square(&wantSquare, &x.x)
	squareCount := uint64(1 + xWords[0]&3)
	wantSquareN := x.x
	for range squareCount {
		fiat.Square(&wantSquareN, &wantSquareN)
	}

	var got fiat.MontgomeryDomainFieldElement
	addMontgomery(&got, &x.x, &y.x)
	if got != wantAdd {
		t.Fatalf("add mismatch for x=%x y=%x: got %x want %x", xWords, yWords, got, wantAdd)
	}
	assertCanonicalMontgomery(t, &got)
	subMontgomery(&got, &x.x, &y.x)
	if got != wantSub {
		t.Fatalf("sub mismatch for x=%x y=%x: got %x want %x", xWords, yWords, got, wantSub)
	}
	assertCanonicalMontgomery(t, &got)
	mulMontgomery(&got, &x.x, &y.x)
	if got != wantMul {
		t.Fatalf("mul mismatch for x=%x y=%x: got %x want %x", xWords, yWords, got, wantMul)
	}
	assertCanonicalMontgomery(t, &got)
	mulByB3Montgomery(&got, &x.x)
	if got != wantMulByB3 {
		t.Fatalf("mul by 21 mismatch for x=%x: got %x want %x", xWords, got, wantMulByB3)
	}
	assertCanonicalMontgomery(t, &got)
	squareMontgomery(&got, &x.x)
	if got != wantSquare {
		t.Fatalf("square mismatch for x=%x: got %x want %x", xWords, got, wantSquare)
	}
	assertCanonicalMontgomery(t, &got)
	squareMontgomeryN(&got, &x.x, squareCount)
	if got != wantSquareN {
		t.Fatalf("square n mismatch for x=%x n=%d: got %x want %x", xWords, squareCount, got, wantSquareN)
	}
	assertCanonicalMontgomery(t, &got)

	got = x.x
	addMontgomery(&got, &got, &y.x)
	if got != wantAdd {
		t.Fatalf("add x alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = y.x
	addMontgomery(&got, &x.x, &got)
	if got != wantAdd {
		t.Fatalf("add y alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = x.x
	subMontgomery(&got, &got, &y.x)
	if got != wantSub {
		t.Fatalf("sub x alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = y.x
	subMontgomery(&got, &x.x, &got)
	if got != wantSub {
		t.Fatalf("sub y alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = x.x
	mulMontgomery(&got, &got, &y.x)
	if got != wantMul {
		t.Fatalf("mul x alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = y.x
	mulMontgomery(&got, &x.x, &got)
	if got != wantMul {
		t.Fatalf("mul y alias mismatch for x=%x y=%x", xWords, yWords)
	}
	got = x.x
	squareMontgomery(&got, &got)
	if got != wantSquare {
		t.Fatalf("square alias mismatch for x=%x", xWords)
	}
	got = x.x
	mulMontgomery(&got, &got, &got)
	if got != wantSquare {
		t.Fatalf("mul double alias mismatch for x=%x", xWords)
	}
	got = x.x
	mulByB3Montgomery(&got, &got)
	if got != wantMulByB3 {
		t.Fatalf("mul by 21 alias mismatch for x=%x", xWords)
	}
	got = x.x
	squareMontgomeryN(&got, &got, squareCount)
	if got != wantSquareN {
		t.Fatalf("square n alias mismatch for x=%x n=%d", xWords, squareCount)
	}
}

func assertCanonicalMontgomery(t *testing.T, x *fiat.MontgomeryDomainFieldElement) {
	t.Helper()
	var words fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&words, x)
	if !LessThanModulusWords([4]uint64(words)) {
		t.Fatalf("backend returned non-canonical value %x", words)
	}
}

func randomCanonicalFieldWords(rng *rand.Rand) [4]uint64 {
	for {
		words := [4]uint64{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		if LessThanModulusWords(words) {
			return words
		}
	}
}
