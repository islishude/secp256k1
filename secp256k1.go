package secp256k1

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"io"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	// PrivateKeySize is the byte length of a canonical secp256k1 private key.
	PrivateKeySize = 32
	// PublicKeyCompressedSize is the byte length of a SEC 1 compressed public key.
	PublicKeyCompressedSize = 33
	// PublicKeyUncompressedSize is the byte length of a SEC 1 uncompressed public key.
	PublicKeyUncompressedSize = 65
	// RecoverableSignatureSize is the byte length of r || s || recovery-id.
	RecoverableSignatureSize = 65

	recoverableSignatureRecIDAt = 64
)

var (
	errInvalidPrivateKey = errors.New("secp256k1: invalid private key")
	errInvalidPublicKey  = errors.New("secp256k1: invalid public key")
	errInvalidSignature  = errors.New("secp256k1: invalid signature")
)

// PrivateKey is a secp256k1 signing key represented as a scalar modulo the
// curve order.
type PrivateKey struct {
	d scalar.Element
}

// PublicKey is a secp256k1 verification key represented by an affine curve
// point.
type PublicKey struct {
	x field.Element
	y field.Element
}

// GenerateKey reads random bytes until it produces a valid non-zero private
// scalar less than the secp256k1 group order.
func GenerateKey(reader io.Reader) (*PrivateKey, error) {
	if reader == nil {
		reader = rand.Reader
	}
	for {
		var b [PrivateKeySize]byte
		if _, err := io.ReadFull(reader, b[:]); err != nil {
			return nil, err
		}
		// Rejection is extremely rare, but keeps the distribution uniform over
		// the valid scalar range [1, n-1].
		key, err := NewPrivateKey(b)
		if err == nil {
			return key, nil
		}
	}
}

// NewPrivateKey parses a canonical 32-byte big-endian private key.
func NewPrivateKey(b [PrivateKeySize]byte) (*PrivateKey, error) {
	if scalar.IsZeroBytes(&b) || !scalar.LessThanOrder(&b) {
		return nil, errInvalidPrivateKey
	}
	var d scalar.Element
	if !d.SetBytes(&b) {
		return nil, errInvalidPrivateKey
	}
	return &PrivateKey{d: d}, nil
}

// Bytes returns the canonical 32-byte big-endian private key encoding.
func (k *PrivateKey) Bytes() [PrivateKeySize]byte {
	return k.d.Bytes()
}

// Public derives the matching public key by multiplying the base point by the
// private scalar.
func (k *PrivateKey) Public() *PublicKey {
	p := scalarBaseMult(&k.d)
	return publicKeyFromPoint(&p)
}

// SignDigest signs a 32-byte message digest with deterministic RFC6979 ECDSA.
//
// The returned signature is r || s || recovery-id. The s value is normalized to
// the low half of the group order so the signature has a unique canonical form.
func (k *PrivateKey) SignDigest(digest [32]byte) ([RecoverableSignatureSize]byte, error) {
	privBytes := k.d.Bytes()
	nonce := newNonceRFC6979(privBytes, digest)
	e := new(scalar.Element).SetBytesModOrder(&digest)

	for {
		kBytes := nonce.Next()
		var nonceScalar scalar.Element
		if !nonceScalar.SetBytes(&kBytes) {
			nonce.Reject()
			continue
		}

		rPoint := scalarBaseMult(&nonceScalar)
		rx, ry, ok := rPoint.affine()
		if !ok {
			nonce.Reject()
			continue
		}
		// ECDSA defines r as x(R) mod n. Recovery needs to know whether the
		// original field x-coordinate was r or r+n, so keep that overflow bit.
		rxBytes := rx.Bytes()
		xOverflow := byte(0)
		if !scalar.LessThanOrder(&rxBytes) {
			xOverflow = 1
		}
		rBytes := scalar.SetBytesModOrder(rxBytes)
		if scalar.IsZeroBytes(&rBytes) {
			nonce.Reject()
			continue
		}

		var r, rd, sum, kinv, s scalar.Element
		if !r.SetBytes(&rBytes) {
			nonce.Reject()
			continue
		}
		rd.Mul(&r, &k.d)
		sum.Add(e, &rd)
		kinv.Inv(&nonceScalar)
		s.Mul(&kinv, &sum)
		if s.IsZero() {
			nonce.Reject()
			continue
		}

		// The recovery id encodes the y parity and the x-coordinate overflow.
		recid := byte(0)
		if ry.IsOdd() {
			recid |= 1
		}
		recid |= xOverflow << 1
		if s.IsHigh() {
			// Low-S normalization replaces s with n-s. That is equivalent to
			// using -R, so the y-parity bit must be flipped for recovery.
			s.Neg(&s)
			recid ^= 1
		}

		var sig [RecoverableSignatureSize]byte
		copy(sig[:32], rBytes[:])
		sBytes := s.Bytes()
		copy(sig[32:64], sBytes[:])
		sig[recoverableSignatureRecIDAt] = recid
		return sig, nil
	}
}

