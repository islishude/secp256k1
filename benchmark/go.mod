module github.com/islishude/secp256k1/benchmark

go 1.26.3

replace github.com/islishude/secp256k1 => ../

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1
	github.com/ethereum/go-ethereum v1.17.3
	github.com/islishude/secp256k1 v0.0.0-00010101000000-000000000000
)
