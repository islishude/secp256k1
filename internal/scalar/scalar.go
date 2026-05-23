package scalar

import (
	"encoding/binary"
	"math/bits"

	fiat "github.com/islishude/secp256k1/internal/fiat/secp256k1montgomeryscalar"
	"github.com/islishude/secp256k1/internal/field"
)

// Size is the byte length of a secp256k1 scalar.
const Size = 32

const (
	orderLimb0 uint64 = 0xffffffffffffffff
	orderLimb1 uint64 = 0xfffffffffffffffe
	orderLimb2 uint64 = 0xbaaedce6af48a03b
	orderLimb3 uint64 = 0xbfd25e8cd0364141

	// p - n, where:
	// p = secp256k1 field modulus
	// n = secp256k1 group order
	fieldMinusOrder0 uint64 = field.ModuleLimb0 - orderLimb0
	fieldMinusOrder1 uint64 = field.ModuleLimb1 - orderLimb1
	fieldMinusOrder2 uint64 = field.ModuleLimb2 - orderLimb2
	fieldMinusOrder3 uint64 = field.ModuleLimb3 - orderLimb3
)

// Order is the secp256k1 group order n.
var Order = [Size]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
	0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b,
	0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41,
}

const (
	halfOrder0 uint64 = 0x7fffffffffffffff
	halfOrder1 uint64 = 0xffffffffffffffff
	halfOrder2 uint64 = 0x5d576e7357a4501d
	halfOrder3 uint64 = 0xdfe92f46681b20a0
)

// HalfOrder is floor(n/2). It is used to enforce low-S ECDSA signatures.
var HalfOrder = [Size]byte{
	0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x5d, 0x57, 0x6e, 0x73, 0x57, 0xa4, 0x50, 0x1d,
	0xdf, 0xe9, 0x2f, 0x46, 0x68, 0x1b, 0x20, 0xa0,
}

var (
	endoNegLambda = mustElement([Size]byte{
		0xac, 0x9c, 0x52, 0xb3, 0x3f, 0xa3, 0xcf, 0x1f,
		0x5a, 0xd9, 0xe3, 0xfd, 0x77, 0xed, 0x9b, 0xa4,
		0xa8, 0x80, 0xb9, 0xfc, 0x8e, 0xc7, 0x39, 0xc2,
		0xe0, 0xcf, 0xc8, 0x10, 0xb5, 0x12, 0x83, 0xcf,
	})
	endoNegB1 = mustElement([Size]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xe4, 0x43, 0x7e, 0xd6, 0x01, 0x0e, 0x88, 0x28,
		0x6f, 0x54, 0x7f, 0xa9, 0x0a, 0xbf, 0xe4, 0xc3,
	})
	endoNegB2 = mustElement([Size]byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
		0x8a, 0x28, 0x0a, 0xc5, 0x07, 0x74, 0x34, 0x6d,
		0xd7, 0x65, 0xcd, 0xa8, 0x3d, 0xb1, 0x56, 0x2c,
	})
	endoZ1Words = [4]uint64{
		0x3daa8a1471e8ca7f,
		0xe86c90e49284eb15,
		0x3086d221a7d46bcd,
		0x0000000000000000,
	}
	endoZ2Words = [4]uint64{
		0x221208ac9df506c6,
		0x6f547fa90abfe4c4,
		0xe4437ed6010e8828,
		0x0000000000000000,
	}
)

// Element is an integer modulo the secp256k1 group order.
//
// Values are stored in Montgomery form so multiplication, squaring, and
// inversion can use fiat-crypto's generated routines directly.
type Element struct {
	x fiat.MontgomeryDomainFieldElement
}

// LessThanOrder reports whether b is a canonical scalar encoding.
func LessThanOrder(b *[Size]byte) bool {
	return wordsLessThanOrder(bytesToWords(b))
}

