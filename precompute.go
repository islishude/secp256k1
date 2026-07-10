package secp256k1

import "github.com/islishude/secp256k1/internal/field"

//go:generate go run ./cmd/genprecomp

func newGeneratorAffineTableW4() [64][16]affinePoint {
	var table [64][16]affinePoint
	base := generator
	for i := range table {
		table[i][0].infinity = 1

		var points [15]point
		points[0].set(&base)
		for j := 1; j < len(points); j++ {
			points[j].add(&points[j-1], &base)
		}
		affine := batchNormalize15(points)
		for j := 1; j < len(table[i]); j++ {
			table[i][j] = affine[j-1]
		}

		// Move to the next 4-bit window: base *= 16.
		for range 4 {
			base.double(&base)
		}
	}
	return table
}

func newGeneratorAffineTableW5() [baseWindows][baseTableSize]affinePoint {
	var table [baseWindows][baseTableSize]affinePoint
	base := generator
	for i := range table {
		var points [baseTableSize]point
		points[0].set(&base)
		for j := 1; j < len(points); j++ {
			points[j].add(&points[j-1], &base)
		}
		affine := batchNormalize16(points)
		for j := range table[i] {
			table[i][j] = affine[j]
		}

		for range baseWindow {
			base.double(&base)
		}
	}
	return table
}

func newAffineOddTable(p *point) [varWNAFTableSize]affinePoint {
	var points [varWNAFTableSize]point
	points[0].set(p)
	if len(points) > 1 {
		var twoP point
		twoP.double(p)
		for i := 1; i < len(points); i++ {
			points[i].add(&points[i-1], &twoP)
		}
	}
	return batchNormalize(points)
}

func newGeneratorWNAFTable() [generatorWNAFSize]affinePoint {
	var points [generatorWNAFSize]point
	points[0].set(&generator)
	if len(points) > 1 {
		var twoG point
		twoG.double(&generator)
		for i := 1; i < len(points); i++ {
			points[i].add(&points[i-1], &twoG)
		}
	}
	return batchNormalizeGenerator(points)
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

func newGeneratorEndomorphismWNAFTable(table *[generatorWNAFSize]affinePoint) [generatorWNAFSize]affinePoint {
	var out [generatorWNAFSize]affinePoint
	for i := range table {
		out[i].y.Set(&table[i].y)
		out[i].infinity = table[i].infinity
		out[i].x.Mul(&table[i].x, &endoBeta)
	}
	return out
}

func loadGeneratorAffineTableW5(words *[baseWindows][baseTableSize][8]uint64) [baseWindows][baseTableSize]affinePoint {
	var table [baseWindows][baseTableSize]affinePoint
	for i := range table {
		for j := range table[i] {
			table[i][j].x.SetMontgomeryWords([4]uint64{
				words[i][j][0], words[i][j][1], words[i][j][2], words[i][j][3],
			})
			table[i][j].y.SetMontgomeryWords([4]uint64{
				words[i][j][4], words[i][j][5], words[i][j][6], words[i][j][7],
			})
		}
	}
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

func batchNormalize(points [varWNAFTableSize]point) [varWNAFTableSize]affinePoint {
	var prefixes [varWNAFTableSize]field.Element
	var acc field.Element
	acc.SetOne()
	for i := range points {
		prefixes[i].Set(&acc)
		acc.Mul(&acc, &points[i].z)
	}

	var accInv field.Element
	accInv.Inv(&acc)

	var out [varWNAFTableSize]affinePoint
	for i := len(points) - 1; i >= 0; i-- {
		var zInv, z2, z3 field.Element
		zInv.Mul(&accInv, &prefixes[i])
		accInv.Mul(&accInv, &points[i].z)

		z2.Square(&zInv)
		z3.Mul(&z2, &zInv)
		out[i].x.Mul(&points[i].x, &z2)
		out[i].y.Mul(&points[i].y, &z3)
	}
	return out
}

func batchNormalizeGenerator(points [generatorWNAFSize]point) [generatorWNAFSize]affinePoint {
	var prefixes [generatorWNAFSize]field.Element
	var acc field.Element
	acc.SetOne()
	for i := range points {
		prefixes[i].Set(&acc)
		acc.Mul(&acc, &points[i].z)
	}

	var accInv field.Element
	accInv.Inv(&acc)

	var out [generatorWNAFSize]affinePoint
	for i := len(points) - 1; i >= 0; i-- {
		var zInv, z2, z3 field.Element
		zInv.Mul(&accInv, &prefixes[i])
		accInv.Mul(&accInv, &points[i].z)

		z2.Square(&zInv)
		z3.Mul(&z2, &zInv)
		out[i].x.Mul(&points[i].x, &z2)
		out[i].y.Mul(&points[i].y, &z3)
	}
	return out
}

func batchNormalize15(points [15]point) [15]affinePoint {
	var prefixes [15]field.Element
	var acc field.Element
	acc.SetOne()
	for i := range points {
		prefixes[i].Set(&acc)
		acc.Mul(&acc, &points[i].z)
	}

	var accInv field.Element
	accInv.Inv(&acc)

	var out [15]affinePoint
	for i := len(points) - 1; i >= 0; i-- {
		var zInv, z2, z3 field.Element
		zInv.Mul(&accInv, &prefixes[i])
		accInv.Mul(&accInv, &points[i].z)

		z2.Square(&zInv)
		z3.Mul(&z2, &zInv)
		out[i].x.Mul(&points[i].x, &z2)
		out[i].y.Mul(&points[i].y, &z3)
	}
	return out
}

func batchNormalize16(points [16]point) [16]affinePoint {
	var prefixes [16]field.Element
	var acc field.Element
	acc.SetOne()
	for i := range points {
		prefixes[i].Set(&acc)
		acc.Mul(&acc, &points[i].z)
	}

	var accInv field.Element
	accInv.Inv(&acc)

	var out [16]affinePoint
	for i := len(points) - 1; i >= 0; i-- {
		var zInv, z2, z3 field.Element
		zInv.Mul(&accInv, &prefixes[i])
		accInv.Mul(&accInv, &points[i].z)
		z2.Square(&zInv)
		z3.Mul(&z2, &zInv)
		out[i].x.Mul(&points[i].x, &z2)
		out[i].y.Mul(&points[i].y, &z3)
	}
	return out
}
