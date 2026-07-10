# secp256k1 Sign/Verify Benchmark

This directory benchmarks ECDSA digest signing and verification for three
secp256k1 implementations:

- Local: `github.com/islishude/secp256k1`
- Decred: `github.com/decred/dcrd/dcrec/secp256k1/v4`
- Geth: `github.com/ethereum/go-ethereum/crypto/secp256k1`

## How to run

```sh
go test -bench=. -benchmem -count=10
```

## Environment

| Item        | Value                              |
| ----------- | ---------------------------------- |
| Go          | `go version go1.26.4 darwin/arm64` |
| GOOS        | `darwin`                           |
| GOARCH      | `arm64`                            |
| CGO_ENABLED | `1`                                |
| CPU         | `Apple M3 Max`                     |

## Notes

- The benchmark measures 32-byte digest-level ECDSA sign and verify operations.
- Compact signing means a non-recoverable 64-byte `r || s` signature.
- Recoverable signing means a 65-byte signature with recovery metadata. The
  exact byte layout is implementation-specific.
- Geth's low-level package is benchmarked for recoverable signing and verify;
  it does not expose a separate compact signing API in this benchmark.
- Keys, digests, public keys, and signatures are prepared before timing starts.
- Local results below use the default fiat arithmetic backend. Repeated-key
  verification promotes itself from wNAF/GLV to the adaptive comb path after
  eight valid verifications.
- The fixture derives each implementation's public key directly from the fixed
  private key, then separately checks verification and recovery.
- Results are medians of ten runs on this machine. Different CPUs, Go versions,
  or CGO settings can change the ranking, especially for geth.

## Results

| Operation        | Implementation | Median ns/op |  B/op | allocs/op |
| ---------------- | -------------- | -----------: | ----: | --------: |
| Compact sign     | Local          |       24,600 |     0 |         0 |
| Compact sign     | Decred         |       33,745 | 1,512 |        28 |
| Recoverable sign | Local          |       24,681 |     0 |         0 |
| Recoverable sign | Decred         |       34,562 | 1,592 |        29 |
| Recoverable sign | Geth           |       17,154 |   164 |         3 |
| Verify           | Local          |       20,898 |     0 |         0 |
| Verify           | Decred         |      124,515 |   568 |        12 |
| Verify           | Geth           |       18,105 |     0 |         0 |

## Conclusions

- Geth remains the fastest implementation for recoverable signing and
  verification on this machine.
- Local compact signing is about 1.37x faster than Decred compact signing while
  using zero benchmark-time allocations.
- Local recoverable signing is about 1.40x faster than Decred recoverable
  signing while using zero benchmark-time allocations.
- Local verification is about 5.96x faster than Decred verification while using
  zero benchmark-time allocations.
- Local recoverable signing is about 1.44x slower than geth, and Local
  verification is about 1.15x slower than geth on this machine.

The staged local before/after measurements, including hot/cold public-key
workloads and internal microbenchmarks, are recorded under `../docs/perf/`.
