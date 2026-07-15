//go:build amd64 && secp256k1_asm

package cpufeat

// HasADXAndBMI2 is initialized once from public CPU state. GOAMD64 levels are
// intentionally not consulted because GOAMD64=v3 does not require ADX.
var HasADXAndBMI2 = detectADXAndBMI2()

func detectADXAndBMI2() bool {
	maxLeaf, _, _, _ := cpuid(0, 0)
	if maxLeaf < 7 {
		return false
	}
	_, ebx, _, _ := cpuid(7, 0)
	return hasADXAndBMI2(ebx)
}
