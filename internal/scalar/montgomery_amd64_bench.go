//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

package scalar

import (
	"github.com/islishude/secp256k1/internal/amd64bench"
	"github.com/islishude/secp256k1/internal/cpufeat"
)

func init() {
	amd64ScalarKernels = amd64ScalarKernelSet{
		invVartime: amd64bench.Enabled("invvartime") && cpufeat.HasADXAndBMI2,
	}
}
