package secp256k1

import (
	"crypto/sha256"

	"github.com/islishude/secp256k1/internal/scalar"
)

const maxHMACPayloadSize = sha256.Size + 1 + PrivateKeySize + scalar.Size

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

	n.k = hmacRFC6979Init(&n.k, &n.v, 0x00, priv, &h1)
	n.v = hmacRFC6979V(&n.k, &n.v)
	n.k = hmacRFC6979Init(&n.k, &n.v, 0x01, priv, &h1)
	n.v = hmacRFC6979V(&n.k, &n.v)

	return n
}

func (n *nonceRFC6979) Next() [scalar.Size]byte {
	for {
		n.v = hmacRFC6979V(&n.k, &n.v)
		if scalar.LessThanOrder(&n.v) && n.v != [sha256.Size]byte{} {
			return n.v
		}
		// Invalid candidates are fed back into the DRBG, as RFC6979 requires.
		n.Reject()
	}
}

func (n *nonceRFC6979) Reject() {
	n.k = hmacRFC6979Reject(&n.k, &n.v)
	n.v = hmacRFC6979V(&n.k, &n.v)
}

func (n *nonceRFC6979) Destroy() {
	clear(n.k[:])
	clear(n.v[:])
}

func initHMACPadsK32(
	inner *[sha256.BlockSize + maxHMACPayloadSize]byte,
	outer *[sha256.BlockSize + sha256.Size]byte,
	key *[sha256.Size]byte,
) {
	for i := range sha256.Size {
		k := key[i]
		inner[i] = k ^ 0x36
		outer[i] = k ^ 0x5c
	}

	for i := sha256.Size; i < sha256.BlockSize; i++ {
		inner[i] = 0x36
		outer[i] = 0x5c
	}
}

// hmacRFC6979V computes:
//
//	HMAC-SHA256(k, v)
func hmacRFC6979V(
	key *[sha256.Size]byte,
	v *[sha256.Size]byte,
) [sha256.Size]byte {
	var inner [sha256.BlockSize + maxHMACPayloadSize]byte
	var outer [sha256.BlockSize + sha256.Size]byte

	initHMACPadsK32(&inner, &outer, key)

	n := sha256.BlockSize
	copy(inner[n:], v[:])
	n += sha256.Size

	innerHash := sha256.Sum256(inner[:n])
	copy(outer[sha256.BlockSize:], innerHash[:])

	return sha256.Sum256(outer[:])
}

// hmacRFC6979Reject computes:
//
//	HMAC-SHA256(k, v || 0x00)
func hmacRFC6979Reject(
	key *[sha256.Size]byte,
	v *[sha256.Size]byte,
) [sha256.Size]byte {
	var inner [sha256.BlockSize + maxHMACPayloadSize]byte
	var outer [sha256.BlockSize + sha256.Size]byte

	initHMACPadsK32(&inner, &outer, key)

	n := sha256.BlockSize
	copy(inner[n:], v[:])
	n += sha256.Size

	inner[n] = 0x00
	n++

	innerHash := sha256.Sum256(inner[:n])
	copy(outer[sha256.BlockSize:], innerHash[:])

	return sha256.Sum256(outer[:])
}

// hmacRFC6979Init computes:
//
//	HMAC-SHA256(k, v || tag || priv || h1)
//
// where tag is either 0x00 or 0x01.
func hmacRFC6979Init(
	key *[sha256.Size]byte,
	v *[sha256.Size]byte,
	tag byte,
	priv *[PrivateKeySize]byte,
	h1 *[scalar.Size]byte,
) [sha256.Size]byte {
	var inner [sha256.BlockSize + maxHMACPayloadSize]byte
	var outer [sha256.BlockSize + sha256.Size]byte

	initHMACPadsK32(&inner, &outer, key)

	n := sha256.BlockSize

	copy(inner[n:], v[:])
	n += sha256.Size

	inner[n] = tag
	n++

	copy(inner[n:], priv[:])
	n += PrivateKeySize

	copy(inner[n:], h1[:])
	n += scalar.Size

	innerHash := sha256.Sum256(inner[:n])
	copy(outer[sha256.BlockSize:], innerHash[:])

	return sha256.Sum256(outer[:])
}
