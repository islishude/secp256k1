package scalar

import (
	"math/big"
	"testing"
)

func FuzzElementArithmetic(f *testing.F) {
	max := fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140")
	nearHalf := fromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	f.Add([]byte{}, []byte{})
	f.Add([]byte{0x01}, []byte{0x02})
	f.Add(Order[:], HalfOrder[:])
	f.Add(max[:], nearHalf[:])

	f.Fuzz(func(t *testing.T, xInput, yInput []byte) {
		xRaw := fuzzBytesToScalar(xInput)
		yRaw := fuzzBytesToScalar(yInput)

		wantXCanonical := new(big.Int).SetBytes(xRaw[:]).Cmp(orderBig) < 0
		wantYCanonical := new(big.Int).SetBytes(yRaw[:]).Cmp(orderBig) < 0
		var rawX, rawY Element
		if got := rawX.SetBytes(&xRaw); got != wantXCanonical {
			t.Fatalf("SetBytes(%x) = %v, want %v", xRaw, got, wantXCanonical)
		}
		if got := rawY.SetBytes(&yRaw); got != wantYCanonical {
			t.Fatalf("SetBytes(%x) = %v, want %v", yRaw, got, wantYCanonical)
		}

		xBytes := SetBytesModOrder(&xRaw)
		yBytes := SetBytesModOrder(&yRaw)
		if got, want := xBytes, scalarBigToBytes(new(big.Int).Mod(new(big.Int).SetBytes(xRaw[:]), orderBig)); got != want {
			t.Fatalf("SetBytesModOrder(%x) = %x, want %x", xRaw, got, want)
		}
		if got, want := yBytes, scalarBigToBytes(new(big.Int).Mod(new(big.Int).SetBytes(yRaw[:]), orderBig)); got != want {
			t.Fatalf("SetBytesModOrder(%x) = %x, want %x", yRaw, got, want)
		}

		var x, y, got Element
		if !x.SetBytes(&xBytes) || !y.SetBytes(&yBytes) {
			t.Fatal("reduced scalar did not parse")
		}
		bx := new(big.Int).SetBytes(xBytes[:])
		by := new(big.Int).SetBytes(yBytes[:])

		got.Add(&x, &y)
		assertScalarBig(t, &got, new(big.Int).Mod(new(big.Int).Add(bx, by), orderBig))

		got.Sub(&x, &y)
		assertScalarBig(t, &got, new(big.Int).Mod(new(big.Int).Sub(bx, by), orderBig))

		got.Neg(&x)
		assertScalarBig(t, &got, new(big.Int).Mod(new(big.Int).Neg(bx), orderBig))

		got.Mul(&x, &y)
		assertScalarBig(t, &got, new(big.Int).Mod(new(big.Int).Mul(bx, by), orderBig))

		got.Square(&x)
		assertScalarBig(t, &got, new(big.Int).Mod(new(big.Int).Mul(bx, bx), orderBig))

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
	})
}

func fuzzBytesToScalar(in []byte) [Size]byte {
	var out [Size]byte
	if len(in) > Size {
		in = in[:Size]
	}
	copy(out[Size-len(in):], in)
	return out
}

func scalarBigToBytes(n *big.Int) [Size]byte {
	var out [Size]byte
	b := n.Bytes()
	copy(out[Size-len(b):], b)
	return out
}
