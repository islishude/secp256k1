# P8 arithmetic experiments

Date: 2026-07-10  
Go: go1.26.4  
GOOS/GOARCH: darwin/arm64  
CPU: Apple M3 Max

P8 was treated as an experiment gate, not as authorization to replace the
audited fiat backend without a measured whole-workload benefit.

## Pure-Go pseudo-Mersenne field candidate

A build-tagged canonical 4x64 implementation used `math/bits` multiplication and
fixed-count reductions based on `2^256 = 2^32 + 977 (mod p)`. It passed the field
tests and the full repository test suite under the experiment tag, but regressed
every relevant benchmark:

| Operation        |          fiat |     candidate |
| ---------------- | ------------: | ------------: |
| Field multiply   | about 12.0 ns | about 34.4 ns |
| Field square     | about 10.9 ns | about 34.9 ns |
| Field add        |  about 4.7 ns |  about 9.9 ns |
| Hot verify       |   about 31 us |   about 90 us |
| Recoverable sign |   about 25 us |   about 62 us |

The candidate and its backend abstraction were deleted. Keeping the abstraction
alone also enlarged Go's inlining cost and regressed whole point operations even
when the fiat implementation remained selected.

## Constant-time safegcd scalar inverse candidate

The repository's fiat output includes `Divstep`, `DivstepPrecomp`, and `Msat`.
A fixed 741-step Bernstein-Yang wrapper was implemented from the standard fiat
wrapper construction, including sign normalization, precomp multiplication, zero
handling, and aliasing tests. It matched the addition-chain inverse over 5,000
deterministic random values but measured about 13.4 us versus about 5.6 us for
the existing addition-chain inverse. It was deleted and never wired into signing.

Reference wrapper construction: [primefield fiat wrapper source](https://doc.servo.org/src/primefield/macros/fiat.rs.html#199-280).

## Decision

The default field/scalar arithmetic remains fiat-crypto. No experimental build
tag or alternate production backend remains. Architecture assembly was not
started because the pure-Go candidates failed the required whole-workload gain
and an assembly backend would require a separate architecture matrix and audit.
