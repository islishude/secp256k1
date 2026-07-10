package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

func TestAddAffineWNAFVartimeMatchesAddAffine(t *testing.T) {
	var infinity point
	infinity.setInfinity()
	negGenerator := generator
	negGenerator.y.Neg(&negGenerator.y)
	doubledGenerator := generator
	doubledGenerator.double(&doubledGenerator)

	tests := []struct {
		name string
		p    point
		q    point
	}{
		{name: "infinity accumulator", p: infinity, q: generator},
		{name: "same point", p: generator, q: generator},
		{name: "opposite point", p: generator, q: negGenerator},
		{name: "different point", p: doubledGenerator, q: generator},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertAddAffineVartimeMatches(t, &tc.p, &tc.q.x, &tc.q.y)
		})
	}

	state := uint64(0xa4093822299f31d0)
	for range 1_000 {
		var pWords, qWords [4]uint64
		for i := range pWords {
			state = state*6364136223846793005 + 1442695040888963407
			pWords[i] = state
			state = state*6364136223846793005 + 1442695040888963407
			qWords[i] = state
		}
		var ps, qs scalar.Element
		ps.SetWordsModOrder(pWords)
		qs.SetWordsModOrder(qWords)
		if ps.IsZero() {
			ps.SetOne()
		}
		if qs.IsZero() {
			qs.SetOne()
		}
		p := scalarBaseMult(&ps)
		q := scalarBaseMult(&qs)
		qx, qy, ok := q.affine()
		if !ok {
			t.Fatal("random affine point is infinity")
		}
		assertAddAffineVartimeMatches(t, &p, &qx, &qy)
	}
}

func assertAddAffineVartimeMatches(t *testing.T, p *point, x, y *field.Element) {
	t.Helper()
	var got, want point
	got.addAffineWNAFVartime(p, x, y)
	want.addAffine(p, x, y)
	gotX, gotY, gotOK := got.affine()
	wantX, wantY, wantOK := want.affine()
	if gotOK != wantOK || gotOK && (!gotX.Equal(&wantX) || !gotY.Equal(&wantY)) {
		t.Fatalf("mixed-add mismatch: got (%x, %x, %v), want (%x, %x, %v)",
			gotX.Bytes(), gotY.Bytes(), gotOK, wantX.Bytes(), wantY.Bytes(), wantOK)
	}
}

func TestPointDoubleMulMatchesDouble(t *testing.T) {
	var infinity point
	infinity.setInfinity()
	assertPointDoubleMulMatches(t, &infinity)
	assertPointDoubleMulMatches(t, &generator)

	state := uint64(0xbe5466cf34e90c6c)
	for range 2_000 {
		var words [4]uint64
		for i := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[i] = state
		}
		var k scalar.Element
		k.SetWordsModOrder(words)
		if k.IsZero() {
			k.SetOne()
		}
		p := scalarBaseMult(&k)
		assertPointDoubleMulMatches(t, &p)
	}
}

func assertPointDoubleMulMatches(t *testing.T, p *point) {
	t.Helper()
	var got, want point
	got.double(p)
	want.doubleSquare(p)
	gotX, gotY, gotOK := got.affine()
	wantX, wantY, wantOK := want.affine()
	if gotOK != wantOK || gotOK && (!gotX.Equal(&wantX) || !gotY.Equal(&wantY)) {
		t.Fatal("point doubling formulas disagree")
	}
}
