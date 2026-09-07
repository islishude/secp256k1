package secp256k1

// Reference implementations are retained only as independent test and benchmark oracles.
import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

func (p *point) doubleSquare(q *point) *point {
	if q.isInfinity() || q.y.IsZero() {
		return p.setInfinity()
	}

	x1, y1, z1 := q.x, q.y, q.z
	var xx, yy, yyyy, s, m, t field.Element

	// Jacobian doubling formula for curves with a = 0.
	xx.Square(&x1)
	yy.Square(&y1)
	yyyy.Square(&yy)

	t.Add(&x1, &yy)
	t.Square(&t)
	t.Sub(&t, &xx)
	t.Sub(&t, &yyyy)
	s.Double(&t)

	m.Double(&xx)
	m.Add(&m, &xx)

	var x3, y3, z3 field.Element
	x3.Square(&m)
	t.Double(&s)
	x3.Sub(&x3, &t)

	t.Sub(&s, &x3)
	y3.Mul(&m, &t)
	t.Double(&yyyy)
	t.Double(&t)
	t.Double(&t)
	y3.Sub(&y3, &t)

	z3.Mul(&y1, &z1)
	z3.Double(&z3)

	p.x.Set(&x3)
	p.y.Set(&y3)
	p.z.Set(&z3)
	return p
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

func scalarMultVartime(p *point, k *[32]byte) point {
	var r point
	r.setInfinity()
	for i := range 256 {
		// Left-to-right double-and-add over the scalar bits.
		var doubled point
		doubled.doubleVartime(&r)
		var sum point
		sum.addVartime(&doubled, p)
		r.selectPoint(&doubled, &sum, uint64(bitAt(k, i)))
	}
	return r
}

func scalarMultAffineVartime(p *point, k *[32]byte) point {
	var r point
	r.setInfinity()
	if p.isInfinity() {
		return r
	}
	for i := range 256 {
		var doubled point
		doubled.doubleVartime(&r)
		var sum point
		sum.addAffineVartime(&doubled, &p.x, &p.y)
		r.selectPoint(&doubled, &sum, uint64(bitAt(k, i)))
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
		r.doubleVartime(&r)
		addGeneratorWNAFPointVartime(&r, &generatorWNAFTable, k1aDigits[i]*k1aSign)
		addGeneratorWNAFPointVartime(&r, &generatorEndoWNAFTable, k1bDigits[i]*k1bSign)
		addVariableWNAFPointVartime(&r, p2Table, k2aDigits[i]*k2aSign)
		addVariableWNAFPointVartime(&r, p2EndoTable, k2bDigits[i]*k2bSign)
	}
	return r
}

func newGeneratorAffineTableW4() [64][16]affinePoint {
	var table [64][16]affinePoint
	base := generator
	for i := range table {
		table[i][0].infinity = 1

		var points [15]point
		points[0].set(&base)
		for j := 1; j < len(points); j++ {
			points[j].addVartime(&points[j-1], &base)
		}
		affine := batchNormalize15(points)
		for j := 1; j < len(table[i]); j++ {
			table[i][j] = affine[j-1]
		}

		// Move to the next 4-bit window: base *= 16.
		for range 4 {
			base.doubleVartime(&base)
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
			points[j].addVartime(&points[j-1], &base)
		}
		affine := batchNormalize16(points)
		for j := range table[i] {
			table[i][j] = affine[j]
		}

		for range baseWindow {
			base.doubleVartime(&base)
		}
	}
	return table
}

func newGeneratorWNAFTable() [generatorWNAFSize]affinePoint {
	var points [generatorWNAFSize]point
	points[0].set(&generator)
	if len(points) > 1 {
		var twoG point
		twoG.doubleVartime(&generator)
		for i := 1; i < len(points); i++ {
			points[i].addVartime(&points[i-1], &twoG)
		}
	}
	return batchNormalizeGenerator(points)
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

func signedWNAFVartime(k *scalar.Element, window int) ([257]int16, int, int16) {
	sign := int16(1)
	words := k.Words()
	if scalar.IsHighWords(words) {
		words = scalar.NegWords(words)
		sign = -1
	}
	naf, length := wnafVartime(&words, window)
	return naf, length, sign
}

func wnafVartime(words *[4]uint64, window int) ([257]int16, int) {
	var out [257]int16
	length := 0
	for i := 0; i < len(out) && !isZeroWords(words); i++ {
		if (*words)[0]&1 == 1 {
			digit := int((*words)[0] & ((1 << window) - 1))
			if digit > 1<<(window-1) {
				digit -= 1 << window
			}
			out[i] = int16(digit)
			if digit > 0 {
				subSmall(words, uint64(digit))
			} else {
				addSmall(words, uint64(-digit))
			}
		}
		shr1(words)
		length = i + 1
	}
	return out, length
}

func isZeroWords(words *[4]uint64) bool {
	return words[0]|words[1]|words[2]|words[3] == 0
}

func addSmall(words *[4]uint64, v uint64) {
	words[0] += v
	if words[0] >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]++
		if words[i] != 0 {
			return
		}
	}
}

func subSmall(words *[4]uint64, v uint64) {
	old := words[0]
	words[0] -= v
	if old >= v {
		return
	}
	for i := 1; i < len(words); i++ {
		words[i]--
		if words[i] != ^uint64(0) {
			return
		}
	}
}

func shr1(words *[4]uint64) {
	for i := range len(words) - 1 {
		words[i] = (words[i] >> 1) | (words[i+1] << 63)
	}
	words[len(words)-1] >>= 1
}

func bitAt(k *[32]byte, i int) byte {
	return (k[i/8] >> uint(7-i%8)) & 1
}

func nibbleAt(k *[32]byte, i int) byte {
	b := k[31-i/2]
	if i%2 == 0 {
		return b & 0x0f
	}
	return b >> 4
}
