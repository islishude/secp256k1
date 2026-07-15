# Assembly backend external audit checklist

Last consolidated: 2026-07-15

Status: implementation evidence is complete; independent reviewer sign-off is
pending. The `secp256k1_asm` backends must remain opt-in until this checklist is
completed for the reviewed commit.

## Review scope

| Area | Current files |
| --- | --- |
| Shared fixed-base logic and data | `scalar_mult.go`, `zz_precompute_base_w5.go`, `zz_precompute_base_w6_asm.go` |
| ARM64 arithmetic and selector | `internal/field/montgomery_arm64.{go,s}`, `internal/scalar/montgomery_arm64.{go,s}`, `scalar_select_arm64.{go,s}` |
| AMD64 dispatch, arithmetic, and selector | `internal/cpufeat/`, `internal/field/montgomery_amd64*`, `scalar_select_amd64*` |
| Generators and automated audit | `cmd/genprecomp/`, `asm/`, `scripts/check-amd64-asm.sh`, `scripts/check-vartime.sh` |

Performance evidence and rejected-candidate rationale are summarized in
[`performance.md`](performance.md). Rejected implementations are not part of
the review except where this checklist explicitly requires confirming their
absence.

## Build and dependency isolation

- [ ] Confirm the default build is pure Go, uses fiat field/scalar arithmetic
  and the signed-W5 fixed-base table, and does not link tagged assembly symbols.
- [ ] Confirm `secp256k1_asm` selects assembly only on ARM64 or AMD64 and that
  unsupported architectures still compile through the default implementation.
- [ ] Run `make check-main-deps`; confirm the root module has no third-party
  module dependencies.
- [ ] Run `make check-asm-module`; confirm avo v0.6.0 and its dependencies stay
  isolated in `asm/go.mod`.
- [ ] Run `make generate-check` from a clean tree; confirm fiat, addition-chain,
  precomputation, CPUID, field-kernel, and selector generation is reproducible.
- [ ] Confirm each production build links exactly one fixed-base table: W5 by
  default and W6 only for tagged ARM64/AMD64 builds.
- [ ] Confirm no public API, type layout, signature encoding, RFC6979 behavior,
  recovery semantics, or verification-comb width changes under the build tag.

## Shared constant-time fixed-base path

- [ ] Confirm signing scalars are split into exactly 43 public-position,
  six-bit windows; signed digits stay in `[-31, 32]`, magnitudes in `[0, 32]`,
  and carry in `{0, 1}`.
- [ ] Confirm the highest window consumes only bits 252 through 255 plus carry,
  produces no terminal carry, and reconstructs boundary scalars exactly.
- [ ] Confirm every selector invocation reads all 32 packed affine points in
  the selected public window, including magnitude zero; the secret magnitude
  never controls an address, branch, or loop count.
- [ ] Confirm sign application, zero-digit handling, first-window infinity, and
  subsequent complete mixed additions use masks and remain correct for equal,
  opposite, and infinity cases.
- [ ] Confirm local scalar words are cleared before return and the W6 table is
  exactly 43 x 32 x 64 = 88,064 bytes.
- [ ] Independently regenerate all 1,376 W6 affine points and compare their
  canonical Montgomery limbs with `zz_precompute_base_w6_asm.go`.

## ARM64 backend

### Field and scalar arithmetic

- [ ] Confirm all operands are canonical four-limb little-endian Montgomery
  values modulo `p = 2^256 - 2^32 - 977`, with `R = 2^256`.
- [ ] Confirm Mul loads both operands before writing output, covers all 16
  cross-products, performs four valid Montgomery reductions, produces a result
  below `2p`, and finishes with one fixed conditional subtraction. Verify all
  alias arrangements.
- [ ] Confirm Mul contains 20 `MUL`, 20 `UMULH`, no call, no branch, no R18 use,
  and no operand-dependent address or instruction count.
- [ ] Review Add, MulByB3, Square, and field/scalar SquareN for canonical output,
  alias safety, fixed memory access, and correct reduction bounds. SquareN may
  branch only on its public static addition-chain count, including `n == 0`.
- [ ] Confirm field Sub and scalar Mul/Square remain fiat calls and no unreviewed
  ARM64 symbol is routed into production.

### NEON W6 selector

- [ ] Confirm `selectGeneratorW6` uses the expected Go ABI, `NOSPLIT`, no calls,
  no R18, and stores exactly one 64-byte result.
