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
| Local          |       26253 |       25805 |       25850 |       26018 |       25990 |     25983 |    0 |         0 |
| Decred         |       31661 |       32305 |       31679 |       31559 |       31632 |     31767 | 1512 |        28 |
| Geth           |       15548 |       15564 |       15629 |       15729 |       15733 |     15641 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       39986 |       39968 |       40019 |       40013 |       40057 |     40009 |    0 |         0 |
| Decred         |      108433 |      108542 |      108444 |      108679 |      108452 |    108510 |  568 |        12 |
| Geth           |       18176 |       18349 |       18189 |       18223 |       18411 |     18270 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation in both operations on this machine. It is about 1.7x faster than local signing and about 2.2x faster than local verification.
- Local signing is faster than Decred signing while using zero allocations. Decred allocates 1512 B/op across 28 allocations.
- Local verification is substantially faster than Decred verification. It is about 2.7x faster while using zero allocations.
- The local implementation has zero benchmark-time allocations for both sign and verify.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSign-16              44270             26253 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              46464             25805 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              46408             25850 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              46401             26018 ns/op               0 B/op          0 allocs/op
BenchmarkLocalSign-16              46000             25990 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            29971             39986 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            29956             39968 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            29940             40019 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            29908             40013 ns/op               0 B/op          0 allocs/op
BenchmarkLocalVerify-16            29986             40057 ns/op               0 B/op          0 allocs/op
BenchmarkDecredSign-16             37906             31661 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             36792             32305 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             37556             31679 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             37944             31559 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredSign-16             37718             31632 ns/op            1512 B/op         28 allocs/op
BenchmarkDecredVerify-16           10000            108433 ns/op             568 B/op         12 allocs/op
BenchmarkDecredVerify-16           10000            108542 ns/op             568 B/op         12 allocs/op
BenchmarkDecredVerify-16           10000            108444 ns/op             568 B/op         12 allocs/op
BenchmarkDecredVerify-16           10000            108679 ns/op             568 B/op         12 allocs/op
BenchmarkDecredVerify-16           10000            108452 ns/op             568 B/op         12 allocs/op
BenchmarkGethSign-16               77763             15548 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               78229             15564 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               75800             15629 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               76623             15729 ns/op             164 B/op          3 allocs/op
BenchmarkGethSign-16               77146             15733 ns/op             164 B/op          3 allocs/op
BenchmarkGethVerify-16             66776             18176 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             66868             18349 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             67657             18189 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65671             18223 ns/op               0 B/op          0 allocs/op
BenchmarkGethVerify-16             65973             18411 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/islishude/secp256k1/benchmark        35.726s
```