// VerifyDigest reports whether sig is a valid recoverable ECDSA signature for
// digest under pub.
func VerifyDigest(pub *PublicKey, digest [32]byte, sig [RecoverableSignatureSize]byte) bool {
	if pub == nil || !pub.isValid() {
		return false
	}
	r, s, ok := parseSignatureScalars(sig)
	if !ok {
		return false
	}

	var w, u1, u2, e scalar.Element
	w.Inv(&s)
	e.SetBytesModOrder(&digest)
	u1.Mul(&e, &w)
	u2.Mul(&r, &w)

	// ECDSA verification checks that x((e/s)G + (r/s)Q) mod n equals r.
	pubPoint := pointFromPublicKey(pub)
	sum := doubleScalarBaseMult(&u1, &pubPoint, &u2)
	if sum.isInfinity() {
		return false
	}
	x, _, _ := sum.affine()
	xBytes := scalar.SetBytesModOrder(x.Bytes())
	rBytes := r.Bytes()
	return subtle.ConstantTimeCompare(xBytes[:], rBytes[:]) == 1
}

// RecoverDigest reconstructs the public key that produced sig over digest.
func RecoverDigest(digest [32]byte, sig [RecoverableSignatureSize]byte) (*PublicKey, error) {
	r, s, ok := parseSignatureScalars(sig)
	if !ok {
		return nil, errInvalidSignature
	}
	recid := sig[recoverableSignatureRecIDAt]
	xBytes := r.Bytes()
	if recid>>1 == 1 {
		// The high recovery bit means the ephemeral point used x = r + n.
		var ok bool
		xBytes, ok = addOrder(xBytes)
		if !ok {
			return nil, errInvalidSignature
		}
	}
	if !field.LessThanModulus(&xBytes) {
		return nil, errInvalidSignature
	}

	var x, y, rhs, x2, seven field.Element
	if !x.SetBytes(&xBytes) {
		return nil, errInvalidSignature
	}
	x2.Square(&x)
	rhs.Mul(&x2, &x)
	seven.SetUint64(7)
	rhs.Add(&rhs, &seven)
	if !y.Sqrt(&rhs) {
		return nil, errInvalidSignature
	}
	// The low recovery bit selects which of the two square roots is the
	// ephemeral point's y-coordinate.
	if y.IsOdd() != (recid&1 == 1) {
		y.Neg(&y)
	}

	var rPoint point
	rPoint.setAffine(&x, &y)
	if !isOnCurve(&x, &y) {
		return nil, errInvalidSignature
	}

	var e, rInv scalar.Element
	e.SetBytesModOrder(&digest)
	rInv.Inv(&r)

	// Rearranging s = k^-1(e + rd) gives Q = dG = r^-1(sR - eG).
	sBytes := s.Bytes()
	sR := scalarMultAffine(&rPoint, &sBytes)
	eG := scalarBaseMult(&e)
	var negEG point
	negEG.neg(&eG)
	var q point
	q.add(&sR, &negEG)
	rInvBytes := rInv.Bytes()
	q = scalarMult(&q, &rInvBytes)
	if q.isInfinity() {
		return nil, errInvalidSignature
	}
	pub := publicKeyFromPoint(&q)
	if pub == nil || !VerifyDigest(pub, digest, sig) {
		return nil, errInvalidSignature
	}
	return pub, nil
}

