// Package cpufeat contains the small, dependency-free feature detector used by
// opt-in assembly backends.
package cpufeat

const (
	leaf7BMI2 = uint32(1 << 8)
	leaf7ADX  = uint32(1 << 19)
)

func hasADXAndBMI2(leaf7EBX uint32) bool {
	const required = leaf7ADX | leaf7BMI2
	return leaf7EBX&required == required
}
