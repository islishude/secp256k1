package secp256k1

import (
	"crypto/rand"
	"io"

	"github.com/islishude/secp256k1/internal/scalar"
)

// PrivateKey is a secp256k1 signing key represented as a scalar modulo the
// curve order.
type PrivateKey struct {
	d     scalar.Element
	valid bool
}

// GeneratePrivateKey reads random bytes until it produces a valid non-zero
// private scalar less than the secp256k1 group order.
func GeneratePrivateKey(reader io.Reader) (*PrivateKey, error) {
	if reader == nil {
		reader = rand.Reader
	}
	for {
		var b [PrivateKeySize]byte
		if _, err := io.ReadFull(reader, b[:]); err != nil {
			return nil, err
		}
		key, err := parsePrivateKeyBytes(&b)
		clear(b[:])
		if err == nil {
			return key, nil
		}
	}
}

// ParsePrivateKey parses a canonical 32-byte big-endian private key.
func ParsePrivateKey(b []byte) (*PrivateKey, error) {
	if len(b) != PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}
	return parsePrivateKeyBytes((*[PrivateKeySize]byte)(b))
}

func parsePrivateKeyBytes(b *[PrivateKeySize]byte) (*PrivateKey, error) {
	if scalar.IsZeroBytes(b) || !scalar.LessThanOrder(b) {
		return nil, ErrInvalidPrivateKey
	}
	var d scalar.Element
	d.SetBytesUnchecked(b)
	return &PrivateKey{d: d, valid: true}, nil
}

func (k *PrivateKey) isValid() bool {
	return k != nil && k.valid && !k.d.IsZero()
}

// Destroy clears k's scalar material and marks it invalid. Go does not provide a
// hard memory-erasure guarantee, so this is a best-effort cleanup API.
func (k *PrivateKey) Destroy() {
	if k == nil {
		return
	}
	k.d.SetZero()
	k.valid = false
}

// Bytes returns the canonical 32-byte big-endian private key encoding.
func (k *PrivateKey) Bytes() ([PrivateKeySize]byte, error) {
	if !k.isValid() {
		return [PrivateKeySize]byte{}, ErrInvalidPrivateKey
	}
	return k.d.Bytes(), nil
}

// PublicKey derives the matching public key by multiplying the base point by the
// private scalar.
func (k *PrivateKey) PublicKey() (PublicKey, error) {
	if !k.isValid() {
		return PublicKey{}, ErrInvalidPrivateKey
	}
	x, y, ok := scalarBaseMultAffine(&k.d)
	if !ok {
		return PublicKey{}, ErrInvalidPrivateKey
	}
	pub := newPublicKey(&x, &y)
	if !pub.isValid() {
		return PublicKey{}, ErrInvalidPrivateKey
	}
	return pub, nil
}
