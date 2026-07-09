package benchmark

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/islishude/secp256k1"
)

func TestCrossCheck(t *testing.T) {
	t.Parallel()

	for i := range 100 {
		t.Run(fmt.Sprintf("round-%d", i), func(t *testing.T) {
			priv, err := secp256k1.GeneratePrivateKey(nil)
			if err != nil {
				t.Fatal(err)
			}

			var msg [secp256k1.DigestSize]byte
			if _, err := io.ReadFull(rand.Reader, msg[:]); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(msg[:])
			sig, err := priv.SignDigest(digest)
			if err != nil {
				t.Fatal(err)
			}
			recSig, err := priv.SignRecoverableDigest(digest)
			if err != nil {
				t.Fatal(err)
			}

			privBytes, err := priv.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			ethpriv, err := crypto.ToECDSA(privBytes[:])
			if err != nil {
				t.Fatal(err)
			}

			pubBytes := crypto.FromECDSAPub(ethpriv.Public().(*ecdsa.PublicKey))
			valid := crypto.VerifySignature(pubBytes[:], digest[:], sig.Bytes())
			if !valid {
				t.Fatal("invalid signature from geth implementation")
			}

			sig2, err := crypto.Sign(digest[:], ethpriv)
			if err != nil {
				t.Fatal(err)
			}

			pub, err := priv.PublicKey()
			if err != nil {
				t.Fatal(err)
			}
			var gethSignature secp256k1.Signature
			copy(gethSignature[:], sig2[:secp256k1.SignatureSize])
			valid = secp256k1.VerifyDigest(pub, digest, gethSignature)
			if !valid {
				t.Fatal("invalid signature from local implementation")
			}

			rec, err := crypto.Ecrecover(digest[:], sig2)
			if err != nil {
				t.Fatal(err)
			}
			recovered, err := secp256k1.RecoverDigest(digest, recSig)
			if err != nil {
				t.Fatal(err)
			}
			recoveredBytes, err := recovered.BytesUncompressed()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(recoveredBytes[:], pubBytes) {
				t.Fatal("local recovered public key does not match")
			}
			if !bytes.Equal(rec, pubBytes) {
				t.Fatal("geth recovered public key does not match")
			}
		})
	}
}
