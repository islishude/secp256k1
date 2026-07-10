package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func TestWNAFHalfMatchesFullWNAF(t *testing.T) {
	state := uint64(0x243f6a8885a308d3)
	for range 2_000 {
		var words [4]uint64
		for i := range words {
			state = state*2862933555777941757 + 3037000493
			words[i] = state
		}
		var k scalar.Element
		k.SetWordsModOrder(words)
		old1, old2 := scalar.SplitEndomorphism(&k)
		new1, new2 := scalar.SplitEndomorphismVartimeWords(k.Words())
		assertHalfWNAFMatches(t, &old1, new1, generatorWNAFWindow)
		assertHalfWNAFMatches(t, &old2, new2, generatorWNAFWindow)
		assertHalfWNAFMatches(t, &old1, new1, varWNAFWindow)
		assertHalfWNAFMatches(t, &old2, new2, varWNAFWindow)
	}
}

func assertHalfWNAFMatches(t *testing.T, old *scalar.Element, compact scalar.SplitScalar, window int) {
	t.Helper()
	oldNAF, oldLen, oldSign := signedWNAFVartime(old, window)
	var got [halfWNAFSize]int16
	gotLen := signedWNAFHalfVartime(&got, compact, window)
	if gotLen != oldLen {
		t.Fatalf("wNAF length = %d, want %d for %+v", gotLen, oldLen, compact)
	}
	for i := range gotLen {
		if got[i] != oldNAF[i]*oldSign {
			t.Fatalf("wNAF[%d] = %d, want %d for %+v", i, got[i], oldNAF[i]*oldSign, compact)
		}
	}
}

func TestDoubleScalarWordsMatchesLegacy(t *testing.T) {
	var qScalar scalar.Element
	qScalar.SetUint64(42)
	q := scalarBaseMult(&qScalar)
	table := newAffineOddTable(&q)
	endoTable := newEndomorphismWNAFTable(&table)

	state := uint64(0x13198a2e03707344)
	for range 500 {
		var k1Words, k2Words [4]uint64
		for i := range k1Words {
			state = state*6364136223846793005 + 1442695040888963407
			k1Words[i] = state
			state = state*6364136223846793005 + 1442695040888963407
			k2Words[i] = state
		}
		var k1, k2 scalar.Element
		k1.SetWordsModOrder(k1Words)
		k2.SetWordsModOrder(k2Words)
		got := doubleScalarBaseMultPrecomputedVartime(&k1, &k2, &table, &endoTable)
		want := doubleScalarBaseMultPrecomputedLegacyVartime(&k1, &k2, &table, &endoTable)
		gotX, gotY, gotOK := got.affine()
		wantX, wantY, wantOK := want.affine()
		if gotOK != wantOK || gotOK && (!gotX.Equal(&wantX) || !gotY.Equal(&wantY)) {
			t.Fatalf("double-scalar mismatch for %x/%x", k1.Bytes(), k2.Bytes())
		}
	}
}

func TestDoubleScalarCombMatchesWNAF(t *testing.T) {
	var qScalar scalar.Element
	qScalar.SetUint64(42)
	q := scalarBaseMult(&qScalar)
	wnafTable := newAffineOddTable(&q)
	endoTable := newEndomorphismWNAFTable(&wnafTable)
	combTable := newVerifyCombTable(&q)

	state := uint64(0xa4093822299f31d0)
	for range 500 {
		var k1Words, k2Words [4]uint64
		for i := range k1Words {
			state = state*6364136223846793005 + 1442695040888963407
			k1Words[i] = state
			state = state*6364136223846793005 + 1442695040888963407
			k2Words[i] = state
		}
		var k1, k2 scalar.Element
		k1.SetWordsModOrder(k1Words)
		k2.SetWordsModOrder(k2Words)
		got := doubleScalarBaseMultCombVartime(&k1, &k2, &combTable)
		want := doubleScalarBaseMultPrecomputedVartime(&k1, &k2, &wnafTable, &endoTable)
		gotX, gotY, gotOK := got.affine()
		wantX, wantY, wantOK := want.affine()
		if gotOK != wantOK || gotOK && (!gotX.Equal(&wantX) || !gotY.Equal(&wantY)) {
			t.Fatalf("comb mismatch for %x/%x", k1.Bytes(), k2.Bytes())
		}
	}
}
