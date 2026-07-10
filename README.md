# secp256k1

A pure Go secp256k1 ECDSA implementation with private key generation, public
key parsing and encoding, 32-byte digest signing, verification, and recoverable
signature public key recovery.

## Features

- secp256k1 curve point arithmetic using Jacobian coordinates to reduce field inversions.
- Deterministic RFC6979 nonces, so signing the same digest with the same key is stable.
- Separate 64-byte ECDSA signatures and 65-byte recoverable signatures.
- Strict DER signature parsing and encoding.
- Low-S signing and canonical verification to reduce ECDSA signature malleability.
- SEC 1 compressed and uncompressed public key parsing.
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
	priv, err := secp256k1.GeneratePrivateKey(nil)
	if err != nil {
		panic(err)
	}
	defer priv.Destroy()

	digest := sha256.Sum256([]byte("message"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		panic(err)
	}

	pub, err := priv.PublicKey()
	if err != nil {
		panic(err)
	}
	if !secp256k1.VerifyCanonicalDigest(pub, digest, sig) {
		panic("invalid signature")
	}

	recoverable, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		panic(err)
	}
	recovered, err := secp256k1.RecoverDigest(digest, recoverable)
	if err != nil {
		panic(err)
	}

	compressed, err := pub.BytesCompressed()
	if err != nil {
		panic(err)
	}
	fmt.Printf("compressed public key: %x\n", compressed[:])
	fmt.Printf("recovered matches: %v\n", recovered.Equal(pub))
}
```

## API Overview

| API                                                | Description                                                                                                            |
| -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --- | -------------------- | --- | ------------------------------ |
| `DigestSize`, `Digest`                             | Define the fixed digest width used by digest-level APIs.                                                               |
| `SignatureSize`, `Signature`                       | Define the fixed 64-byte `r                                                                                            |     | s` signature format. |
| `RecoverableSignatureSize`, `RecoverableSignature` | Define the fixed 65-byte `r                                                                                            |     | s                    |     | recovery-id` signature format. |
| `ParseSignature([]byte)`                           | Parses a 64-byte compact signature and rejects zero or out-of-range scalars.                                           |
| `ParseRecoverableSignature([]byte)`                | Parses a 65-byte recoverable signature and rejects invalid recovery ids or non-canonical scalars.                      |
| `ParseDERSignature([]byte)`                        | Parses a strict DER ECDSA signature and rejects non-minimal or malformed encodings.                                    |
| `(Signature).Bytes()`                              | Returns the signature as a byte slice for interoperability helpers.                                                    |
| `(Signature).BytesDER()`                           | Returns a strict DER ECDSA signature encoding.                                                                         |
| `(RecoverableSignature).Bytes()`                   | Returns the recoverable signature as a byte slice for interoperability helpers.                                        |
| `(RecoverableSignature).Signature()`               | Returns the non-recoverable `Signature` portion.                                                                       |
| `GeneratePrivateKey(reader)`                       | Generates a valid private key from a random source; uses `crypto/rand.Reader` when `reader == nil`.                    |
| `ParsePrivateKey([]byte)`                          | Parses a 32-byte big-endian private key and rejects zero or values greater than or equal to the group order.           |
| `(*PrivateKey).Bytes()`                            | Returns the 32-byte private key encoding or `ErrInvalidPrivateKey`.                                                    |
| `(*PrivateKey).Destroy()`                          | Best-effort cleanup of private scalar material and invalidation of the key.                                            |
| `(*PrivateKey).PublicKey()`                        | Derives the public key with base-point multiplication and prepares it for verification.                                |
| `(*PrivateKey).SignDigest(Digest)`                 | Creates an RFC6979 deterministic low-S `Signature` over a digest.                                                      |
| `(*PrivateKey).SignRecoverableDigest(Digest)`      | Creates an RFC6979 deterministic low-S `RecoverableSignature`.                                                         |
| `VerifyDigest(pub, digest, sig)`                   | Verifies a 64-byte signature and accepts mathematically valid high-S signatures using public-input variable-time code. |
| `VerifyCanonicalDigest(pub, digest, sig)`          | Verifies a 64-byte signature and rejects high-S signatures using public-input variable-time code.                      |
| `RecoverDigest(digest, recoverableSig)`            | Recovers the public key from a 65-byte recoverable signature.                                                          |
| `ParsePublicKey([]byte)`                           | Parses a SEC 1 compressed or uncompressed public key and prepares it for verification.                                 |
| `(PublicKey).BytesCompressed()`                    | Returns the 33-byte compressed public key encoding or `ErrInvalidPublicKey`.                                           |
| `(PublicKey).BytesUncompressed()`                  | Returns the 65-byte uncompressed public key encoding or `ErrInvalidPublicKey`.                                         |

## Signature Formats

`Signature` is a fixed 64-byte array:

```text
0..31   r
32..63  s
```

`RecoverableSignature` is a fixed 65-byte array:

```text
0..31   r
32..63  s
64      recovery-id
```

The low bit of `recovery-id` records the parity of the ephemeral point `R`'s
y-coordinate. The high bit records whether `R.x` was reconstructed as `r + n`.
The `s` value produced by signing is normalized into the low half of the group
order, so the recovery parity bit may be flipped during normalization.

`ParseDERSignature` accepts strict DER only: short-form sequence length, two
positive minimal INTEGER values, no trailing data, and `r/s` in `[1, n-1]`.

Digest-level APIs accept exactly 32 bytes. They interpret that value as the
ECDSA message digest integer for secp256k1; APIs that accept arbitrary-length
messages or digests should perform the ECDSA leftmost-bit truncation before
calling this package.

## Development

Run tests:

```sh
go test ./...
```

Run the full local test target:

```sh
make test
```

Run the benchmark submodule:

```sh
cd benchmark
go test -bench=. -benchmem -count=5
```

Run DER fuzz or the optional timing smoke test:

```sh
make fuzz-smoke
make ct-smoke
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
repository does not claim a third-party security audit. `PrivateKey.Destroy`
and signing-path cleanup are best-effort only; Go does not guarantee hard memory
erasure. Signing and public-key derivation are intended to avoid secret-indexed
table lookups and secret-dependent low-S branching; verification uses
public-input variable-time wNAF/GLV code. Before using this library for
production funds or high-value keys, validate it against your audit standards,
test vectors, side-channel requirements, and higher-level protocol rules.

## License

MIT. License details for fiat-crypto generated code are in `internal/fiat`.
