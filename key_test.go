package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func TestPrivateKeyOnePublicKey(t *testing.T) {
	var one [32]byte
	one[31] = 1
	priv, err := ParsePrivateKey(one[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	got, err := pub.BytesCompressed()
	if err != nil {
		t.Fatal(err)
	}
	want := [33]byte{
		0x02,
		0x79, 0xbe, 0x66, 0x7e, 0xf9, 0xdc, 0xbb, 0xac,
		0x55, 0xa0, 0x62, 0x95, 0xce, 0x87, 0x0b, 0x07,
		0x02, 0x9b, 0xfc, 0xdb, 0x2d, 0xce, 0x28, 0xd9,
		0x59, 0xf2, 0x81, 0x5b, 0x16, 0xf8, 0x17, 0x98,
	}
	if got != want {
		t.Fatalf("unexpected compressed public key\n got %x\nwant %x", got, want)
	}
}

func TestZeroValueAndDestroyedKeys(t *testing.T) {
	digest := sha256.Sum256([]byte("secp256k1 zero values"))
	var sig Signature
	var priv PrivateKey
	if _, err := priv.SignDigest(digest); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("zero private key SignDigest error = %v", err)
	}
	if _, err := priv.SignRecoverableDigest(digest); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("zero private key SignRecoverableDigest error = %v", err)
	}
	if _, err := priv.Bytes(); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("zero private key Bytes error = %v", err)
	}
	if _, err := priv.PublicKey(); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("zero private key PublicKey error = %v", err)
	}

	var pub PublicKey
	if _, err := pub.BytesCompressed(); !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("zero public key BytesCompressed error = %v", err)
	}
	if _, err := pub.BytesUncompressed(); !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("zero public key BytesUncompressed error = %v", err)
	}
	if VerifyDigest(pub, digest, sig) {
		t.Fatal("zero public key verified")
	}
	if VerifyCanonicalDigest(pub, digest, sig) {
		t.Fatal("zero public key canonical-verified")
	}
	if pub.Equal(PublicKey{}) {
		t.Fatal("zero public keys compared equal")
	}

	missingPrecompute := PublicKey{x: generator.x, y: generator.y, valid: true}
	if _, err := missingPrecompute.BytesCompressed(); !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("public key without precompute BytesCompressed error = %v", err)
	}
	if VerifyDigest(missingPrecompute, digest, sig) {
		t.Fatal("public key without precompute verified")
	}

	keyBytes := must32("01")
	parsed, err := ParsePrivateKey(keyBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	parsed.Destroy()
	if _, err := parsed.SignDigest(digest); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("destroyed private key SignDigest error = %v", err)
	}
	if _, err := parsed.Bytes(); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatalf("destroyed private key Bytes error = %v", err)
	}
}

func TestPrivateKeyBounds(t *testing.T) {
	if _, err := ParsePrivateKey(make([]byte, PrivateKeySize)); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatal("accepted zero private key")
	}
	if _, err := ParsePrivateKey(scalar.Order[:]); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatal("accepted order as private key")
	}
	if _, err := ParsePrivateKey(nil); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatal("accepted empty private key")
	}
	if _, err := ParsePrivateKey(make([]byte, PrivateKeySize-1)); !errors.Is(err, ErrInvalidPrivateKey) {
		t.Fatal("accepted short private key")
	}
}

func TestGeneratePrivateKeyRejectsInvalidCandidates(t *testing.T) {
	var valid [PrivateKeySize]byte
	valid[31] = 1

	input := make([]byte, 0, 3*PrivateKeySize)
	input = append(input, make([]byte, PrivateKeySize)...)
	input = append(input, scalar.Order[:]...)
	input = append(input, valid[:]...)

	priv, err := GeneratePrivateKey(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	got, err := priv.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if got != valid {
		t.Fatalf("GeneratePrivateKey returned %x, want %x", got, valid)
	}

	priv, err = GeneratePrivateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !priv.isValid() {
		t.Fatal("GeneratePrivateKey returned an invalid key")
	}
}

func TestGeneratePrivateKeyReadError(t *testing.T) {
	if _, err := GeneratePrivateKey(bytes.NewReader(nil)); err == nil {
		t.Fatal("GeneratePrivateKey succeeded with an empty reader")
	}
}

func TestPrivateKeyBytesRoundTrip(t *testing.T) {
	keyBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(keyBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	got, err := priv.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if got != keyBytes {
		t.Fatalf("PrivateKey.Bytes() = %x, want %x", got, keyBytes)
	}
	reparsed, err := ParsePrivateKey(got[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	reparsedPub, err := reparsed.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(reparsedPub) {
		t.Fatal("reparsed private key derived a different public key")
	}
}
