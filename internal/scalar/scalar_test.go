package scalar

import (
	"math/big"
	"testing"
)

var orderBig = new(big.Int).SetBytes(Order[:])

func TestElementArithmeticAgainstBig(t *testing.T) {
	values := [][Size]byte{
		fromHex("0000000000000000000000000000000000000000000000000000000000000000"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000001"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000002"),
		fromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}

	for _, xb := range values {
		for _, yb := range values {
			var x, y, got Element
			if !x.SetBytes(&xb) || !y.SetBytes(&yb) {
				t.Fatalf("SetBytes failed")
			}
			bx := new(big.Int).SetBytes(xb[:])
			by := new(big.Int).SetBytes(yb[:])

			got.Add(&x, &y)
			want := new(big.Int).Add(bx, by)
			want.Mod(want, orderBig)
			assertScalarBig(t, &got, want)

			got.Sub(&x, &y)
			want.Sub(bx, by)
			want.Mod(want, orderBig)
			assertScalarBig(t, &got, want)

			got.Mul(&x, &y)
			want.Mul(bx, by)
			want.Mod(want, orderBig)
			assertScalarBig(t, &got, want)
		}
	}
}

func TestElementInv(t *testing.T) {
	values := [][Size]byte{
		fromHex("0000000000000000000000000000000000000000000000000000000000000001"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000002"),
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}
	var one Element
	one.SetOne()
	for _, xb := range values {
		var x, inv, product Element
		if !x.SetBytes(&xb) {
			t.Fatalf("SetBytes failed")
		}
		inv.Inv(&x)
		product.Mul(&x, &inv)
		if !product.Equal(&one) {
			t.Fatalf("inverse mismatch for %x", xb)
		}
	}
}

func TestReductionAndRejectsNonCanonicalEncoding(t *testing.T) {
	var x Element
	if x.SetBytes(&Order) {
		t.Fatal("accepted order as canonical scalar")
	}
	got := SetBytesModOrder(Order)
	want := [Size]byte{}
	if got != want {
		t.Fatalf("reduction mismatch: got %x want %x", got, want)
	}
}

func assertScalarBig(t *testing.T, got *Element, want *big.Int) {
	t.Helper()
	b := got.Bytes()
	gotBig := new(big.Int).SetBytes(b[:])
	if gotBig.Cmp(want) != 0 {
		t.Fatalf("got %x want %x", gotBig, want)
	}
}

func fromHex(s string) [Size]byte {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad hex")
	}
	var out [Size]byte
	b := n.Bytes()
	copy(out[Size-len(b):], b)
	return out
}
