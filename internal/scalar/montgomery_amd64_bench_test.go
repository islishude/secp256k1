//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

package scalar

import (
	"testing"

	"github.com/islishude/secp256k1/internal/amd64bench"
	"github.com/islishude/secp256k1/internal/cpufeat"
)

func TestAMD64V2BenchmarkKernelSelection(t *testing.T) {
	want := amd64ScalarKernelSet{
		invVartime: amd64bench.Enabled("invvartime") && cpufeat.HasADXAndBMI2,
	}
	if amd64ScalarKernels != want {
		t.Fatalf("benchmark selection %q produced %+v, want %+v", amd64bench.Selection(), amd64ScalarKernels, want)
	}
}
