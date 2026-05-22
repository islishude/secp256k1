# secp256k1 Sign/Verify Benchmark

This directory benchmarks ECDSA digest signing and verification for three
secp256k1 implementations:

- Local: `github.com/islishude/secp256k1`
- Decred: `github.com/decred/dcrd/dcrec/secp256k1/v4`
- Geth: `github.com/ethereum/go-ethereum/crypto/secp256k1`

## How to Run

```sh
go test -bench=. -benchmem -count=5
```

## Environment

| Item        | Value                              |
| ----------- | ---------------------------------- |
| Go          | `go version go1.26.3 darwin/arm64` |
| GOOS        | `darwin`                           |
| GOARCH      | `arm64`                            |
| CGO_ENABLED | `1`                                |
| CPU         | `Apple M3 Max`                     |

## Notes

- The benchmark measures 32-byte digest-level ECDSA sign and verify operations.
- Keys, digests, public keys, and signatures are prepared before timing starts.
- Results are from this machine only. Different CPUs, Go versions, or CGO settings can change the ranking, especially for geth.

## Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       27152 |       26691 |       26586 |       26786 |       26797 |     26802 |    0 |         0 |
| Decred         |       32722 |       32728 |       32707 |       32678 |       32803 |     32728 | 1512 |        28 |
| Geth           |       15840 |       15703 |       15776 |       15714 |       15846 |     15776 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       67147 |       67519 |       67307 |       67445 |       67474 |     67378 |    0 |         0 |
| Decred         |      108785 |      108598 |      108569 |      108438 |      108867 |    108651 | 1289 |        23 |
| Geth           |       18447 |       18545 |       18239 |       18391 |       18408 |     18406 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation in both operations on this machine. It is about 1.7x faster than local signing and about 3.7x faster than local verification.
- Local signing is faster than Decred signing while using zero allocations. Decred allocates 1512 B/op across 28 allocations.
- Local verification is faster than Decred verification and uses zero allocations, but remains significantly slower than geth's CGO-backed implementation.
- The local implementation has zero benchmark-time allocations for both sign and verify.
- Compared with the previous run, local signing and verification both improved substantially, especially verification.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSign-16              42981             27152 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              45015             26691 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44869             26586 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44635             26786 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44836             26797 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17811             67147 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17611             67519 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17854             67307 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17751             67445 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17752             67474 ns/op               0 B/op          0 allocs/op
BenchmarkDecredSign-16             36644             32722 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36632             32728 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36578             32707 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36588             32678 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36465             32803 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredVerify-16           10000            108785 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108598 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108569 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108438 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108867 ns/op            1289 B/op         23 allocs/op
BenchmarkGethSign-16               75270             15840 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               75324             15703 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               74751             15776 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               76370             15714 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               76173             15846 ns/op             164 B/op          3 allocs/op
BenchmarkGethVerify-16             64726             18447 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65821             18545 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             66280             18239 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65541             18391 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65088             18408 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/islishude/secp256k1/benchmark        39.520s
```
