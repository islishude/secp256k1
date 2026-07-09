package secp256k1

import (
	"crypto/sha256"
	"testing"
)

var (
	benchmarkSignatureSink            Signature
	benchmarkRecoverableSignatureSink RecoverableSignature
	benchmarkPublicKeySink            PublicKey
	benchmarkVerifyResult             bool
)

func BenchmarkSignDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 sign"))

	b.ReportAllocs()
	b.SetBytes(32)

	for b.Loop() {
		sig, err := priv.SignDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSignatureSink = sig
	}
}

func BenchmarkSignRecoverableDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 recoverable sign"))

	b.ReportAllocs()
	b.SetBytes(32)

	for b.Loop() {
		sig, err := priv.SignRecoverableDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRecoverableSignatureSink = sig
	}
}

func BenchmarkVerifyDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 verify"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(32)
	for b.Loop() {
		benchmarkVerifyResult = VerifyDigest(pub, digest, sig)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}

func BenchmarkRecoverDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 recover"))
	sig, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	want, err := priv.PublicKey()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(32)
	for b.Loop() {
		pub, err := RecoverDigest(digest, sig)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkPublicKeySink = pub
	}
	if !benchmarkPublicKeySink.Equal(want) {
		b.Fatal("recovery failed")
	}
}
