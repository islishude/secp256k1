//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

package field

import (
	"os"
	"testing"

	"github.com/islishude/secp256k1/internal/cpufeat"
)

func TestAMD64BenchmarkKernelSelection(t *testing.T) {
	selection := os.Getenv("SECP256K1_AMD64_BENCH_KERNEL")
	want := amd64KernelSet{}
	switch selection {
	case "none":
	case "mul":
		if cpufeat.HasADXAndBMI2 {
			want.mul = true
		}
	case "square":
		if cpufeat.HasADXAndBMI2 {
			want.square = true
		}
	default:
		t.Fatalf("test requires a benchmark kernel selection, got %q", selection)
	}
	if amd64Kernels != want {
		t.Fatalf("benchmark selection %q produced %+v, want %+v", selection, amd64Kernels, want)
	}
}
