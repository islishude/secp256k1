# Performance and backend status

Last updated: 2026-09-08

This document is the maintained summary of the repository's performance work.
Measurements are evidence for the decisions below, not portable performance
promises: Go versions, CPUs, operating systems, thermals, and runner placement
all affect absolute values.

## Current backends

| Build | Field arithmetic | Fixed-base multiplication | Status |
| --- | --- | --- | --- |
| Default, all architectures | Generated fiat-crypto Go | Constant-time signed-W5 table | Enabled |
| `arm64 && secp256k1_asm` | Assembly Add, Mul, MulByB3, Square, and SquareN; fiat Sub; assembly scalar SquareN | Constant-time signed-W6 table with a fixed NEON scan | Opt-in; independent review pending |
| `amd64 && secp256k1_asm` | ADX+BMI2 Mul and Square after explicit CPUID detection; fiat fallback and all other operations | Constant-time signed-W6 table with a baseline-SSE2 scan | Opt-in; independent review pending |

The build tag never changes the public API, signature format, RFC6979 behavior,
or verification semantics. AMD64 does not infer ADX support from `GOAMD64`; a
tagged binary falls back to fiat Mul and Square unless both ADX and BMI2 are
present. The W6 selector only needs baseline AMD64/SSE2.

Verification is allowed to use public-input variable-time algorithms. A public
key starts with a small shared cache handle. Its first call reaching
double-scalar multiplication builds wNAF/GLV tables; after eight such calls, one
caller builds a width-7 comb table. Concurrent callers continue on the immutable
wNAF table while promotion runs. Publication replaces the table pointer, allowing
the old table to be reclaimed after its last reader finishes. Hot verification
loads the published comb table without incrementing the use counter. Signing and
private-key derivation continue to use fixed-window, full-table-scan paths.

## Retained measurements

### Default Go path

The initial and retained-default measurements below were collected on an Apple
M3 Max with Go 1.26.4. The retained values include the adaptive verification
comb and the signed-W5 fixed-base path.

| Benchmark | Initial baseline | Retained default | Delta |
| --- | ---: | ---: | ---: |
| VerifyHotPublicKey | 36,432 ns | 20,651 ns | -43.3% |
| VerifyParseCompressedCold | 70,652 ns | 64,868 ns | -8.2% |
| VerifyParseUncompressedCold | 66,295 ns | 60,374 ns | -8.9% |
| SignRecoverable | 28,619 ns | 24,694 ns | -13.7% |

These changes kept hot verification and signing at zero benchmark-time
allocations. These historical measurements used eager wNAF construction and
retained both representations after promotion. The review fixes described below replace that
cache layout.

### ARM64 opt-in path

The final ARM64 sprint used the same M3 Max and Go 1.26.4. The comparison is
against the immediately preceding signed-W6 tagged backend, so it isolates the
retained base-field Mul stage.

| Benchmark | W6 baseline | Final ARM64 | Delta |
| --- | ---: | ---: | ---: |
| FieldMul | 10.48 ns | 8.7315 ns | -16.68% |
| ScalarBaseMultProjective | 8,270.5 ns | 7,607.0 ns | -8.02% |
| SignRecoverable | 16,774.5 ns | 15,367.5 ns | -8.39% |
| VerifyHotPublicKey | 19,672.0 ns | 17,464.5 ns | -11.22% |
| VerifyParseCompressedCold | 56,510.0 ns | 49,735.5 ns | -11.99% |
| VerifyParseUncompressedCold | 53,427.0 ns | 47,148.0 ns | -11.75% |

All listed hot operations remained at 0 B/op and 0 allocs/op. The W6 table is
88,064 bytes, 34,816 bytes larger than W5 and below the retained 40 KiB growth
limit. Only one fixed-base table is linked into a production binary.

### AMD64 opt-in path

The accepted implementation is base-field Mul/Square plus W6. The final cleanup
run used Intel Xeon Platinum 8370C runners, pinned every benchmark process to
one CPU and one Go P, and measured ten alternating default/tagged pairs. The
hard gate uses the median of the ten paired percentage deltas.

