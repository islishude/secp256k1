package secp256k1

import "math/bits"

//go:generate go run ./cmd/genprecomp

func newAffineOddTable(p *point) [varWNAFTableSize]affinePoint {
	var points [varWNAFTableSize]point
	points[0].set(p)
	if len(points) > 1 {
		var twoP point
		twoP.doubleVartime(p)
		for i := 1; i < len(points); i++ {
			points[i].addVartime(&points[i-1], &twoP)
		}
	}
	return batchNormalize(points)
}

func newEndomorphismWNAFTable(table *[varWNAFTableSize]affinePoint) [varWNAFTableSize]affinePoint {
	var out [varWNAFTableSize]affinePoint
	for i := range table {
		out[i].y.Set(&table[i].y)
		out[i].infinity = table[i].infinity
		out[i].x.Mul(&table[i].x, &endoBeta)
	}
	return out
}

func newVerifyCombTable(p *point) [verifyCombTableSize]affinePoint {
	var bases [verifyCombWidth]point
	bases[0].set(p)
	for i := 1; i < len(bases); i++ {
		bases[i].set(&bases[i-1])
		for range verifyCombRows {
			bases[i].doubleVartime(&bases[i])
		}
	}

	var projective [verifyCombTableSize - 1]point
	for digit := 1; digit < verifyCombTableSize; digit++ {
		bit := bits.TrailingZeros(uint(digit))
		previous := digit & (digit - 1)
		if previous == 0 {
			projective[digit-1].set(&bases[bit])
		} else {
			projective[digit-1].addVartime(&projective[previous-1], &bases[bit])
		}
	}

	normalized := batchNormalizeVerifyComb(projective)
	var table [verifyCombTableSize]affinePoint
	table[0].infinity = 1
	copy(table[1:], normalized[:])
	return table
}

func loadGeneratorWNAFTables(
	words *[generatorWNAFSize][8]uint64,
	endoXWords *[generatorWNAFSize][4]uint64,
) ([generatorWNAFSize]affinePoint, [generatorWNAFSize]affinePoint) {
	var table, endoTable [generatorWNAFSize]affinePoint
	for i := range table {
		xWords := [4]uint64{words[i][0], words[i][1], words[i][2], words[i][3]}
		yWords := [4]uint64{words[i][4], words[i][5], words[i][6], words[i][7]}
		table[i].x.SetMontgomeryWords(xWords)
		table[i].y.SetMontgomeryWords(yWords)
		endoTable[i].x.SetMontgomeryWords(endoXWords[i])
		endoTable[i].y.SetMontgomeryWords(yWords)
	}
	return table, endoTable
}

func loadGeneratorCombTable(words *[verifyCombTableSize - 1][8]uint64) [verifyCombTableSize]affinePoint {
	var table [verifyCombTableSize]affinePoint
	table[0].infinity = 1
	for i := range words {
		table[i+1].x.SetMontgomeryWords([4]uint64{
			words[i][0], words[i][1], words[i][2], words[i][3],
		})
		table[i+1].y.SetMontgomeryWords([4]uint64{
			words[i][4], words[i][5], words[i][6], words[i][7],
		})
	}
	return table
}
