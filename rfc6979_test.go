package secp256k1

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func mustDecode32(t *testing.T, s string) [32]byte {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("invalid hex %q: %v", s, err)
	}
	if len(b) != 32 {
		t.Fatalf("invalid 32-byte hex length: got %d", len(b))
	}
	return [32]byte(b)
}

func digestSHA256(msg string) Digest {
	return sha256.Sum256([]byte(msg))
}

func hmacSHA256Reference(key []byte, chunks ...[]byte) [sha256.Size]byte {
	h := hmac.New(sha256.New, key)
	for _, chunk := range chunks {
		_, _ = h.Write(chunk)
	}
	var out [sha256.Size]byte
	h.Sum(out[:0])
	return out
}

func TestNonceRFC6979_RFC6979SHA256KVectors(t *testing.T) {
	// These k values are from RFC 6979 Appendix A.2.5, ECDSA 256-bit
	// prime-field test vectors, using:
	//
	//   x = C9AFA9D845BA75166B5C215767B1D6934E50C3DB36E89B127B8A622B120F6721
	//
	// The RFC curve is NIST P-256, not secp256k1. However, these two cases are
	// still valid for testing this RFC6979 HMAC-DRBG step because:
	//
	//   1. qlen == hlen == 256.
	//   2. SHA256("sample") and SHA256("test") are both already below the
	//      secp256k1 group order, so bits2octets is unchanged.
	//   3. The private scalar is also valid for secp256k1.
	//
	// Do not use the RFC r/s values as secp256k1 signature vectors. Only the
	// nonce k is being tested here.
	priv := mustDecode32(t,
		"c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721",
	)

	tests := []struct {
		name   string
		digest Digest
		wantK  [sha256.Size]byte
	}{
		{
			name:   `RFC6979/SHA256("sample")`,
			digest: digestSHA256("sample"),
			wantK: mustDecode32(t,
				"a6e3c57dd01abe90086538398355dd4c3b17aa873382b0f24d6129493d8aad60",
			),
		},
		{
			name:   `RFC6979/SHA256("test")`,
			digest: digestSHA256("test"),
			wantK: mustDecode32(t,
				"d16b6ae827f17175e040871a1c7ec3500192c4c92677336ec2537acaee0008e0",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			n := newNonceRFC6979(&priv, &test.digest)
			got := n.Next()

			if got != test.wantK {
				t.Fatalf("nonce mismatch\nwant %x\n got %x", test.wantK, got)
			}
		})
	}
}

func TestNonceRFC6979_Secp256k1RegressionVectors(t *testing.T) {
	// These are secp256k1-specific regression vectors generated from the
	// RFC6979 algorithm with SHA-256 and the secp256k1 group order.
	//
	// They complement the RFC vectors above by covering simple private scalar
	// values and bits2octets reduction edge cases.
	tests := []struct {
		name      string
		priv      [PrivateKeySize]byte
		digest    Digest
		wantNonce [sha256.Size]byte
	}{
		{
			name:   `priv=1,SHA256("sample")`,
			priv:   mustDecode32(t, "0000000000000000000000000000000000000000000000000000000000000001"),
			digest: digestSHA256("sample"),
			wantNonce: mustDecode32(t,
				"0f23d7a2ba580b716ff2a03d43e26b3148eea2eb3a1fc6e7abf7cef3877b35be",
			),
		},
		{
			name:   `priv=1,SHA256("test")`,
			priv:   mustDecode32(t, "0000000000000000000000000000000000000000000000000000000000000001"),
			digest: digestSHA256("test"),
			wantNonce: mustDecode32(t,
				"a548b6eb514cb0fb71b14c30edc0d8f218b1b85f2d019bf43830121c3d729fac",
			),
		},
		{
			name: "priv=bitcoin-example,SHA256(sample)",
			priv: mustDecode32(t,
				"1e99423a4ed27608a15a2616a2b0e9e52ced330ac530edcc32c8ffc6a526aedd",
			),
			digest: digestSHA256("sample"),
			wantNonce: mustDecode32(t,
				"d549fa2991bcb4b371d2011fd1548023f848d53e7a8fd6098d50bd533ca63e89",
			),
		},
		{
			name: "digest=order,reduces-to-zero",
			priv: mustDecode32(t,
				"0000000000000000000000000000000000000000000000000000000000000001",
			),
			digest: mustDecode32(t,
				"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
			),
			wantNonce: mustDecode32(t,
				"010497d369b3d525ca15ec29c104a694210bb59ff6cabfc10afe6df0283896df",
			),
		},
		{
			name: "digest=order+1,reduces-to-one",
			priv: mustDecode32(t,
				"0000000000000000000000000000000000000000000000000000000000000001",
			),
			digest: mustDecode32(t,
				"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364142",
			),
			wantNonce: mustDecode32(t,
				"9a409dab05968059da3efb323dc67c96f234571b965fd39810ca0643fbb795ac",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			n := newNonceRFC6979(&test.priv, &test.digest)
			got := n.Next()

			if got != test.wantNonce {
				t.Fatalf("nonce mismatch\nwant %x\n got %x", test.wantNonce, got)
			}
		})
	}
}

