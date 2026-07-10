# Performance baseline

This baseline was captured before changing production code for the performance
work. It is the comparison point for the staged optimizations in this branch.

```text
Commit: 8ae73ae24bede4f26b6ddaeae7a726e76795431b
Date: 2026-07-10
Go: go1.26.4
GOOS/GOARCH: darwin/arm64
CPU: Apple M3 Max
CGO_ENABLED: 1
Main-module dependencies: none
```

The raw ten-run summary is in `baseline-main.txt`. Representative medians are:

| Benchmark                              |      Baseline |            Allocations |
| -------------------------------------- | ------------: | ---------------------: |
| VerifyHotPublicKey                     |  36,432 ns/op |    0 B/op, 0 allocs/op |
| VerifyParseCompressedCold              |  70,652 ns/op | 9,472 B/op, 1 alloc/op |
| VerifyParseUncompressedCold            |  66,295 ns/op | 9,472 B/op, 1 alloc/op |
| SignCompact                            |  28,650 ns/op |    0 B/op, 0 allocs/op |
| SignRecoverable                        |  28,619 ns/op |    0 B/op, 0 allocs/op |
| RecoverDigest                          | 301,810 ns/op | 9,472 B/op, 1 alloc/op |
| PublicKeyDerive                        |  52,152 ns/op | 9,472 B/op, 1 alloc/op |
| ScalarInv                              |   5,693 ns/op |    0 B/op, 0 allocs/op |
| SplitEndomorphism                      |   141.5 ns/op |    0 B/op, 0 allocs/op |
| SignedWNAF256                          |   939.6 ns/op |    0 B/op, 0 allocs/op |
| DoubleScalarBaseMultPrecomputedVartime |  30,488 ns/op |    0 B/op, 0 allocs/op |
| ScalarBaseMultProjective               |  16,961 ns/op |    0 B/op, 0 allocs/op |
| NewAffineOddTable                      |  28,907 ns/op |    0 B/op, 0 allocs/op |
| AddAffine                              |   182.9 ns/op |    0 B/op, 0 allocs/op |
| CurveYFromX                            |   4,858 ns/op |    0 B/op, 0 allocs/op |

Commands:

```sh
go test ./...
go test -run '^$' -bench 'Benchmark(VerifyHotPublicKey|VerifyParseCompressedCold|VerifyParseUncompressedCold|SignCompact|SignRecoverable|RecoverDigest|PublicKeyDerive|ScalarInv|SplitEndomorphism|SignedWNAF256|DoubleScalarBaseMultPrecomputedVartime|ScalarBaseMultProjective|NewAffineOddTable|AddAffine|CurveYFromX)$' -benchmem -count=10 .
go test -run '^$' -bench '^BenchmarkVerifyHotPublicKey$' -benchtime=3s -cpuprofile /tmp/secp256k1-verify-baseline.cpu .
go test -run '^$' -bench '^BenchmarkSignRecoverable$' -benchtime=3s -cpuprofile /tmp/secp256k1-sign-baseline.cpu .
```

The profiles confirm that field multiplication/squaring dominates both paths.
Scalar inversion accounts for about 14% of hot verification, while constant-time
base-point multiplication accounts for about 57% of signing. This supports the
planned order: public scalar inversion and GLV/wNAF first, then base-point work.
