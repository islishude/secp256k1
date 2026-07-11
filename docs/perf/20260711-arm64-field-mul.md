# ARM64 field multiplication result

Date: 2026-07-11  
Go: go1.26.4  
GOOS/GOARCH: darwin/arm64  
CPU: Apple M3 Max  
Baseline: retained signed-W6 working tree after `4aa13ce94af3`

## Result

The opt-in `secp256k1_asm` backend now uses a fixed-instruction ARM64
Montgomery multiplication kernel for the secp256k1 base field. It reached the
16,000 ns recoverable-signing target without the planned fused mixed-add
fallback. The retained ten-run median is 15,367.5 ns/op with 0 B/op and
0 allocs/op.

| Tagged benchmark | W6 baseline | Field-Mul result | Delta | Gate |
| --- | ---: | ---: | ---: | --- |
| FieldMul | 10.48 ns | 8.7315 ns | -16.68% | pass: >=12% |
| FieldInv | 2,976.0 ns | 2,929.5 ns | -1.56% | pass: no regression |
| AddCompleteMixedGo | 158.6 ns | 145.1 ns | -8.51% | informational |
| ScalarBaseMultProjective | 8,270.5 ns | 7,607.0 ns | -8.02% | pass: >=4% |
| SignRecoverable | 16,774.5 ns | 15,367.5 ns | -8.39% | pass: <=16,000 ns and >=2.5% |
| VerifyHotPublicKey | 19,672.0 ns | 17,464.5 ns | -11.22% | pass |
| VerifyParseCompressedCold | 56,510.0 ns | 49,735.5 ns | -11.99% | pass |
| VerifyParseUncompressedCold | 53,427.0 ns | 47,148.0 ns | -11.75% | pass |

The default build remains on its existing fiat/W5 production path: the new
declaration, implementation, and build selection are confined to
`arm64 && secp256k1_asm`. An order-balanced adjacent run of the W6 baseline and
final default binaries measured -0.03% signing, -0.31% fixed-base, -0.32% hot
verification, +0.20% compressed cold verification, and -0.29% uncompressed
cold verification. The benchmark submodule test suite passed; its default
backend documentation therefore remains unchanged.

## Implementation and bounds

The kernel first loads both four-limb operands, so `out == x`, `out == y`, and
`x == y` are safe. Four fixed Comba rows produce the full eight-limb product.
For `R = 2^256`, the modulus is `p = R - c`, where `c = 2^32 + 977`, and the
Montgomery factor is `c^-1 mod 2^64`. Each of four reduction steps chooses
`q = t0*c^-1 mod 2^64`; adding `q*R` and subtracting `q*c` cancels the current
low limb exactly. Since the input product is below `p^2`, the four-step result
is below `2p`, so one fixed conditional subtraction produces a canonical field
element.

The 560-byte function contains 20 `MUL`, 20 `UMULH`, four input `LDP`, two
output `STP`, no calls, and no branches. Its only addresses are the public
argument pointers and fixed constant symbols. Final normalization uses a fixed
subtraction chain and four `CSEL` instructions.

The tagged test binary grew from 6,357,378 to 6,357,586 bytes, a 208-byte net
increase. W6 table and selector sizes remain 88,064 and 1,024 bytes.

## Validation

- The kernel was compared with fiat over zero, one, two, `p-1`, the maximum
  canonical raw Montgomery limbs, every aliasing arrangement, and 100,000
  deterministic random input pairs.
- Existing Square/SquareN, inversion, fixed-base W4/W5/W6 oracle, 1,000-random
  scalar, compact signature, and recoverable signature golden tests pass.
- Default/tagged non-cached race tests, vet, golangci-lint, generation
  reproducibility, two 10-second fuzz smokes, constant-time smokes, dependency
  and variable-time audits, benchmark-module tests, and Linux ARM64/AMD64
  cross-builds pass.
- A five-second profile measured 16,065 ns/op under sustained load. Field
  multiplication was 33.84% flat, fixed-base multiplication 47.91% cumulative,
  complete mixed addition 38.59%, affine conversion 19.20%, and the selector
  6.46%. Final acceptance uses the specified ten-run median above.

Raw benchmark and audit evidence is in
[`20260711-arm64-field-mul.txt`](20260711-arm64-field-mul.txt). The external
review checklist is in
[`../audit/20260711-arm64-field-mul-checklist.md`](../audit/20260711-arm64-field-mul-checklist.md).

The backend remains opt-in. Independent external review is still required
before considering default enablement.
