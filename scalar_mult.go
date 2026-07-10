package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	baseWindow           = 5
	baseWindows          = (256 + baseWindow - 1) / baseWindow
	baseTableSize        = 1 << (baseWindow - 1)
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

func scalarBaseMultProjectiveW5Affine(k *scalar.Element, table *[baseWindows][baseTableSize]affinePoint) projectivePoint {
	words := k.Words()
	defer clear(words[:])
	var r projectivePoint
	r.setInfinity()
	var carry uint64
	for i := range table {
		value := uint64(fixedWindowDigit(&words, uint(i), baseWindow)) + carry
		negative := (value + (1 << (baseWindow - 1))) >> baseWindow
		negativeMask := uint64(0) - negative
		digitBits := value - negative*(1<<baseWindow)
		magnitude := (digitBits ^ negativeMask) + negative

		selected := table[i][0]
		for j := 2; j <= len(table[i]); j++ {
			// The digit is secret. Scan the whole table and select with masks.
			selected.selectPoint(&selected, &table[i][j-1], equalByte(byte(magnitude), byte(j)))
		}
		var negY field.Element
		negY.Neg(&selected.y)
		selected.y.Select(&selected.y, &negY, negative)
		var sum projectivePoint
		sum.addCompleteMixed(&r, &selected)
		r.selectPoint(&r, &sum, equalByte(byte(magnitude), 0)^1)
		carry = negative
	}
	return r
}

func scalarBaseMultProjectiveW4(k *scalar.Element, table *[64][16]affinePoint) projectivePoint {
	b := k.Bytes()
	defer clear(b[:])
	var r projectivePoint
	r.setInfinity()
	for i := range table {
		digit := nibbleAt(&b, i)
		selected := table[i][1]
		for j := 2; j < len(table[i]); j++ {
			// Scan the whole window table and conditionally select instead of
			// indexing by the secret scalar nibble.
			selected.selectPoint(&selected, &table[i][j], equalByte(digit, byte(j)))
		}
		var sum projectivePoint
		sum.addCompleteMixed(&r, &selected)
		r.selectPoint(&r, &sum, equalByte(digit, 0)^1)
	}
	return r
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

func scalarMult(p *point, k *[32]byte) point {
	var r point
	r.setInfinity()
	for i := range 256 {
		// Left-to-right double-and-add over the scalar bits.
		var doubled point
		doubled.double(&r)
		var sum point
		sum.add(&doubled, p)
		r.selectPoint(&doubled, &sum, uint64(bitAt(k, i)))
	}
	return r
}

func scalarMultAffine(p *point, k *[32]byte) point {
	var r point
	r.setInfinity()
	if p.isInfinity() {
		return r
	}
	for i := range 256 {
		var doubled point
		doubled.double(&r)
		var sum point
		sum.addAffine(&doubled, &p.x, &p.y)
		r.selectPoint(&doubled, &sum, uint64(bitAt(k, i)))
	}
	return r
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
		r.double(&r)
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
		r.double(&r)
		addGeneratorWNAFPointVartime(&r, &generatorWNAFTable, k1aNAF[i])
		addGeneratorWNAFPointVartime(&r, &generatorEndoWNAFTable, k1bNAF[i])
		addVariableWNAFPointVartime(&r, p2Table, k2aNAF[i])
		addVariableWNAFPointVartime(&r, p2EndoTable, k2bNAF[i])
	}
	return r
}

func doubleScalarBaseMultPrecomputedLegacyVartime(k1, k2 *scalar.Element, p2Table, p2EndoTable *[varWNAFTableSize]affinePoint) point {
	k1a, k1b := scalar.SplitEndomorphism(k1)
	k2a, k2b := scalar.SplitEndomorphism(k2)
	k1aNAF, k1aLen, k1aSign := signedWNAFVartime(&k1a, generatorWNAFWindow)
	k1bNAF, k1bLen, k1bSign := signedWNAFVartime(&k1b, generatorWNAFWindow)
	k2aNAF, k2aLen, k2aSign := signedWNAFVartime(&k2a, varWNAFWindow)
	k2bNAF, k2bLen, k2bSign := signedWNAFVartime(&k2b, varWNAFWindow)
	n := max(k1aLen, k1bLen, k2aLen, k2bLen)
	k1aDigits := k1aNAF[:n]
	k1bDigits := k1bNAF[:n]
	k2aDigits := k2aNAF[:n]
	k2bDigits := k2bNAF[:n]

	var r point
	r.setInfinity()
	for i := n - 1; i >= 0; i-- {
		r.double(&r)
		addGeneratorWNAFPointVartime(&r, &generatorWNAFTable, k1aDigits[i]*k1aSign)
		addGeneratorWNAFPointVartime(&r, &generatorEndoWNAFTable, k1bDigits[i]*k1bSign)
		addVariableWNAFPointVartime(&r, p2Table, k2aDigits[i]*k2aSign)
		addVariableWNAFPointVartime(&r, p2EndoTable, k2bDigits[i]*k2bSign)
	}
	return r
}
