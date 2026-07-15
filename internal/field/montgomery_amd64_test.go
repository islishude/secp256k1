//go:build amd64 && secp256k1_asm

package field

import (
	"math/rand"
	"testing"

	"github.com/islishude/secp256k1/internal/cpufeat"
)

func TestAMD64FeatureDispatch(t *testing.T) {
	want := amd64KernelSet{
		add:    cpufeat.HasADXAndBMI2,
		sub:    cpufeat.HasADXAndBMI2,
		mul:    cpufeat.HasADXAndBMI2,
		square: cpufeat.HasADXAndBMI2,
	}
	if amd64Kernels != want {
		t.Fatalf("dispatch=%+v, want %+v", amd64Kernels, want)
	}
	t.Logf("CPUID ADX+BMI2=%t", cpufeat.HasADXAndBMI2)
}

func TestAMD64AddSubAssemblyMatchesFiat(t *testing.T) {
	original := amd64Kernels
	amd64Kernels = amd64KernelSet{add: true, sub: true}
	t.Cleanup(func() { amd64Kernels = original })

	edges := montgomeryBackendEdges()
	for _, x := range edges {
		for _, y := range edges {
			checkMontgomeryBackend(t, x, y)
		}
	}
	rng := rand.New(rand.NewSource(3))
	for range 100_000 {
		checkMontgomeryBackend(t, randomCanonicalFieldWords(rng), randomCanonicalFieldWords(rng))
	}
}

func TestAMD64FiatFallback(t *testing.T) {
	original := amd64Kernels
	amd64Kernels = amd64KernelSet{}
	t.Cleanup(func() { amd64Kernels = original })

	checkMontgomeryBackend(t,
		[4]uint64{0xfffffffefffffc2e, ^uint64(0), ^uint64(0), ^uint64(0)},
		[4]uint64{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	)
}
