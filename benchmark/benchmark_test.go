package benchmark

import (
	"bytes"
	"crypto/sha256"
	"testing"

	decredsecp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	decredecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	gethsecp256k1 "github.com/ethereum/go-ethereum/crypto/secp256k1"
	localsecp256k1 "github.com/islishude/secp256k1"
)

var (
	benchFixture = newFixture()

	localSignatureSink             localsecp256k1.Signature
	localRecoverableSignatureSink  localsecp256k1.RecoverableSignature
	decredSignatureSink            [localsecp256k1.SignatureSize]byte
	decredRecoverableSignatureSink []byte
	gethRecoverableSignatureSink   []byte
	verifySink                     bool
)

type fixture struct {
	privateKeyBytes [32]byte
	digest          localsecp256k1.Digest

	localPrivateKey           *localsecp256k1.PrivateKey
	localPublicKey            localsecp256k1.PublicKey
	localSignature            localsecp256k1.Signature
	localRecoverableSignature localsecp256k1.RecoverableSignature

	decredPrivateKey           *decredsecp256k1.PrivateKey
	decredPublicKey            *decredsecp256k1.PublicKey
	decredSignature            *decredecdsa.Signature
	decredRecoverableSignature []byte

	gethPublicKey []byte
	gethSignature []byte
}

func newFixture() fixture {
	privateKeyBytes := [32]byte{
		0x1e, 0x99, 0x42, 0x3a, 0x4e, 0xd2, 0x76, 0x08,
		0xa1, 0x5a, 0x26, 0x16, 0xb4, 0xc1, 0xb5, 0xd1,
		0xf7, 0x65, 0xa9, 0xf6, 0xa5, 0xf5, 0xa2, 0xd8,
		0xe8, 0x1f, 0x6f, 0x8a, 0x6a, 0x88, 0xb8, 0xd8,
	}
	digest := sha256.Sum256([]byte("secp256k1 benchmark digest"))

	localPrivateKey, err := localsecp256k1.ParsePrivateKey(privateKeyBytes[:])
	if err != nil {
		panic(err)
	}
	localPublicKey, err := localPrivateKey.PublicKey()
	if err != nil {
		panic(err)
	}
	localSignature, err := localPrivateKey.SignDigest(digest)
	if err != nil {
		panic(err)
	}
	localRecoverableSignature, err := localPrivateKey.SignRecoverableDigest(digest)
	if err != nil {
		panic(err)
	}

	decredPrivateKey := decredsecp256k1.PrivKeyFromBytes(privateKeyBytes[:])
	decredPublicKey := decredPrivateKey.PubKey()
	decredSignature := decredecdsa.Sign(decredPrivateKey, digest[:])
	decredRecoverableSignature := decredecdsa.SignCompact(decredPrivateKey, digest[:], false)

	gethSignature, err := gethsecp256k1.Sign(digest[:], privateKeyBytes[:])
	if err != nil {
		panic(err)
	}
	gethCurve := gethsecp256k1.S256()
	gethPubX, gethPubY := gethCurve.ScalarBaseMult(privateKeyBytes[:])
	gethPublicKey := gethCurve.Marshal(gethPubX, gethPubY)

	return fixture{
		privateKeyBytes:            privateKeyBytes,
		digest:                     digest,
		localPrivateKey:            localPrivateKey,
		localPublicKey:             localPublicKey,
		localSignature:             localSignature,
		localRecoverableSignature:  localRecoverableSignature,
		decredPrivateKey:           decredPrivateKey,
		decredPublicKey:            decredPublicKey,
		decredSignature:            decredSignature,
		decredRecoverableSignature: decredRecoverableSignature,
		gethPublicKey:              gethPublicKey,
		gethSignature:              gethSignature,
	}
}

