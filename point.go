package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
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
	generator      = newGeneratorPoint()
	generatorTable = newGeneratorTable()
)

// pointTable stores 4-bit windows of the generator. Entry [i][j] is
// j * (16^i * G), letting base-point multiplication consume one nibble at a
// time.
type pointTable [64][16]point

// point is a Jacobian-coordinate curve point. The point at infinity is encoded
// with z = 0.
type point struct {
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

func newGeneratorTable() pointTable {
	var table pointTable
	base := generator
	for i := range table {
		table[i][0].setInfinity()
		table[i][1].set(&base)
		for j := 2; j < len(table[i]); j++ {
			table[i][j].add(&table[i][j-1], &base)
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

func scalarBaseMult(k *scalar.Element) point {
	b := k.Bytes()
	var r point
	r.setInfinity()
	for i := range generatorTable {
		digit := nibbleAt(&b, i)
		var selected point
		selected.setInfinity()
		for j := range generatorTable[i] {
			// Scan the whole window table and conditionally select to avoid
			// branching on the secret scalar nibble.
			selected.selectPoint(&selected, &generatorTable[i][j], equalByte(digit, byte(j)))
		}
		r.add(&r, &selected)
	}
	return r
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
	k1Bytes := k1.Bytes()
	k2Bytes := k2.Bytes()

	// Precompute G + P once so each bit position needs at most one addition.
	var gPlusP2 point
	gPlusP2.add(&generator, p2)

	var gPlusP2X, gPlusP2Y field.Element
	gPlusP2IsInfinity := gPlusP2.isInfinity()
	if !gPlusP2IsInfinity {
		gPlusP2X, gPlusP2Y, _ = gPlusP2.affine()
	}

	var r point
	r.setInfinity()
	for i := range 256 {
		r.double(&r)

		k1Bit := bitAt(&k1Bytes, i)
		k2Bit := bitAt(&k2Bytes, i)
		switch {
		case k1Bit == 1 && k2Bit == 1:
			if !gPlusP2IsInfinity {
				r.addAffine(&r, &gPlusP2X, &gPlusP2Y)
			}
		case k1Bit == 1:
			r.addAffine(&r, &generator.x, &generator.y)
		case k2Bit == 1:
			r.addAffine(&r, &p2.x, &p2.y)
		}
	}
	return r
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

func pointFromPublicKey(pub *PublicKey) point {
	var p point
	p.setAffine(&pub.x, &pub.y)
	return p
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
