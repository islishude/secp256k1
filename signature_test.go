package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func TestSignVerifyRecoverDigest(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("secp256k1 deterministic recoverable signature"))

	sig1, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if sig1 != sig2 {
		t.Fatal("RFC6979 signature is not deterministic")
	}
	recSig1, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	recSig2, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if recSig1 != recSig2 {
		t.Fatal("recoverable RFC6979 signature is not deterministic")
	}
	wantR := must32("6a223f22d601fdadc0f3a3c166c947d19bf54375d7897d8eabb989ea6a2c4731")
	wantS := must32("498457b17443ac530fcedfd6e5d719460e628993c783f810300919052cf2acdf")
	var wantRecoverable RecoverableSignature
	copy(wantRecoverable[:32], wantR[:])
	copy(wantRecoverable[32:64], wantS[:])
	if recSig1 != wantRecoverable {
		t.Fatalf("recoverable signature changed\n got %x\nwant %x", recSig1, wantRecoverable)
	}
	if recSig1.Signature() != sig1 {
		t.Fatal("recoverable and non-recoverable signatures disagree")
	}
	if !bytes.Equal(sig1.Bytes(), sig1[:]) {
		t.Fatal("Signature.Bytes returned different bytes")
	}
	if !bytes.Equal(recSig1.Bytes(), recSig1[:]) {
		t.Fatal("RecoverableSignature.Bytes returned different bytes")
	}
	if !VerifyDigest(pub, digest, sig1) {
		t.Fatal("signature does not verify")
	}
	if !VerifyCanonicalDigest(pub, digest, sig1) {
		t.Fatal("canonical signature does not verify")
	}

	recovered, err := RecoverDigest(digest, recSig1)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Equal(pub) {
		t.Fatal("recovered public key mismatch")
	}

	var sBytes [32]byte
	copy(sBytes[:], sig1[32:64])
	if bytes.Compare(sBytes[:], scalar.HalfOrder[:]) > 0 {
		t.Fatalf("signature is not low-S: %x", sBytes)
	}

	highS := sig1
	makeSignatureHighS(&highS)
	if !VerifyDigest(pub, digest, highS) {
		t.Fatal("mathematically valid high-S signature did not verify")
	}
	if VerifyCanonicalDigest(pub, digest, highS) {
		t.Fatal("canonical verification accepted high-S signature")
	}
	highSRecoverable := recSig1
	makeRecoverableSignatureHighS(&highSRecoverable)
	if _, err := RecoverDigest(digest, highSRecoverable); err == nil {
		t.Fatal("recovered high-S recoverable signature")
	}

	recidMutated := recSig1
	recidMutated[recoverableSignatureRecIDAt] ^= 1
	if !VerifyDigest(pub, digest, recidMutated.Signature()) {
		t.Fatal("ordinary verification depends on recovery id")
	}
	if recovered, err := RecoverDigest(digest, recidMutated); err == nil && recovered.Equal(pub) {
		t.Fatal("mutated recovery id recovered the original public key")
	}

	tampered := sig1
	tampered[10] ^= 1
	if VerifyDigest(pub, digest, tampered) {
		t.Fatal("tampered r verified")
	}
	tampered = sig1
	tampered[63] ^= 1
	if VerifyDigest(pub, digest, tampered) {
		t.Fatal("tampered s verified")
	}
	badRecoverable := recSig1
	badRecoverable[64] = 4
	if _, err := RecoverDigest(digest, badRecoverable); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("invalid recid did not return ErrInvalidSignature")
	}
}

func TestPublicKeyRepeatedVerify(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("secp256k1 public key verifier"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if !VerifyDigest(pub, digest, sig) {
			t.Fatal("public key verifier rejected valid signature")
		}
		if !VerifyCanonicalDigest(pub, digest, sig) {
			t.Fatal("public key canonical verifier rejected valid signature")
		}
	}
}

func TestSignatureRejectsInvalidScalars(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("secp256k1 invalid signature scalars"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	recSig, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*Signature)
	}{
		{
			name: "zero r",
			mutate: func(sig *Signature) {
				clear(sig[:32])
			},
		},
		{
			name: "zero s",
			mutate: func(sig *Signature) {
				clear(sig[32:64])
			},
		},
		{
			name: "r equals order",
			mutate: func(sig *Signature) {
				copy(sig[:32], scalar.Order[:])
			},
		},
		{
			name: "s equals order",
			mutate: func(sig *Signature) {
				copy(sig[32:64], scalar.Order[:])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			badSig := sig
			tc.mutate(&badSig)
			if VerifyDigest(pub, digest, badSig) {
				t.Fatal("invalid signature verified")
			}
			if VerifyCanonicalDigest(pub, digest, badSig) {
				t.Fatal("invalid signature canonical-verified")
			}

			badRecSig := recSig
			badRecSigBase := badRecSig.Signature()
			tc.mutate(&badRecSigBase)
			copy(badRecSig[:SignatureSize], badRecSigBase[:])
			if _, err := RecoverDigest(digest, badRecSig); err == nil {
				t.Fatal("invalid signature recovered")
			}
		})
	}
}

