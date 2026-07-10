package secp256k1

import "github.com/islishude/secp256k1/internal/scalar"

// RecoverDigest reconstructs the public key that produced sig over digest.
func RecoverDigest(digest Digest, sig RecoverableSignature) (PublicKey, error) {
	r, s, recid, ok := parseRecoverableSignature(&sig)
	if !ok {
		return PublicKey{}, ErrInvalidSignature
	}
	xWords := r.Words()
	if recid>>1 == 1 {
		var ok bool
		xWords, ok = scalar.AddOrderWords(xWords)
		if !ok {
			return PublicKey{}, ErrInvalidSignature
		}
	}

	x, y, ok := affineFromXWords(&xWords, recid&1 == 1)
	if !ok {
		return PublicKey{}, ErrInvalidSignature
	}

	var rPoint point
	rPoint.setAffine(&x, &y)

	var e, rInv, u1, u2 scalar.Element
	e.SetBytesModOrder(&digest)
	rInv.InvVartime(&r)

	// Rearranging s = k^-1(e + rd) gives
	// Q = r^-1(sR - eG) = (-e/r)G + (s/r)R. All recovery inputs
	// are public, so use the variable-time double-scalar path directly.
	u1.Mul(&e, &rInv)
	u1.Neg(&u1)
	u2.Mul(&s, &rInv)
	q := doubleScalarBaseMultVartime(&u1, &rPoint, &u2)
	if q.isInfinity() {
		return PublicKey{}, ErrInvalidSignature
	}
	pub, ok := publicKeyFromPoint(&q)
	if !ok || !pub.verifyDigest(digest, (*[SignatureSize]byte)(sig[:SignatureSize]), true) {
		return PublicKey{}, ErrInvalidSignature
	}
	return pub, nil
}

func parseRecoverableSignature(sig *RecoverableSignature) (scalar.Element, scalar.Element, byte, bool) {
	recid := sig[recoverableSignatureRecIDAt]
	if recid > 3 {
		return scalar.Element{}, scalar.Element{}, 0, false
	}
	r, s, ok := parseSignature((*[SignatureSize]byte)(sig[:SignatureSize]), true)
	if !ok {
		return scalar.Element{}, scalar.Element{}, 0, false
	}
	return r, s, recid, true
}
