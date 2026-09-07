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
	if pub.precomputed.table.Load().comb {
		t.Fatal("comb table built before reuse threshold")
	}
	if !VerifyDigest(pub, digest, sig) {
		t.Fatal("comb verification failed")
	}
	if !pub.precomputed.table.Load().comb {
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
	if !pub.precomputed.table.Load().comb {
		t.Fatal("concurrent verification did not build comb table")
	}
}

func TestPublicKeyCachesAreLazyAndShared(t *testing.T) {
	var key [32]byte
	key[31] = 1
	priv, err := ParsePrivateKey(key[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := pub.BytesCompressed()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParsePublicKey(encoded[:])
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Equal(pub) {
		t.Fatal("encoding changed key")
	}
	if parsed.precomputed.table.Load() != nil || pub.precomputed.table.Load() != nil {
		t.Fatal("encoding or derivation built a table")
	}
	if VerifyDigest(pub, Digest{}, Signature{}) {
		t.Fatal("accepted invalid signature")
	}
	if pub.precomputed.table.Load() != nil {
		t.Fatal("invalid scalars built a table")
	}
	copyOfPub := pub
	sig, err := priv.SignDigest(Digest{})
	if err != nil {
		t.Fatal(err)
	}
	for range verifyCombBuildAfter {
		if !VerifyDigest(copyOfPub, Digest{}, sig) {
			t.Fatal("verification failed")
		}
	}
	old := pub.precomputed.table.Load()
	if old == nil || old.comb {
		t.Fatal("missing wNAF table")
	}
	oldContents := *old
	if !VerifyDigest(pub, Digest{}, sig) {
		t.Fatal("promotion failed")
	}
	if pub.precomputed.table.Load() == old || !copyOfPub.precomputed.table.Load().comb {
		t.Fatal("copies did not share promotion")
	}
	if *old != oldContents {
		t.Fatal("promotion mutated table used by existing readers")
	}
	uses := pub.precomputed.verifyUses.Load()
	for range 10 {
		if !VerifyDigest(copyOfPub, Digest{}, sig) {
			t.Fatal("hot verification failed")
		}
	}
	if pub.precomputed.verifyUses.Load() != uses {
		t.Fatal("hot path still increments counter")
	}
}

func TestVerifyWhileAnotherCallerPromotes(t *testing.T) {
	var key [32]byte
	key[31] = 1
	priv, err := ParsePrivateKey(key[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	sig, err := priv.SignDigest(Digest{})
	if err != nil {
		t.Fatal(err)
	}
	pub.verificationTable()
	pub.precomputed.verifyUses.Store(verifyCombBuildAfter)
	// Model a builder paused before publication: other callers must complete on wNAF.
	pub.precomputed.promoting.Store(true)
	if !VerifyDigest(pub, Digest{}, sig) {
		t.Fatal("verification during promotion failed")
	}
	if pub.precomputed.table.Load().comb {
		t.Fatal("second caller built another table")
	}
}
