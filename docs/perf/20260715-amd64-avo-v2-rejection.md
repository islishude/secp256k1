# AMD64 avo v2 optimization rejection report

Date: 2026-07-15

Branch: `codex/amd64-avo-asm-v2`

Accepted baseline: `4ac5bc4` on `codex/amd64-avo-asm`

Accepted baseline run: [GitHub Actions 29345965487](https://github.com/islishude/secp256k1/actions/runs/29345965487)

Terminal experimental candidate: `d4dc372`

Cleanup commit: `b6a2a91`

Benchmark-method commit: `2dfad4d`

Final cleanup run: [GitHub Actions 29388877833](https://github.com/islishude/secp256k1/actions/runs/29388877833)

## Decision

The v2 candidate set is rejected. None of its production kernels, routing,
generator code, benchmark selectors, or assembly declarations are retained.
The final `asm/`, `internal/field`, `internal/scalar`, and W6 production files
are byte-for-byte identical to accepted baseline `4ac5bc4`.

The generated public-input scalar inversion was a strong verification-only
candidate, but scalar Montgomery failed its group thresholds and field Add/Sub
did not supply the missing signing gain. The terminal combination improved
`SignRecoverable` by only 1.88% at `GOAMD64=v1` and 1.40% at v3, below the 3%
incremental requirement at both levels. The plan therefore requires deletion
of every v2 production candidate, including the independently successful
public-input inversion.

The accepted AMD64 backend remains unchanged: opt-in `secp256k1_asm`, explicit
ADX+BMI2 CPUID dispatch, base-field Mul/Square, and the fixed-length SSE2 W6
selector. Default builds remain pure Go.

## Final cleanup validation

The final cleanup run passed every quality, correctness, platform, assembly,
and performance job. Both performance jobs ran on Intel Xeon Platinum 8370C
CPUs, reported ADX+BMI2, allowed CPUs 0-3, and pinned each benchmark process to
CPU 0 with one Go P.

Independent medians remain useful descriptive values; the hard gate is the
median of the ten same-iteration percentage deltas. Negative values are
faster.

| GOAMD64 | Benchmark | Default median | Tagged median | Independent delta | Paired median delta | Allocations | Gate |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| v1 | SignRecoverable | 46,263.0 ns | 36,897.5 ns | -20.24% | -20.26% | 0 B / 0 | pass: >=10% |
| v1 | VerifyHotPublicKey | 36,357.5 ns | 31,575.5 ns | -13.15% | -13.14% | 0 B / 0 | pass: >=10% |
| v3 | SignRecoverable | 46,116.5 ns | 36,641.5 ns | -20.55% | -20.54% | 0 B / 0 | pass: >=10% |
| v3 | VerifyHotPublicKey | 36,026.0 ns | 31,412.5 ns | -12.81% | -12.80% | 0 B / 0 | pass: >=10% |

Every other gate-tracked paired median improved. The narrowest non-gate gain
was 11.69%, so no end-to-end workload approached the 1% regression limit. Native
Linux disassembly artifacts at v1 and v3 contain retained field Mul/Square and
W6 evidence and contain none of the six rejected v2 symbols.

## Measurement method

Each experimental run compared the accepted tagged backend with one candidate
or candidate combination in alternating order ten times on the same Ubuntu
runner. `GOAMD64=v1` and v3 are evaluated independently; absolute values from
different jobs or runs are never compared. All hot signing and verification
samples remained 0 B/op and 0 allocs/op.

The v2 runs were:

- [29375703485](https://github.com/islishude/secp256k1/actions/runs/29375703485): initial scalar CIOS Montgomery group.
- [29377187861](https://github.com/islishude/secp256k1/actions/runs/29377187861): XMM complement schedule and first full public inversion.
- [29378465358](https://github.com/islishude/secp256k1/actions/runs/29378465358): GPR complement schedule and generation-time rotation.
- [29379875980](https://github.com/islishude/secp256k1/actions/runs/29379875980): dedicated scalar Comba square.
- [29381506099](https://github.com/islishude/secp256k1/actions/runs/29381506099): public inversion plus conditional field Add/Sub.

## Stage one: scalar Montgomery

The initial scalar CIOS implementation made Mul fast enough, but failed the
Square and SquareN group gates. It also missed the v3 signing requirement by
0.03 percentage point. Negative deltas are faster.

| GOAMD64 | Scalar Mul | Scalar Square | Scalar SquareN | ScalarInv | SignRecoverable | VerifyHotPublicKey |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| v1 | -18.86% | -14.71% | -7.61% | -11.62% | -3.05% | -0.09% |
| v3 | -27.29% | -15.22% | -7.71% | -11.39% | -2.97% | -0.13% |

An XMM-resident complement schedule in run 29377187861 was slower and was
discarded. A GPR-resident schedule improved individual operations but still
failed SquareN, v3 ScalarInv, and signing:

| GOAMD64 | Scalar Mul | Scalar Square | Scalar SquareN | ScalarInv | SignRecoverable | VerifyHotPublicKey |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| v1 | -17.43% | -15.71% | -7.40% | -11.56% | -2.87% | +0.04% |
| v3 | -20.03% | -18.98% | -5.30% | -9.90% | -2.27% | -0.11% |

The final dedicated Comba-square experiment regressed SquareN and ScalarInv,
especially at v3, and did not improve signing materially:

| GOAMD64 | Scalar Mul | Scalar Square | Scalar SquareN | ScalarInv | SignRecoverable |
| --- | ---: | ---: | ---: | ---: | ---: |
| v1 | -17.30% | -2.84% | +9.42% | +2.78% | +0.65% |
| v3 | -7.80% | +35.05% | +17.04% | +8.33% | -0.19% |

The scalar Mul/Square/SquareN group was therefore deleted in full rather than
retaining a partial routing that violated the group acceptance rules.

## Stage two: public-input inversion

The full four-limb Stein binary-GCD assembly loop passed its own performance
requirements. It used fixed addresses and only public signature scalar data
controlled its variable-time branches. The terminal run measured:

| GOAMD64 | ScalarInvVartime accepted | Candidate | Delta | Verify accepted | Candidate | Delta | Sign delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| v1 | 5,426.5 ns | 2,586.0 ns | -52.34% | 32,633.5 ns | 29,831.0 ns | -8.59% | +0.21% |
| v3 | 3,876.5 ns | 2,081.0 ns | -46.32% | 25,488.5 ns | 23,776.5 ns | -6.72% | +0.14% |

Recovery and cold verification did not regress. Signing, as expected, was
essentially unchanged, so the final double-3% target still required stage
three. Although this kernel passed its local gate, it is absent from the final
tree because the overall v2 target was not achieved.

## Stage three: base-field Add/Sub

Add passed its 10% microbenchmark threshold, but Sub was slower than the
accepted fiat path on both CPUs. The combined field candidate therefore failed
the stage gate before considering the final end-to-end result.

| GOAMD64 | Field Add accepted | Candidate | Delta | Field Sub accepted | Candidate | Delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| v1 | 6.25 ns | 5.00 ns | -19.99% | 3.82 ns | 4.38 ns | +14.57% |
| v3 | 4.12 ns | 3.25 ns | -21.18% | 2.53 ns | 2.91 ns | +15.00% |

Relative to the retained public-inversion stage, Add/Sub improved signing by
2.09%/1.53% and hot verification by 1.56%/0.32% at v1/v3. Relative to the
accepted backend, the full terminal combination produced:

| GOAMD64 | Sign accepted | Candidate | Delta | Verify accepted | Candidate | Delta | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| v1 | 35,497.0 ns | 34,830.5 ns | -1.88% | 32,633.5 ns | 29,367.0 ns | -10.01% | reject: sign <3% |
| v3 | 28,737.5 ns | 28,336.0 ns | -1.40% | 25,488.5 ns | 23,701.0 ns | -7.01% | reject: sign <3% |

The terminal experimental binary still exceeded the original default-to-tagged
10% gates, but that does not satisfy the new accepted-to-v2 double-3% rule.
Those default-to-terminal deltas were 25.10%/23.36% at v1 and 21.04%/16.02%
at v3 for signing/verification respectively.

## Correctness and assembly evidence

All experimental commits passed default and tagged full/race tests, generator
checks, dependency and vartime audits, Linux and Windows AMD64 v1/v3, and
Darwin AMD64 v1/v3 cross-builds. Candidate arithmetic covered edge values,
canonical outputs, all alias forms, and 100,000 deterministic oracle cases.

During the terminal experiment, field Add/Sub were branchless 137-byte and
101-byte symbols. The public inversion symbol was 1,678 bytes and used the
audited public-data-dependent Stein loop. These sizes document the rejected
artifacts only; final disassembly checks require all six v2 symbol names to be
absent from the linked root binary.

The final variable-time audit allows `InvVartime` production calls only from
verification and recovery and rejects any remaining public-inversion assembly
caller. No variable-time secret path was added.

## Retained CI framework

The AMD64 workflow continues to run ten interleaved default/tagged samples for
v1 and v3 and records field, scalar, W5/W6, Add/Sub, end-to-end workloads, CPU
features, affinity, symbol sizes, disassembly, and tagged
signing/verification CPU profiles. Benchmark processes are pinned to one CPU
with one Go P. The hard gate uses the median of the ten paired per-iteration
deltas, still requiring default-to-tagged double 10%, 0 B/op, 0 allocs/op, and
at most 1% paired-median regression elsewhere. Independent medians remain in
the artifact for transparency.

The previously accepted per-kernel and W6 medians remain in the artifact as
diagnostic reports rather than re-acceptance gates. This avoids invalidating
the immutable accepted decision when GitHub assigns a different CPU: in run
29381506099 the Intel Xeon Platinum 8573C measured retained FieldMul at 13.10%
while the complete accepted backend still comfortably passed both final
end-to-end gates. The original acceptance evidence remains run 29345965487.

The first cleanup run,
[29382922561](https://github.com/islishude/secp256k1/actions/runs/29382922561),
made the statistical issue concrete: v3 verification had bimodal default and
tagged samples, so dividing two independently selected medians reported 8.55%,
while the median of the ten paired deltas was 10.45%. The paired test uses the
actual samples from that run as a regression fixture; it does not lower the
10% requirement.

No width-8/width-7 change, W7 table, fused point arithmetic, AVX2 selector,
public API change, RFC6979 change, or signature encoding change was made.

Raw compact evidence is in
[`20260715-amd64-avo-v2-rejection.txt`](20260715-amd64-avo-v2-rejection.txt).
The external cleanup checklist is in
[`../audit/20260715-amd64-avo-v2-rejection-checklist.md`](../audit/20260715-amd64-avo-v2-rejection-checklist.md).
