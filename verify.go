package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

// VerifyDigest reports whether sig is a mathematically valid ECDSA signature for
// digest under pub. High-S signatures are accepted by this function.
//
// Verification only uses public inputs and may use variable-time algorithms.
func VerifyDigest(pub PublicKey, digest Digest, sig Signature) bool {
	return pub.verifyDigest(digest, (*[SignatureSize]byte)(sig[:]), false)
}

// VerifyCanonicalDigest reports whether sig is a valid low-S ECDSA signature for
// digest under pub.
//
// Verification only uses public inputs and may use variable-time algorithms.
func VerifyCanonicalDigest(pub PublicKey, digest Digest, sig Signature) bool {
	return pub.verifyDigest(digest, (*[SignatureSize]byte)(sig[:]), true)
}

func (p PublicKey) verifyDigest(digest Digest, sig *[SignatureSize]byte, requireLowS bool) bool {
	if !p.isValid() {
		return false
	}
	r, s, ok := parseSignature(sig, requireLowS)
	if !ok {
		return false
	}

	var w, u1, u2, e scalar.Element
	w.InvVartime(&s)
	e.SetBytesModOrder(&digest)
	u1.Mul(&e, &w)
	u2.Mul(&r, &w)

	// ECDSA verification checks that x((e/s)G + (r/s)Q) mod n equals r.
	sum := p.doubleScalarBaseMultVartime(&u1, &u2)
	if sum.isInfinity() {
		return false
	}
	return jacobianXEqualsScalar(&sum, &r)
}

func (p PublicKey) doubleScalarBaseMultVartime(k1, k2 *scalar.Element) point {
	table := p.precomputed.table.Load()
	if table == nil {
		table = p.verificationTable()
	}
	if !table.comb && p.precomputed.verifyUses.Add(1) > verifyCombBuildAfter &&
		p.precomputed.promoting.CompareAndSwap(false, true) {
		// Only the builder pays the promotion cost. Concurrent callers keep
		// using the immutable wNAF table until the comb table is published.
		table = p.buildVerifyCombTable()
	}
	if table.comb {
		return doubleScalarBaseMultCombVartime(k1, k2, (*[verifyCombTableSize]affinePoint)(table.points[:verifyCombTableSize]))
	}
	wnaf, endo := table.wnafTables()
	return doubleScalarBaseMultPrecomputedVartime(k1, k2, wnaf, endo)
}

// Keep the large precomputation workspace off the hot verification stack.
//
//go:noinline
func (p PublicKey) buildVerifyCombTable() *publicKeyTable {
	var q point
	q.setAffine(&p.x, &p.y)
	next := new(publicKeyTable)
	comb := newVerifyCombTable(&q)
	copy(next.points[:], comb[:])
	next.comb = true
	p.precomputed.table.Store(next)
	return next
}

func jacobianXEqualsScalar(p *point, x *scalar.Element) bool {
	if p.isInfinity() {
		return false
	}

	xWords := x.Words()
	if jacobianXEqualsFieldElementWords(p, &xWords) {
		return true
	}

	xPlusOrder, ok := scalar.AddOrderWords(xWords)
	return ok && jacobianXEqualsFieldElementWords(p, &xPlusOrder)
}

func jacobianXEqualsFieldElementWords(p *point, xWords *[4]uint64) bool {
	var x, z2, expected field.Element
	x.SetNonMontgomeryWords(*xWords)
	z2.Square(&p.z)
	expected.Mul(&x, &z2)
	return expected.Equal(&p.x)
}

func parseSignature(sig *[SignatureSize]byte, requireLowS bool) (scalar.Element, scalar.Element, bool) {
	rBytes := (*[scalar.Size]byte)(sig[:scalar.Size])
	sBytes := (*[scalar.Size]byte)(sig[scalar.Size:SignatureSize])
	if scalar.IsZeroBytes(rBytes) || scalar.IsZeroBytes(sBytes) ||
		!scalar.LessThanOrder(rBytes) || !scalar.LessThanOrder(sBytes) ||
		(requireLowS && scalar.IsHighBytes(sBytes)) {
		return scalar.Element{}, scalar.Element{}, false
	}
	var r, s scalar.Element
	r.SetBytesUnchecked(rBytes)
	s.SetBytesUnchecked(sBytes)
	return r, s, true
}
