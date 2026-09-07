package secp256k1

import (
	"runtime"

	"github.com/islishude/secp256k1/internal/scalar"
)

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
	clearSecretBytes(privBytes[:])
	defer nonce.Destroy()
	var e scalar.Element
	e.SetBytesModOrder(&digest)
	for {
		kBytes := nonce.Next()
		sig, ok := k.signWithNonce(&e, &kBytes, recoverable)
		if ok {
			return sig, nil
		}
		nonce.Reject()
	}
}

// signingScalars groups the controllable scalar copies for one nonce attempt.
// Cleanup runs on both successful and rejected attempts.
type signingScalars struct {
	nonce, r, rd, sum, kinv, s, negS scalar.Element
}

//go:noinline
func (s *signingScalars) destroy() {
	*s = signingScalars{}
	runtime.KeepAlive(s)
}

func (k *PrivateKey) signWithNonce(e *scalar.Element, kBytes *[scalar.Size]byte, recoverable bool) (RecoverableSignature, bool) {
	defer clearSecretBytes(kBytes[:])
	var v signingScalars
	defer v.destroy()
	v.nonce.SetBytesUnchecked(kBytes)
	rx, ry, ok := scalarBaseMultAffine(&v.nonce)
	if !ok {
		return RecoverableSignature{}, false
	}
	// ECDSA r is x(R) mod n. The recovery id retains x's overflow bit.
	rxWords := rx.NonMontgomeryWords()
	xOverflow := byte(0)
	if recoverable && !scalar.LessThanOrderWords(rxWords) {
		xOverflow = 1
	}
	v.r.SetWordsModOrder(rxWords)
	if v.r.IsZero() {
		return RecoverableSignature{}, false
	}
	v.rd.Mul(&v.r, &k.d)
	v.sum.Add(e, &v.rd)
	v.kinv.Inv(&v.nonce)
	v.s.Mul(&v.kinv, &v.sum)
	if v.s.IsZero() {
		return RecoverableSignature{}, false
	}
	recid := byte(0)
	if recoverable {
		if ry.IsOdd() {
			recid = 1
		}
		recid |= xOverflow << 1
	}
	// Negating s is equivalent to negating R; flip recovery parity as well.
	highS := v.s.IsHighChoice()
	v.negS.Neg(&v.s)
	v.s.Select(&v.s, &v.negS, highS)
	if recoverable {
		recid ^= byte(highS)
	}
	var sig RecoverableSignature
	v.r.PutBytes((*[scalar.Size]byte)(sig[:scalar.Size]))
	v.s.PutBytes((*[scalar.Size]byte)(sig[scalar.Size : 2*scalar.Size]))
	sig[recoverableSignatureRecIDAt] = recid
	return sig, true
}
