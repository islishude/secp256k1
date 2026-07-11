# ARM64 signing stretch result

Date: 2026-07-11  
Go: go1.26.4  
GOOS/GOARCH: darwin/arm64  
CPU: Apple M3 Max  
Baseline revision: `73ea04a`

## Result

The opt-in `secp256k1_asm` backend now meets the 19,500 ns recoverable-signing
stretch target while preserving zero benchmark-time allocations. The final
retained ten-run median is 18,542.5 ns/op.

| Benchmark | Baseline median | Final median | Delta | Allocations |
| --- | ---: | ---: | ---: | ---: |
| SignRecoverable | 23,884.0 ns | 18,542.5 ns | -22.36% | 0 B, 0 allocs |
| ScalarBaseMultProjective | 13,027.5 ns | 9,720.5 ns | -25.38% | 0 B, 0 allocs |
| ScalarInv | 5,590.0 ns | 4,248.0 ns | -24.01% | 0 B, 0 allocs |
| FieldInv | 3,920.5 ns | 3,022.5 ns | -22.91% | 0 B, 0 allocs |
| ScalarSquareN/64 | 1,205.0 ns | 844.7 ns | -29.90% | 0 B, 0 allocs |
| FieldSquareN/64 | 904.9 ns | 673.3 ns | -25.59% | 0 B, 0 allocs |

An adjacent baseline/final verification run showed no hot-path regression and
improved both cold parse workloads:

| Benchmark | Baseline median | Final median | Delta |
| --- | ---: | ---: | ---: |
| VerifyHotPublicKey | 19,523.0 ns | 19,527.5 ns | +0.02% |
| VerifyParseCompressedCold | 59,128.5 ns | 57,299.0 ns | -3.09% |
| VerifyParseUncompressedCold | 55,408.5 ns | 52,760.0 ns | -4.78% |

## Retained changes

- Repeated squaring in generated field and scalar addition chains now uses
  register-resident ARM64 `SquareN` loops. Their loop counts are public static
  addition-chain constants.
- Complete mixed addition uses a specialized multiplication by `3*b = 21`.
  It measured 2.88 ns versus 10.73 ns for a general field multiplication.
- The signed W5 generator path still scans every one of its sixteen packed
  entries, but the opt-in ARM64 implementation performs the scan with a fully
  unrolled NEON sequence. The secret digit is used only by `CMP`/`CSETM`, never
  as an address or branch condition.
- The default path keeps the original inline Go table scan. Against an extracted
  baseline tree, its ten-run medians changed by +0.22% for base multiplication,
  +0.92% for signing, -0.14% for hot verification, -1.18% for compressed cold
  verification, and -0.14% for uncompressed cold verification.

## Rejected and skipped candidates

- The standalone scalar Square assembly candidate improved its microbenchmark
  by 15.4%, below the required 20% gate. It was removed; scalar `Square` still
  calls fiat-crypto, while the independently successful `SquareN` remains.
- W6 retuning was skipped in this phase because W5 reached its signing target.
  It was subsequently evaluated as an isolated follow-up and retained for the
  opt-in ARM64 backend; see [`20260711-arm64-w6.md`](20260711-arm64-w6.md).
  Fused complete-mixed-add assembly remains out of scope. The verification comb
  remains width 7 and retains the existing per-public-key memory policy.

## Security and validation

- Field/scalar Square, SquareN, and multiplication by 21 were checked against
  fiat/general-multiplication oracles over edge cases, aliasing cases, and at
  least 100,000 deterministic random inputs.
- W5 selection was checked for all 52 windows and magnitudes 0 through 16;
  fixed-base multiplication retains the existing 1,000-random-scalar W4/W5/W6
  differential test.
- Default and tagged builds produce the same checked-in deterministic
  recoverable-signature golden value.
- Disassembly contains no conditional branches in field Square, multiplication
  by 21, or W5 selection. `SquareN` contains only public-count `CBZ`/`JMP`
  instructions, and the selector reads the table through a fixed incrementing
  pointer.
- Default/tagged race tests, vet, golangci-lint, constant-time smoke, fuzz smoke,
  dependency checks, generation reproducibility, benchmark-module tests, and
  Linux ARM64/AMD64 cross-builds passed.

The backend remains behind `secp256k1_asm`; independent review is still required
before considering default enablement. Exact benchmark output is stored in
`20260711-arm64-sign-sprint.txt`.
