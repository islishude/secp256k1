# Performance result: 2026-07-10 working tree

```text
Commit: working tree based on 8ae73ae24bede4f26b6ddaeae7a726e76795431b
Go: go1.26.4
GOOS/GOARCH: darwin/arm64
CPU: Apple M3 Max
CGO_ENABLED: 1
Command: go test -run '^$' -bench '<final performance set>' -benchmem -count=10 .
Baseline commit: 8ae73ae24bede4f26b6ddaeae7a726e76795431b
```

| Benchmark                              | Baseline median | Final median |           Delta | Final allocations |
| -------------------------------------- | --------------: | -----------: | --------------: | ----------------: |
| VerifyHotPublicKey                     |       36,432 ns |    30,143 ns |          -17.3% |     0 B, 0 allocs |
| VerifyParseCompressedCold              |       70,652 ns |    64,415 ns |           -8.8% |  9,472 B, 1 alloc |
| VerifyParseUncompressedCold            |       66,295 ns |    60,053 ns |           -9.4% |  9,472 B, 1 alloc |
| SignCompact                            |       28,650 ns |    24,706 ns |          -13.8% |     0 B, 0 allocs |
| SignRecoverable                        |       28,619 ns |    24,632 ns |          -13.9% |     0 B, 0 allocs |
| RecoverDigest                          |      301,810 ns |   127,136 ns |          -57.9% |  9,472 B, 1 alloc |
| PublicKeyDerive                        |       52,152 ns |    47,616 ns |           -8.7% |  9,472 B, 1 alloc |
| ScalarInv                              |        5,693 ns |     5,614 ns |           -1.4% |     0 B, 0 allocs |
| ScalarInvVartime                       |             n/a |     2,397 ns |             n/a |     0 B, 0 allocs |
| SplitEndomorphism                      |        141.5 ns |     151.5 ns |     oracle only |     0 B, 0 allocs |
| SplitEndomorphismVartimeWords          |             n/a |      99.3 ns |             n/a |     0 B, 0 allocs |
| SignedWNAF256                          |        939.6 ns |     931.5 ns |     oracle only |     0 B, 0 allocs |
| SignedWNAFHalf                         |             n/a |     332.4 ns |             n/a |     0 B, 0 allocs |
| DoubleScalarBaseMultPrecomputedVartime |       30,488 ns |    27,962 ns |           -8.3% |     0 B, 0 allocs |
| ScalarBaseMultProjective               |       16,961 ns |    13,491 ns |          -20.5% |     0 B, 0 allocs |
| AddAffine                              |        182.9 ns |     180.9 ns |     oracle only |     0 B, 0 allocs |
| AddAffineWNAFVartime                   |             n/a |     167.4 ns |             n/a |     0 B, 0 allocs |
| PointDoubleSquare                      |        157.2 ns |     157.2 ns |     oracle only |     0 B, 0 allocs |
| PointDouble                            |             n/a |     144.4 ns | -8.1% vs oracle |     0 B, 0 allocs |
| CurveYFromX                            |        4,858 ns |     4,742 ns |           -2.4% |     0 B, 0 allocs |

Result summary:

- Verification and signing both improved without adding allocations.
- P1's 33,000 ns hot-verify goal was met after subsequent work; the later
  29,000/25,000/23,000 ns engineering goals were not met on this machine.
- The 23,500 ns signing goal was missed by about 4.8%; the result remains a
  stable 13.9% improvement over baseline.
- Static table loading measured roughly 86-89 us versus roughly 1.36 ms for
  dynamic construction. The benchmark's large-array escape allocations are not
  representative of package-level global initialization.
- Generated table headers record parameters, generator-source SHA-256, and table
  data SHA-256. A mutable git commit id is intentionally omitted so committing a
  generated file does not make the next deterministic `go generate` dirty.
- Main-module dependencies remain empty and the public API is unchanged.
