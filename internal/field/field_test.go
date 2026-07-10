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

func TestMulByB3Differential(t *testing.T) {
	var b3 Element
	b3.SetUint64(21)
	check := func(i int, x *Element) {
		t.Helper()
		var want, got Element
		want.Mul(x, &b3)
		got.MulByB3(x)
		if !got.Equal(&want) {
			t.Fatalf("MulByB3 mismatch at input %d: got %x want %x", i, got.Bytes(), want.Bytes())
		}
		got.Set(x)
		got.MulByB3(&got)
		if !got.Equal(&want) {
			t.Fatalf("aliased MulByB3 mismatch at input %d: got %x want %x", i, got.Bytes(), want.Bytes())
		}
	}

	edges := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e"),
	}
	for i, xb := range edges {
		var x Element
		if !x.SetBytes(&xb) {
			t.Fatalf("SetBytes failed for edge %d", i)
		}
		check(i, &x)
	}

	state := uint64(0x13198a2e03707344)
	for i := range 100_000 {
		var words [4]uint64
		for j := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[j] = state
		}
		if !LessThanModulusWords(words) {
			words[3] = 0
		}
		var x Element
		x.SetNonMontgomeryWords(words)
		check(i+len(edges), &x)
	}
}

func TestSquareAndSquareNDifferential(t *testing.T) {
	check := func(i int, x *Element) {
		t.Helper()
		var squareWant, squareGot Element
		squareWant.Mul(x, x)
		squareGot.Square(x)
		if !squareGot.Equal(&squareWant) {
			t.Fatalf("Square mismatch at input %d: got %x want %x", i, squareGot.Bytes(), squareWant.Bytes())
		}
		squareGot.Set(x)
		squareGot.Square(&squareGot)
		if !squareGot.Equal(&squareWant) {
			t.Fatalf("aliased Square mismatch at input %d: got %x want %x", i, squareGot.Bytes(), squareWant.Bytes())
		}

		var squareNWant, squareNGot Element
		squareNWant.Mul(&squareWant, &squareWant)
		squareNGot.SquareN(x, 2)
		if !squareNGot.Equal(&squareNWant) {
			t.Fatalf("SquareN mismatch at input %d: got %x want %x", i, squareNGot.Bytes(), squareNWant.Bytes())
		}
		squareNGot.Set(x)
		squareNGot.SquareN(&squareNGot, 2)
		if !squareNGot.Equal(&squareNWant) {
			t.Fatalf("aliased SquareN mismatch at input %d: got %x want %x", i, squareNGot.Bytes(), squareNWant.Bytes())
		}
	}

	edges := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e"),
	}
	for i, xb := range edges {
		var x Element
		if !x.SetBytes(&xb) {
			t.Fatalf("SetBytes failed for edge %d", i)
		}
		check(i, &x)
		for _, n := range []int{-1, 0, 1, 64} {
			var want, got Element
			want.Set(&x)
			for range max(n, 0) {
				want.Mul(&want, &want)
			}
			got.SquareN(&x, n)
			if !got.Equal(&want) {
				t.Fatalf("SquareN(%d) mismatch for edge %d: got %x want %x", n, i, got.Bytes(), want.Bytes())
			}
		}
	}

	state := uint64(0xa4093822299f31d0)
	for i := range 100_000 {
		var words [4]uint64
		for j := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[j] = state
		}
		if !LessThanModulusWords(words) {
			words[3] = 0
		}
		var x Element
		x.SetNonMontgomeryWords(words)
		check(i+len(edges), &x)
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

func TestWordEncodingHelpers(t *testing.T) {
	values := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("02"),
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e"),
	}

	if LessThanModulusWords(bytesToWords(&Modulus)) {
		t.Fatal("LessThanModulusWords accepted modulus")
	}

	for _, vb := range values {
		words := bytesToWords(&vb)
		if !LessThanModulusWords(words) {
			t.Fatalf("LessThanModulusWords(%x) = false", vb)
		}

		var got Element
		got.SetNonMontgomeryWords(words)
		if gotBytes := got.Bytes(); gotBytes != vb {
			t.Fatalf("SetNonMontgomeryWords(%x).Bytes() = %x", vb, gotBytes)
		}
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
