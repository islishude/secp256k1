package scalar

import (
	"encoding/binary"

	fiat "github.com/islishude/secp256k1/internal/fiat/secp256k1montgomeryscalar"
)

// Size is the byte length of a secp256k1 scalar.
const Size = 32

// Order is the secp256k1 group order n.
var Order = [Size]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
	0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b,
	0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41,
}

// HalfOrder is floor(n/2). It is used to enforce low-S ECDSA signatures.
var HalfOrder = [Size]byte{
	0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x5d, 0x57, 0x6e, 0x73, 0x57, 0xa4, 0x50, 0x1d,
	0xdf, 0xe9, 0x2f, 0x46, 0x68, 0x1b, 0x20, 0xa0,
}

// Element is an integer modulo the secp256k1 group order.
//
// Values are stored in Montgomery form so multiplication, squaring, and
// inversion can use fiat-crypto's generated routines directly.
type Element struct {
	x fiat.MontgomeryDomainFieldElement
}

// LessThanOrder reports whether b is a canonical scalar encoding.
func LessThanOrder(b *[Size]byte) bool {
	return lessThan(b, &Order)
}

// IsZeroBytes reports whether b is the all-zero scalar encoding.
func IsZeroBytes(b *[Size]byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// SetBytesModOrder reduces b modulo the group order and returns its canonical
// 32-byte big-endian encoding.
func SetBytesModOrder(b [Size]byte) [Size]byte {
	if !LessThanOrder(&b) {
		// Inputs are 32 bytes and n is close to 2^256, so at most one
		// subtraction is needed.
		b = subtract(b, Order)
	}
	return b
}

// Set assigns z = x.
func (z *Element) Set(x *Element) *Element {
	z.x = x.x
	return z
}

// SetZero assigns z = 0.
func (z *Element) SetZero() *Element {
	var in fiat.NonMontgomeryDomainFieldElement
	fiat.ToMontgomery(&z.x, &in)
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
		panic("scalar: uint64 out of range")
	}
	return z
}

// SetBytes parses a canonical 32-byte big-endian scalar.
func (z *Element) SetBytes(b *[Size]byte) bool {
	if !LessThanOrder(b) {
		return false
	}
	var le [Size]byte
	// fiat-crypto generated code consumes little-endian limbs, while the public
	// package uses big-endian byte strings.
	reverseBytes(le[:], b[:])
	var in fiat.NonMontgomeryDomainFieldElement
	fiat.FromBytes((*[4]uint64)(&in), (*[Size]uint8)(&le))
	fiat.ToMontgomery(&z.x, &in)
	return true
}

// SetBytesModOrder assigns z to b reduced modulo the group order.
func (z *Element) SetBytesModOrder(b *[Size]byte) *Element {
	reduced := SetBytesModOrder(*b)
	ok := z.SetBytes(&reduced)
	if !ok {
		panic("scalar: reduction failed")
	}
	return z
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

// IsHigh reports whether z is greater than n/2.
func (z *Element) IsHigh() bool {
	b := z.Bytes()
	return greaterThan(&b, &HalfOrder)
}

// Equal reports whether z and x are the same scalar.
func (z *Element) Equal(x *Element) bool {
	return z.Bytes() == x.Bytes()
}

// Add assigns z = x + y mod n.
func (z *Element) Add(x, y *Element) *Element {
	fiat.Add(&z.x, &x.x, &y.x)
	return z
}

// Sub assigns z = x - y mod n.
func (z *Element) Sub(x, y *Element) *Element {
	fiat.Sub(&z.x, &x.x, &y.x)
	return z
}

// Neg assigns z = -x mod n.
func (z *Element) Neg(x *Element) *Element {
	fiat.Opp(&z.x, &x.x)
	return z
}

// Mul assigns z = x*y mod n.
func (z *Element) Mul(x, y *Element) *Element {
	fiat.Mul(&z.x, &x.x, &y.x)
	return z
}

// Square assigns z = x^2 mod n.
func (z *Element) Square(x *Element) *Element {
	fiat.Square(&z.x, &x.x)
	return z
}

// SquareN assigns z = x^(2^n) mod n.
func (z *Element) SquareN(x *Element, n int) *Element {
	z.Set(x)
	for range n {
		z.Square(z)
	}
	return z
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

func greaterThan(a, b *[Size]byte) bool {
	for i := range Size {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

func subtract(a, b [Size]byte) [Size]byte {
	var out [Size]byte
	borrow := 0
	for i := Size - 1; i >= 0; i-- {
		v := int(a[i]) - int(b[i]) - borrow
		if v < 0 {
			v += 256
			borrow = 1
		} else {
			borrow = 0
		}
		out[i] = byte(v)
	}
	return out
}

func reverseBytes(dst, src []byte) {
	for i := range src {
		dst[len(src)-1-i] = src[i]
	}
}
