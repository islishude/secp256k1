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
| Local          |       28579 |       28238 |       28587 |       28330 |       28281 |     28403 |    0 |         0 |
| Decred         |       31778 |       31822 |       31781 |       31788 |       31935 |     31821 | 1512 |        28 |

## Recoverable Sign Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       28526 |       28354 |       28337 |       28080 |       28175 |     28294 |    0 |         0 |
| Decred         |       31859 |       31684 |       31632 |       31816 |       31961 |     31790 | 1592 |        29 |
| Geth           |       15789 |       15889 |       15793 |       15845 |       15822 |     15828 |  164 |         3 |

## Verify Results

| Implementation | Run 1 ns/op | Run 2 ns/op | Run 3 ns/op | Run 4 ns/op | Run 5 ns/op | Avg ns/op | B/op | allocs/op |
| -------------- | ----------: | ----------: | ----------: | ----------: | ----------: | --------: | ---: | --------: |
| Local          |       35733 |       35721 |       35693 |       35674 |       35724 |     35709 |    0 |         0 |
| Decred         |      111512 |      112812 |      112248 |      113169 |      112376 |    112423 |  568 |        12 |
| Geth           |       18323 |       18331 |       18456 |       18585 |       18457 |     18430 |    0 |         0 |

## Conclusions

- Geth is the fastest implementation for recoverable signing and verification
  on this machine.
- Local compact signing is about 1.1x faster than Decred compact signing while
  using zero benchmark-time allocations.
- Local recoverable signing is about 1.1x faster than Decred recoverable
  signing while using zero benchmark-time allocations.
- Local verification is about 3.1x faster than Decred verification while using
  zero benchmark-time allocations.
- Local verification is about 1.9x slower than geth verification on this
  machine.

## Raw Output

```text
goos: darwin
goarch: arm64
pkg: github.com/islishude/secp256k1/benchmark
cpu: Apple M3 Max
BenchmarkLocalSignCompact-16         	   41001	     28579 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42414	     28238 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   41942	     28587 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42427	     28330 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignCompact-16         	   42339	     28281 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   42538	     28526 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   42415	     28354 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   42098	     28337 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   42350	     28080 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalSignRecoverable-16     	   42566	     28175 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33600	     35733 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33696	     35721 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33541	     35693 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33642	     35674 ns/op	       0 B/op	       0 allocs/op
BenchmarkLocalVerify-16              	   33675	     35724 ns/op	       0 B/op	       0 allocs/op
BenchmarkDecredSignCompact-16        	   38130	     31778 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   38149	     31822 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   37422	     31781 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   37514	     31788 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignCompact-16        	   37316	     31935 ns/op	    1512 B/op	      28 allocs/op
BenchmarkDecredSignRecoverable-16    	   37822	     31859 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   37984	     31684 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   37974	     31632 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   37314	     31816 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredSignRecoverable-16    	   37704	     31961 ns/op	    1592 B/op	      29 allocs/op
BenchmarkDecredVerify-16             	   10000	    111512 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000	    112812 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000	    112248 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000	    113169 ns/op	     568 B/op	      12 allocs/op
BenchmarkDecredVerify-16             	   10000	    112376 ns/op	     568 B/op	      12 allocs/op
BenchmarkGethSignRecoverable-16      	   76198	     15789 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   75644	     15889 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   76005	     15793 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   75171	     15845 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethSignRecoverable-16      	   75681	     15822 ns/op	     164 B/op	       3 allocs/op
BenchmarkGethVerify-16               	   65664	     18323 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   67465	     18331 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   64137	     18456 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   64329	     18585 ns/op	       0 B/op	       0 allocs/op
BenchmarkGethVerify-16               	   66952	     18457 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/islishude/secp256k1/benchmark	48.214s
```
