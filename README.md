# secp256k1

A pure Go secp256k1 ECDSA implementation with private key generation, public
key parsing and encoding, 32-byte digest signing, verification, and recoverable
signature public key recovery.

## Features

- secp256k1 curve point arithmetic using Jacobian coordinates to reduce field inversions.
- Deterministic RFC6979 nonces, so signing the same digest with the same key is stable.
- Low-S normalization to reduce ECDSA signature malleability.
- SEC 1 compressed and uncompressed public key parsing.
- 65-byte recoverable signature format: `r || s || recovery-id`.
- Field and scalar arithmetic backed by fiat-crypto generated Montgomery routines.

## Installation

```sh
go get github.com/islishude/secp256k1
```

The module currently declares `go 1.26.3`.

## Quick Start

```go
package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/islishude/secp256k1"
)

func main() {
	priv, err := secp256k1.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	digest := sha256.Sum256([]byte("message"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		panic(err)
	}

	pub := priv.Public()
	if !secp256k1.VerifyDigest(pub, digest, sig) {
		panic("invalid signature")
	}

	recovered, err := secp256k1.RecoverDigest(digest, sig)
	if err != nil {
		panic(err)
	}

	compressed := pub.BytesCompressed()
	fmt.Printf("compressed public key: %x\n", compressed[:])
	fmt.Printf("recovered matches: %v\n", recovered.Equal(pub))
}
```

## API Overview

| API                                  | Description                                                                                                  |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `GenerateKey(reader)`                | Generates a valid private key from a random source; uses `crypto/rand.Reader` when `reader == nil`.          |
| `NewPrivateKey([32]byte)`            | Parses a 32-byte big-endian private key and rejects zero or values greater than or equal to the group order. |
| `(*PrivateKey).Public()`             | Derives the public key with base-point multiplication.                                                       |
| `(*PrivateKey).SignDigest([32]byte)` | Creates an RFC6979 deterministic recoverable ECDSA signature over a 32-byte digest.                          |
| `VerifyDigest(pub, digest, sig)`     | Verifies a digest and 65-byte signature.                                                                     |
| `RecoverDigest(digest, sig)`         | Recovers the public key from a recoverable signature.                                                        |
| `ParsePublicKey([]byte)`             | Parses a SEC 1 compressed or uncompressed public key.                                                        |
| `(*PublicKey).BytesCompressed()`     | Returns the 33-byte compressed public key encoding.                                                          |
| `(*PublicKey).BytesUncompressed()`   | Returns the 65-byte uncompressed public key encoding.                                                        |

## Signature Format

`SignDigest` returns a fixed 65-byte array:

```text
0..31   r
32..63  s
64      recovery-id
```

The low bit of `recovery-id` records the parity of the ephemeral point `R`'s
y-coordinate. The high bit records whether `R.x` was reconstructed as `r + n`.
The `s` value is normalized into the low half of the group order, so the
recovery parity bit may be flipped during normalization.

## Development

Run tests:

```sh
go test ./...
```

Run the benchmark submodule:

```sh
cd benchmark
go test -bench=. -benchmem -count=5
```

Regenerate low-level generated code:

```sh
go generate ./internal/fiat
go generate ./internal/addchain
```

Generation uses the Docker image `ghcr.io/islishude/fiat-crypto-go-tool`. See
`internal/fiat/README.md` and `internal/addchain/README.md` for details.

## Security Notes

This library implements common secp256k1 ECDSA digest-level APIs, but this
repository does not claim a third-party security audit. Before using it for
production funds or high-value keys, validate it against your audit standards,
test vectors, side-channel requirements, and higher-level protocol rules.

## License

MIT. License details for fiat-crypto generated code are in `internal/fiat`.
