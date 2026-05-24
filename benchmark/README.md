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
- The local signing benchmark uses `SignRecoverableDigest`; compact
  `SignDigest` is not benchmarked in this directory.
- Keys, digests, public keys, and signatures are prepared before timing starts.
- Results are from this machine only. Different CPUs, Go versions, or CGO settings can change the ranking, especially for geth.

## Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       29053 |       28648 |       28654 |       28440 |       28608 |     28681 |    0 |         0 |
| Decred         |       32408 |       32935 |       32701 |       32591 |       33040 |     32735 | 1512 |        28 |
| Geth           |       15981 |       16003 |       16059 |       16234 |       16103 |     16076 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       36477 |       36490 |       36577 |       36945 |       36525 |     36603 |    0 |         0 |
| Decred         |      117737 |      116691 |      112653 |      113671 |      113607 |    114872 |  568 |        12 |
| Geth           |       18846 |       18954 |       18583 |       18461 |       18572 |     18683 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation in both operations on this machine. It is about 1.8x faster than local recoverable signing and about 2.0x faster than local verification.
- Local recoverable signing is about 1.1x faster than Decred signing while using zero allocations. Decred allocates 1512 B/op across 28 allocations.
- Local verification is substantially faster than Decred verification. It is about 3.1x faster while using zero allocations.
- The local implementation has zero benchmark-time allocations for recoverable sign and verify.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSignRecoverable-16    	   40363	     29053 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16    	   41450	     28648 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16    	   41584	     28654 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16    	   42111	     28440 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16    	   42102	     28608 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16             	   32884	     36477 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16             	   32925	     36490 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16             	   32929	     36577 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16             	   31707	     36945 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16             	   32985	     36525 ns/op	       0 B/op	       0 allocs/op
BenchmarkDecredSign-16              	   37021	     32408 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16              	   36255	     32935 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16              	   36847	     32701 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16              	   36739	     32591 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16              	   36957	     33040 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredVerify-16            	    9943	    117737 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16            	   10000	    116691 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16            	   10000	    112653 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16            	   10000	    113671 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16            	   10000	    113607 ns/op	     568 B/op	      12 allocs/op
BenchmarkGethSign-16                	   75022	     15981 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16                	   75340	     16003 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16                	   75237	     16059 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16                	   74929	     16234 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16                	   74017	     16103 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethVerify-16              	   64305	     18846 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16              	   64557	     18954 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16              	   65808	     18583 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16              	   67634	     18461 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16              	   64347	     18572 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/islishude/secp256k1/benchmark	36.551s
```
