//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

package field

import (
	"os"

	"github.com/islishude/secp256k1/internal/cpufeat"
)

// This file is compiled only by the dedicated performance workflow. Unlike a
// _test.go file, it is also present when the root benchmark imports field.
// Production secp256k1_asm builds cannot observe this environment variable.
func init() {
	selection := os.Getenv("SECP256K1_AMD64_BENCH_KERNEL")
	if selection == "" || selection == "all" {
		return
	}
	amd64Kernels = amd64KernelSet{}
	switch selection {
	case "none":
		return
	case "mul":
		amd64Kernels.mul = cpufeat.HasADXAndBMI2
	case "square":
		amd64Kernels.square = cpufeat.HasADXAndBMI2
	default:
		panic("unknown SECP256K1_AMD64_BENCH_KERNEL value: " + selection)
	}
}