// IsZeroBytes reports whether b is the all-zero scalar encoding.
func IsZeroBytes(b *[Size]byte) bool {
	return *b == [Size]byte{}
}

// FieldElementLessThanOrder reports whether x's canonical field value is less
// than the secp256k1 group order.
func FieldElementLessThanOrder(x *field.Element) bool {
	return wordsLessThanOrder(x.NonMontgomeryWords())
}

// SetBytesModOrder reduces b modulo the group order and returns its canonical
// 32-byte big-endian encoding.
//
// Since b is exactly 32 bytes and Order is close to 2^256, at most one
// subtraction is needed.
func SetBytesModOrder(b *[Size]byte) (out [Size]byte) {
	return wordsToBytes(reduceWordsModOrder(bytesToWords(b)))
}

func bytesToWords(b *[Size]byte) [4]uint64 {
	// fiat-crypto generated code stores limbs little-endian, while the public
	// package uses conventional big-endian byte strings.
	return [4]uint64{
		binary.BigEndian.Uint64(b[24:32]),
		binary.BigEndian.Uint64(b[16:24]),
		binary.BigEndian.Uint64(b[8:16]),
		binary.BigEndian.Uint64(b[0:8]),
	}
}

func wordsToBytes(words [4]uint64) (out [Size]byte) {
	putWordsBytes(&out, words)
	return out
}

func putWordsBytes(out *[Size]byte, words [4]uint64) {
	binary.BigEndian.PutUint64(out[0:8], words[3])
	binary.BigEndian.PutUint64(out[8:16], words[2])
	binary.BigEndian.PutUint64(out[16:24], words[1])
	binary.BigEndian.PutUint64(out[24:32], words[0])
}

func wordsLessThanOrder(words [4]uint64) bool {
	if words[3] != orderLimb0 {
		return words[3] < orderLimb0
	}
	if words[2] != orderLimb1 {
		return words[2] < orderLimb1
	}
	if words[1] != orderLimb2 {
		return words[1] < orderLimb2
	}
	return words[0] < orderLimb3
}

func reduceWordsModOrder(words [4]uint64) [4]uint64 {
	var borrow uint64

	d0, borrow := bits.Sub64(words[0], orderLimb3, 0)
	d1, borrow := bits.Sub64(words[1], orderLimb2, borrow)
	d2, borrow := bits.Sub64(words[2], orderLimb1, borrow)
	d3, borrow := bits.Sub64(words[3], orderLimb0, borrow)

	// borrow == 0: words >= Order, use d = words - Order.
	// borrow == 1: words <  Order, use original words.
	mask := uint64(0) - borrow

	return [4]uint64{
		(d0 &^ mask) | (words[0] & mask),
		(d1 &^ mask) | (words[1] & mask),
		(d2 &^ mask) | (words[2] & mask),
		(d3 &^ mask) | (words[3] & mask),
	}
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
		panic("scalar: uint64 out of range")
	}
	return z
}

// SetBytes parses a canonical 32-byte big-endian scalar.
func (z *Element) SetBytes(b *[Size]byte) bool {
	if !LessThanOrder(b) {
		return false
	}
	z.SetBytesUnchecked(b)
	return true
}

// SetBytesUnchecked assigns z from a canonical 32-byte big-endian scalar.
//
// The caller must ensure b is less than the group order.
func (z *Element) SetBytesUnchecked(b *[Size]byte) *Element {
	return z.setWordsUnchecked(bytesToWords(b))
}

func (z *Element) setWordsUnchecked(words [4]uint64) *Element {
	in := fiat.NonMontgomeryDomainFieldElement{
		words[0], words[1], words[2], words[3],
	}
	fiat.ToMontgomery(&z.x, &in)
	return z
}

// SetBytesModOrder assigns z to b reduced modulo the group order.
func (z *Element) SetBytesModOrder(b *[Size]byte) *Element {
	return z.setWordsUnchecked(reduceWordsModOrder(bytesToWords(b)))
}

