# AMD64 avo assembly external audit checklist

Date: 2026-07-14

Scope: `amd64 && secp256k1_asm` CPUID dispatch, retained base-field Mul/Square kernels, and signed-W6 fixed-base selector

Status: implementation evidence complete; independent reviewer sign-off pending

## Generation and dependency isolation

- [ ] Confirm `asm/go.mod` pins `github.com/mmcloughlin/avo v0.6.0` and
  generator-only dependencies do not enter the root module.
- [ ] Re-run `make check-asm-module generate-check` and confirm the committed Go
  stubs and assembly are reproducible without a diff.
- [ ] Confirm Dependabot covers `/asm` separately and the root module still has
  exactly one module in `go list -m all`.

## Feature dispatch and fallback

- [ ] Confirm CPUID first checks maximum basic leaf, then reads leaf 7 subleaf 0
  EBX bits 8 (BMI2) and 19 (ADX), and caches the combined result once.
- [ ] Confirm `GOAMD64` is not used as an ADX proxy and both features are
  required before any ADX/BMI2 symbol is called.
- [ ] Confirm the feature branch depends only on public hardware state and the
  no-feature path calls the existing fiat Mul and Square implementations.
- [ ] Confirm the `SECP256K1_AMD64_BENCH_KERNEL` override is unavailable unless
  the dedicated `secp256k1_amd64_bench` measurement tag is also supplied.

## Representation and arithmetic

- [ ] Confirm operands are canonical four-limb little-endian Montgomery values
  modulo `p = 2^256 - 2^32 - 977`, with `R = 2^256`.
- [ ] Confirm each general product uses four fixed CIOS rows, preserves the
  fifth carry limb, and accounts for all 16 operand cross-products.
- [ ] Confirm `fieldK0 = (2^32 + 977)^-1 mod 2^64`; each reduction row cancels
  its low limb and performs the corresponding Montgomery word shift.
- [ ] Confirm the result bound is below `2p` and the fixed subtraction plus
  `CMOVQCS` chain returns a canonical value in `[0,p)`.

## ABI, aliasing, and control flow

- [ ] Confirm all generated declarations are `//go:noescape`, use Go's expected
  AMD64 ABI wrapper, preserve required registers, and contain no calls.
- [ ] Confirm both inputs are fully staged before arithmetic: Mul keeps the
  first input in four general registers and the second in two SSE2 registers,
  with no stack frame. Confirm this preserves `out == x`, `out == y`, `x == y`,
  and repeated-square in-place operation.
- [ ] Confirm Mul and Square each contain 20 `MULXQ`, 20 `ADCXQ`, and 20
  `ADOXQ`, with no branches.
- [ ] Confirm Mul additionally contains exactly two `MOVDQU` input loads and
  two `PSRLDQ` extractions, and that SSE2 is available on every AMD64 target.
- [ ] Confirm operand limbs never influence an address, loop count, branch, or
  instruction count.
- [ ] Review both Go and native GNU disassembly artifacts for instruction
  decoding, symbols, ABI, branches, memory operands, and function sizes.

## Signed-W6 fixed-base path

- [ ] Confirm the generated 43 x 32 x 64-byte W6 table is shared by the ARM64
  and AMD64 tagged implementations, while default and AMD64 W5-benchmark
  builds retain the 52 x 16 x 64-byte W5 table.
- [ ] Confirm a production binary links exactly one fixed-base table. Check the
  W6-over-W5 data increase is 34,816 bytes, below the 40 KiB limit.
- [ ] Independently regenerate every W6 affine point and compare all 1,376
  packed entries with the committed table in Montgomery representation.
- [ ] Confirm the AMD64 selector uses only baseline AMD64/SSE2 instructions,
  reads all 32 entries in the selected public window, and executes the same
  instruction sequence for every secret magnitude.
- [ ] Confirm magnitude zero is handled only by the later constant-time point
  selection; the selector deliberately returns entry one for magnitudes zero
  and one and never indexes memory with the magnitude.
- [ ] Confirm native disassembly has 132 `MOVDQU`, 31 `CMP`, 31 `SETE`, no
  branch, and no secret-dependent address in `selectGeneratorW6`.
- [ ] Confirm signed recoding always executes 43 windows, handles the highest
  carry, applies sign with constant-time field selection, and erases the local
  scalar words on return.

## Correctness and integration

- [ ] Re-run fiat differential tests over zero, one, `p-1`, limb carry/borrow
  boundaries, all alias arrangements, and 100,000 deterministic random pairs.
- [ ] Re-run SquareN zero and generated-chain counts, base-field fuzzing, CPUID
  mask tests, forced-fallback tests, signature goldens, and public-key oracles.
- [ ] Re-run W5/W6 dynamic-table differential tests, every selector window and
  magnitude 0 through 32, highest-window carry, boundary scalars, and at least
  1,000 deterministic random scalar comparisons against W4/W5/W6 oracles.
- [ ] Confirm default builds still use fiat, ARM64 tagged routing is unchanged,
  and no public API, type, RFC6979 behavior, or signature encoding changed.
- [ ] Confirm Add/Sub, scalar assembly, fused point arithmetic, W7, and new
  variable-time secret paths are absent from the AMD64 change set.

## Performance and platform evidence

- [ ] Reproduce ten interleaved same-runner medians for `GOAMD64=v1` and `v3`.
  Check every field microbenchmark, each isolated kernel's end-to-end
  contribution, final signing/verification, allocations, and other workloads.
- [ ] Confirm the rejected stack-backed, SquareN, and multiply-by-21 candidates
  are absent and retained Mul/Square satisfy all recorded thresholds.
- [ ] Confirm W6 entered only after field-only signing missed 10%, improves
  fixed-base multiplication by at least 5% and signing by at least 3%, and
  does not regress verification by more than 1% at either GOAMD64 level.
- [ ] Re-run default/tagged race tests, lint/vet, fuzz, constant-time smoke,
  dependency/vartime audits, benchmark-module tests, native Linux/Windows AMD64,
  and Darwin AMD64 cross-builds.

Performance evidence is recorded in
[`../perf/20260714-amd64-avo.md`](../perf/20260714-amd64-avo.md).
Independent audit sign-off remains mandatory before considering default enablement.