func TestBenchmarkFixtures(t *testing.T) {
	localPublicKeyBytes, err := benchFixture.localPublicKey.BytesUncompressed()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(localPublicKeyBytes[:], benchFixture.decredPublicKey.SerializeUncompressed()) {
		t.Fatal("local and decred public keys differ")
	}
	if !bytes.Equal(localPublicKeyBytes[:], benchFixture.gethPublicKey) {
		t.Fatal("local and geth public keys differ")
	}

	if !localsecp256k1.VerifyDigest(benchFixture.localPublicKey, benchFixture.digest, benchFixture.localSignature) {
		t.Fatal("local signature does not verify")
	}
	recoveredLocalPublicKey, err := localsecp256k1.RecoverDigest(benchFixture.digest, benchFixture.localRecoverableSignature)
	if err != nil {
		t.Fatal(err)
	}
	recoveredLocalPublicKeyBytes, err := recoveredLocalPublicKey.BytesUncompressed()
	if err != nil {
		t.Fatal(err)
	}
	if recoveredLocalPublicKeyBytes != localPublicKeyBytes {
		t.Fatal("local recoverable signature recovers a different public key")
	}
	if !benchFixture.decredSignature.Verify(benchFixture.digest[:], benchFixture.decredPublicKey) {
		t.Fatal("decred signature does not verify")
	}
	recoveredDecredPublicKey, _, err := decredecdsa.RecoverCompact(benchFixture.decredRecoverableSignature, benchFixture.digest[:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(localPublicKeyBytes[:], recoveredDecredPublicKey.SerializeUncompressed()) {
		t.Fatal("decred recoverable signature recovers a different public key")
	}
	if !gethsecp256k1.VerifySignature(benchFixture.gethPublicKey, benchFixture.digest[:], benchFixture.gethSignature[:localsecp256k1.SignatureSize]) {
		t.Fatal("geth signature does not verify")
	}
	recoveredGethPublicKey, err := gethsecp256k1.RecoverPubkey(benchFixture.digest[:], benchFixture.gethSignature)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(localPublicKeyBytes[:], recoveredGethPublicKey) {
		t.Fatal("geth recoverable signature recovers a different public key")
	}
}

func decredSignatureCompact(sig *decredecdsa.Signature) [localsecp256k1.SignatureSize]byte {
	var out [localsecp256k1.SignatureSize]byte
	r, s := sig.R(), sig.S()
	r.PutBytesUnchecked(out[:32])
	s.PutBytesUnchecked(out[32:])
	return out
}

func BenchmarkLocalSignCompact(b *testing.B) {
	b.ReportAllocs()
	privateKey := benchFixture.localPrivateKey
	digest := benchFixture.digest

	for b.Loop() {
		signature, err := privateKey.SignDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		localSignatureSink = signature
	}
}

func BenchmarkLocalSignRecoverable(b *testing.B) {
	b.ReportAllocs()
	privateKey := benchFixture.localPrivateKey
	digest := benchFixture.digest

	for b.Loop() {
		signature, err := privateKey.SignRecoverableDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		localRecoverableSignatureSink = signature
	}
}

func BenchmarkLocalVerify(b *testing.B) {
	b.ReportAllocs()
	publicKey := benchFixture.localPublicKey
	digest := benchFixture.digest
	signature := benchFixture.localSignature

	for b.Loop() {
		verifySink = localsecp256k1.VerifyDigest(publicKey, digest, signature)
	}
	if !verifySink {
		b.Fatal("verification failed")
	}
}

func BenchmarkDecredSignCompact(b *testing.B) {
	b.ReportAllocs()
	privateKey := benchFixture.decredPrivateKey
	digest := benchFixture.digest[:]

	for b.Loop() {
		decredSignatureSink = decredSignatureCompact(decredecdsa.Sign(privateKey, digest))
	}
}

func BenchmarkDecredSignRecoverable(b *testing.B) {
	b.ReportAllocs()
	privateKey := benchFixture.decredPrivateKey
	digest := benchFixture.digest[:]

	for b.Loop() {
		decredRecoverableSignatureSink = decredecdsa.SignCompact(privateKey, digest, false)
	}
}

func BenchmarkDecredVerify(b *testing.B) {
	b.ReportAllocs()
	publicKey := benchFixture.decredPublicKey
	digest := benchFixture.digest[:]
	signature := benchFixture.decredSignature

	for b.Loop() {
		verifySink = signature.Verify(digest, publicKey)
	}
	if !verifySink {
		b.Fatal("verification failed")
	}
}

func BenchmarkGethSignRecoverable(b *testing.B) {
	b.ReportAllocs()
	privateKey := benchFixture.privateKeyBytes[:]
	digest := benchFixture.digest[:]

	for b.Loop() {
		signature, err := gethsecp256k1.Sign(digest, privateKey)
		if err != nil {
			b.Fatal(err)
		}
		gethRecoverableSignatureSink = signature
	}
}

func BenchmarkGethVerify(b *testing.B) {
	b.ReportAllocs()
	publicKey := benchFixture.gethPublicKey
	digest := benchFixture.digest[:]
	signature := benchFixture.gethSignature[:localsecp256k1.SignatureSize]

	for b.Loop() {
		verifySink = gethsecp256k1.VerifySignature(publicKey, digest, signature)
	}
	if !verifySink {
		b.Fatal("verification failed")
	}
}