// SetFieldElementModOrder assigns z to x reduced modulo the group order.
func (z *Element) SetFieldElementModOrder(x *field.Element) *Element {
	return z.setWordsUnchecked(reduceWordsModOrder(x.NonMontgomeryWords()))
}

// Bytes returns the canonical 32-byte big-endian encoding of z.
func (z *Element) Bytes() [Size]byte {
	var out [Size]byte
	z.PutBytes(&out)
	return out
}

// PutBytes writes the canonical 32-byte big-endian encoding of z to out.
func (z *Element) PutBytes(out *[Size]byte) {
	var words fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&words, &z.x)
	putWordsBytes(out, [4]uint64{words[0], words[1], words[2], words[3]})
}

// IsZero reports whether z is 0.
func (z *Element) IsZero() bool {
	return z.x == fiat.MontgomeryDomainFieldElement{}
}

// IsHigh reports whether z is greater than n/2.
func (z *Element) IsHigh() bool {
	var out fiat.NonMontgomeryDomainFieldElement
	fiat.FromMontgomery(&out, &z.x)
	if out[3] != halfOrder0 {
		return out[3] > halfOrder0
	}
	if out[2] != halfOrder1 {
		return out[2] > halfOrder1
	}
	if out[1] != halfOrder2 {
		return out[1] > halfOrder2
	}
	return out[0] > halfOrder3
}

// Equal reports whether z and x are the same scalar.
func (z *Element) Equal(x *Element) bool {
	return z.x == x.x
}

