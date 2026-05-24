package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

func FuzzParsePublicKey(f *testing.F) {
	var one [32]byte
	one[31] = 1
	priv, err := ParsePrivateKey(one[:])
	if err != nil {
		f.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		f.Fatal(err)
	}
	compressed, err := pub.BytesCompressed()
	if err != nil {
		f.Fatal(err)
	}
	uncompressed, err := pub.BytesUncompressed()
	if err != nil {
		f.Fatal(err)
	}

	f.Add(compressed[:])
	f.Add(uncompressed[:])
	f.Add([]byte{})
	f.Add([]byte{0x02})
	f.Add(append([]byte{0x02}, make([]byte, field.Size)...))
	f.Add(append([]byte{0x04}, make([]byte, 64)...))

	f.Fuzz(func(t *testing.T, encoded []byte) {
		pub, err := ParsePublicKey(encoded)
		if err != nil {
			return
		}

		compressed, err := pub.BytesCompressed()
		if err != nil {
			t.Fatalf("compressed encoding failed: %v", err)
		}
		parsedCompressed, err := ParsePublicKey(compressed[:])
		if err != nil {
			t.Fatalf("compressed encoding did not parse: %v", err)
		}
		if !pub.Equal(parsedCompressed) {
			t.Fatalf("compressed round trip mismatch for %x", encoded)
		}

		uncompressed, err := pub.BytesUncompressed()
		if err != nil {
			t.Fatalf("uncompressed encoding failed: %v", err)
		}
		parsedUncompressed, err := ParsePublicKey(uncompressed[:])
		if err != nil {
			t.Fatalf("uncompressed encoding did not parse: %v", err)
		}
		if !pub.Equal(parsedUncompressed) {
			t.Fatalf("uncompressed round trip mismatch for %x", encoded)
		}
		if !parsedCompressed.Equal(parsedUncompressed) {
			t.Fatalf("compressed/uncompressed mismatch for %x", encoded)
		}
	})
}

func FuzzSignVerifyRecoverDigest(f *testing.F) {
	addSignVerifySeed(f, must32("01"), sha256.Sum256([]byte("fuzz secp256k1 one")))
	addSignVerifySeed(f, must32("2a"), sha256.Sum256([]byte("fuzz secp256k1 forty-two")))
	addSignVerifySeed(f, must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8"), sha256.Sum256([]byte("fuzz secp256k1 known key")))
	addSignVerifySeed(f, [PrivateKeySize]byte{}, sha256.Sum256([]byte("fuzz secp256k1 zero key")))
	addSignVerifySeed(f, scalar.Order, sha256.Sum256([]byte("fuzz secp256k1 order key")))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < PrivateKeySize+DigestSize {
			return
		}

		var keyBytes [PrivateKeySize]byte
		var digest Digest
		copy(keyBytes[:], input[:PrivateKeySize])
		copy(digest[:], input[PrivateKeySize:PrivateKeySize+DigestSize])

		priv, err := ParsePrivateKey(keyBytes[:])
		if err != nil {
			return
		}
		pub, err := priv.PublicKey()
		if err != nil {
			t.Fatalf("PublicKey failed: %v", err)
		}

		sig1, err := priv.SignDigest(digest)
		if err != nil {
			t.Fatalf("SignDigest failed: %v", err)
		}
		sig2, err := priv.SignDigest(digest)
		if err != nil {
			t.Fatalf("second SignDigest failed: %v", err)
		}
		if sig1 != sig2 {
			t.Fatalf("signature is not deterministic\n first %x\nsecond %x", sig1, sig2)
		}
		if !VerifyDigest(pub, digest, sig1) {
			t.Fatalf("signature did not verify\nkey %x\ndigest %x\nsig %x", keyBytes, digest, sig1)
		}
		if !VerifyCanonicalDigest(pub, digest, sig1) {
			t.Fatalf("canonical signature did not verify\nkey %x\ndigest %x\nsig %x", keyBytes, digest, sig1)
		}

		recSig, err := priv.SignRecoverableDigest(digest)
		if err != nil {
			t.Fatalf("SignRecoverableDigest failed: %v", err)
		}
		if recSig.Signature() != sig1 {
			t.Fatalf("recoverable signature does not match ordinary signature\nsig %x\nrec %x", sig1, recSig)
		}
		recovered, err := RecoverDigest(digest, recSig)
		if err != nil {
			t.Fatalf("RecoverDigest failed: %v\nkey %x\ndigest %x\nsig %x", err, keyBytes, digest, recSig)
		}
		if !recovered.Equal(pub) {
			t.Fatalf("recovered public key mismatch\nkey %x\ndigest %x\nsig %x", keyBytes, digest, recSig)
		}

		var sBytes [32]byte
		copy(sBytes[:], sig1[32:64])
		if bytes.Compare(sBytes[:], scalar.HalfOrder[:]) > 0 {
			t.Fatalf("signature is not low-S: %x", sBytes)
		}
	})
}

func FuzzRecoverDigest(f *testing.F) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		f.Fatal(err)
	}
	digest := sha256.Sum256([]byte("fuzz secp256k1 recover"))
	sig, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(digest[:], sig[:])
	f.Add(make([]byte, DigestSize), make([]byte, RecoverableSignatureSize))
	f.Add([]byte{}, []byte{})

	f.Fuzz(func(t *testing.T, digestBytes, sigBytes []byte) {
		if len(digestBytes) != DigestSize || len(sigBytes) != RecoverableSignatureSize {
			return
		}

		var digest Digest
		var sig RecoverableSignature
		copy(digest[:], digestBytes)
		copy(sig[:], sigBytes)

		recovered, err := RecoverDigest(digest, sig)
		if err != nil {
			return
		}
		if !VerifyCanonicalDigest(recovered, digest, sig.Signature()) {
			t.Fatalf("recovered key does not verify signature\ndigest %x\nsig %x", digest, sig)
		}

		compressed, err := recovered.BytesCompressed()
		if err != nil {
			t.Fatalf("recovered key compressed encoding failed: %v", err)
		}
		parsed, err := ParsePublicKey(compressed[:])
		if err != nil {
			t.Fatalf("recovered key compressed encoding did not parse: %v", err)
		}
		if !recovered.Equal(parsed) {
			t.Fatalf("recovered key compressed round trip mismatch")
		}
	})
}

func addSignVerifySeed(f *testing.F, key [PrivateKeySize]byte, digest Digest) {
	seed := make([]byte, PrivateKeySize+DigestSize)
	copy(seed[:PrivateKeySize], key[:])
	copy(seed[PrivateKeySize:], digest[:])
	f.Add(seed)
}
