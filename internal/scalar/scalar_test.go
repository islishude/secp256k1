package scalar

import (
	"math/big"
	"testing"

	"github.com/islishude/secp256k1/internal/field"
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

func TestSquareMatchesMulDifferential(t *testing.T) {
	check := func(i int, words [4]uint64) {
		t.Helper()
		var x, want, got Element
		x.SetWordsModOrder(words)
		want.Mul(&x, &x)
		got.Square(&x)
		if !got.Equal(&want) {
			t.Fatalf("Square mismatch at input %d: got %x want %x", i, got.Bytes(), want.Bytes())
		}
		got.Set(&x)
		got.Square(&got)
		if !got.Equal(&want) {
			t.Fatalf("aliased Square mismatch at input %d: got %x want %x", i, got.Bytes(), want.Bytes())
		}

		var squareNWant Element
		squareNWant.Mul(&want, &want)
		got.SquareN(&x, 2)
		if !got.Equal(&squareNWant) {
			t.Fatalf("SquareN mismatch at input %d: got %x want %x", i, got.Bytes(), squareNWant.Bytes())
		}
		got.Set(&x)
		got.SquareN(&got, 2)
		if !got.Equal(&squareNWant) {
			t.Fatalf("aliased SquareN mismatch at input %d: got %x want %x", i, got.Bytes(), squareNWant.Bytes())
		}
	}

	edges := [][4]uint64{
		{},
		{1, 0, 0, 0},
		{orderLimb3 - 1, orderLimb2, orderLimb1, orderLimb0},
	}
	for i, words := range edges {
		check(i, words)
		var x Element
		x.SetWordsModOrder(words)
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

	state := uint64(0x243f6a8885a308d3)
	for i := range 100_000 {
		var words [4]uint64
		for j := range words {
			state = state*6364136223846793005 + 1442695040888963407
			words[j] = state
		}
		check(i+len(edges), words)
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
	got := SetBytesModOrder(&Order)
	want := [Size]byte{}
	if got != want {
		t.Fatalf("reduction mismatch: got %x want %x", got, want)
	}
}

func TestFieldElementModOrder(t *testing.T) {
	orderMinusOne := new(big.Int).Sub(new(big.Int).Set(orderBig), big.NewInt(1))
	orderPlusOne := new(big.Int).Add(new(big.Int).Set(orderBig), big.NewInt(1))
	pMinusOne := new(big.Int).Sub(new(big.Int).SetBytes(field.Modulus[:]), big.NewInt(1))

	tests := []struct {
		name string
		in   [Size]byte
	}{
		{name: "zero", in: fromHex("00")},
		{name: "one", in: fromHex("01")},
		{name: "order minus one", in: scalarBigToBytes(orderMinusOne)},
		{name: "order", in: Order},
		{name: "order plus one", in: scalarBigToBytes(orderPlusOne)},
		{name: "p minus one", in: scalarBigToBytes(pMinusOne)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var x field.Element
			if !x.SetBytes(&tc.in) {
				t.Fatalf("field SetBytes(%x) failed", tc.in)
			}

			var got Element
			got.SetFieldElementModOrder(&x)
			want := scalarBigToBytes(new(big.Int).Mod(new(big.Int).SetBytes(tc.in[:]), orderBig))
			if gotBytes := got.Bytes(); gotBytes != want {
				t.Fatalf("SetFieldElementModOrder(%x) = %x, want %x", tc.in, gotBytes, want)
			}

			wantLess := new(big.Int).SetBytes(tc.in[:]).Cmp(orderBig) < 0
			if gotLess := FieldElementLessThanOrder(&x); gotLess != wantLess {
				t.Fatalf("FieldElementLessThanOrder(%x) = %v, want %v", tc.in, gotLess, wantLess)
			}
		})
	}
}

func TestIsHigh(t *testing.T) {
	halfPlusOne := scalarBigToBytes(new(big.Int).Add(new(big.Int).SetBytes(HalfOrder[:]), big.NewInt(1)))
	orderMinusOne := scalarBigToBytes(new(big.Int).Sub(new(big.Int).Set(orderBig), big.NewInt(1)))
	tests := []struct {
		name string
		in   [Size]byte
		want bool
	}{
		{name: "zero", in: fromHex("00"), want: false},
		{name: "low with large least significant limb", in: fromHex("000000000000000000000000000000000000000000000000ffffffffffffffff"), want: false},
		{name: "half order", in: HalfOrder, want: false},
		{name: "half order plus one", in: halfPlusOne, want: true},
		{name: "order minus one", in: orderMinusOne, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var x Element
			if !x.SetBytes(&tc.in) {
				t.Fatalf("SetBytes(%x) failed", tc.in)
			}
			if got := x.IsHigh(); got != tc.want {
				t.Fatalf("IsHigh(%x) = %v, want %v", tc.in, got, tc.want)
			}
			if got := x.IsHighChoice() == 1; got != tc.want {
				t.Fatalf("IsHighChoice(%x) = %v, want %v", tc.in, got, tc.want)
			}
			if got := IsHighBytes(&tc.in); got != tc.want {
				t.Fatalf("IsHighBytes(%x) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestWordHelpers(t *testing.T) {
	values := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("02"),
		HalfOrder,
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}

	if LessThanOrderWords(bytesToWords(&Order)) {
		t.Fatal("LessThanOrderWords accepted order")
	}

	for _, vb := range values {
		var v Element
		if !v.SetBytes(&vb) {
			t.Fatalf("SetBytes(%x) failed", vb)
		}
		words := v.Words()
		if !LessThanOrderWords(words) {
			t.Fatalf("LessThanOrderWords(%x) = false", vb)
		}

		var neg Element
		neg.Neg(&v)
		if got, want := wordsToBytes(NegWords(words)), neg.Bytes(); got != want {
			t.Fatalf("NegWords(%x) = %x, want %x", vb, got, want)
		}
	}
}

func TestSelect(t *testing.T) {
	oneBytes := fromHex("01")
	twoBytes := fromHex("02")
	var one, two, got Element
	if !one.SetBytes(&oneBytes) || !two.SetBytes(&twoBytes) {
		t.Fatal("SetBytes failed")
	}
	got.Select(&one, &two, 0)
	if got.Bytes() != oneBytes {
		t.Fatal("Select choice 0 did not select first scalar")
	}
	got.Select(&one, &two, 1)
	if got.Bytes() != twoBytes {
		t.Fatal("Select choice 1 did not select second scalar")
	}
	got.Select(&one, &two, 3)
	if got.Bytes() != twoBytes {
		t.Fatal("Select did not mask choice to low bit")
	}
}

func TestSplitEndomorphismAgainstBig(t *testing.T) {
	lambdaBytes := fromHex("5363ad4cc05c30e0a5261c028812645a122e22ea20816678df02967c1b23bd72")
	lambdaBig := new(big.Int).SetBytes(lambdaBytes[:])
	halfOrderBig := new(big.Int).Rsh(new(big.Int).Set(orderBig), 1)
	values := [][Size]byte{
		fromHex("0000000000000000000000000000000000000000000000000000000000000000"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000001"),
		fromHex("0000000000000000000000000000000000000000000000000000000000000002"),
		lambdaBytes,
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}

	for _, kb := range values {
		var k Element
		if !k.SetBytes(&kb) {
			t.Fatalf("SetBytes failed")
		}
		k1, k2 := SplitEndomorphism(&k)
		k1Bytes := k1.Bytes()
		k2Bytes := k2.Bytes()
		k1Big := new(big.Int).SetBytes(k1Bytes[:])
		k2Big := new(big.Int).SetBytes(k2Bytes[:])

		got := new(big.Int).Mul(k2Big, lambdaBig)
		got.Add(got, k1Big)
		got.Mod(got, orderBig)
		want := new(big.Int).SetBytes(kb[:])
		if got.Cmp(want) != 0 {
			t.Fatalf("split relation mismatch for %x: got %x want %x", kb, got, want)
		}

		for _, part := range []*big.Int{k1Big, k2Big} {
			if part.Cmp(halfOrderBig) > 0 {
				part.Sub(orderBig, part)
			}
			if part.BitLen() > 130 {
				t.Fatalf("split component too large for %x: bit length %d", kb, part.BitLen())
			}
		}
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
