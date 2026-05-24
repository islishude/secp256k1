package secp256k1

import "errors"

const (
	// DigestSize is the byte length of a message digest accepted by the
	// digest-level ECDSA APIs.
	DigestSize = 32
	// PrivateKeySize is the byte length of a canonical secp256k1 private key.
	PrivateKeySize = 32
	// PublicKeyCompressedSize is the byte length of a SEC 1 compressed public key.
	PublicKeyCompressedSize = 33
	// PublicKeyUncompressedSize is the byte length of a SEC 1 uncompressed public key.
	PublicKeyUncompressedSize = 65
	// SignatureSize is the byte length of r || s.
	SignatureSize = 64
	// RecoverableSignatureSize is the byte length of r || s || recovery-id.
	RecoverableSignatureSize = 65

	recoverableSignatureRecIDAt = 64
)

var (
	// ErrInvalidPrivateKey reports a nil, zero, destroyed, malformed, or out of
	// range private key.
	ErrInvalidPrivateKey = errors.New("secp256k1: invalid private key")
	// ErrInvalidPublicKey reports a zero, malformed, or off-curve public key.
	ErrInvalidPublicKey = errors.New("secp256k1: invalid public key")
	// ErrInvalidSignature reports malformed ECDSA signature material.
	ErrInvalidSignature = errors.New("secp256k1: invalid signature")
)

// Signature is a fixed-width ECDSA signature encoded as r || s.
type Signature [SignatureSize]byte

// Bytes returns sig as a byte slice.
func (sig Signature) Bytes() []byte {
	return sig[:]
}

// Digest is the fixed-width message digest accepted by the digest-level ECDSA APIs.
type Digest = [DigestSize]byte

// RecoverableSignature is a fixed-width ECDSA signature encoded as
// r || s || recovery-id.
type RecoverableSignature [RecoverableSignatureSize]byte

// Bytes returns sig as a byte slice.
func (sig RecoverableSignature) Bytes() []byte {
	return sig[:]
}

// Signature returns the non-recoverable r || s portion of sig.
func (sig RecoverableSignature) Signature() Signature {
	var out Signature
	copy(out[:], sig[:SignatureSize])
	return out
}
