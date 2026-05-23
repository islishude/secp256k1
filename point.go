package secp256k1

import (
	"encoding/binary"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

const (
	varWNAFWindow       = 8
	varWNAFTableSize    = 1 << (varWNAFWindow - 2)
	generatorWNAFWindow = 8
	generatorWNAFSize   = 1 << (generatorWNAFWindow - 2)
)

var (
	gxBytes = [32]byte{
		0x79, 0xbe, 0x66, 0x7e, 0xf9, 0xdc, 0xbb, 0xac,
		0x55, 0xa0, 0x62, 0x95, 0xce, 0x87, 0x0b, 0x07,
		0x02, 0x9b, 0xfc, 0xdb, 0x2d, 0xce, 0x28, 0xd9,
		0x59, 0xf2, 0x81, 0x5b, 0x16, 0xf8, 0x17, 0x98,
	}
	gyBytes = [32]byte{
		0x48, 0x3a, 0xda, 0x77, 0x26, 0xa3, 0xc4, 0x65,
		0x5d, 0xa4, 0xfb, 0xfc, 0x0e, 0x11, 0x08, 0xa8,
		0xfd, 0x17, 0xb4, 0x48, 0xa6, 0x85, 0x54, 0x19,
		0x9c, 0x47, 0xd0, 0x8f, 0xfb, 0x10, 0xd4, 0xb8,
	}
	endoBeta               = newEndomorphismBeta()
	secp256k1B3            = fieldElementUint64(21)
	generator              = newGeneratorPoint()
	generatorAffineTable   = newGeneratorAffineTable()
	generatorWNAFTable     = newGeneratorWNAFTable()
	generatorEndoWNAFTable = newGeneratorEndomorphismWNAFTable(&generatorWNAFTable)
)

// point is a Jacobian-coordinate curve point. The point at infinity is encoded
// with z = 0.
type point struct {
	x field.Element
	y field.Element
	z field.Element
}

type affinePoint struct {
	x        field.Element
	y        field.Element
	infinity uint64
}

type projectivePoint struct {
	x field.Element
	y field.Element
	z field.Element
}

func newGeneratorPoint() point {
	var x, y field.Element
	if !x.SetBytes(&gxBytes) || !y.SetBytes(&gyBytes) {
		panic("secp256k1: invalid generator")
	}
	var g point
	g.setAffine(&x, &y)
	return g
}

func newEndomorphismBeta() field.Element {
	b := [32]byte{
		0x7a, 0xe9, 0x6a, 0x2b, 0x65, 0x7c, 0x07, 0x10,
		0x6e, 0x64, 0x47, 0x9e, 0xac, 0x34, 0x34, 0xe9,
		0x9c, 0xf0, 0x49, 0x75, 0x12, 0xf5, 0x89, 0x95,
		0xc1, 0x39, 0x6c, 0x28, 0x71, 0x95, 0x01, 0xee,
	}
	var beta field.Element
	if !beta.SetBytes(&b) {
		panic("secp256k1: invalid endomorphism beta")
	}
	return beta
}

func fieldElementUint64(v uint64) field.Element {
	var out field.Element
	out.SetUint64(v)
	return out
}

func newGeneratorAffineTable() [64][16]affinePoint {
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

func (p *point) set(q *point) *point {
	p.x.Set(&q.x)
	p.y.Set(&q.y)
	p.z.Set(&q.z)
	return p
}

func (p *point) setInfinity() *point {
	p.x.SetZero()
	p.y.SetZero()
	p.z.SetZero()
	return p
}

func (p *point) setAffine(x, y *field.Element) *point {
	p.x.Set(x)
	p.y.Set(y)
	p.z.SetOne()
	return p
}

func (p *point) isInfinity() bool {
	return p.z.IsZero()
}

func (p *point) affine() (field.Element, field.Element, bool) {
	if p.isInfinity() {
		var x, y field.Element
		return x, y, false
	}
	var zInv, z2, z3, x, y field.Element
	// Jacobian to affine: x = X/Z^2 and y = Y/Z^3.
	zInv.Inv(&p.z)
	z2.Square(&zInv)
	z3.Mul(&z2, &zInv)
	x.Mul(&p.x, &z2)
	y.Mul(&p.y, &z3)
	return x, y, true
}

func (p *point) double(q *point) *point {
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

func (p *point) add(p1, p2 *point) *point {
	if p1.isInfinity() {
		return p.set(p2)
	}
	if p2.isInfinity() {
		return p.set(p1)
	}

	x1, y1, z1 := p1.x, p1.y, p1.z
	x2, y2, z2 := p2.x, p2.y, p2.z

	var z1z1, z2z2, u1, u2, s1, s2 field.Element
	var h, r, i, j, v, t field.Element

	// Add two Jacobian points without converting to affine coordinates.
	z1z1.Square(&z1)
	z2z2.Square(&z2)
	u1.Mul(&x1, &z2z2)
	u2.Mul(&x2, &z1z1)

	t.Mul(&z2, &z2z2)
	s1.Mul(&y1, &t)
	t.Mul(&z1, &z1z1)
	s2.Mul(&y2, &t)

	h.Sub(&u2, &u1)
	r.Sub(&s2, &s1)
	if h.IsZero() {
		if r.IsZero() {
			// Same affine point.
			return p.double(p1)
		}
		// Same x-coordinate and opposite y-coordinate.
		return p.setInfinity()
	}

	i.Double(&h)
	i.Square(&i)
	j.Mul(&h, &i)
	r.Double(&r)
	v.Mul(&u1, &i)

	var x3, y3, z3 field.Element
	x3.Square(&r)
	x3.Sub(&x3, &j)
	t.Double(&v)
	x3.Sub(&x3, &t)

	t.Sub(&v, &x3)
	y3.Mul(&r, &t)
	t.Mul(&s1, &j)
	t.Double(&t)
	y3.Sub(&y3, &t)

	t.Add(&z1, &z2)
	t.Square(&t)
	t.Sub(&t, &z1z1)
	t.Sub(&t, &z2z2)
	z3.Mul(&t, &h)

	p.x.Set(&x3)
	p.y.Set(&y3)
	p.z.Set(&z3)
	return p
}

func (p *point) addAffine(p1 *point, x2, y2 *field.Element) *point {
	if p1.isInfinity() {
		return p.setAffine(x2, y2)
	}

	x1, y1, z1 := p1.x, p1.y, p1.z

	var z1z1, u2, s2 field.Element
	var h, r, i, j, v, t field.Element

	// Mixed addition saves work when the second point already has z = 1.
	z1z1.Square(&z1)
	u2.Mul(x2, &z1z1)

	t.Mul(&z1, &z1z1)
	s2.Mul(y2, &t)

	h.Sub(&u2, &x1)
	r.Sub(&s2, &y1)
	if h.IsZero() {
		if r.IsZero() {
			return p.double(p1)
		}
		return p.setInfinity()
	}

	i.Double(&h)
	i.Square(&i)
	j.Mul(&h, &i)
	r.Double(&r)
	v.Mul(&x1, &i)

	var x3, y3, z3 field.Element
	x3.Square(&r)
	x3.Sub(&x3, &j)
	t.Double(&v)
	x3.Sub(&x3, &t)

	t.Sub(&v, &x3)
	y3.Mul(&r, &t)
	t.Mul(&y1, &j)
	t.Double(&t)
	y3.Sub(&y3, &t)

	z3.Mul(&z1, &h)
	z3.Double(&z3)

	p.x.Set(&x3)
	p.y.Set(&y3)
	p.z.Set(&z3)
	return p
}

func (p *point) addMixed(p1 *point, p2 *affinePoint) *point {
	if p2.infinity == 1 {
		return p.set(p1)
	}
	return p.addAffine(p1, &p2.x, &p2.y)
}

func (p *point) neg(q *point) *point {
	p.x.Set(&q.x)
	p.y.Neg(&q.y)
	p.z.Set(&q.z)
	return p
}

func (p *point) selectPoint(x, y *point, choice uint64) *point {
	p.x.Select(&x.x, &y.x, choice)
	p.y.Select(&x.y, &y.y, choice)
	p.z.Select(&x.z, &y.z, choice)
	return p
}

func (p *affinePoint) selectPoint(x, y *affinePoint, choice uint64) *affinePoint {
	mask := uint64(0) - (choice & 1)
	p.x.Select(&x.x, &y.x, choice)
	p.y.Select(&x.y, &y.y, choice)
	p.infinity = (x.infinity &^ mask) | (y.infinity & mask)
	return p
}

func (p *projectivePoint) setInfinity() *projectivePoint {
	p.x.SetZero()
	p.y.SetOne()
	p.z.SetZero()
	return p
}

func (p *projectivePoint) selectPoint(x, y *projectivePoint, choice uint64) *projectivePoint {
	p.x.Select(&x.x, &y.x, choice)
	p.y.Select(&x.y, &y.y, choice)
	p.z.Select(&x.z, &y.z, choice)
	return p
}

// addCompleteMixed implements the complete mixed addition formula for
// j-invariant 0 curves over projective coordinates, specialized to b = 7.
func (p *projectivePoint) addCompleteMixed(p1 *projectivePoint, p2 *affinePoint) *projectivePoint {
	var t0, t1, t2, t3, t4 field.Element
	var x3, y3, z3 field.Element

	t0.Mul(&p1.x, &p2.x)
	t1.Mul(&p1.y, &p2.y)
	t3.Add(&p2.x, &p2.y)
	t4.Add(&p1.x, &p1.y)
	t3.Mul(&t3, &t4)
	t4.Add(&t0, &t1)
	t3.Sub(&t3, &t4)
	t4.Mul(&p2.y, &p1.z)
	t4.Add(&t4, &p1.y)
	y3.Mul(&p2.x, &p1.z)
	y3.Add(&y3, &p1.x)
	x3.Add(&t0, &t0)
	t0.Add(&x3, &t0)
	t2.Mul(&secp256k1B3, &p1.z)
	z3.Add(&t1, &t2)
	t1.Sub(&t1, &t2)
	y3.Mul(&secp256k1B3, &y3)
	x3.Mul(&t4, &y3)
	t2.Mul(&t3, &t1)
	x3.Sub(&t2, &x3)
	y3.Mul(&y3, &t0)
	t1.Mul(&t1, &z3)
	y3.Add(&t1, &y3)
	t0.Mul(&t0, &t3)
	z3.Mul(&z3, &t4)
	z3.Add(&z3, &t0)

	p.x.Set(&x3)
	p.y.Set(&y3)
	p.z.Set(&z3)
	return p
}

func (p *projectivePoint) jacobian() point {
	var out point
	if p.z.IsZero() {
		out.setInfinity()
		return out
	}
	var zInv, x, y field.Element
	zInv.Inv(&p.z)
	x.Mul(&p.x, &zInv)
	y.Mul(&p.y, &zInv)
	out.setAffine(&x, &y)
	return out
}

func scalarBaseMult(k *scalar.Element) point {
	b := k.Bytes()
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

func doubleScalarBaseMult(k1 *scalar.Element, p2 *point, k2 *scalar.Element) point {
	p2Table := newAffineOddTable(p2)
	p2EndoTable := newEndomorphismWNAFTable(&p2Table)
	return doubleScalarBaseMultPrecomputed(k1, k2, &p2Table, &p2EndoTable)
}

func doubleScalarBaseMultPrecomputed(k1, k2 *scalar.Element, p2Table, p2EndoTable *[varWNAFTableSize]affinePoint) point {
	k1a, k1b := scalar.SplitEndomorphism(k1)
	k2a, k2b := scalar.SplitEndomorphism(k2)
	k1aNAF, k1aLen, k1aSign := signedWNAF(&k1a, generatorWNAFWindow)
	k1bNAF, k1bLen, k1bSign := signedWNAF(&k1b, generatorWNAFWindow)
	k2aNAF, k2aLen, k2aSign := signedWNAF(&k2a, varWNAFWindow)
	k2bNAF, k2bLen, k2bSign := signedWNAF(&k2b, varWNAFWindow)
	n := max(k1aLen, k1bLen, k2aLen, k2bLen)
	k1aDigits := k1aNAF[:n]
	k1bDigits := k1bNAF[:n]
	k2aDigits := k2aNAF[:n]
	k2bDigits := k2bNAF[:n]

	var r point
	r.setInfinity()
	for i := n - 1; i >= 0; i-- {
		r.double(&r)
		addGeneratorWNAFPoint(&r, &generatorWNAFTable, k1aDigits[i]*k1aSign)
		addGeneratorWNAFPoint(&r, &generatorEndoWNAFTable, k1bDigits[i]*k1bSign)
		addVariableWNAFPoint(&r, p2Table, k2aDigits[i]*k2aSign)
		addVariableWNAFPoint(&r, p2EndoTable, k2bDigits[i]*k2bSign)
	}
	return r
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

func signedWNAF(k *scalar.Element, window int) ([257]int8, int, int8) {
	sign := int8(1)
	kBytes := k.Bytes()
	if scalar.IsHighBytes(&kBytes) {
		var neg scalar.Element
		neg.Neg(k)
		kBytes = neg.Bytes()
		sign = -1
	}
	naf, length := wnaf(&kBytes, window)
	return naf, length, sign
}

func addVariableWNAFPoint(r *point, table *[varWNAFTableSize]affinePoint, digit int8) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffine(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffine(r, &entry.x, &y)
}

func addGeneratorWNAFPoint(r *point, table *[generatorWNAFSize]affinePoint, digit int8) {
	if digit == 0 {
		return
	}
	if digit > 0 {
		entry := &table[(digit-1)/2]
		r.addAffine(r, &entry.x, &entry.y)
		return
	}

	entry := &table[(-digit-1)/2]
	var y field.Element
	y.Neg(&entry.y)
	r.addAffine(r, &entry.x, &y)
}

func wnaf(k *[32]byte, window int) ([257]int8, int) {
	words := scalarWords(k)
	var out [257]int8
	length := 0
	for i := 0; i < len(out) && !isZeroWords(&words); i++ {
		if words[0]&1 == 1 {
			digit := int(words[0] & ((1 << window) - 1))
			if digit > 1<<(window-1) {
				digit -= 1 << window
			}
			out[i] = int8(digit)
			if digit > 0 {
				subSmall(&words, uint64(digit))
			} else {
				addSmall(&words, uint64(-digit))
			}
		}
		shr1(&words)
		length = i + 1
	}
	return out, length
}

func scalarWords(k *[32]byte) [4]uint64 {
	return [4]uint64{
		binary.BigEndian.Uint64(k[24:32]),
		binary.BigEndian.Uint64(k[16:24]),
		binary.BigEndian.Uint64(k[8:16]),
		binary.BigEndian.Uint64(k[0:8]),
	}
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

func equalByte(x, y byte) uint64 {
	v := uint64(x ^ y)
	v |= v >> 4
	v |= v >> 2
	v |= v >> 1
	return (v ^ 1) & 1
}

func isOnCurve(x, y *field.Element) bool {
	var yy, xx, rhs, seven field.Element
	yy.Square(y)
	xx.Square(x)
	rhs.Mul(&xx, x)
	seven.SetUint64(7)
	rhs.Add(&rhs, &seven)
	return yy.Equal(&rhs)
}
