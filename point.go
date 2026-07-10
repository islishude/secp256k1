package secp256k1

import "github.com/islishude/secp256k1/internal/field"

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

func (p *point) double(q *point) *point {
	if q.isInfinity() || q.y.IsZero() {
		return p.setInfinity()
	}
	x1, y1, z1 := q.x, q.y, q.z
	var xx, yy, yyyy, s, m, t field.Element
	xx.Square(&x1)
	yy.Square(&y1)
	yyyy.Square(&yy)
	s.Mul(&x1, &yy)
	s.Double(&s)
	s.Double(&s)
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

// addAffineWNAFVartime adds a public affine wNAF table point to p1 using an
// incomplete mixed-add formula. Exceptional equal/opposite points fall back to
// the complete behavior in addAffine. It must only be used on public inputs.
func (p *point) addAffineWNAFVartime(p1 *point, x2, y2 *field.Element) *point {
	if p1.isInfinity() {
		return p.setAffine(x2, y2)
	}
	x1, y1, z1 := p1.x, p1.y, p1.z
	var z1z1, u2, s2, h, r field.Element
	var hh, hhh, v, t field.Element

	z1z1.Square(&z1)
	u2.Mul(x2, &z1z1)
	t.Mul(&z1, &z1z1)
	s2.Mul(y2, &t)
	h.Sub(&u2, &x1)
	r.Sub(&s2, &y1)
	if h.IsZero() {
		return p.addAffine(p1, x2, y2)
	}

	// Unscaled Jacobian mixed addition:
	// X3 = R^2 - H^3 - 2*X1*H^2
	// Y3 = R*(X1*H^2 - X3) - Y1*H^3
	// Z3 = Z1*H
	hh.Square(&h)
	hhh.Mul(&h, &hh)
	v.Mul(&x1, &hh)

	var x3, y3, z3 field.Element
	x3.Square(&r)
	x3.Sub(&x3, &hhh)
	t.Double(&v)
	x3.Sub(&x3, &t)

	t.Sub(&v, &x3)
	y3.Mul(&r, &t)
	t.Mul(&y1, &hhh)
	y3.Sub(&y3, &t)

	z3.Mul(&z1, &h)
	p.x.Set(&x3)
	p.y.Set(&y3)
	p.z.Set(&z3)
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

func (p *projectivePoint) setAffine(q *affinePoint) *projectivePoint {
	p.x.Set(&q.x)
	p.y.Set(&q.y)
	p.z.SetOne()
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

func (p *projectivePoint) affine() (field.Element, field.Element, bool) {
	if p.z.IsZero() {
		var x, y field.Element
		return x, y, false
	}
	var zInv, x, y field.Element
	zInv.Inv(&p.z)
	x.Mul(&p.x, &zInv)
	y.Mul(&p.y, &zInv)
	return x, y, true
}
