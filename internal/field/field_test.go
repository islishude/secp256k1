package field

import (
	"math/big"
	"testing"
)

var modulusBig = new(big.Int).SetBytes(Modulus[:])

func TestElementArithmeticAgainstBig(t *testing.T) {
	values := [][Size]byte{
		fromHex("0000000000000000000000000000000000000000000000000000000000000000"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000001"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000002"),
		fromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e"),
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
			want.Mod(want, modulusBig)
			assertFieldBig(t, &got, want)

			got.Sub(&x, &y)
			want.Sub(bx, by)
			want.Mod(want, modulusBig)
			assertFieldBig(t, &got, want)

			got.Mul(&x, &y)
			want.Mul(bx, by)
			want.Mod(want, modulusBig)
			assertFieldBig(t, &got, want)
		}
	}
}

func TestElementInvAndSqrt(t *testing.T) {
	values := [][Size]byte{
		fromHex("0000000000000000000000000000000000000000000000000000000000000001"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000002"),
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e"),
	}
	var one Element
	one.SetOne()
	for _, xb := range values {
		var x, inv, product, square, root, check Element
		if !x.SetBytes(&xb) {
			t.Fatalf("SetBytes failed")
		}
		inv.Inv(&x)
		product.Mul(&x, &inv)
		if !product.Equal(&one) {
			t.Fatalf("inverse mismatch for %x", xb)
		}

		square.Square(&x)
		if !root.Sqrt(&square) {
			t.Fatalf("sqrt failed for square of %x", xb)
		}
		check.Square(&root)
		if !check.Equal(&square) {
			t.Fatalf("sqrt candidate mismatch for %x", xb)
		}
	}
}

func TestRejectsNonCanonicalEncoding(t *testing.T) {
	var x Element
	if x.SetBytes(&Modulus) {
		t.Fatal("accepted modulus as canonical field element")
	}
}

func assertFieldBig(t *testing.T, got *Element, want *big.Int) {
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
	n.FillBytes(out[:])
	return out
}
