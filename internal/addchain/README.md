The code generated from this directory uses github.com/mmcloughlin/addchain to
produce fixed-exponent routines for:

    field inverse: p - 2
    field square root: (p + 1) / 4
    scalar inverse: n - 2

The checked-in files are generated artifacts, so normal builds and tests don't
require addchain. To regenerate them, run Docker and then:

    go generate

The generator runs addchain from:

    ghcr.io/islishude/fiat-crypto-go-tool@sha256:fa7d326be7c8ce419573e4f3165a88a41dfb0d29276887dd3c9de643f2723424

The shared pin is maintained in `../gentool/tool.go`. It selects a manifest
containing Linux ARM64 and AMD64 images. Regeneration with this digest reproduced
the checked-in arithmetic unchanged during the 2026-09-08 review fixes.

Set FIAT_CRYPTO_GO_TOOL_IMAGE only to deliberately override the audited pin,
for example with an immutable mirror digest. Set ADDCHAIN_BIN
only when intentionally bypassing Docker with a local addchain binary.
