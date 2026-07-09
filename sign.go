package secp256k1

import "github.com/islishude/secp256k1/internal/scalar"

// SignDigest signs a message digest with deterministic RFC6979 ECDSA.
//
// The returned signature is r || s. The s value is normalized to the low half of
// the group order so the signature has a unique canonical form.
func (k *PrivateKey) SignDigest(digest Digest) (Signature, error) {
	sig, err := k.signDigest(digest, false)
	if err != nil {
		return Signature{}, err
	}
	return sig.Signature(), nil
}

// SignRecoverableDigest signs a message digest and returns r || s || recovery-id.
func (k *PrivateKey) SignRecoverableDigest(digest Digest) (RecoverableSignature, error) {
	sig, err := k.signDigest(digest, true)
	if err != nil {
		return RecoverableSignature{}, err
	}
	return sig, nil
}

func (k *PrivateKey) signDigest(digest Digest, recoverable bool) (RecoverableSignature, error) {
	if !k.isValid() {
		return RecoverableSignature{}, ErrInvalidPrivateKey
	}

	privBytes := k.d.Bytes()
	nonce := newNonceRFC6979(&privBytes, &digest)

	var e scalar.Element
	e.SetBytesModOrder(&digest)

	for {
		kBytes := nonce.Next()
		var nonceScalar scalar.Element
		nonceScalar.SetBytesUnchecked(&kBytes)

		rx, ry, ok := scalarBaseMultAffine(&nonceScalar)
		if !ok {
			nonce.Reject()
			continue
		}
		// ECDSA defines r as x(R) mod n. Recovery needs to know whether the
		// original field x-coordinate was r or r+n, so keep that overflow bit.
		xOverflow := byte(0)
		rxWords := rx.NonMontgomeryWords()
		if recoverable && !scalar.LessThanOrderWords(rxWords) {
			xOverflow = 1
		}

		var r, rd, sum, kinv, s scalar.Element
		r.SetWordsModOrder(rxWords)
		if r.IsZero() {
			nonce.Reject()
			continue
		}

		rd.Mul(&r, &k.d)
		sum.Add(&e, &rd)
		kinv.Inv(&nonceScalar)
		s.Mul(&kinv, &sum)
		if s.IsZero() {
			nonce.Reject()
			continue
		}

		// The recovery id encodes the y parity and the x-coordinate overflow.
		recid := byte(0)
		if recoverable {
			if ry.IsOdd() {
				recid |= 1
			}
			recid |= xOverflow << 1
		}
		// Low-S normalization replaces s with n-s. That is equivalent to
		// using -R, so the y-parity bit must be flipped for recovery.
		highS := s.IsHighChoice()
		var negS scalar.Element
		negS.Neg(&s)
		s.Select(&s, &negS, highS)
		if recoverable {
			recid ^= byte(highS)
		}

		var sig RecoverableSignature
		r.PutBytes((*[scalar.Size]byte)(sig[:scalar.Size]))
		s.PutBytes((*[scalar.Size]byte)(sig[scalar.Size : 2*scalar.Size]))
		sig[recoverableSignatureRecIDAt] = recid
		clear(privBytes[:])
		nonce.Destroy()
		return sig, nil
	}
}
