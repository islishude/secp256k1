package field

import (
	"encoding/binary"

	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

// Size is the byte length of a secp256k1 field element.
const Size = 32

// Modulus is the secp256k1 base-field prime p = 2^256 - 2^32 - 977.
var Modulus = [Size]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xfc, 0x2f,
}

const (
	ModuleLimb0 uint64 = 0xffffffffffffffff
	ModuleLimb1 uint64 = 0xffffffffffffffff
	ModuleLimb2 uint64 = 0xffffffffffffffff
	ModuleLimb3 uint64 = 0xfffffffefffffc2f
)

// Element is an element of the secp256k1 base field modulo p.
//
// Values are stored in Montgomery form so multiplication and squaring can use
// fiat-crypto's generated routines directly.
type Element struct {
	x fiat.MontgomeryDomainFieldElement
}

// LessThanModulus reports whether b is a canonical field encoding.
func LessThanModulus(b *[Size]byte) bool {
	x0 := binary.BigEndian.Uint64(b[0:8])
	if x0 != ModuleLimb0 {
		return x0 < ModuleLimb0
	}

	x1 := binary.BigEndian.Uint64(b[8:16])
	if x1 != ModuleLimb1 {
		return x1 < ModuleLimb1
	}

	x2 := binary.BigEndian.Uint64(b[16:24])
	if x2 != ModuleLimb2 {
		return x2 < ModuleLimb2
	}

	x3 := binary.BigEndian.Uint64(b[24:32])
	return x3 < ModuleLimb3
}

// Set assigns z = x.
func (z *Element) Set(x *Element) *Element {
	z.x = x.x
	return z
}

// SetZero assigns z = 0.
func (z *Element) SetZero() *Element {
	clear(z.x[:])
	return z
}

// SetOne assigns z = 1.
func (z *Element) SetOne() *Element {
	fiat.SetOne(&z.x)
	return z
}

// SetUint64 assigns z = v.
func (z *Element) SetUint64(v uint64) *Element {
	var b [Size]byte
	binary.BigEndian.PutUint64(b[24:], v)
	ok := z.SetBytes(&b)
	if !ok {
		panic("field: uint64 out of range")
	}
	return z
}

// SetBytes parses a canonical 32-byte big-endian field element.
func (z *Element) SetBytes(b *[Size]byte) bool {
	if !LessThanModulus(b) {
		return false
	}
	var in fiat.NonMontgomeryDomainFieldElement
	// fiat-crypto generated code stores limbs little-endian, while the public
	// API uses conventional big-endian encodings.
	in[0] = binary.BigEndian.Uint64(b[24:32])
	in[1] = binary.BigEndian.Uint64(b[16:24])
	in[2] = binary.BigEndian.Uint64(b[8:16])
	in[3] = binary.BigEndian.Uint64(b[0:8])
	fiat.ToMontgomery(&z.x, &in)
	return true
}

// NonMontgomeryWords returns the canonical non-Montgomery little-endian limbs of z.
func (z *Element) NonMontgomeryWords() [4]uint64 {
	var out fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&out, &z.x)
	return [4]uint64{out[0], out[1], out[2], out[3]}
}

// Bytes returns the canonical 32-byte big-endian encoding of z.
func (z *Element) Bytes() [Size]byte {
	out := z.NonMontgomeryWords()
	var be [Size]byte
	binary.BigEndian.PutUint64(be[0:8], out[3])
	binary.BigEndian.PutUint64(be[8:16], out[2])
	binary.BigEndian.PutUint64(be[16:24], out[1])
	binary.BigEndian.PutUint64(be[24:32], out[0])
	return be
}

// IsZero reports whether z is 0.
func (z *Element) IsZero() bool {
	return z.x == fiat.MontgomeryDomainFieldElement{}
}

// IsOdd reports whether z's canonical integer encoding is odd.
func (z *Element) IsOdd() bool {
	b := z.Bytes()
	return b[Size-1]&1 == 1
}

// Equal reports whether z and x are the same field element.
func (z *Element) Equal(x *Element) bool {
	return z.x == x.x
}

// Select assigns z = x when choice == 0 and z = y when choice == 1.
func (z *Element) Select(x, y *Element, choice uint64) *Element {
	mask := uint64(0) - (choice & 1)
	for i := range z.x {
		z.x[i] = (x.x[i] &^ mask) | (y.x[i] & mask)
	}
	return z
}

// Add assigns z = x + y mod p.
func (z *Element) Add(x, y *Element) *Element {
	fiat.Add(&z.x, &x.x, &y.x)
	return z
}

// Sub assigns z = x - y mod p.
func (z *Element) Sub(x, y *Element) *Element {
	fiat.Sub(&z.x, &x.x, &y.x)
	return z
}

// Neg assigns z = -x mod p.
func (z *Element) Neg(x *Element) *Element {
	fiat.Opp(&z.x, &x.x)
	return z
}

// Double assigns z = 2*x mod p.
func (z *Element) Double(x *Element) *Element {
	return z.Add(x, x)
}

// Mul assigns z = x*y mod p.
func (z *Element) Mul(x, y *Element) *Element {
	fiat.Mul(&z.x, &x.x, &y.x)
	return z
}

// Square assigns z = x^2 mod p.
func (z *Element) Square(x *Element) *Element {
	fiat.Square(&z.x, &x.x)
	return z
}

// SquareN assigns z = x^(2^n) mod p.
func (z *Element) SquareN(x *Element, n int) *Element {
	z.Set(x)
	for range n {
		z.Square(z)
	}
	return z
}

// Sqrt attempts to assign z to a square root of x.
func (z *Element) Sqrt(x *Element) bool {
	z.sqrtCandidate(x)
	var check Element
	// Re-square the candidate because non-residues also produce a field value.
	check.Square(z)
	return check.Equal(x)
}
