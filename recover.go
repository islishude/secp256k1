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

	var e, rInv scalar.Element
	e.SetBytesModOrder(&digest)
	rInv.Inv(&r)

	// Rearranging s = k^-1(e + rd) gives Q = dG = r^-1(sR - eG).
	sR := scalarMultAffineScalar(&rPoint, &s)
	eG := scalarBaseMult(&e)
	var negEG point
	negEG.neg(&eG)
	var q point
	q.add(&sR, &negEG)
	q = scalarMultScalar(&q, &rInv)
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
