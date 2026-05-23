package benchemark

import (
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
			priv, err := secp256k1.GenerateKey(nil)
			if err != nil {
				t.Fatal(err)
			}

			var msg [32]byte
			if _, err := io.ReadFull(rand.Reader, msg[:]); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(msg[:])
			sig, err := priv.SignDigest(digest)
			if err != nil {
				t.Fatal(err)
			}

			privBytes := priv.Bytes()
			ethpriv, err := crypto.ToECDSA(privBytes[:])
			if err != nil {
				t.Fatal(err)
			}

			pubBytes := crypto.FromECDSAPub(ethpriv.Public().(*ecdsa.PublicKey))
			valid := crypto.VerifySignature(pubBytes[:], digest[:], sig[:64])
			if !valid {
				t.Fatal("invalid signature from geth implementation")
			}

			sig2, err := crypto.Sign(digest[:], ethpriv)
			if err != nil {
				t.Fatal(err)
			}

			valid = secp256k1.VerifyDigest(priv.Public(), digest, [65]byte(sig2))
			if !valid {
				t.Fatal("invalid signature from local implementation")
			}

			rec, err := crypto.Ecrecover(digest[:], sig2)
			if err != nil {
				t.Fatal(err)
			}
			pub, err := secp256k1.RecoverDigest(digest, sig)
			if err != nil {
				t.Fatal(err)
			}
			if pub.BytesUncompressed() != [65]byte(rec) {
				t.Fatal("recovered public key does not match")
			}
		})
	}
}
