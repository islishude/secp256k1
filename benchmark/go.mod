module github.com/islishude/secp256k1/benchmark

go 1.26.3

replace github.com/islishude/secp256k1 => ../

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1
	github.com/ethereum/go-ethereum v1.17.4
	github.com/islishude/secp256k1 v0.0.0-00010101000000-000000000000
)

require (
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
