package secp256k1

import (
	"crypto/sha256"
	"testing"
)

func BenchmarkSignDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(privBytes)
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 sign"))

	b.ReportAllocs()
	b.SetBytes(32)

	for b.Loop() {
		_, err := priv.SignDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(privBytes)
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 verify"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	pub := priv.Public()

	b.ReportAllocs()
	b.SetBytes(32)
	var benchmarkVerifyResult bool
	for b.Loop() {
		benchmarkVerifyResult = VerifyDigest(pub, digest, sig)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}
