package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func TestSignWithNonceCleansAcceptedAndRejectedAttempts(t *testing.T) {
	var privBytes [32]byte
	privBytes[31] = 1
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	var one, zero, minusR scalar.Element
	one.SetOne()
	minusR.SetFieldElementModOrder(&generator.x)
	minusR.Neg(&minusR)
	for _, tc := range []struct {
		name     string
		nonce, e scalar.Element
		want     bool
	}{
		{"accepted", one, one, true}, {"zero nonce", zero, one, false}, {"zero s", one, minusR, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nonceBytes := tc.nonce.Bytes()
			_, ok := priv.signWithNonce(&tc.e, &nonceBytes, true)
			if ok != tc.want {
				t.Fatalf("accepted=%v want=%v", ok, tc.want)
			}
			if nonceBytes != [32]byte{} {
				t.Fatal("nonce bytes survived attempt")
			}
			got, err := priv.Bytes()
			if err != nil || got != privBytes {
				t.Fatal("cleanup damaged private key")
			}
		})
	}
}
