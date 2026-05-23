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
	k [32]byte
	v [32]byte
}

func newNonceRFC6979(priv, digest *[32]byte) nonceRFC6979 {
	var n nonceRFC6979
	for i := range n.v {
		n.v[i] = 0x01
	}

	// RFC6979 section 3.2 initializes an HMAC-DRBG from the private key and
	// the message digest reduced modulo the group order.
	h1 := scalar.SetBytesModOrder(digest)
	n.k = hmacDigest(n.k[:], n.v[:], rfc6979Zero[:], priv[:], h1[:])
	n.v = hmacDigest(n.k[:], n.v[:])
	n.k = hmacDigest(n.k[:], n.v[:], rfc6979One[:], priv[:], h1[:])
	n.v = hmacDigest(n.k[:], n.v[:])
	return n
}

func (n *nonceRFC6979) Next() [32]byte {
	for {
		n.v = hmacDigest(n.k[:], n.v[:])
		if scalar.LessThanOrder(&n.v) && n.v != [32]byte{} {
			return n.v
		}
		// Invalid candidates are fed back into the DRBG, as RFC6979 requires.
		n.Reject()
	}
}

func (n *nonceRFC6979) Reject() {
	n.k = hmacDigest(n.k[:], n.v[:], rfc6979Zero[:])
	n.v = hmacDigest(n.k[:], n.v[:])
}

func hmacDigest(key []byte, chunks ...[]byte) [32]byte {
	var inner [sha256.BlockSize + maxHMACPayloadSize]byte
	var outer [sha256.BlockSize + sha256.Size]byte

	// First build the default HMAC pads for a zero key.
	for i := range sha256.BlockSize {
		inner[i] = 0x36
		outer[i] = 0x5c
	}

	// Then XOR in the actual key material.
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
		if len(chunk) > len(inner)-n {
			panic("secp256k1: RFC6979 HMAC input too large")
		}
		n += copy(inner[n:], chunk)
	}

	innerHash := sha256.Sum256(inner[:n])
	copy(outer[sha256.BlockSize:], innerHash[:])

	return sha256.Sum256(outer[:sha256.BlockSize+sha256.Size])
}