func TestHMACRFC6979Functions_MatchCryptoHMAC(t *testing.T) {
	key := mustDecode32(t,
		"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	)
	v := mustDecode32(t,
		"202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
	)
	priv := mustDecode32(t,
		"1e99423a4ed27608a15a2616a2b0e9e52ced330ac530edcc32c8ffc6a526aedd",
	)
	h1raw := mustDecode32(t,
		"af2bdbe1aa9b6ec1e2ade1d694f41fc71a831d0268e9891562113d8a62add1bf",
	)
	h1 := [scalar.Size]byte(h1raw)

	zero := [1]byte{0x00}
	one := [1]byte{0x01}

	if got, want := hmacRFC6979V(&key, &v), hmacSHA256Reference(key[:], v[:]); got != want {
		t.Fatalf("hmacRFC6979V mismatch\nwant %x\n got %x", want, got)
	}

	if got, want := hmacRFC6979Reject(&key, &v), hmacSHA256Reference(key[:], v[:], zero[:]); got != want {
		t.Fatalf("hmacRFC6979Reject mismatch\nwant %x\n got %x", want, got)
	}

	if got, want := hmacRFC6979Init(&key, &v, 0x00, &priv, &h1), hmacSHA256Reference(key[:], v[:], zero[:], priv[:], h1[:]); got != want {
		t.Fatalf("hmacRFC6979Init tag=0x00 mismatch\nwant %x\n got %x", want, got)
	}

	if got, want := hmacRFC6979Init(&key, &v, 0x01, &priv, &h1), hmacSHA256Reference(key[:], v[:], one[:], priv[:], h1[:]); got != want {
		t.Fatalf("hmacRFC6979Init tag=0x01 mismatch\nwant %x\n got %x", want, got)
	}
}

func TestNonceRFC6979Reject_MatchesReferenceTransition(t *testing.T) {
	n := nonceRFC6979{
		k: mustDecode32(t,
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		),
		v: mustDecode32(t,
			"202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
		),
	}

	oldV := n.v
	zero := [1]byte{0x00}

	wantK := hmacSHA256Reference(n.k[:], oldV[:], zero[:])
	wantV := hmacSHA256Reference(wantK[:], oldV[:])

	n.Reject()

	if n.k != wantK {
		t.Fatalf("Reject k mismatch\nwant %x\n got %x", wantK, n.k)
	}
	if n.v != wantV {
		t.Fatalf("Reject v mismatch\nwant %x\n got %x", wantV, n.v)
	}
}

func TestNonceRFC6979DestroyClearsState(t *testing.T) {
	n := nonceRFC6979{
		k: mustDecode32(t,
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		),
		v: mustDecode32(t,
			"0101010101010101010101010101010101010101010101010101010101010101",
		),
	}

	n.Destroy()

	if n.k != [sha256.Size]byte{} {
		t.Fatalf("k was not cleared: %x", n.k)
	}
	if n.v != [sha256.Size]byte{} {
		t.Fatalf("v was not cleared: %x", n.v)
	}
}
