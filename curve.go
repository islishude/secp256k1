package secp256k1

import "github.com/islishude/secp256k1/internal/field"

var (
	gxBytes = [32]byte{
		0x79, 0xbe, 0x66, 0x7e, 0xf9, 0xdc, 0xbb, 0xac,
		0x55, 0xa0, 0x62, 0x95, 0xce, 0x87, 0x0b, 0x07,
		0x02, 0x9b, 0xfc, 0xdb, 0x2d, 0xce, 0x28, 0xd9,
		0x59, 0xf2, 0x81, 0x5b, 0x16, 0xf8, 0x17, 0x98,
	}
	gyBytes = [32]byte{
		0x48, 0x3a, 0xda, 0x77, 0x26, 0xa3, 0xc4, 0x65,
		0x5d, 0xa4, 0xfb, 0xfc, 0x0e, 0x11, 0x08, 0xa8,
		0xfd, 0x17, 0xb4, 0x48, 0xa6, 0x85, 0x54, 0x19,
		0x9c, 0x47, 0xd0, 0x8f, 0xfb, 0x10, 0xd4, 0xb8,
	}
	endoBeta               = newEndomorphismBeta()
	secp256k1B3            = fieldElementUint64(21)
	generator              = newGeneratorPoint()
	generatorAffineTable   = newGeneratorAffineTable()
	generatorWNAFTable     = newGeneratorWNAFTable()
	generatorEndoWNAFTable = newGeneratorEndomorphismWNAFTable(&generatorWNAFTable)
)

func newGeneratorPoint() point {
	var x, y field.Element
	if !x.SetBytes(&gxBytes) || !y.SetBytes(&gyBytes) {
		panic("secp256k1: invalid generator")
	}
	var g point
	g.setAffine(&x, &y)
	return g
}

func newEndomorphismBeta() field.Element {
	b := [32]byte{
		0x7a, 0xe9, 0x6a, 0x2b, 0x65, 0x7c, 0x07, 0x10,
		0x6e, 0x64, 0x47, 0x9e, 0xac, 0x34, 0x34, 0xe9,
		0x9c, 0xf0, 0x49, 0x75, 0x12, 0xf5, 0x89, 0x95,
		0xc1, 0x39, 0x6c, 0x28, 0x71, 0x95, 0x01, 0xee,
	}
	var beta field.Element
	if !beta.SetBytes(&b) {
		panic("secp256k1: invalid endomorphism beta")
	}
	return beta
}

func fieldElementUint64(v uint64) field.Element {
	var out field.Element
	out.SetUint64(v)
	return out
}

func isOnCurve(x, y *field.Element) bool {
	var yy, xx, rhs, seven field.Element
	yy.Square(y)
	xx.Square(x)
	rhs.Mul(&xx, x)
	seven.SetUint64(7)
	rhs.Add(&rhs, &seven)
	return yy.Equal(&rhs)
}

func affineFromXBytes(xBytes *[32]byte, wantOdd bool) (field.Element, field.Element, bool) {
	var x field.Element
	if !x.SetBytes(xBytes) {
		return field.Element{}, field.Element{}, false
	}
	y, ok := curveYFromX(&x, wantOdd)
	if !ok {
		return field.Element{}, field.Element{}, false
	}
	return x, y, true
}

func curveYFromX(x *field.Element, wantOdd bool) (field.Element, bool) {
	var y, rhs, x2, seven field.Element
	x2.Square(x)
	rhs.Mul(&x2, x)
	seven.SetUint64(7)
	rhs.Add(&rhs, &seven)
	if !y.Sqrt(&rhs) {
		return field.Element{}, false
	}
	if y.IsOdd() != wantOdd {
		y.Neg(&y)
	}
	if !isOnCurve(x, &y) {
		return field.Element{}, false
	}
	return y, true
}