- [ ] Confirm its 1,024-byte body contains 32 consecutive `VLD1` table reads,
  31 `CMP`/`CSETM` mask constructions, and no branch.
- [ ] Confirm the secret magnitude is used only in compare/mask operations; all
  pointers advance monotonically through public memory.

## AMD64 backend

### Feature dispatch and field kernels

- [ ] Confirm CPUID checks the maximum basic leaf before leaf 7 subleaf 0, then
  requires both EBX bit 8 (BMI2) and bit 19 (ADX). `GOAMD64` must not be used as
  an ADX proxy.
- [ ] Confirm feature detection is cached from public CPU state and CPUs missing
  either feature call fiat Mul and Square without reaching ADX/BMI2 symbols.
- [ ] Confirm Mul stages all four limbs of both operands before output: one
  operand in general registers and one in two SSE2 registers, with no stack
  frame and complete alias safety.
- [ ] Confirm Mul and Square each use the fixed four-row CIOS schedule and
  contain 20 `MULXQ`, 20 `ADCXQ`, 20 `ADOXQ`, four conditional moves, no call,
  and no branch. Mul must also contain exactly two unaligned SSE2 loads and two
  64-bit lane extractions.
- [ ] Run `scripts/check-amd64-asm.sh` at `GOAMD64=v1` and `v3`; inspect both Go
  and GNU disassembly, symbol sizes, ABI wrappers, memory operands, and branch
  absence.

### SSE2 W6 selector and rejected-symbol absence

- [ ] Confirm the selector uses baseline AMD64/SSE2 only, reads all 32 entries,
  and handles magnitude zero without secret-indexed memory.
- [ ] Confirm native source/disassembly contains 132 unaligned 128-bit moves,
  31 compares, 31 equality masks, and no branch; the Go recoding loop always
  executes 43 windows.
- [ ] Confirm linked tagged binaries contain retained field Mul/Square and W6
  symbols but no AMD64 field Add/Sub/SquareN/MulByB3, scalar
  Mul/Square/SquareN, or assembly `InvVartime` candidate symbol.
- [ ] Confirm `SECP256K1_AMD64_BENCH_KERNEL` can affect only builds carrying the
  dedicated `secp256k1_amd64_bench` measurement tag.

## Correctness, side-channel, and platform evidence

- [ ] Differential-test every retained arithmetic operation against fiat over
  zero, one, modulus boundaries, carry/borrow edges, canonical maximums, every
  alias arrangement, and at least 100,000 deterministic random inputs.
- [ ] Test W5/W6 generated tables, every W6 window and magnitude, highest-window
  carry, boundary scalars, and at least 1,000 deterministic random scalar
  comparisons against independent W4/W5/dynamic-W6 oracles.
- [ ] Run default and tagged signature goldens, full tests, non-cached race
  tests, field fuzzing, and the opt-in constant-time smoke test.
- [ ] Run `make vartime-audit`; confirm production `InvVartime` reaches only
  verification/recovery public inputs and no variable-time path reaches
  signing, private-key derivation, or RFC6979.
- [ ] Confirm tagged ARM64 passes native Linux and macOS, tagged AMD64 passes
  native Linux and Windows at v1/v3, and Darwin AMD64 cross-builds at v1/v3.
- [ ] Review CI disassembly and benchmark artifacts for the exact commit under
  review; do not substitute absolute measurements from a different runner.

## Performance acceptance

- [ ] Reproduce the ARM64 retained stages on controlled hardware and confirm no
  stable default-path regression and zero allocations in signing/hot verify.
- [ ] Reproduce ten alternating AMD64 default/tagged pairs at v1 and v3 on one
  pinned CPU and one Go P. Require at least 10% paired-median improvement in
  SignRecoverable and VerifyHotPublicKey, zero allocations in every sample, and
  no more than 1% paired-median regression in the other gated workloads.
- [ ] Treat isolated kernel and W5/W6 results as diagnostics, not substitutes
  for the final end-to-end gate.

## Sign-off

- Reviewed commit: `________________`
- Reviewer: `________________`
- Review date: `________________`
- [ ] All findings are resolved or explicitly accepted with rationale.
- [ ] The reviewer approves the audited backend for the proposed deployment
  status. Default enablement requires a separate, explicit change after sign-off.
