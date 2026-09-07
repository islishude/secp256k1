package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	baseWindow           = 5
	baseWindows          = (256 + baseWindow - 1) / baseWindow
	baseTableSize        = 1 << (baseWindow - 1)
	baseWindowW6         = 6
	baseWindowsW6        = (256 + baseWindowW6 - 1) / baseWindowW6
	baseTableSizeW6      = 1 << (baseWindowW6 - 1)
	verifyCombWidth      = 7
	verifyCombRows       = (256 + verifyCombWidth - 1) / verifyCombWidth
	verifyCombTableSize  = 1 << verifyCombWidth
	verifyCombBuildAfter = 8
)

func scalarBaseMult(k *scalar.Element) point {
	x, y, ok := scalarBaseMultAffine(k)
	if !ok {
		var out point
		out.setInfinity()
		return out
	}
	var out point
	out.setAffine(&x, &y)
	return out
}

func scalarBaseMultAffine(k *scalar.Element) (field.Element, field.Element, bool) {
	r := scalarBaseMultProjective(k)
	return r.affine()
}

func fixedWindowDigit(words *[4]uint64, windowIndex, window uint) byte {
	bit := windowIndex * window
	wordIndex := bit / 64
	shift := bit % 64
	digit := words[wordIndex] >> shift
	if shift+window > 64 && wordIndex+1 < uint(len(words)) {
		digit |= words[wordIndex+1] << (64 - shift)
	}
	return byte(digit & ((1 << window) - 1))
}

func doubleScalarBaseMultVartime(k1 *scalar.Element, p2 *point, k2 *scalar.Element) point {
	p2Table := newAffineOddTable(p2)
	p2EndoTable := newEndomorphismWNAFTable(&p2Table)
	return doubleScalarBaseMultPrecomputedVartime(k1, k2, &p2Table, &p2EndoTable)
}

// doubleScalarBaseMultCombVartime computes k1*G + k2*P using fixed-base comb
// tables. The scalars, table indices, and both points are public inputs.
func doubleScalarBaseMultCombVartime(k1, k2 *scalar.Element, p2Table *[verifyCombTableSize]affinePoint) point {
	k1Words := k1.Words()
	k2Words := k2.Words()
	var r point
	r.setInfinity()
	for row := verifyCombRows - 1; row >= 0; row-- {
		r.doubleVartime(&r)
		addCombPointVartime(&r, &generatorCombTable, verifyCombDigit(&k1Words, row))
		addCombPointVartime(&r, p2Table, verifyCombDigit(&k2Words, row))
	}
	return r
}

func verifyCombDigit(words *[4]uint64, row int) byte {
	var digit byte
	for column := range verifyCombWidth {
		bit := row + column*verifyCombRows
		if bit < 256 {
			digit |= byte((words[bit/64]>>uint(bit%64))&1) << uint(column)
		}
	}
	return digit
}

func addCombPointVartime(r *point, table *[verifyCombTableSize]affinePoint, digit byte) {
	if digit == 0 {
		return
	}
	entry := &table[digit]
	r.addAffineWNAFVartime(r, &entry.x, &entry.y)
}

// doubleScalarBaseMultPrecomputedVartime computes k1*G + k2*P with variable-time
// wNAF and table lookups. It is only for verification/recovery inputs.
func doubleScalarBaseMultPrecomputedVartime(k1, k2 *scalar.Element, p2Table, p2EndoTable *[varWNAFTableSize]affinePoint) point {
	k1a, k1b := scalar.SplitEndomorphismVartimeWords(k1.Words())
	k2a, k2b := scalar.SplitEndomorphismVartimeWords(k2.Words())
	var k1aNAF, k1bNAF, k2aNAF, k2bNAF [halfWNAFSize]int16
	k1aLen := signedWNAFHalfVartime(&k1aNAF, k1a, generatorWNAFWindow)
	k1bLen := signedWNAFHalfVartime(&k1bNAF, k1b, generatorWNAFWindow)
	k2aLen := signedWNAFHalfVartime(&k2aNAF, k2a, varWNAFWindow)
	k2bLen := signedWNAFHalfVartime(&k2bNAF, k2b, varWNAFWindow)
	n := max(k1aLen, k1bLen, k2aLen, k2bLen)

	var r point
	r.setInfinity()
	for i := n - 1; i >= 0; i-- {
		r.doubleVartime(&r)
		addGeneratorWNAFPointVartime(&r, &generatorWNAFTable, k1aNAF[i])
		addGeneratorWNAFPointVartime(&r, &generatorEndoWNAFTable, k1bNAF[i])
		addVariableWNAFPointVartime(&r, p2Table, k2aNAF[i])
		addVariableWNAFPointVartime(&r, p2EndoTable, k2bNAF[i])
	}
	return r
}
