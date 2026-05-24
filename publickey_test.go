package secp256k1

import (
	"testing"

	"github.com/islishude/secp256k1/internal/field"
)

func TestPublicKeyEncodingRoundTrip(t *testing.T) {
	var keyBytes [32]byte
	keyBytes[31] = 42
	priv, err := ParsePrivateKey(keyBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := pub.BytesCompressed()
	if err != nil {
		t.Fatal(err)
	}
	parsedCompressed, err := ParsePublicKey(compressed[:])
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsedCompressed) {
		t.Fatal("compressed round trip mismatch")
	}
	uncompressed, err := pub.BytesUncompressed()
	if err != nil {
		t.Fatal(err)
	}
	parsedUncompressed, err := ParsePublicKey(uncompressed[:])
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsedUncompressed) {
		t.Fatal("uncompressed round trip mismatch")
	}
}

func TestParsePublicKeyRejectsInvalidEncodings(t *testing.T) {
	compressedBadPrefix := make([]byte, PublicKeyCompressedSize)
	compressedBadPrefix[0] = 0x05

	compressedXModulus := make([]byte, PublicKeyCompressedSize)
	compressedXModulus[0] = 0x02
	copy(compressedXModulus[1:], field.Modulus[:])

	uncompressedBadPrefix := make([]byte, PublicKeyUncompressedSize)
	uncompressedBadPrefix[0] = 0x05

	uncompressedXModulus := make([]byte, PublicKeyUncompressedSize)
	uncompressedXModulus[0] = 0x04
	copy(uncompressedXModulus[1:33], field.Modulus[:])
	copy(uncompressedXModulus[33:], gyBytes[:])

	uncompressedYModulus := make([]byte, PublicKeyUncompressedSize)
	uncompressedYModulus[0] = 0x04
	copy(uncompressedYModulus[1:33], gxBytes[:])
	copy(uncompressedYModulus[33:], field.Modulus[:])

	uncompressedOffCurve := make([]byte, PublicKeyUncompressedSize)
	uncompressedOffCurve[0] = 0x04
	copy(uncompressedOffCurve[1:33], gxBytes[:])

	tests := []struct {
		name string
		in   []byte
	}{
		{name: "empty", in: nil},
		{name: "short compressed", in: []byte{0x02}},
		{name: "bad compressed prefix", in: compressedBadPrefix},
		{name: "compressed x equals modulus", in: compressedXModulus},
		{name: "bad uncompressed prefix", in: uncompressedBadPrefix},
		{name: "uncompressed x equals modulus", in: uncompressedXModulus},
		{name: "uncompressed y equals modulus", in: uncompressedYModulus},
		{name: "uncompressed off curve", in: uncompressedOffCurve},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if pub, err := ParsePublicKey(tc.in); err == nil {
				t.Fatalf("ParsePublicKey(%x) = %#v, want error", tc.in, pub)
			}
		})
	}
}
