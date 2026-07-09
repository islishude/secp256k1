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
- The fixture derives each implementation's public key directly from the fixed
  private key, then separately checks verification and recovery.
- Results are from this machine only. Different CPUs, Go versions, or CGO
  settings can change the ranking, especially for geth.

## Compact Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       28167 |       28158 |       28094 |       28125 |       28208 |     28150 |    0 |         0 |
| Decred         |       32248 |       32481 |       32108 |       32271 |       32198 |     32261 | 1512 |        28 |

## Recoverable Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       28383 |       28206 |       28133 |       28111 |       28239 |     28214 |    0 |         0 |
| Decred         |       32474 |       32345 |       32403 |       32377 |       32415 |     32403 | 1592 |        29 |
| Geth           |       15817 |       15861 |       15871 |       15868 |       15864 |     15856 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       36672 |       36678 |       36675 |       36695 |       36660 |     36676 |    0 |         0 |
| Decred         |      115510 |      115552 |      114880 |      114773 |      112786 |    114700 |  568 |        12 |
| Geth           |       18459 |       18579 |       18475 |       18544 |       18289 |     18469 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation for recoverable signing and verification
  on this machine.
- Local compact signing is about 1.15x faster than Decred compact signing while
  using zero benchmark-time allocations.
- Local recoverable signing is about 1.15x faster than Decred recoverable
  signing while using zero benchmark-time allocations.
- Local verification is about 3.13x faster than Decred verification while using
  zero benchmark-time allocations.
- Local verification is about 1.98x slower than geth verification on this
  machine.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSignCompact-16          	   42482 	    28167 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16          	   42135 	    28158 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16          	   42540 	    28094 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16          	   42669 	    28125 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16          	   42626 	    28208 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16      	   42589 	    28383 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16      	   42580 	    28206 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16      	   42637 	    28133 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16      	   42844 	    28111 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16      	   42763 	    28239 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16               	   32774 	    36672 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16               	   32726 	    36678 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16               	   32749 	    36675 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16               	   32698 	    36695 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16               	   32683 	    36660 ns/op	       0 B/op	       0 allocs/op
BenchmarkDecredSignCompact-16         	   36879 	    32248 ns/op	 1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16         	   37302 	    32481 ns/op	 1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16         	   37440 	    32108 ns/op	 1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16         	   37150 	    32271 ns/op	 1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16         	   37358 	    32198 ns/op	 1512 B/op	      28 allocs/op
BenchmarkDecredSignRecoverable-16     	   37281 	    32474 ns/op	 1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16     	   37431 	    32345 ns/op	 1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16     	   37630 	    32403 ns/op	 1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16     	   36631 	    32377 ns/op	 1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16     	   36922 	    32415 ns/op	 1592 B/op	      29 allocs/op
BenchmarkDecredVerify-16              	  10000 	   115510 ns/op	    568 B/op	      12 allocs/op
BenchmarkDecredVerify-16              	   9859 	   115552 ns/op	    568 B/op	      12 allocs/op
BenchmarkDecredVerify-16              	   9964 	   114880 ns/op	    568 B/op	      12 allocs/op
BenchmarkDecredVerify-16              	  10000 	   114773 ns/op	    568 B/op	      12 allocs/op
BenchmarkDecredVerify-16              	  10000 	   112786 ns/op	    568 B/op	      12 allocs/op
BenchmarkGethSignRecoverable-16       	   75866 	    15817 ns/op	    164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16       	   75585 	    15861 ns/op	    164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16       	   75505 	    15871 ns/op	    164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16       	   75751 	    15868 ns/op	    164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16       	   75440 	    15864 ns/op	    164 B/op	       3 allocs/op
BenchmarkGethVerify-16                	   65779 	    18459 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16                	   65872 	    18579 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16                	   65626 	    18475 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16                	   65612 	    18544 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16                	   65259 	    18289 ns/op	       0 B/op	       0 allocs/op
PASS
ok 	github.com/islishude/secp256k1/benchmark	48.528s
```
