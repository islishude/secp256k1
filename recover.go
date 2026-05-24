package secp256k1

import (
	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

// RecoverDigest reconstructs the public key that produced sig over digest.
func RecoverDigest(digest Digest, sig RecoverableSignature) (PublicKey, error) {
	r, s, recid, ok := parseRecoverableSignature(&sig)
	if !ok {
		return PublicKey{}, ErrInvalidSignature
	}
	xBytes := r.Bytes()
	if recid>>1 == 1 {
		var ok bool
		xBytes, ok = scalar.AddOrder(xBytes)
		if !ok {
			return PublicKey{}, ErrInvalidSignature
		}
	}
	if !field.LessThanModulus(&xBytes) {
		return PublicKey{}, ErrInvalidSignature
	}

	x, y, ok := affineFromXBytes(&xBytes, recid&1 == 1)
	if !ok {
		return PublicKey{}, ErrInvalidSignature
	}

	var rPoint point
	rPoint.setAffine(&x, &y)

	var e, rInv scalar.Element
	e.SetBytesModOrder(&digest)
	rInv.Inv(&r)

	// Rearranging s = k^-1(e + rd) gives Q = dG = r^-1(sR - eG).
	sBytes := s.Bytes()
	sR := scalarMultAffine(&rPoint, &sBytes)
	eG := scalarBaseMult(&e)
	var negEG point
	negEG.neg(&eG)
	var q point
	q.add(&sR, &negEG)
	rInvBytes := rInv.Bytes()
	q = scalarMult(&q, &rInvBytes)
	if q.isInfinity() {
		return PublicKey{}, ErrInvalidSignature
	}
	pub, ok := publicKeyFromPoint(&q)
	if !ok || !VerifyCanonicalDigest(pub, digest, sig.Signature()) {
		return PublicKey{}, ErrInvalidSignature
	}
	return pub, nil
}

func parseRecoverableSignature(sig *RecoverableSignature) (scalar.Element, scalar.Element, byte, bool) {
	recid := sig[recoverableSignatureRecIDAt]
	if recid > 3 {
		return scalar.Element{}, scalar.Element{}, 0, false
	}
	baseSig := sig.Signature()
	r, s, ok := parseSignature(&baseSig, true)
	if !ok {
		return scalar.Element{}, scalar.Element{}, 0, false
	}
	return r, s, recid, true
}
