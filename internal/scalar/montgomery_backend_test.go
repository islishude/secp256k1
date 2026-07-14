package scalar

import (
	"math/rand"
	"testing"

	fiat "github.com/islishude/secp256k1/internal/fiat/scalarfield"
)

func TestScalarMontgomeryBackendMatchesFiat(t *testing.T) {
	edges := scalarMontgomeryBackendEdges()
	for _, x := range edges {
		for _, y := range edges {
			checkScalarMontgomeryBackend(t, x, y)
		}
	}

	rng := rand.New(rand.NewSource(2))
	for range 100_000 {
		x := randomCanonicalScalarWords(rng)
		y := randomCanonicalScalarWords(rng)
		checkScalarMontgomeryBackend(t, x, y)
	}
}

func TestScalarMontgomerySquareNCountsMatchFiat(t *testing.T) {
	counts := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 26, 60, 64}
	for _, xWords := range scalarMontgomeryBackendEdges() {
		var x Element
		x.SetWords(xWords)
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
			assertCanonicalScalarMontgomery(t, &got)

			got = x.x
			squareMontgomeryN(&got, &got, count)
			if got != want {
				t.Fatalf("square n alias mismatch for x=%x n=%d", xWords, count)
			}
		}
	}
}

func scalarMontgomeryBackendEdges() [][4]uint64 {
	return [][4]uint64{
		{},
		{1},
		{2},
		{^uint64(0)},
		{0, ^uint64(0)},
		{0, 0, ^uint64(0)},
		{orderLimb3 - 1, orderLimb2, orderLimb1, orderLimb0},
		{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	}
}

func checkScalarMontgomeryBackend(t *testing.T, xWords, yWords [4]uint64) {
	t.Helper()
	var x, y Element
	x.SetWordsModOrder(xWords)
	y.SetWordsModOrder(yWords)

	var wantMul, wantSquare fiat.MontgomeryDomainFieldElement
	fiat.Mul(&wantMul, &x.x, &y.x)
	fiat.Square(&wantSquare, &x.x)
	squareCount := uint64(1 + xWords[0]&7)
	wantSquareN := x.x
	for range squareCount {
		fiat.Square(&wantSquareN, &wantSquareN)
	}

	var got fiat.MontgomeryDomainFieldElement
	mulMontgomery(&got, &x.x, &y.x)
	if got != wantMul {
		t.Fatalf("mul mismatch for x=%x y=%x: got %x want %x", xWords, yWords, got, wantMul)
	}
	assertCanonicalScalarMontgomery(t, &got)

	squareMontgomery(&got, &x.x)
	if got != wantSquare {
		t.Fatalf("square mismatch for x=%x: got %x want %x", xWords, got, wantSquare)
	}
	assertCanonicalScalarMontgomery(t, &got)

	squareMontgomeryN(&got, &x.x, squareCount)
	if got != wantSquareN {
		t.Fatalf("square n mismatch for x=%x n=%d: got %x want %x", xWords, squareCount, got, wantSquareN)
	}
	assertCanonicalScalarMontgomery(t, &got)

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
	mulMontgomery(&got, &got, &got)
	if got != wantSquare {
		t.Fatalf("mul double alias mismatch for x=%x", xWords)
	}
	got = x.x
	squareMontgomery(&got, &got)
	if got != wantSquare {
		t.Fatalf("square alias mismatch for x=%x", xWords)
	}
	got = x.x
	squareMontgomeryN(&got, &got, squareCount)
	if got != wantSquareN {
		t.Fatalf("square n alias mismatch for x=%x n=%d", xWords, squareCount)
	}
}

func assertCanonicalScalarMontgomery(t *testing.T, x *fiat.MontgomeryDomainFieldElement) {
	t.Helper()
	var words fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&words, x)
	if !LessThanOrderWords([4]uint64(words)) {
		t.Fatalf("backend returned non-canonical value %x", words)
	}
}

func randomCanonicalScalarWords(rng *rand.Rand) [4]uint64 {
	for {
		words := [4]uint64{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		if LessThanOrderWords(words) {
			return words
		}
	}
}
