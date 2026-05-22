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
| Local          |       36872 |       35947 |       35877 |       35863 |       36289 |     36170 |    0 |         0 |
| Decred         |       34266 |       34614 |       33747 |       34053 |       33951 |     34126 | 1512 |        28 |
| Geth           |       16145 |       16061 |       15985 |       15951 |       16232 |     16075 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       93144 |       92408 |       91301 |       92147 |       92156 |     92231 |    0 |         0 |
| Decred         |      113312 |      115938 |      111763 |      112333 |      114787 |    113627 | 1289 |        23 |
| Geth           |       18540 |       18723 |       18502 |       18752 |       18602 |     18624 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation in both operations on this machine. It is about 2.2x faster than local signing and about 5.0x faster than local verification.
- Local signing is close to Decred signing in throughput, while using zero allocations. Decred is slightly faster in this run, but allocates 1512 B/op across 28 allocations.
- Local verification is faster than Decred verification and uses zero allocations, but remains significantly slower than geth's CGO-backed implementation.
- The local implementation now has zero benchmark-time allocations for both sign and verify.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSign-16       	   32514	     36872 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSign-16       	   33484	     35947 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSign-16       	   33403	     35877 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSign-16       	   33493	     35863 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSign-16       	   32980	     36289 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16     	   13071	     93144 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16     	   12919	     92408 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16     	   13018	     91301 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16     	   12998	     92147 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16     	   13053	     92156 ns/op	       0 B/op	       0 allocs/op
BenchmarkDecredSign-16      	   34999	     34266 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16      	   33756	     34614 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16      	   35628	     33747 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16      	   35402	     34053 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSign-16      	   35136	     33951 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredVerify-16    	   10000	    113312 ns/op	    1289 B/op	      23 allocs/op
BenchmarkDecredVerify-16    	   10000	    115938 ns/op	    1289 B/op	      23 allocs/op
BenchmarkDecredVerify-16    	   10000	    111763 ns/op	    1289 B/op	      23 allocs/op
BenchmarkDecredVerify-16    	   10000	    112333 ns/op	    1289 B/op	      23 allocs/op
BenchmarkDecredVerify-16    	   10000	    114787 ns/op	    1289 B/op	      23 allocs/op
BenchmarkGethSign-16        	   73521	     16145 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16        	   74608	     16061 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16        	   75058	     15985 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16        	   74269	     15951 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSign-16        	   72765	     16232 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethVerify-16      	   67321	     18540 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16      	   62517	     18723 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16      	   65482	     18502 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16      	   64302	     18752 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16      	   63904	     18602 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/islishude/secp256k1/benchmark	40.547s
```
