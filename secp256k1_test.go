package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"

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