| GOAMD64 | Benchmark | Default median | Tagged median | Paired median delta |
| --- | --- | ---: | ---: | ---: |
| v1 | SignRecoverable | 46,263.0 ns | 36,897.5 ns | -20.26% |
| v1 | VerifyHotPublicKey | 36,357.5 ns | 31,575.5 ns | -13.14% |
| v3 | SignRecoverable | 46,116.5 ns | 36,641.5 ns | -20.54% |
| v3 | VerifyHotPublicKey | 36,026.0 ns | 31,412.5 ns | -12.80% |

All four rows remained at 0 B/op and 0 allocs/op. The original acceptance run is
[GitHub Actions 29345965487](https://github.com/islishude/secp256k1/actions/runs/29345965487);
the paired-gate cleanup validation is
[GitHub Actions 29388877833](https://github.com/islishude/secp256k1/actions/runs/29388877833).

## Measurement and CI policy

The consolidated [`CI` workflow](../.github/workflows/ci.yaml) separates
correctness from performance:

- Root, generator, benchmark-module, race, fuzz, and tagged-backend correctness
  checks run on pull requests and `main` pushes.
- Compatibility benchmarks run five times on Linux AMD64, Linux ARM64, macOS
  ARM64, and Windows AMD64, including the benchmark module's cross-check tests.
- The expensive AMD64 paired performance gate runs on pull requests and manual
  dispatches after the quality job passes. It runs independently for
  `GOAMD64=v1` and `v3`.
- Signing and hot verification must each improve by at least 10%, with every
  sample at 0 B/op and 0 allocs/op. Nine other end-to-end workloads may regress
  by at most 1% by paired median.
- Per-kernel and W5/W6 comparisons remain diagnostic reports. The accepted
  backend is gated end to end so runner-to-runner CPU changes cannot rewrite an
  earlier acceptance decision.

On a Linux AMD64 host with ADX+BMI2, the workflow steps can be reproduced with:

```sh
output=$(mktemp -d)
GOAMD64=v1 ./scripts/benchmark-amd64.sh environment "$output"
GOAMD64=v1 ./scripts/benchmark-amd64.sh benchmarks "$output"
GOAMD64=v1 ./scripts/benchmark-amd64.sh profiles "$output"
GOAMD64=v1 ./scripts/benchmark-amd64.sh report "$output"
GOAMD64=v1 ./scripts/benchmark-amd64.sh gate "$output"
```

Do not compare absolute values from different jobs or machines. Alternate the
two binaries on one runner, keep sample positions paired, pin one CPU and one Go
P, and retain the runner description with the results.

## Rejected or superseded experiments

Only conclusions that still constrain current work are retained here:

- A pure-Go pseudo-Mersenne field backend was roughly three times slower than
  fiat and was removed. Its abstraction alone also harmed inlining.
- A constant-time safegcd scalar inverse was correct but measured about 13.4 us
  versus about 5.6 us for the retained addition chain.
- Generator wNAF window 14 and the earlier Go signed-W6 table missed their
  size/performance gates. Default production remains signed W5; W6 is used only
  by the opt-in assembly builds where its whole-workload gate passed.
- ARM64 standalone scalar Square missed its 20% microbenchmark gate. Fused
  complete-mixed-add assembly was not needed after the retained stages met the
  signing targets.
- AMD64 stack-backed Montgomery kernels were too slow. Register SquareN and
  MulByB3 improved their microbenchmarks but regressed an unaffected hot path by
  more than 1%, so only Mul and Square remain.
- The AMD64 v2 scalar Montgomery group failed its combined gates. A public-input
  assembly inversion improved verification, but the terminal inversion plus
  field Add/Sub candidate improved signing by only 1.88% at v1 and 1.40% at v3,
  below the required 3% incremental gain. All v2 production symbols, routing,
  and generator code were removed.

## Historical evidence

The former date-stamped reports, audit transcripts, raw benchmark outputs, and
rejection logs were consolidated into this document. They remain recoverable
from Git history:

| Commit | Evidence |
| --- | --- |
| `2ae7217` | Initial baseline, window matrices, and rejected P8 experiments |
| `73ea04a` | Adaptive comb and first ARM64 assembly results |
| `4aa13ce` | ARM64 repeated-squaring and fixed-base sprint |
| `37eab34` | Final ARM64 W6 and field-Mul evidence |
| `245020e` | Accepted AMD64 avo backend, v2 rejection, and CI artifacts |

The maintained external-review scope is in [`audit.md`](audit.md).

## Review fixes: cache lifecycle and measurement

`BenchmarkParsePublicKeyOnly` isolates key decoding. `BenchmarkVerifyPublicKeyLifetime`
includes parsing and 1, 8, 9, 16, or 32 verifications, including the switch cost.
`BenchmarkVerifyConcurrentPromotion` measures eight concurrent callers at the
switch, and `BenchmarkVerifyHotPublicKeyParallel` measures shared-key throughput.
The hot verification benchmarks explicitly warm the cache before starting the
measurement; lifecycle benchmarks retain all initialization costs.

The threshold remains eight: promotion costs about eight to nine saved wNAF
operations on the reviewed M3 Max. This is a reuse heuristic, not a guarantee that
a key used exactly nine times benefits from promotion. The change focuses on
avoiding unused tables, releasing the old representation, and keeping concurrent
readers off the promotion lock. No background goroutine is created.

The paired gate requires at least ten matched samples for every required workload.
Every sample must contain finite positive ns/op and finite nonnegative B/op and
allocs/op. Missing metrics are shown as missing in reports and rejected by gates.
Signing and hot verification require zero allocations in every paired sample.

Fixed-size batch normalization implementations are emitted from one algorithm in
`cmd/genprecomp`. Production variants and test-only variants are generated into
separate files. Reference scalar multiplication and older fixed-base algorithms
live in `reference_test.go` and are excluded from production package builds.

### Local before/after measurements

Measured on 2026-09-08, Apple M3 Max, macOS ARM64, Go 1.27.1. Baseline is
`478ba48`, with the same lifecycle benchmarks and explicit hot-key warmup added
for comparison. Ten alternating before/after pairs used precompiled test binaries,
`GOMAXPROCS=1`, and `-test.benchtime=150ms`. Concurrent measurements used five pairs,
`GOMAXPROCS=4`, and 200 ms. No other test or generation jobs ran during these pairs.
CPU affinity was not pinned on macOS; these are local diagnostics, not the native
AMD64 release gate. Percentages below are medians of paired percentage changes.

| Workload | Default delta | ARM64 asm delta |
| --- | ---: | ---: |
| ParsePublicKeyOnly | -86.10% | -88.62% |
| VerifyPublicKeyLifetime/1 | +2.13% | -1.54% |
| VerifyPublicKeyLifetime/8 | -0.80% | -1.58% |
| VerifyPublicKeyLifetime/9 | -0.20% | -0.65% |
| VerifyPublicKeyLifetime/16 | +0.66% | -1.59% |
| VerifyPublicKeyLifetime/32 | +1.14% | -0.62% |
| RecoverDigest | -0.59% | -0.83% |
| VerifyHotPublicKey | +0.29% | -1.08% |
| VerifyParseCompressedCold | +1.30% | -2.86% |
| SignRecoverable | +1.32% | -0.88% |
| PublicKeyDerive | -63.06% | -69.34% |
| VerifyConcurrentPromotion | -14.74% | -16.74% |
| VerifyHotPublicKeyParallel | -1.71% | -1.33% |

Every signing and hot-verification sample remained at 0 B/op and 0 allocs/op.
Parsing/derivation now allocates a 32-byte handle instead of 9,472 bytes.
First verification adds a 9,472-byte table; promotion allocates a replacement of
that size and makes the old table eligible for GC after readers finish. Consequently
cold parse-and-verify/recovery totals 9,504 B/op in two allocations, versus the
former 9,472 B/op in one. A promoted key retains one representation instead of two.

The large parse/derive improvements and the concurrent-promotion gains are the
main measured benefits. Small changes around +/-2% remain noise-sensitive; the
default signing and single-use verification increases are recorded explicitly.
The promotion threshold remains eight. Large builder workspaces are isolated in
a non-inlined helper; the hot wrapper's ARM64 stack frame is 240 bytes.
