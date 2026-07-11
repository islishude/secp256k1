package secp256k1

import "testing"

func TestGeneratedPrecomputeMatchesDynamicBuilders(t *testing.T) {
	wnaf := newGeneratorWNAFTable()
	endo := newGeneratorEndomorphismWNAFTable(&wnaf)
	for i := range wnaf {
		if !wnaf[i].x.Equal(&generatorWNAFTable[i].x) ||
			!wnaf[i].y.Equal(&generatorWNAFTable[i].y) {
			t.Fatalf("generated generator wNAF table mismatch at %d", i)
		}
		if !endo[i].x.Equal(&generatorEndoWNAFTable[i].x) ||
			!endo[i].y.Equal(&generatorEndoWNAFTable[i].y) {
			t.Fatalf("generated endomorphism wNAF table mismatch at %d", i)
		}
	}

	comb := newVerifyCombTable(&generator)
	for i := range comb {
		if comb[i].infinity != generatorCombTable[i].infinity ||
			!comb[i].x.Equal(&generatorCombTable[i].x) ||
			!comb[i].y.Equal(&generatorCombTable[i].y) {
			t.Fatalf("generated generator comb table mismatch at %d", i)
		}
	}
}