// IsHighBytes reports whether b is greater than n/2.
func IsHighBytes(b *[Size]byte) bool {
	x0 := binary.BigEndian.Uint64(b[0:8])
	if x0 != halfOrder0 {
		return x0 > halfOrder0
	}

	x1 := binary.BigEndian.Uint64(b[8:16])
	if x1 != halfOrder1 {
		return x1 > halfOrder1
	}

	x2 := binary.BigEndian.Uint64(b[16:24])
	if x2 != halfOrder2 {
		return x2 > halfOrder2
	}

	x3 := binary.BigEndian.Uint64(b[24:32])
	return x3 > halfOrder3
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

// SplitEndomorphism returns k1 and k2 such that k = k1 + k2*lambda mod n.
//
// The returned scalars are balanced around half the bit length of k. Negative
// halves are represented modulo n, so callers that care about scalar length
// should convert values over n/2 to their negation and flip the point sign.
func SplitEndomorphism(k *Element) (Element, Element) {
	kWords := elementWords(k)
	c1 := mul512Rsh320Round(kWords, endoZ1Words)
	c2 := mul512Rsh320Round(kWords, endoZ2Words)

	var c1B1, c2B2, k1, k2, k2Lambda Element
	c1B1.Mul(&c1, &endoNegB1)
	c2B2.Mul(&c2, &endoNegB2)
	k2.Add(&c1B1, &c2B2)

	k2Lambda.Mul(&k2, &endoNegLambda)
	k1.Add(k, &k2Lambda)
	return k1, k2
}

func mustElement(b [Size]byte) Element {
	var e Element
	if !e.SetBytes(&b) {
		panic("scalar: invalid constant")
	}
	return e
}

func mul512Rsh320Round(n1Digits, n2Digits [4]uint64) Element {
	var r1, r2, r3, r4, r5, r6, r7, c uint64

	c, _ = bits.Mul64(n2Digits[0], n1Digits[0])
	c, r1 = mulAdd64(n2Digits[0], n1Digits[1], c)
	c, r2 = mulAdd64(n2Digits[0], n1Digits[2], c)
	r4, r3 = mulAdd64(n2Digits[0], n1Digits[3], c)

	c, _ = mulAdd64(n2Digits[1], n1Digits[0], r1)
	c, r2 = mulAdd64Carry(n2Digits[1], n1Digits[1], r2, c)
	c, r3 = mulAdd64Carry(n2Digits[1], n1Digits[2], r3, c)
	r5, r4 = mulAdd64Carry(n2Digits[1], n1Digits[3], r4, c)

	c, _ = mulAdd64(n2Digits[2], n1Digits[0], r2)
	c, r3 = mulAdd64Carry(n2Digits[2], n1Digits[1], r3, c)
	c, r4 = mulAdd64Carry(n2Digits[2], n1Digits[2], r4, c)
	r6, r5 = mulAdd64Carry(n2Digits[2], n1Digits[3], r5, c)

	c, _ = mulAdd64(n2Digits[3], n1Digits[0], r3)
	c, r4 = mulAdd64Carry(n2Digits[3], n1Digits[1], r4, c)
	c, r5 = mulAdd64Carry(n2Digits[3], n1Digits[2], r5, c)
	r7, r6 = mulAdd64Carry(n2Digits[3], n1Digits[3], r6, c)

	roundBit := r4 >> 63
	r2, r1, r0 := r7, r6, r5

	r0, c = bits.Add64(r0, roundBit, 0)
	r1, c = bits.Add64(r1, 0, c)
	r2, r3 = bits.Add64(r2, 0, c)

	var b [Size]byte
	binary.BigEndian.PutUint64(b[0:8], r3)
	binary.BigEndian.PutUint64(b[8:16], r2)
	binary.BigEndian.PutUint64(b[16:24], r1)
	binary.BigEndian.PutUint64(b[24:32], r0)
	return mustElement(b)
}

func elementWords(k *Element) [4]uint64 {
	b := k.Bytes()
	return [4]uint64{
		binary.BigEndian.Uint64(b[24:32]),
		binary.BigEndian.Uint64(b[16:24]),
		binary.BigEndian.Uint64(b[8:16]),
		binary.BigEndian.Uint64(b[0:8]),
	}
}

func mulAdd64(digit1, digit2, m uint64) (hi, lo uint64) {
	var c uint64
	hi, lo = bits.Mul64(digit1, digit2)
	lo, c = bits.Add64(lo, m, 0)
	hi += c
	return hi, lo
}

func mulAdd64Carry(digit1, digit2, m, c uint64) (hi, lo uint64) {
	var c2 uint64
	hi, lo = mulAdd64(digit1, digit2, m)
	lo, c2 = bits.Add64(lo, c, 0)
	hi, _ = bits.Add64(hi, 0, c2)
	return hi, lo
}

// AddOrder returns r + n and whether the result is less than p - n.
func AddOrder(r [32]byte) ([32]byte, bool) {
	x0 := binary.BigEndian.Uint64(r[0:8])
	x1 := binary.BigEndian.Uint64(r[8:16])
	x2 := binary.BigEndian.Uint64(r[16:24])
	x3 := binary.BigEndian.Uint64(r[24:32])

	var carry uint64

	y3, carry := bits.Add64(x3, orderLimb3, 0)
	y2, carry := bits.Add64(x2, orderLimb2, carry)
	y1, carry := bits.Add64(x1, orderLimb1, carry)
	y0, _ := bits.Add64(x0, orderLimb0, carry)

	var out [32]byte
	binary.BigEndian.PutUint64(out[0:8], y0)
	binary.BigEndian.PutUint64(out[8:16], y1)
	binary.BigEndian.PutUint64(out[16:24], y2)
	binary.BigEndian.PutUint64(out[24:32], y3)

	// ok iff r < p - n.
	//
	// This also implies no carry from r + n, because:
	// p - n < 2^256 - n.
	var borrow uint64
	_, borrow = bits.Sub64(x3, fieldMinusOrder3, 0)
	_, borrow = bits.Sub64(x2, fieldMinusOrder2, borrow)
	_, borrow = bits.Sub64(x1, fieldMinusOrder1, borrow)
	_, borrow = bits.Sub64(x0, fieldMinusOrder0, borrow)

	return out, borrow == 1
}
