package secp256k1

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"testing"
)

func standardHMACDigest(key []byte, chunks ...[]byte) [sha256.Size]byte {
	mac := hmac.New(sha256.New, key)
	for _, chunk := range chunks {
		_, _ = mac.Write(chunk)
	}
	var out [sha256.Size]byte
	mac.Sum(out[:0])
	return out
}

func TestHMACDigestMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name   string
		key    []byte
		chunks [][]byte
	}{
		{
			name:   "rfc6979 init shape",
			key:    bytes.Repeat([]byte{0x00}, sha256.Size),
			chunks: [][]byte{bytes.Repeat([]byte{0x01}, sha256.Size), rfc6979Zero[:], bytes.Repeat([]byte{0x02}, PrivateKeySize), bytes.Repeat([]byte{0x03}, sha256.Size)},
		},
		{
			name:   "long key",
			key:    bytes.Repeat([]byte{0x11}, sha256.BlockSize+1),
			chunks: [][]byte{[]byte("secp256k1"), []byte("rfc6979")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hmacDigestFast(tc.key, tc.chunks...)
			want := standardHMACDigest(tc.key, tc.chunks...)
			if got != want {
				t.Fatalf("hmacDigest mismatch\n got %x\nwant %x", got, want)
			}
		})
	}
}
