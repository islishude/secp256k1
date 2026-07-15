//go:build amd64 && secp256k1_asm

package scalar

import (
	"testing"

	"github.com/islishude/secp256k1/internal/cpufeat"
)

func TestAMD64ScalarFeatureDispatch(t *testing.T) {
	want := amd64ScalarKernelSet{
		invVartime: cpufeat.HasADXAndBMI2,
	}
	if amd64ScalarKernels != want {
		t.Fatalf("dispatch=%+v, want %+v", amd64ScalarKernels, want)
	}
	t.Logf("scalar CPUID ADX+BMI2=%t", cpufeat.HasADXAndBMI2)
}

func TestAMD64ScalarFiatFallback(t *testing.T) {
	original := amd64ScalarKernels
	amd64ScalarKernels = amd64ScalarKernelSet{}
	t.Cleanup(func() { amd64ScalarKernels = original })

	input := [4]uint64{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff}
	checkInvVartimeWordsBackend(t, 0, reduceWordsModOrder(input))
}
