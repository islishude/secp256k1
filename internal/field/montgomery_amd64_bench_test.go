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
	if cpufeat.HasADXAndBMI2 {
		switch selection {
		case "mul":
			want.mul = true
		case "square":
			want.square = true
		default:
			t.Fatalf("test requires one benchmark kernel, got %q", selection)
		}
	}
	if amd64Kernels != want {
		t.Fatalf("benchmark selection %q produced %+v, want %+v", selection, amd64Kernels, want)
	}
}
