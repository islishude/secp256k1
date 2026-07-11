# ARM64 W6 fixed-base external audit checklist

Date: 2026-07-11  
Scope: `arm64 && secp256k1_asm` signed-W6 fixed-base multiplication  
Status: implementation evidence complete; independent reviewer sign-off pending

## Formula and recoding boundaries

- [ ] Confirm the scalar is represented as four little-endian 64-bit words and
  split into 43 public-position, 6-bit windows.
- [ ] Confirm signed recoding yields digits in `[-31, 32]`, magnitude in
  `[0, 32]`, and carry in `{0, 1}`.
- [ ] Confirm window 42 reads only scalar bits 252 through 255 plus incoming
  carry, so its value is at most 16 and terminal carry is zero.
- [ ] Confirm zero digits do not alter the accumulator and the first window can
  initialize infinity without an incomplete-addition exception.
- [ ] Confirm all later additions use the existing complete mixed-add formula,
  including infinity and equal/opposite-point cases, and retain the specialized
  Montgomery multiplication by `3*b = 21`.
- [ ] Confirm no RFC6979, scalar inversion, verification-comb, public API, or
  signature-encoding behavior is in scope.

## ABI and register use

- [ ] Confirm `selectGeneratorW6` has Go ABI declaration
  `func(out *[8]uint64, table *[32][8]uint64, magnitude uint64)` with
  `//go:noescape` and an assembly frame of `$0-24` marked `NOSPLIT`.
- [ ] Confirm R0 is the output pointer, R1 is the monotonically advanced table
  pointer, R2 is the secret magnitude, and R3 is only the comparison mask.
- [ ] Confirm V0-V3 hold the selected 64-byte record, V4 holds the broadcast
  mask, and V5-V8 hold each candidate record.
- [ ] Confirm the selector does not use platform-reserved R18, does not call,
  and stores exactly one 64-byte result.

## Table access and control flow

- [ ] Confirm the table shape is 43 x 32 x 8 `uint64`, each affine point is a
  64-byte Montgomery `(x,y)` record, and total data is 88,064 bytes.
- [ ] Confirm every invocation performs exactly 32 consecutive 64-byte `VLD1`
  reads regardless of magnitude, including magnitude zero.
- [ ] Confirm magnitude enters only 31 `CMP`/`CSETM` mask constructions and
  never address calculation, loop count, or branch condition.
- [ ] Confirm selector disassembly has 32 `VLD1`, 31 `CMP`, 31 `CSETM`, zero
  branches, and a 1,024-byte text symbol.
- [ ] Confirm the Go recoding loop has a public 43-iteration bound; its only
  special-case branch is the public window index `i == 0`.

## Generated data and build isolation

- [ ] Re-run `go generate ./...` twice and compare all three generated SHA-256
  values.
- [ ] Confirm shared generated data contains only wNAF, endomorphism, and
  width-7 verification comb tables.
- [ ] Confirm default/non-ARM64 builds contain only W5 fixed-base data and the
  tagged ARM64 build contains only W6 fixed-base data.
- [ ] Independently regenerate all 1,376 W6 affine points and compare their
  Montgomery limbs with `zz_precompute_base_w6_arm64.go`.
- [ ] Check the generated header parameter and data hashes before review
  sign-off.

## Test and performance evidence

- [ ] Run default and tagged full tests and non-cached race tests.
- [ ] Run W5/W6 dynamic-table differential tests, all selector windows and
  magnitudes, highest-window/carry recoding, scalar boundaries, and at least
  1,000 deterministic random scalar oracle comparisons.
- [ ] Run compact/recoverable signature golden tests in both builds.
- [ ] Run vet, golangci-lint, fuzz smoke, constant-time smoke, dependency and
  variable-time audits, generation consistency, and Linux ARM64/AMD64 builds.
- [ ] Reproduce the ten-run medians and confirm 0 B/op, 0 allocs/op, <=18,000
  ns/op signing, >=3% signing improvement, and >=5% fixed-base improvement.
- [ ] Confirm the W6-over-W5 data increase is 34,816 bytes and tagged binary
  increase is 32,896 bytes.

Independent audit sign-off is mandatory before proposing default enablement.

The subsequent general field multiplication kernel has its own review scope in
[`20260711-arm64-field-mul-checklist.md`](20260711-arm64-field-mul-checklist.md).
