//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

package scalar

import (
	"github.com/islishude/secp256k1/internal/amd64bench"
	"github.com/islishude/secp256k1/internal/cpufeat"
)

func init() {
	amd64ScalarKernels = amd64ScalarKernelSet{
		mul:     amd64bench.Enabled("scalar-mul") && cpufeat.HasADXAndBMI2,
		square:  amd64bench.Enabled("scalar-square") && cpufeat.HasADXAndBMI2,
		squareN: amd64bench.Enabled("scalar-squaren") && cpufeat.HasADXAndBMI2,
	}
}
