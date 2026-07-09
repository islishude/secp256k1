package secp256k1

const maxDERSignatureSize = 72

// ParseSignature parses a fixed-width compact ECDSA signature encoded as r || s.
func ParseSignature(b []byte) (Signature, error) {
	if len(b) != SignatureSize {
		return Signature{}, ErrInvalidSignature
	}
	var sig Signature
	copy(sig[:], b)
	if _, _, ok := parseSignature((*[SignatureSize]byte)(sig[:]), false); !ok {
		return Signature{}, ErrInvalidSignature
	}
	return sig, nil
}

// ParseRecoverableSignature parses r || s || recovery-id. Only low-S signatures
// are accepted, matching RecoverDigest.
func ParseRecoverableSignature(b []byte) (RecoverableSignature, error) {
	if len(b) != RecoverableSignatureSize {
		return RecoverableSignature{}, ErrInvalidSignature
	}
	var sig RecoverableSignature
	copy(sig[:], b)
	if _, _, _, ok := parseRecoverableSignature(&sig); !ok {
		return RecoverableSignature{}, ErrInvalidSignature
	}
	return sig, nil
}

// BytesDER returns sig encoded as a strict DER ECDSA signature.
func (sig Signature) BytesDER() ([]byte, error) {
	if _, _, ok := parseSignature((*[SignatureSize]byte)(sig[:]), false); !ok {
		return nil, ErrInvalidSignature
	}

	var der [maxDERSignatureSize]byte
	der[0] = 0x30
	n := 2
	n += putDERInteger(der[n:], sig[:32])
	n += putDERInteger(der[n:], sig[32:])
	der[1] = byte(n - 2)

	out := make([]byte, n)
	copy(out, der[:n])
	return out, nil
}

// ParseDERSignature parses a strict DER ECDSA signature. High-S signatures are
// accepted; use VerifyCanonicalDigest to enforce low-S during verification.
func ParseDERSignature(der []byte) (Signature, error) {
	if len(der) < 8 || len(der) > maxDERSignatureSize ||
		der[0] != 0x30 || int(der[1]) != len(der)-2 {
		return Signature{}, ErrInvalidSignature
	}

	r, pos, ok := parseDERInteger(der, 2)
	if !ok {
		return Signature{}, ErrInvalidSignature
	}
	s, pos, ok := parseDERInteger(der, pos)
	if !ok || pos != len(der) {
		return Signature{}, ErrInvalidSignature
	}

	var sig Signature
	copy(sig[32-len(r):32], r)
	copy(sig[SignatureSize-len(s):], s)
	if _, _, ok := parseSignature((*[SignatureSize]byte)(sig[:]), false); !ok {
		return Signature{}, ErrInvalidSignature
	}
	return sig, nil
}

func putDERInteger(out []byte, value []byte) int {
	for len(value) > 1 && value[0] == 0 {
		value = value[1:]
	}

	out[0] = 0x02
	if value[0]&0x80 != 0 {
		out[1] = byte(len(value) + 1)
		out[2] = 0
		copy(out[3:], value)
		return len(value) + 3
	}

	out[1] = byte(len(value))
	copy(out[2:], value)
	return len(value) + 2
}

func parseDERInteger(der []byte, pos int) ([]byte, int, bool) {
	if pos+2 > len(der) || der[pos] != 0x02 {
		return nil, 0, false
	}
	length := int(der[pos+1])
	start := pos + 2
	end := start + length
	if length == 0 || end > len(der) {
		return nil, 0, false
	}

	value := der[start:end]
	if value[0]&0x80 != 0 {
		return nil, 0, false
	}
	if len(value) > 1 && value[0] == 0 && value[1]&0x80 == 0 {
		return nil, 0, false
	}
	if len(value) > 33 {
		return nil, 0, false
	}
	if len(value) == 33 {
		if value[0] != 0 || value[1]&0x80 == 0 {
			return nil, 0, false
		}
		value = value[1:]
	}

	return value, end, true
}
