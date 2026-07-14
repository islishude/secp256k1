//go:build amd64 && secp256k1_asm

package field

import (
	"testing"

	"github.com/islishude/secp256k1/internal/cpufeat"
)

func TestAMD64FeatureDispatch(t *testing.T) {
	if useADXAndBMI2 != cpufeat.HasADXAndBMI2 {
		t.Fatalf("dispatch=%t, CPUID ADX+BMI2=%t", useADXAndBMI2, cpufeat.HasADXAndBMI2)
	}
	t.Logf("CPUID ADX+BMI2=%t", cpufeat.HasADXAndBMI2)
}

func TestAMD64FiatFallback(t *testing.T) {
	original := useADXAndBMI2
	useADXAndBMI2 = false
	t.Cleanup(func() { useADXAndBMI2 = original })

	checkMontgomeryBackend(t,
		[4]uint64{0xfffffffefffffc2e, ^uint64(0), ^uint64(0), ^uint64(0)},
		[4]uint64{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	)
}
