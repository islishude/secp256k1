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
| Local          |       28308 |       28226 |       28331 |       28345 |       28799 |     28402 |    0 |         0 |
| Decred         |       32841 |       33431 |       32881 |       32946 |       32900 |     33000 | 1512 |        28 |

## Recoverable Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       29253 |       29203 |       28680 |       28768 |       28913 |     28963 |    0 |         0 |
| Decred         |       33393 |       33388 |       33694 |       32825 |       33765 |     33413 | 1592 |        29 |
| Geth           |       16021 |       15986 |       15928 |       15982 |       16134 |     16010 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       36146 |       36400 |       35774 |       35868 |       35735 |     35985 |    0 |         0 |
| Decred         |      115670 |      112606 |      114400 |      113454 |      112438 |    113714 |  568 |        12 |
| Geth           |       18541 |       18862 |       18896 |       18673 |       18840 |     18762 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation for recoverable signing and verification
  on this machine.
- Local compact signing is about 1.16x faster than Decred compact signing while
  using zero benchmark-time allocations.
- Local recoverable signing is about 1.15x faster than Decred recoverable
  signing while using zero benchmark-time allocations.
- Local verification is about 3.16x faster than Decred verification while using
  zero benchmark-time allocations.
- Local verification is about 1.92x slower than geth verification on this
  machine.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSignCompact-16         	   42079 	    28308 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42292 	    28226 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42326 	    28331 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42342 	    28345 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42446 	    28799 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   41343 	    29253 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   40996 	    29203 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   41944 	    28680 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   41457 	    28768 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   40826 	    28913 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33416 	    36146 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   32823 	    36400 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33538 	    35774 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33201 	    35868 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33514 	    35735 ns/op	       0 B/op	       0 allocs/op
BenchmarkDecredSignCompact-16        	   37105 	    32841 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   36370 	    33431 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   36700 	    32881 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   35674 	    32946 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   36117 	    32900 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignRecoverable-16    	   35533 	    33393 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   37029 	    33388 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   36117 	    33694 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   36019 	    32825 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   35709 	    33765 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredVerify-16             	   10000 	   115670 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000 	   112606 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000 	   114400 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000 	   113454 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000 	   112438 ns/op	     568 B/op	      12 allocs/op
BenchmarkGethSignRecoverable-16      	   74607 	    16021 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   75393 	    15986 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   75180 	    15928 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   74834 	    15982 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   74624 	    16134 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethVerify-16               	   65312 	    18541 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   62320 	    18862 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   64773 	    18896 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   64491 	    18673 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   65077 	    18840 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/islishude/secp256k1/benchmark	48.704s
```
