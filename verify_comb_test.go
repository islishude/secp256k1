package secp256k1

import (
	"crypto/sha256"
	"sync"
	"testing"
)

func TestVerifyPromotesRepeatedPublicKeyToComb(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("adaptive verification precompute"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}

	for range verifyCombBuildAfter {
		if !VerifyDigest(pub, digest, sig) {
			t.Fatal("wNAF verification failed")
		}
	}
	if pub.precomputed.combTable != nil {
		t.Fatal("comb table built before reuse threshold")
	}
	if !VerifyDigest(pub, digest, sig) {
		t.Fatal("comb verification failed")
	}
	if pub.precomputed.combTable == nil {
		t.Fatal("comb table was not built after reuse threshold")
	}
}

func TestVerifyPromotesPublicKeyConcurrently(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("concurrent adaptive verification precompute"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 4 {
				if !VerifyDigest(pub, digest, sig) {
					t.Error("concurrent verification failed")
				}
			}
		})
	}
	wg.Wait()
	if pub.precomputed.combTable == nil {
		t.Fatal("concurrent verification did not build comb table")
	}
}
