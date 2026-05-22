package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

func TestPrivateKeyOnePublicKey(t *testing.T) {
	var one [32]byte
	one[31] = 1
	priv, err := NewPrivateKey(one)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public()
	got := pub.BytesCompressed()
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

func TestPublicKeyEncodingRoundTrip(t *testing.T) {
	var keyBytes [32]byte
	keyBytes[31] = 42
	priv, err := NewPrivateKey(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public()
	compressed := pub.BytesCompressed()
	parsedCompressed, err := ParsePublicKey(compressed[:])
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsedCompressed) {
		t.Fatal("compressed round trip mismatch")
	}
	uncompressed := pub.BytesUncompressed()
	parsedUncompressed, err := ParsePublicKey(uncompressed[:])
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsedUncompressed) {
		t.Fatal("uncompressed round trip mismatch")
	}
}

func TestGroupOrder(t *testing.T) {
	p := scalarMult(&generator, &scalar.Order)
	if !p.isInfinity() {
		t.Fatal("n*G is not infinity")
	}
}

func TestScalarMultiplicationAgainstBig(t *testing.T) {
	scalars := [][32]byte{
		must32("01"),
		must32("02"),
		must32("03"),
		must32("2a"),
		must32("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
	}
	for _, k := range scalars {
		got := scalarMult(&generator, &k)
		gotAffine := scalarMultAffine(&generator, &k)
		gotX, gotY, ok := got.affine()
		if !ok {
			t.Fatalf("scalar %x produced infinity", k)
		}
		gotAffineX, gotAffineY, ok := gotAffine.affine()
		if !ok {
			t.Fatalf("affine scalar %x produced infinity", k)
		}
		wantX, wantY := bigScalarBaseMult(new(big.Int).SetBytes(k[:]))
		if gotX.Bytes() != bigTo32(wantX) || gotY.Bytes() != bigTo32(wantY) {
			t.Fatalf("scalar %x mismatch\n got (%x,%x)\nwant (%x,%x)",
				k, gotX.Bytes(), gotY.Bytes(), wantX, wantY)
		}
		if gotAffineX.Bytes() != bigTo32(wantX) || gotAffineY.Bytes() != bigTo32(wantY) {
			t.Fatalf("affine scalar %x mismatch\n got (%x,%x)\nwant (%x,%x)",
				k, gotAffineX.Bytes(), gotAffineY.Bytes(), wantX, wantY)
		}
	}
}

func TestDoubleScalarBaseMult(t *testing.T) {
	qBytes := must32("2a")
	q := scalarMultAffine(&generator, &qBytes)
	qx, qy, ok := q.affine()
	if !ok {
		t.Fatal("test point is infinity")
	}
	q.setAffine(&qx, &qy)
	scalars := [][32]byte{
		must32("01"),
		must32("02"),
		must32("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
	}
	for _, k1Bytes := range scalars {
		for _, k2Bytes := range scalars {
			var k1, k2 scalar.Element
			if !k1.SetBytes(&k1Bytes) || !k2.SetBytes(&k2Bytes) {
				t.Fatal("invalid scalar")
			}

			p1 := scalarBaseMult(&k1)
			p2 := scalarMultAffine(&q, &k2Bytes)
			var want point
			want.add(&p1, &p2)
			got := doubleScalarBaseMult(&k1, &q, &k2)

			gotX, gotY, gotOK := got.affine()
			wantX, wantY, wantOK := want.affine()
			if gotOK != wantOK ||
				(gotOK && (gotX.Bytes() != wantX.Bytes() || gotY.Bytes() != wantY.Bytes())) {
				t.Fatalf("double scalar mismatch\nk1=%x\nk2=%x", k1Bytes, k2Bytes)
			}
		}
	}
}

func TestSignVerifyRecoverDigest(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(privBytes)
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
	if !VerifyDigest(priv.Public(), digest, sig1) {
		t.Fatal("signature does not verify")
	}
	recovered, err := RecoverDigest(digest, sig1)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Equal(priv.Public()) {
		t.Fatal("recovered public key mismatch")
	}

	var sBytes [32]byte
	copy(sBytes[:], sig1[32:64])
	if bytes.Compare(sBytes[:], scalar.HalfOrder[:]) > 0 {
		t.Fatalf("signature is not low-S: %x", sBytes)
	}

	tampered := sig1
	tampered[10] ^= 1
	if VerifyDigest(priv.Public(), digest, tampered) {
		t.Fatal("tampered r verified")
	}
	tampered = sig1
	tampered[63] ^= 1
	if VerifyDigest(priv.Public(), digest, tampered) {
		t.Fatal("tampered s verified")
	}
	tampered = sig1
	tampered[64] = 4
	if VerifyDigest(priv.Public(), digest, tampered) {
		t.Fatal("invalid recid verified")
	}
	if _, err := RecoverDigest(digest, tampered); err == nil {
		t.Fatal("invalid recid recovered")
	}
}

func TestPrivateKeyBounds(t *testing.T) {
	if _, err := NewPrivateKey([32]byte{}); err == nil {
		t.Fatal("accepted zero private key")
	}
	if _, err := NewPrivateKey(scalar.Order); err == nil {
		t.Fatal("accepted order as private key")
	}
}

func TestGenerateKeyRejectsInvalidCandidates(t *testing.T) {
	var valid [PrivateKeySize]byte
	valid[31] = 1

	input := make([]byte, 0, 3*PrivateKeySize)
	input = append(input, make([]byte, PrivateKeySize)...)
	input = append(input, scalar.Order[:]...)
	input = append(input, valid[:]...)

	priv, err := GenerateKey(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if got := priv.Bytes(); got != valid {
		t.Fatalf("GenerateKey returned %x, want %x", got, valid)
	}
}

func TestGenerateKeyReadError(t *testing.T) {
	if _, err := GenerateKey(bytes.NewReader(nil)); err == nil {
		t.Fatal("GenerateKey succeeded with an empty reader")
	}
}

func TestPrivateKeyBytesRoundTrip(t *testing.T) {
	keyBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	got := priv.Bytes()
	if got != keyBytes {
		t.Fatalf("PrivateKey.Bytes() = %x, want %x", got, keyBytes)
	}
	reparsed, err := NewPrivateKey(got)
	if err != nil {
		t.Fatal(err)
	}
	if !priv.Public().Equal(reparsed.Public()) {
		t.Fatal("reparsed private key derived a different public key")
	}
}

func TestSignatureRejectsInvalidScalars(t *testing.T) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(privBytes)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public()
	digest := sha256.Sum256([]byte("secp256k1 invalid signature scalars"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyDigest(nil, digest, sig) {
		t.Fatal("nil public key verified")
	}

	tests := []struct {
		name   string
		mutate func(*[RecoverableSignatureSize]byte)
	}{
		{
			name: "zero r",
			mutate: func(sig *[RecoverableSignatureSize]byte) {
				clear(sig[:32])
			},
		},
		{
			name: "zero s",
			mutate: func(sig *[RecoverableSignatureSize]byte) {
				clear(sig[32:64])
			},
		},
		{
			name: "r equals order",
			mutate: func(sig *[RecoverableSignatureSize]byte) {
				copy(sig[:32], scalar.Order[:])
			},
		},
		{
			name: "s equals order",
			mutate: func(sig *[RecoverableSignatureSize]byte) {
				copy(sig[32:64], scalar.Order[:])
			},
		},
		{
			name: "recid out of range",
			mutate: func(sig *[RecoverableSignatureSize]byte) {
				sig[recoverableSignatureRecIDAt] = 4
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
			if _, err := RecoverDigest(digest, badSig); err == nil {
				t.Fatal("invalid signature recovered")
			}
		})
	}
}

func TestRecoverDigestRejectsOverflowXCoordinate(t *testing.T) {
	pMinusOrder := new(big.Int).Sub(new(big.Int).Set(bigP), new(big.Int).SetBytes(scalar.Order[:]))
	rBytes := bigTo32(pMinusOrder)

	var digest [32]byte
	var sig [RecoverableSignatureSize]byte
	copy(sig[:32], rBytes[:])
	sig[63] = 1
	sig[recoverableSignatureRecIDAt] = 2

	if _, err := RecoverDigest(digest, sig); err == nil {
		t.Fatal("recovered signature with x-coordinate equal to field modulus")
	}
}

func TestParsePublicKeyRejectsInvalidEncodings(t *testing.T) {
	compressedBadPrefix := make([]byte, PublicKeyCompressedSize)
	compressedBadPrefix[0] = 0x05

	compressedXModulus := make([]byte, PublicKeyCompressedSize)
	compressedXModulus[0] = 0x02
	copy(compressedXModulus[1:], field.Modulus[:])

	uncompressedBadPrefix := make([]byte, PublicKeyUncompressedSize)
	uncompressedBadPrefix[0] = 0x05

	uncompressedXModulus := make([]byte, PublicKeyUncompressedSize)
	uncompressedXModulus[0] = 0x04
	copy(uncompressedXModulus[1:33], field.Modulus[:])
	copy(uncompressedXModulus[33:], gyBytes[:])

	uncompressedYModulus := make([]byte, PublicKeyUncompressedSize)
	uncompressedYModulus[0] = 0x04
	copy(uncompressedYModulus[1:33], gxBytes[:])
	copy(uncompressedYModulus[33:], field.Modulus[:])

	uncompressedOffCurve := make([]byte, PublicKeyUncompressedSize)
	uncompressedOffCurve[0] = 0x04
	copy(uncompressedOffCurve[1:33], gxBytes[:])

	tests := []struct {
		name string
		in   []byte
	}{
		{name: "empty", in: nil},
		{name: "short compressed", in: []byte{0x02}},
		{name: "bad compressed prefix", in: compressedBadPrefix},
		{name: "compressed x equals modulus", in: compressedXModulus},
		{name: "bad uncompressed prefix", in: uncompressedBadPrefix},
		{name: "uncompressed x equals modulus", in: uncompressedXModulus},
		{name: "uncompressed y equals modulus", in: uncompressedYModulus},
		{name: "uncompressed off curve", in: uncompressedOffCurve},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if pub, err := ParsePublicKey(tc.in); err == nil {
				t.Fatalf("ParsePublicKey(%x) = %#v, want error", tc.in, pub)
			}
		})
	}
}

func FuzzParsePublicKey(f *testing.F) {
	var one [32]byte
	one[31] = 1
	priv, err := NewPrivateKey(one)
	if err != nil {
		f.Fatal(err)
	}
	pub := priv.Public()
	compressed := pub.BytesCompressed()
	uncompressed := pub.BytesUncompressed()

	f.Add(compressed[:])
	f.Add(uncompressed[:])
	f.Add([]byte{})
	f.Add([]byte{0x02})
	f.Add(append([]byte{0x02}, make([]byte, 32)...))
	f.Add(append([]byte{0x04}, make([]byte, 64)...))

	f.Fuzz(func(t *testing.T, encoded []byte) {
		pub, err := ParsePublicKey(encoded)
		if err != nil {
			return
		}

		compressed := pub.BytesCompressed()
		parsedCompressed, err := ParsePublicKey(compressed[:])
		if err != nil {
			t.Fatalf("compressed encoding did not parse: %v", err)
		}
		if !pub.Equal(parsedCompressed) {
			t.Fatalf("compressed round trip mismatch for %x", encoded)
		}

		uncompressed := pub.BytesUncompressed()
		parsedUncompressed, err := ParsePublicKey(uncompressed[:])
		if err != nil {
			t.Fatalf("uncompressed encoding did not parse: %v", err)
		}
		if !pub.Equal(parsedUncompressed) {
			t.Fatalf("uncompressed round trip mismatch for %x", encoded)
		}
		if !parsedCompressed.Equal(parsedUncompressed) {
			t.Fatalf("compressed/uncompressed mismatch for %x", encoded)
		}
	})
}

func FuzzSignVerifyRecoverDigest(f *testing.F) {
	addSignVerifySeed(f, must32("01"), sha256.Sum256([]byte("fuzz secp256k1 one")))
	addSignVerifySeed(f, must32("2a"), sha256.Sum256([]byte("fuzz secp256k1 forty-two")))
	addSignVerifySeed(f, must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8"), sha256.Sum256([]byte("fuzz secp256k1 known key")))
	addSignVerifySeed(f, [32]byte{}, sha256.Sum256([]byte("fuzz secp256k1 zero key")))
	addSignVerifySeed(f, scalar.Order, sha256.Sum256([]byte("fuzz secp256k1 order key")))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < PrivateKeySize+32 {
			return
		}

		var keyBytes, digest [32]byte
		copy(keyBytes[:], input[:PrivateKeySize])
		copy(digest[:], input[PrivateKeySize:PrivateKeySize+32])

		priv, err := NewPrivateKey(keyBytes)
		if err != nil {
			return
		}
		pub := priv.Public()

		sig1, err := priv.SignDigest(digest)
		if err != nil {
			t.Fatalf("SignDigest failed: %v", err)
		}
		sig2, err := priv.SignDigest(digest)
		if err != nil {
			t.Fatalf("second SignDigest failed: %v", err)
		}
		if sig1 != sig2 {
			t.Fatalf("signature is not deterministic\n first %x\nsecond %x", sig1, sig2)
		}
		if !VerifyDigest(pub, digest, sig1) {
			t.Fatalf("signature did not verify\nkey %x\ndigest %x\nsig %x", keyBytes, digest, sig1)
		}

		recovered, err := RecoverDigest(digest, sig1)
		if err != nil {
			t.Fatalf("RecoverDigest failed: %v\nkey %x\ndigest %x\nsig %x", err, keyBytes, digest, sig1)
		}
		if !recovered.Equal(pub) {
			t.Fatalf("recovered public key mismatch\nkey %x\ndigest %x\nsig %x", keyBytes, digest, sig1)
		}

		var sBytes [32]byte
		copy(sBytes[:], sig1[32:64])
		if bytes.Compare(sBytes[:], scalar.HalfOrder[:]) > 0 {
			t.Fatalf("signature is not low-S: %x", sBytes)
		}
	})
}

func FuzzRecoverDigest(f *testing.F) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := NewPrivateKey(privBytes)
	if err != nil {
		f.Fatal(err)
	}
	digest := sha256.Sum256([]byte("fuzz secp256k1 recover"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(digest[:], sig[:])
	f.Add(make([]byte, 32), make([]byte, RecoverableSignatureSize))
	f.Add([]byte{}, []byte{})

	f.Fuzz(func(t *testing.T, digestBytes, sigBytes []byte) {
		if len(digestBytes) != 32 || len(sigBytes) != RecoverableSignatureSize {
			return
		}

		var digest [32]byte
		var sig [RecoverableSignatureSize]byte
		copy(digest[:], digestBytes)
		copy(sig[:], sigBytes)

		recovered, err := RecoverDigest(digest, sig)
		if err != nil {
			return
		}
		if !VerifyDigest(recovered, digest, sig) {
			t.Fatalf("recovered key does not verify signature\ndigest %x\nsig %x", digest, sig)
		}

		compressed := recovered.BytesCompressed()
		parsed, err := ParsePublicKey(compressed[:])
		if err != nil {
			t.Fatalf("recovered key compressed encoding did not parse: %v", err)
		}
		if !recovered.Equal(parsed) {
			t.Fatalf("recovered key compressed round trip mismatch")
		}
	})
}

func addSignVerifySeed(f *testing.F, key [PrivateKeySize]byte, digest [32]byte) {
	seed := make([]byte, PrivateKeySize+32)
	copy(seed[:PrivateKeySize], key[:])
	copy(seed[PrivateKeySize:], digest[:])
	f.Add(seed)
}

func must32(s string) [32]byte {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad hex")
	}
	var out [32]byte
	b := n.Bytes()
	copy(out[32-len(b):], b)
	return out
}

var (
	bigP  = new(big.Int).SetBytes([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xfc, 0x2f})
	bigGX = new(big.Int).SetBytes(gxBytes[:])
	bigGY = new(big.Int).SetBytes(gyBytes[:])
)

func bigScalarBaseMult(k *big.Int) (*big.Int, *big.Int) {
	var rx, ry *big.Int
	for i := k.BitLen() - 1; i >= 0; i-- {
		if rx != nil {
			rx, ry = bigPointDouble(rx, ry)
		}
		if k.Bit(i) == 1 {
			rx, ry = bigPointAdd(rx, ry, bigGX, bigGY)
		}
	}
	return rx, ry
}

func bigPointAdd(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	if x1 == nil {
		return new(big.Int).Set(x2), new(big.Int).Set(y2)
	}
	if x2 == nil {
		return new(big.Int).Set(x1), new(big.Int).Set(y1)
	}
	if x1.Cmp(x2) == 0 {
		sumY := new(big.Int).Add(y1, y2)
		sumY.Mod(sumY, bigP)
		if sumY.Sign() == 0 {
			return nil, nil
		}
		return bigPointDouble(x1, y1)
	}
	num := new(big.Int).Sub(y2, y1)
	den := new(big.Int).Sub(x2, x1)
	den.ModInverse(den.Mod(den, bigP), bigP)
	lambda := num.Mul(num, den)
	lambda.Mod(lambda, bigP)
	x3 := new(big.Int).Mul(lambda, lambda)
	x3.Sub(x3, x1)
	x3.Sub(x3, x2)
	x3.Mod(x3, bigP)
	y3 := new(big.Int).Sub(x1, x3)
	y3.Mul(lambda, y3)
	y3.Sub(y3, y1)
	y3.Mod(y3, bigP)
	return x3, y3
}

func bigPointDouble(x1, y1 *big.Int) (*big.Int, *big.Int) {
	if y1.Sign() == 0 {
		return nil, nil
	}
	num := new(big.Int).Mul(x1, x1)
	num.Mul(num, big.NewInt(3))
	den := new(big.Int).Lsh(y1, 1)
	den.ModInverse(den.Mod(den, bigP), bigP)
	lambda := num.Mul(num, den)
	lambda.Mod(lambda, bigP)
	x3 := new(big.Int).Mul(lambda, lambda)
	twoX := new(big.Int).Lsh(x1, 1)
	x3.Sub(x3, twoX)
	x3.Mod(x3, bigP)
	y3 := new(big.Int).Sub(x1, x3)
	y3.Mul(lambda, y3)
	y3.Sub(y3, y1)
	y3.Mod(y3, bigP)
	return x3, y3
}

func bigTo32(n *big.Int) [32]byte {
	var out [32]byte
	b := n.Bytes()
	copy(out[32-len(b):], b)
	return out
}
