package secp256k1

import "github.com/islishude/secp256k1/internal/scalar"

// PreparedPublicKey stores precomputed tables for repeated verification under a
// single public key.
type PreparedPublicKey struct {
	publicKey     PublicKey
	wnafTable     [varWNAFTableSize]affinePoint
	endoWNAFTable [varWNAFTableSize]affinePoint
	valid         bool
}

// VerifyDigest reports whether sig is a mathematically valid ECDSA signature for
// digest under pub. High-S signatures are accepted by this function.
func VerifyDigest(pub PublicKey, digest Digest, sig Signature) bool {
	prepared, err := pub.Prepare()
	if err != nil {
		return false
	}
	return prepared.VerifyDigest(digest, sig)
}

// VerifyCanonicalDigest reports whether sig is a valid low-S ECDSA signature for
// digest under pub.
func VerifyCanonicalDigest(pub PublicKey, digest Digest, sig Signature) bool {
	prepared, err := pub.Prepare()
	if err != nil {
		return false
	}
	return prepared.VerifyCanonicalDigest(digest, sig)
}

// Prepare builds precomputed verification tables for repeated use of p.
func (p PublicKey) Prepare() (PreparedPublicKey, error) {
	if !p.isValid() {
		return PreparedPublicKey{}, ErrInvalidPublicKey
	}
	var point point
	point.setAffine(&p.x, &p.y)
	wnafTable := newAffineOddTable(&point)
	return PreparedPublicKey{
		publicKey:     p,
		wnafTable:     wnafTable,
		endoWNAFTable: newEndomorphismWNAFTable(&wnafTable),
		valid:         true,
	}, nil
}

func (p *PreparedPublicKey) isValid() bool {
	return p != nil && p.valid && p.publicKey.isValid()
}

// VerifyDigest reports whether sig is a mathematically valid ECDSA signature for
// digest under p's prepared public key. High-S signatures are accepted.
func (p *PreparedPublicKey) VerifyDigest(digest Digest, sig Signature) bool {
	return p.verifyDigest(digest, sig, false)
}

// VerifyCanonicalDigest reports whether sig is a valid low-S ECDSA signature for
// digest under p's prepared public key.
func (p *PreparedPublicKey) VerifyCanonicalDigest(digest Digest, sig Signature) bool {
	return p.verifyDigest(digest, sig, true)
}

func (p *PreparedPublicKey) verifyDigest(digest Digest, sig Signature, requireLowS bool) bool {
	if !p.isValid() {
		return false
	}
	r, s, ok := parseSignature(&sig, requireLowS)
	if !ok {
		return false
	}

	var w, u1, u2, e scalar.Element
	w.Inv(&s)
	e.SetBytesModOrder(&digest)
	u1.Mul(&e, &w)
	u2.Mul(&r, &w)

	// ECDSA verification checks that x((e/s)G + (r/s)Q) mod n equals r.
	sum := doubleScalarBaseMultPrecomputed(&u1, &u2, &p.wnafTable, &p.endoWNAFTable)
	if sum.isInfinity() {
		return false
	}
	x, _, _ := sum.affine()
	var xScalar scalar.Element
	xScalar.SetFieldElementModOrder(&x)
	return xScalar.Equal(&r)
}

func parseSignature(sig *Signature, requireLowS bool) (scalar.Element, scalar.Element, bool) {
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
