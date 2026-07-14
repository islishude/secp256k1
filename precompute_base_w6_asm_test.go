//go:build (arm64 && secp256k1_asm) || (amd64 && secp256k1_asm && !secp256k1_amd64_w5_bench)

package secp256k1

import (
	"math/big"
	"testing"

	"github.com/islishude/secp256k1/internal/field"
)

func TestGeneratedW6TableMatchesDynamicBuilder(t *testing.T) {
	w6 := newGeneratorAffineTableW6ForTest()
	for i := range w6 {
		for j := range w6[i] {
			words := generatorAffineTableW6Words[i][j]
			var x, y field.Element
			x.SetMontgomeryWords([4]uint64{words[0], words[1], words[2], words[3]})
			y.SetMontgomeryWords([4]uint64{words[4], words[5], words[6], words[7]})
			if !w6[i][j].x.Equal(&x) || !w6[i][j].y.Equal(&y) {
				t.Fatalf("generated W6 table mismatch at [%d][%d]", i, j)
			}
		}
	}
}

func TestSelectGeneratorW6(t *testing.T) {
	for window := range generatorAffineTableW6Words {
		table := &generatorAffineTableW6Words[window]
		for magnitude := uint64(0); magnitude <= baseTableSizeW6; magnitude++ {
			var got [8]uint64
			selectGeneratorW6(&got, table, magnitude)
			wantIndex := max(magnitude, 1) - 1
			if want := table[wantIndex]; got != want {
				t.Fatalf("window %d magnitude %d selected %x, want %x", window, magnitude, got, want)
			}
		}
	}
}

func TestW6SignedRecodingHighestWindowCarry(t *testing.T) {
	words := [4]uint64{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}
	var reconstructed big.Int
	var carry uint64
	for i := range baseWindowsW6 {
		value := uint64(fixedWindowDigit(&words, uint(i), baseWindowW6)) + carry
		negative := (value + (1 << (baseWindowW6 - 1))) >> baseWindowW6
		digit := int64(value) - int64(negative*(1<<baseWindowW6))
		magnitude := digit
		if magnitude < 0 {
			magnitude = -magnitude
		}
		if magnitude > baseTableSizeW6 {
			t.Fatalf("window %d magnitude %d exceeds table", i, magnitude)
		}
		if i == baseWindowsW6-1 {
			if carry != 1 {
				t.Fatal("highest window did not consume the preceding carry")
			}
			if value != 16 || digit != 16 {
				t.Fatalf("highest window value/digit = %d/%d, want 16/16", value, digit)
			}
		}

		var term big.Int
		term.SetInt64(digit)
		term.Lsh(&term, uint(i*baseWindowW6))
		reconstructed.Add(&reconstructed, &term)
		carry = negative
	}
	if carry != 0 {
		t.Fatalf("terminal carry = %d, want 0", carry)
	}
	var want big.Int
	want.Lsh(big.NewInt(1), 256)
	want.Sub(&want, big.NewInt(1))
	if reconstructed.Cmp(&want) != 0 {
		t.Fatalf("reconstructed scalar = %x, want %x", &reconstructed, &want)
	}
}

func TestW6TableGrowthLimit(t *testing.T) {
	const (
		w5Bytes = baseWindows * baseTableSize * 8 * 8
		w6Bytes = baseWindowsW6 * baseTableSizeW6 * 8 * 8
	)
	if growth := w6Bytes - w5Bytes; growth > 40*1024 {
		t.Fatalf("W6 table growth = %d bytes, limit %d", growth, 40*1024)
	}
}
