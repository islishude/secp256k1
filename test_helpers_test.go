package secp256k1

import "math/big"

func must32(s string) [32]byte {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad hex")
	}
	var out [32]byte
	n.FillBytes(out[:])
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
	n.FillBytes(out[:])
	return out
}
