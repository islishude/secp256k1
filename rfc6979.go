package secp256k1

import (
	"crypto/sha256"

	"github.com/islishude/secp256k1/internal/scalar"
)

const maxHMACPayloadSize = sha256.Size + 1 + PrivateKeySize + scalar.Size

var (
	rfc6979Zero = [1]byte{0x00}
	rfc6979One  = [1]byte{0x01}
)

type nonceRFC6979 struct {
	k [sha256.Size]byte
	v [sha256.Size]byte
}

func newNonceRFC6979(priv *[PrivateKeySize]byte, digest *Digest) nonceRFC6979 {
	var n nonceRFC6979
	for i := range n.v {
		n.v[i] = 0x01
	}

	// RFC6979 section 3.2 initializes an HMAC-DRBG from the private key and
	// the message digest reduced modulo the group order.
	h1 := scalar.SetBytesModOrder(digest)
	n.k = hmacDigestFast(n.k[:], n.v[:], rfc6979Zero[:], priv[:], h1[:])
	n.v = hmacDigestFast(n.k[:], n.v[:])
	n.k = hmacDigestFast(n.k[:], n.v[:], rfc6979One[:], priv[:], h1[:])
	n.v = hmacDigestFast(n.k[:], n.v[:])
	return n
}

func (n *nonceRFC6979) Next() [scalar.Size]byte {
	for {
		n.v = hmacDigestFast(n.k[:], n.v[:])
		if scalar.LessThanOrder(&n.v) && n.v != [sha256.Size]byte{} {
			return n.v
		}
		// Invalid candidates are fed back into the DRBG, as RFC6979 requires.
		n.Reject()
	}
}

func (n *nonceRFC6979) Reject() {
	n.k = hmacDigestFast(n.k[:], n.v[:], rfc6979Zero[:])
	n.v = hmacDigestFast(n.k[:], n.v[:])
}

func (n *nonceRFC6979) Destroy() {
	clear(n.k[:])
	clear(n.v[:])
}

// hmacDigestFast handles the fixed RFC6979 input shapes without allocating.
func hmacDigestFast(key []byte, chunks ...[]byte) [sha256.Size]byte {
	var inner [sha256.BlockSize + maxHMACPayloadSize]byte
	var outer [sha256.BlockSize + sha256.Size]byte

	for i := range sha256.BlockSize {
		inner[i] = 0x36
		outer[i] = 0x5c
	}

	if len(key) > sha256.BlockSize {
		hashedKey := sha256.Sum256(key)
		for i, b := range hashedKey {
			inner[i] ^= b
			outer[i] ^= b
		}
	} else {
		for i, b := range key {
			inner[i] ^= b
			outer[i] ^= b
		}
	}

	n := sha256.BlockSize
	for _, chunk := range chunks {
		n += copy(inner[n:], chunk)
	}
	innerHash := sha256.Sum256(inner[:n])
	copy(outer[sha256.BlockSize:], innerHash[:])

	return sha256.Sum256(outer[:sha256.BlockSize+sha256.Size])
}
