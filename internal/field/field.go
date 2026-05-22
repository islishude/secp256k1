package field

import (
	"encoding/binary"

	fiat "github.com/islishude/secp256k1/internal/fiat/secp256k1montgomery"
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

// Element is an element of the secp256k1 base field modulo p.
//
// Values are stored in Montgomery form so multiplication and squaring can use
// fiat-crypto's generated routines directly.
type Element struct {
	x fiat.MontgomeryDomainFieldElement
}

// LessThanModulus reports whether b is a canonical field encoding.
func LessThanModulus(b *[Size]byte) bool {
	return lessThan(b, &Modulus)
}

// Set assigns z = x.
func (z *Element) Set(x *Element) *Element {
	z.x = x.x
	return z
}

// SetZero assigns z = 0.
func (z *Element) SetZero() *Element {
	z.x = fiat.MontgomeryDomainFieldElement{}
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
	var le [Size]byte
	// fiat-crypto generated code consumes little-endian limbs, while the public
	// API uses conventional big-endian encodings.
	reverseBytes(le[:], b[:])
	var in fiat.NonMontgomeryDomainFieldElement
	fiat.FromBytes((*[4]uint64)(&in), (*[Size]uint8)(&le))
	fiat.ToMontgomery(&z.x, &in)
	return true
}

// Bytes returns the canonical 32-byte big-endian encoding of z.
func (z *Element) Bytes() [Size]byte {
	var out fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&out, &z.x)
	var le [Size]byte
	fiat.ToBytes((*[Size]uint8)(&le), (*[4]uint64)(&out))
	var be [Size]byte
	reverseBytes(be[:], le[:])
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

func lessThan(a, b *[Size]byte) bool {
	for i := range Size {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return false
}

func reverseBytes(dst, src []byte) {
	for i := range src {
		dst[len(src)-1-i] = src[i]
	}
}
