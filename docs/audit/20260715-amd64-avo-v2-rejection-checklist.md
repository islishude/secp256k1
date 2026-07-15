# AMD64 avo v2 rejection and cleanup audit checklist

Date: 2026-07-15

Scope: v2 scalar Montgomery, public `InvVartime`, and base-field Add/Sub experiments; final cleanup back to accepted `4ac5bc4` production behavior

Status: candidate rejection and final cleanup evidence complete; independent reviewer sign-off pending

## Accepted baseline isolation

- [ ] Confirm branch `codex/amd64-avo-asm-v2` was forked from accepted commit
  `4ac5bc4` and did not alter `codex/amd64-avo-asm` or acceptance run
  [29345965487](https://github.com/islishude/secp256k1/actions/runs/29345965487).
- [ ] Compare final `asm/`, `internal/field`, `internal/scalar`, W6 table, W6
  selector, and generated stubs against `4ac5bc4`; require no diff.
- [ ] Confirm the retained backend is still only base-field Mul/Square plus the
  signed-W6 SSE2 selector behind `secp256k1_asm` and ADX+BMI2 CPUID dispatch.
- [ ] Confirm default builds remain pure Go and root `go.mod` has no third-party
  dependencies.

## Rejected scalar Montgomery group

- [ ] Review run
  [29375703485](https://github.com/islishude/secp256k1/actions/runs/29375703485)
  and confirm initial CIOS Square/SquareN and v3 signing missed their gates.
- [ ] Review runs
  [29377187861](https://github.com/islishude/secp256k1/actions/runs/29377187861),
  [29378465358](https://github.com/islishude/secp256k1/actions/runs/29378465358),
  and [29379875980](https://github.com/islishude/secp256k1/actions/runs/29379875980);
  confirm XMM, GPR, and dedicated-Comba variants did not pass the full scalar
  group and signing requirements at both GOAMD64 levels.
- [ ] Confirm `asm/cmd/scalar`, AMD64 scalar assembly/stubs, backend routing,
  selector code, and candidate-specific tests are absent from the final tree.
- [ ] Confirm final linked binaries have no scalar
  `mulMontgomeryADXAsm`, `squareMontgomeryADXAsm`, or
  `squareMontgomeryNADXAsm` symbols.

## Rejected public-input inversion

- [ ] Confirm the generated Stein loop locally improved ScalarInvVartime by
  52.34%/46.32% and hot verification by 8.59%/6.72% at v1/v3 without a stable
  signing gain.
- [ ] Confirm it was nevertheless deleted because the terminal candidate did
  not meet the overall accepted-to-v2 double-3% gate.
- [ ] Confirm `InvVartime` is again the accepted Go implementation on all
  architectures and no `invVartimeWordsADXAsm` declaration, caller, generator,
  source, stub, or linked symbol remains.
- [ ] Run `make vartime-audit`; confirm the only production `InvVartime` calls
  are in verification and recovery and no variable-time function reaches
  signing, private-key, or RFC6979 paths.

## Rejected base-field Add/Sub

- [ ] Review run
  [29381506099](https://github.com/islishude/secp256k1/actions/runs/29381506099)
  and confirm Add improved 19.99%/21.18%, while Sub regressed 14.57%/15.00%
  at v1/v3.
- [ ] Confirm the terminal accepted-to-v2 signing gains were only 1.88% and
  1.40%, below the 3% requirement, even though verification passed.
- [ ] Confirm Add/Sub generator functions, assembly bodies, stubs, routing,
  v2 environment selector, and candidate tests are absent.
- [ ] Confirm final linked binaries have no field `addMontgomeryADXAsm` or
  `subMontgomeryADXAsm` symbol.

## Generation, CI, and disassembly

- [ ] Re-run `make check-asm-module generate-check`; require no generated diff
  and confirm avo remains pinned to v0.6.0 only in the independent `asm` module.
- [ ] Run the AMD64 workflow at GOAMD64 v1 and v3; require default/tagged full
  and race tests, field fuzz, constant-time smoke, native Windows, Linux
  disassembly, and Darwin cross-builds to pass.
- [ ] Confirm ten-run artifacts retain field/scalar/W5/W6/Add/Sub and root
  workload medians, CPU information and affinity, Go environment, symbols,
  disassembly, and tagged SignRecoverable/VerifyHotPublicKey CPU profiles.
- [ ] Confirm benchmark processes use one pinned CPU and one Go P, and the hard
  gate takes the median of ten same-iteration default/tagged percentage deltas
  rather than dividing two independently bimodal medians.
- [ ] Confirm the final default-to-tagged gate still requires at least 10% for
  SignRecoverable and VerifyHotPublicKey with 0 B/op and 0 allocs/op and no
  tracked end-to-end regression above 1%.
- [ ] Confirm isolated accepted-kernel and W6 data remain report-only: accepted
  evidence is immutable run 29345965487 and absolute results are never compared
  across different runners.
- [ ] Inspect `check-amd64-asm.sh` and its artifacts; require retained Mul,
  Square, and W6 branch/instruction checks plus explicit absence checks for all
  six rejected v2 symbols.

## Interface and scope

- [ ] Confirm public Go APIs and types, deterministic RFC6979 behavior,
  signature encoding, signature goldens, recovery semantics, W6 table, and
  verification comb width are unchanged.
- [ ] Confirm width 8, W7, fused point arithmetic, AVX2 selection, and any new
  variable-time secret path were not introduced.
- [ ] Confirm the report does not treat terminal default-to-candidate gains as
  acceptance: the controlling gate is accepted-to-v2 double 3%.

Evidence and rationale are recorded in
[`../perf/20260715-amd64-avo-v2-rejection.md`](../perf/20260715-amd64-avo-v2-rejection.md).
Final cleanup validation is recorded in
[GitHub Actions run 29388877833](https://github.com/islishude/secp256k1/actions/runs/29388877833).
Independent review remains required before changing the accepted backend.
