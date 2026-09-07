package secp256k1

import (
	"fmt"
	"sync"
	"testing"
)

func BenchmarkParsePublicKeyOnly(b *testing.B) {
	m := newBenchmarkMaterial(b)
	for b.Loop() {
		p, err := ParsePublicKey(m.compressed[:])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkPublicKeySink = p
	}
}

func BenchmarkVerifyPublicKeyLifetime(b *testing.B) {
	m := newBenchmarkMaterial(b)
	for _, count := range []int{1, 8, 9, 16, 32} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			for b.Loop() {
				p, err := ParsePublicKey(m.compressed[:])
				if err != nil {
					b.Fatal(err)
				}
				for range count {
					if !VerifyDigest(p, m.digest, m.signature) {
						b.Fatal("verification failed")
					}
				}
			}
		})
	}
}

func BenchmarkVerifyConcurrentPromotion(b *testing.B) {
	m := newBenchmarkMaterial(b)
	for b.Loop() {
		p, err := ParsePublicKey(m.compressed[:])
		if err != nil {
			b.Fatal(err)
		}
		for range 8 {
			if !VerifyDigest(p, m.digest, m.signature) {
				b.Fatal("verification failed")
			}
		}
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				if !VerifyDigest(p, m.digest, m.signature) {
					b.Error("verification failed")
				}
			})
		}
		wg.Wait()
	}
}

func BenchmarkVerifyHotPublicKeyParallel(b *testing.B) {
	m := newBenchmarkMaterial(b)
	for range 16 {
		if !VerifyDigest(m.publicKey, m.digest, m.signature) {
			b.Fatal("verification failed")
		}
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if !VerifyDigest(m.publicKey, m.digest, m.signature) {
				b.Error("verification failed")
			}
		}
	})
}
