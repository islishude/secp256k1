package field

import (
	"math/big"
	"testing"
)

func FuzzElementArithmetic(f *testing.F) {
	max := fromHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2e")
	large := fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	nearHalf := fromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	f.Add([]byte{}, []byte{})
	f.Add([]byte{0x01}, []byte{0x02})
	f.Add(Modulus[:], max[:])
	f.Add(large[:], nearHalf[:])

	f.Fuzz(func(t *testing.T, xInput, yInput []byte) {
		xRaw := fuzzBytesToField(xInput)
		yRaw := fuzzBytesToField(yInput)

		wantXCanonical := new(big.Int).SetBytes(xRaw[:]).Cmp(modulusBig) < 0
		wantYCanonical := new(big.Int).SetBytes(yRaw[:]).Cmp(modulusBig) < 0
		var rawX, rawY Element
		if got := rawX.SetBytes(&xRaw); got != wantXCanonical {
			t.Fatalf("SetBytes(%x) = %v, want %v", xRaw, got, wantXCanonical)
		}
		if got := rawY.SetBytes(&yRaw); got != wantYCanonical {
			t.Fatalf("SetBytes(%x) = %v, want %v", yRaw, got, wantYCanonical)
		}

		xBytes := fieldBigToBytes(new(big.Int).Mod(new(big.Int).SetBytes(xRaw[:]), modulusBig))
		yBytes := fieldBigToBytes(new(big.Int).Mod(new(big.Int).SetBytes(yRaw[:]), modulusBig))

		var x, y, got Element
		if !x.SetBytes(&xBytes) || !y.SetBytes(&yBytes) {
			t.Fatal("reduced field element did not parse")
		}
		bx := new(big.Int).SetBytes(xBytes[:])
		by := new(big.Int).SetBytes(yBytes[:])

		got.Add(&x, &y)
		assertFieldBig(t, &got, new(big.Int).Mod(new(big.Int).Add(bx, by), modulusBig))

		got.Sub(&x, &y)
		assertFieldBig(t, &got, new(big.Int).Mod(new(big.Int).Sub(bx, by), modulusBig))

		got.Neg(&x)
		assertFieldBig(t, &got, new(big.Int).Mod(new(big.Int).Neg(bx), modulusBig))

		got.Mul(&x, &y)
		assertFieldBig(t, &got, new(big.Int).Mod(new(big.Int).Mul(bx, by), modulusBig))

		var square Element
		square.Square(&x)
		assertFieldBig(t, &square, new(big.Int).Mod(new(big.Int).Mul(bx, bx), modulusBig))

		var inv, product Element
		inv.Inv(&x)
		if x.IsZero() {
			if !inv.IsZero() {
				t.Fatalf("inverse of zero = %x, want zero", inv.Bytes())
			}
		} else {
			product.Mul(&x, &inv)
			var one Element
			one.SetOne()
			if !product.Equal(&one) {
				t.Fatalf("inverse mismatch for %x", xBytes)
			}
		}

		var root, check Element
		if !root.Sqrt(&square) {
			t.Fatalf("sqrt failed for square of %x", xBytes)
		}
		check.Square(&root)
		if !check.Equal(&square) {
			t.Fatalf("sqrt candidate mismatch for %x", xBytes)
		}
	})
}

func fuzzBytesToField(in []byte) [Size]byte {
	var out [Size]byte
	if len(in) > Size {
		in = in[:Size]
	}
	copy(out[Size-len(in):], in)
	return out
}

func fieldBigToBytes(n *big.Int) [Size]byte {
	var out [Size]byte
	b := n.Bytes()
	copy(out[Size-len(b):], b)
	return out
}
