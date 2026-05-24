package secp256k1

import (
	"math/big"
	"testing"

	"github.com/islishude/secp256k1/internal/scalar"
)

func TestGroupOrder(t *testing.T) {
	p := scalarMult(&generator, &scalar.Order)
	if !p.isInfinity() {
		t.Fatal("n*G is not infinity")
	}
}

func TestScalarBaseMultZero(t *testing.T) {
	var zero scalar.Element
	got := scalarBaseMult(&zero)
	if !got.isInfinity() {
		t.Fatal("0*G is not infinity")
	}
}

func TestScalarBaseMultAgainstBig(t *testing.T) {
	orderMinusOne := scalar.Order
	for i := len(orderMinusOne) - 1; i >= 0; i-- {
		if orderMinusOne[i] > 0 {
			orderMinusOne[i]--
			break
		}
		orderMinusOne[i] = 0xff
	}

	scalars := [][32]byte{
		must32("00"),
		must32("01"),
		must32("02"),
		must32("03"),
		must32("2a"),
		must32("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		orderMinusOne,
	}
	for _, kb := range scalars {
		var k scalar.Element
		if !k.SetBytes(&kb) {
			t.Fatalf("SetBytes(%x) failed", kb)
		}
		got := scalarBaseMult(&k)
		wantX, wantY := bigScalarBaseMult(new(big.Int).SetBytes(kb[:]))
		if wantX == nil {
			if !got.isInfinity() {
				t.Fatalf("scalar %x: got non-infinity, want infinity", kb)
			}
			continue
		}
		gotX, gotY, ok := got.affine()
		if !ok {
			t.Fatalf("scalar %x produced infinity", kb)
		}
		if gotX.Bytes() != bigTo32(wantX) || gotY.Bytes() != bigTo32(wantY) {
			t.Fatalf("scalar %x mismatch\n got (%x,%x)\nwant (%x,%x)",
				kb, gotX.Bytes(), gotY.Bytes(), wantX, wantY)
		}
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
	orderMinusOne := scalar.Order
	for i := len(orderMinusOne) - 1; i >= 0; i-- {
		if orderMinusOne[i] > 0 {
			orderMinusOne[i]--
			break
		}
		orderMinusOne[i] = 0xff
	}
	scalars := [][32]byte{
		must32("00"),
		must32("01"),
		must32("02"),
		must32("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		orderMinusOne,
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
