# ARM64 base-field multiplication external audit checklist

Date: 2026-07-11  
Scope: `arm64 && secp256k1_asm` base-field Montgomery multiplication  
Status: implementation evidence complete; independent reviewer sign-off pending

## Representation and arithmetic

- [ ] Confirm operands are canonical four-limb little-endian Montgomery values
  modulo `p = 2^256 - 2^32 - 977` with `R = 2^256`.
- [ ] Confirm the four Comba rows account for all 16 cross-products and every
  carry into the eight-limb accumulator.
- [ ] Confirm `baseFieldK0 = (2^32 + 977)^-1 mod 2^64` and that each reduction
  step computes `q = t0*baseFieldK0 mod 2^64`.
- [ ] Confirm adding `q*R` and subtracting `q*(2^32+977)` is equivalent to
  adding `q*p`, cancels the low limb, and preserves the Montgomery residue.
- [ ] Confirm four reductions divide by `R`, the intermediate is below `2p`,
  and one conditional subtraction yields a value in `[0,p)`.

## ABI, aliasing, and control flow

- [ ] Confirm the Go ABI is `func(out, x, y *MontgomeryDomainFieldElement)` with
  `//go:noescape`, `$0-24`, and `NOSPLIT`.
- [ ] Confirm both inputs are fully loaded before the output pointer is read or
  written, preserving `out == x`, `out == y`, and `x == y` behavior.
- [ ] Confirm R0-R2 hold public pointers, R3-R16/R19-R26 contain only fixed
  arithmetic state, and reserved R18 is unused.
- [ ] Confirm disassembly has 20 `MUL`, 20 `UMULH`, four input `LDP`, two output
  `STP`, zero calls, and zero branches.
- [ ] Confirm no operand limb influences an address, loop bound, branch, or
  instruction count; final selection uses only carry flags and `CSEL`.

## Integration and isolation

- [ ] Confirm default/non-ARM64 builds still call fiat multiplication and only
  `arm64 && secp256k1_asm` resolves `mulMontgomery` to assembly.
- [ ] Confirm W6 recoding, selector, table data, complete mixed-add formula,
  verification comb, RFC6979, APIs, and signature encoding are unchanged.
- [ ] Confirm fused complete-mixed-add was not implemented because the general
  multiplication candidate reached the 16,000 ns signing target.
- [ ] Confirm tagged binary growth is 208 bytes and W6 data remains 88,064
  bytes.

## Test and performance evidence

- [ ] Re-run fiat differential tests for all edge and alias cases plus at least
  100,000 deterministic random pairs.
- [ ] Run default/tagged full and race tests, signature goldens, fixed-base
  oracles, vet, lint, fuzz, constant-time smoke, dependency/vartime audits,
  generation consistency, and Linux ARM64/AMD64 builds.
- [ ] Reproduce the ten-run medians: FieldMul <=8.7315 ns, fixed-base <=7,607
  ns, and recoverable signing <=15,367.5 ns with zero allocations.
- [ ] Confirm default production code is unchanged and tagged hot/cold
  verification has no stable regression.

Independent audit sign-off is mandatory before proposing default enablement.
