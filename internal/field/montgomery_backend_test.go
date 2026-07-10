package field

import (
	"math/rand"
	"testing"

	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

func TestMontgomeryBackendMatchesFiat(t *testing.T) {
	edges := [][4]uint64{
		{},
		{1},
		{2},
		{0xfffffffefffffc2e, ^uint64(0), ^uint64(0), ^uint64(0)},
		{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	}
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

func checkMontgomeryBackend(t *testing.T, xWords, yWords [4]uint64) {
	t.Helper()
	var x, y Element
	x.SetNonMontgomeryWords(xWords)
	y.SetNonMontgomeryWords(yWords)

	var wantAdd, wantSub, wantMul, wantSquare fiat.MontgomeryDomainFieldElement
	fiat.Add(&wantAdd, &x.x, &y.x)
	fiat.Sub(&wantSub, &x.x, &y.x)
	fiat.Mul(&wantMul, &x.x, &y.x)
	fiat.Square(&wantSquare, &x.x)

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
	squareMontgomery(&got, &x.x)
	if got != wantSquare {
		t.Fatalf("square mismatch for x=%x: got %x want %x", xWords, got, wantSquare)
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
