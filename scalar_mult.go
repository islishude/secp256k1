package secp256k1

import "github.com/islishude/secp256k1/internal/scalar"

func scalarBaseMult(k *scalar.Element) point {
	b := k.Bytes()
	defer clear(b[:])
	var r projectivePoint
	r.setInfinity()
	for i := range generatorAffineTable {
		digit := nibbleAt(&b, i)
		selected := generatorAffineTable[i][1]
		for j := 2; j < len(generatorAffineTable[i]); j++ {
			// Scan the whole window table and conditionally select instead of
			// indexing by the secret scalar nibble.
			selected.selectPoint(&selected, &generatorAffineTable[i][j], equalByte(digit, byte(j)))
		}
		var sum projectivePoint
		sum.addCompleteMixed(&r, &selected)
		r.selectPoint(&r, &sum, equalByte(digit, 0)^1)
	}
	return r.jacobian()
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

// doubleScalarBaseMultPrecomputedVartime computes k1*G + k2*P with variable-time
// wNAF and table lookups. It is only for verification/recovery inputs.
func doubleScalarBaseMultPrecomputedVartime(k1, k2 *scalar.Element, p2Table, p2EndoTable *[varWNAFTableSize]affinePoint) point {
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
