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
| Local          |       27222 |       26844 |       26713 |       26814 |       26797 |     26878 |    0 |         0 |
| Decred         |       32887 |       32741 |       32759 |       32648 |       32887 |     32784 | 1512 |        28 |
| Geth           |       15838 |       15809 |       15764 |       15921 |       15857 |     15838 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       67028 |       67491 |       66972 |       67529 |       68134 |     67431 |    0 |         0 |
| Decred         |      108460 |      108804 |      108626 |      108597 |      108497 |    108597 | 1289 |        23 |
| Geth           |       18628 |       18568 |       18448 |       18885 |       18588 |     18623 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation in both operations on this machine. It is about 1.7x faster than local signing and about 3.6x faster than local verification.
- Local signing is faster than Decred signing while using zero allocations. Decred allocates 1512 B/op across 28 allocations.
- Local verification is faster than Decred verification and uses zero allocations, but remains significantly slower than geth's CGO-backed implementation.
- The local implementation has zero benchmark-time allocations for both sign and verify.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSign-16              43176             27222 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44810             26844 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44758             26713 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              44722             26814 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              45027             26797 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17936             67028 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17845             67491 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17936             66972 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17800             67529 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            17614             68134 ns/op               0 B/op          0 allocs/op
BenchmarkDecredSign-16             36106             32887 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36445             32741 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36556             32759 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36621             32648 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36646             32887 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredVerify-16           10000            108460 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108804 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108626 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108597 ns/op            1289 B/op         23 allocs/op
BenchmarkDecredVerify-16           10000            108497 ns/op            1289 B/op         23 allocs/op
BenchmarkGethSign-16               74712             15838 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               76188             15809 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               75249             15764 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               74839             15921 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               75200             15857 ns/op             164 B/op          3 allocs/op
BenchmarkGethVerify-16             63654             18628 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65186             18568 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             64012             18448 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             63954             18885 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             66043             18588 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/islishude/secp256k1/benchmark        35.936s
```