func TestSignatureCompactAndDERParsing(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("secp256k1 der signature"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	recSig, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		t.Fatal(err)
	}

	parsedCompact, err := ParseSignature(sig[:])
	if err != nil {
		t.Fatal(err)
	}
	if parsedCompact != sig {
		t.Fatal("compact signature parse mismatch")
	}
	parsedRecoverable, err := ParseRecoverableSignature(recSig[:])
	if err != nil {
		t.Fatal(err)
	}
	if parsedRecoverable != recSig {
		t.Fatal("recoverable signature parse mismatch")
	}

	der, err := sig.BytesDER()
	if err != nil {
		t.Fatal(err)
	}
	parsedDER, err := ParseDERSignature(der)
	if err != nil {
		t.Fatal(err)
	}
	if parsedDER != sig {
		t.Fatal("DER signature parse mismatch")
	}
	derAgain, err := parsedDER.BytesDER()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(derAgain, der) {
		t.Fatalf("DER did not round trip canonically\n got %x\nwant %x", derAgain, der)
	}

	highS := sig
	makeSignatureHighS(&highS)
	highDER, err := highS.BytesDER()
	if err != nil {
		t.Fatal(err)
	}
	parsedHigh, err := ParseDERSignature(highDER)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyDigest(pub, digest, parsedHigh) {
		t.Fatal("parsed high-S DER signature did not verify permissively")
	}
	if VerifyCanonicalDigest(pub, digest, parsedHigh) {
		t.Fatal("parsed high-S DER signature canonical-verified")
	}
}

func TestParseDERSignatureRejectsInvalidEncoding(t *testing.T) {
	rEqualsOrder := append([]byte{0x30, 0x26, 0x02, 0x21, 0x00}, scalar.Order[:]...)
	rEqualsOrder = append(rEqualsOrder, 0x02, 0x01, 0x01)

	tests := []struct {
		name string
		der  []byte
	}{
		{name: "empty", der: nil},
		{name: "bad sequence tag", der: []byte{0x31, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}},
		{name: "bad sequence length", der: []byte{0x30, 0x07, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}},
		{name: "long form length", der: []byte{0x30, 0x81, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}},
		{name: "trailing data", der: []byte{0x30, 0x07, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01, 0x00}},
		{name: "zero length r", der: []byte{0x30, 0x05, 0x02, 0x00, 0x02, 0x01, 0x01}},
		{name: "negative r", der: []byte{0x30, 0x06, 0x02, 0x01, 0x80, 0x02, 0x01, 0x01}},
		{name: "nonminimal r", der: []byte{0x30, 0x07, 0x02, 0x02, 0x00, 0x01, 0x02, 0x01, 0x01}},
		{name: "zero r", der: []byte{0x30, 0x06, 0x02, 0x01, 0x00, 0x02, 0x01, 0x01}},
		{name: "r equals order", der: rEqualsOrder},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if sig, err := ParseDERSignature(tc.der); err == nil {
				t.Fatalf("ParseDERSignature(%x) = %x, want error", tc.der, sig)
			}
		})
	}
}

func TestParseSignatureRejectsInvalidEncoding(t *testing.T) {
	if _, err := ParseSignature(nil); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("accepted empty compact signature")
	}
	var sig Signature
	sig[31] = 1
	sig[63] = 1
	if _, err := ParseSignature(sig[:SignatureSize-1]); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("accepted short compact signature")
	}
	copy(sig[:32], scalar.Order[:])
	if _, err := ParseSignature(sig[:]); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("accepted compact signature with r equal to order")
	}

	var rec RecoverableSignature
	rec[31] = 1
	rec[63] = 1
	rec[recoverableSignatureRecIDAt] = 4
	if _, err := ParseRecoverableSignature(rec[:]); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("accepted recoverable signature with invalid recovery id")
	}
}

func TestRecoverDigestRejectsOverflowXCoordinate(t *testing.T) {
	pMinusOrder := new(big.Int).Sub(new(big.Int).Set(bigP), new(big.Int).SetBytes(scalar.Order[:]))
	rBytes := bigTo32(pMinusOrder)

	var digest Digest
	var sig RecoverableSignature
	copy(sig[:32], rBytes[:])
	sig[63] = 1
	sig[recoverableSignatureRecIDAt] = 2

	if _, err := RecoverDigest(digest, sig); err == nil {
		t.Fatal("recovered signature with x-coordinate equal to field modulus")
	}
}

func makeSignatureHighS(sig *Signature) {
	s := new(big.Int).SetBytes(sig[32:64])
	s.Sub(new(big.Int).SetBytes(scalar.Order[:]), s)
	s.FillBytes(sig[32:64])
}

func makeRecoverableSignatureHighS(sig *RecoverableSignature) {
	s := sig.Signature()
	makeSignatureHighS(&s)
	copy(sig[:SignatureSize], s[:])
}
