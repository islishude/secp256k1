package secp256k1

import (
	"crypto/subtle"

	"github.com/islishude/secp256k1/internal/field"
)

// PublicKey is a secp256k1 verification key represented by an affine curve
// point.
type PublicKey struct {
	x     field.Element
	y     field.Element
	valid bool
}

// ParsePublicKey parses a SEC 1 compressed or uncompressed public key.
func ParsePublicKey(b []byte) (PublicKey, error) {
	switch len(b) {
	case PublicKeyCompressedSize:
		if b[0] != 0x02 && b[0] != 0x03 {
			return PublicKey{}, ErrInvalidPublicKey
		}
		var xb [field.Size]byte
		copy(xb[:], b[1:])
		x, y, ok := affineFromXBytes(&xb, b[0] == 0x03)
		if !ok {
			return PublicKey{}, ErrInvalidPublicKey
		}
		return newPublicKey(&x, &y), nil
	case PublicKeyUncompressedSize:
		if b[0] != 0x04 {
			return PublicKey{}, ErrInvalidPublicKey
		}
		var xb, yb [field.Size]byte
		copy(xb[:], b[1:33])
		copy(yb[:], b[33:])
		var x, y field.Element
		if !x.SetBytes(&xb) || !y.SetBytes(&yb) || !isOnCurve(&x, &y) {
			return PublicKey{}, ErrInvalidPublicKey
		}
		return newPublicKey(&x, &y), nil
	default:
		return PublicKey{}, ErrInvalidPublicKey
	}
}

// BytesCompressed returns the SEC 1 compressed public key encoding.
func (p PublicKey) BytesCompressed() ([PublicKeyCompressedSize]byte, error) {
	var out [PublicKeyCompressedSize]byte
	if !p.isValid() {
		return out, ErrInvalidPublicKey
	}
	out[0] = 0x02
	if p.y.IsOdd() {
		out[0] = 0x03
	}
	x := p.x.Bytes()
	copy(out[1:], x[:])
	return out, nil
}

// BytesUncompressed returns the SEC 1 uncompressed public key encoding.
func (p PublicKey) BytesUncompressed() ([PublicKeyUncompressedSize]byte, error) {
	var out [PublicKeyUncompressedSize]byte
	if !p.isValid() {
		return out, ErrInvalidPublicKey
	}
	out[0] = 0x04
	x := p.x.Bytes()
	y := p.y.Bytes()
	copy(out[1:33], x[:])
	copy(out[33:], y[:])
	return out, nil
}

// Equal reports whether p and q encode the same public key.
func (p PublicKey) Equal(q PublicKey) bool {
	if !p.isValid() || !q.isValid() {
		return false
	}
	pb, err := p.BytesUncompressed()
	if err != nil {
		return false
	}
	qb, err := q.BytesUncompressed()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(pb[:], qb[:]) == 1
}

func (p PublicKey) isValid() bool {
	return p.valid && isOnCurve(&p.x, &p.y)
}

func publicKeyFromPoint(p *point) (PublicKey, bool) {
	x, y, ok := p.affine()
	if !ok {
		return PublicKey{}, false
	}
	return newPublicKey(&x, &y), true
}

func newPublicKey(x, y *field.Element) PublicKey {
	return PublicKey{
		x:     *x,
		y:     *y,
		valid: true,
	}
}