// ParsePublicKey parses a SEC 1 compressed or uncompressed public key.
func ParsePublicKey(b []byte) (*PublicKey, error) {
	switch len(b) {
	case PublicKeyCompressedSize:
		if b[0] != 0x02 && b[0] != 0x03 {
			return nil, errInvalidPublicKey
		}
		var xb [32]byte
		copy(xb[:], b[1:])
		var x, y, rhs, x2, seven field.Element
		if !x.SetBytes(&xb) {
			return nil, errInvalidPublicKey
		}
		// Compressed keys store x plus one y-parity bit. Recover y by solving
		// y^2 = x^3 + 7 in the field.
		x2.Square(&x)
		rhs.Mul(&x2, &x)
		seven.SetUint64(7)
		rhs.Add(&rhs, &seven)
		if !y.Sqrt(&rhs) {
			return nil, errInvalidPublicKey
		}
		wantOdd := b[0] == 0x03
		if y.IsOdd() != wantOdd {
			y.Neg(&y)
		}
		if !isOnCurve(&x, &y) {
			return nil, errInvalidPublicKey
		}
		return &PublicKey{x: x, y: y}, nil
	case PublicKeyUncompressedSize:
		if b[0] != 0x04 {
			return nil, errInvalidPublicKey
		}
		var xb, yb [32]byte
		copy(xb[:], b[1:33])
		copy(yb[:], b[33:])
		var x, y field.Element
		if !x.SetBytes(&xb) || !y.SetBytes(&yb) || !isOnCurve(&x, &y) {
			return nil, errInvalidPublicKey
		}
		return &PublicKey{x: x, y: y}, nil
	default:
		return nil, errInvalidPublicKey
	}
}

// BytesCompressed returns the SEC 1 compressed public key encoding.
func (p *PublicKey) BytesCompressed() [PublicKeyCompressedSize]byte {
	var out [PublicKeyCompressedSize]byte
	out[0] = 0x02
	if p.y.IsOdd() {
		out[0] = 0x03
	}
	x := p.x.Bytes()
	copy(out[1:], x[:])
	return out
}

// BytesUncompressed returns the SEC 1 uncompressed public key encoding.
func (p *PublicKey) BytesUncompressed() [PublicKeyUncompressedSize]byte {
	var out [PublicKeyUncompressedSize]byte
	out[0] = 0x04
	x := p.x.Bytes()
	y := p.y.Bytes()
	copy(out[1:33], x[:])
	copy(out[33:], y[:])
	return out
}

// Equal reports whether p and q encode the same public key.
func (p *PublicKey) Equal(q *PublicKey) bool {
	if p == nil || q == nil {
		return p == q
	}
	pb := p.BytesUncompressed()
	qb := q.BytesUncompressed()
	return subtle.ConstantTimeCompare(pb[:], qb[:]) == 1
}

func (p *PublicKey) isValid() bool {
	return isOnCurve(&p.x, &p.y)
}

func publicKeyFromPoint(p *point) *PublicKey {
	x, y, ok := p.affine()
	if !ok {
		return nil
	}
	return &PublicKey{x: x, y: y}
}

func parseSignatureScalars(sig [RecoverableSignatureSize]byte) (scalar.Element, scalar.Element, bool) {
	var rBytes, sBytes [32]byte
	copy(rBytes[:], sig[:32])
	copy(sBytes[:], sig[32:64])
	recid := sig[recoverableSignatureRecIDAt]
	if recid > 3 ||
		scalar.IsZeroBytes(&rBytes) || scalar.IsZeroBytes(&sBytes) ||
		!scalar.LessThanOrder(&rBytes) || !scalar.LessThanOrder(&sBytes) {
		return scalar.Element{}, scalar.Element{}, false
	}
	var r, s scalar.Element
	if !r.SetBytes(&rBytes) || !s.SetBytes(&sBytes) {
		return scalar.Element{}, scalar.Element{}, false
	}
	return r, s, true
}

func addOrder(r [32]byte) ([32]byte, bool) {
	var out [32]byte
	carry := 0
	for i := 31; i >= 0; i-- {
		v := int(r[i]) + int(scalar.Order[i]) + carry
		out[i] = byte(v)
		carry = v >> 8
	}
	if carry != 0 {
		return out, false
	}
	return out, field.LessThanModulus(&out)
}
