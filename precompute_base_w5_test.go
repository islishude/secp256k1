//go:build (!arm64 && !amd64) || !secp256k1_asm || (amd64 && secp256k1_amd64_w5_bench)

package secp256k1

import "testing"

func TestGeneratedW5TableMatchesDynamicBuilder(t *testing.T) {
	w5 := newGeneratorAffineTableW5()
	loadedW5 := loadGeneratorAffineTableW5(&generatorAffineTableW5Words)
	for i := range w5 {
		for j := range w5[i] {
			if !w5[i][j].x.Equal(&loadedW5[i][j].x) ||
				!w5[i][j].y.Equal(&loadedW5[i][j].y) {
				t.Fatalf("generated W5 table mismatch at [%d][%d]", i, j)
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
