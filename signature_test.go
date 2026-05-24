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

func TestPreparedPublicKeyVerify(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := pub.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("secp256k1 prepared verifier"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.VerifyDigest(digest, sig) {
		t.Fatal("prepared verifier rejected valid signature")
	}
	if !prepared.VerifyCanonicalDigest(digest, sig) {
		t.Fatal("prepared canonical verifier rejected valid signature")
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
