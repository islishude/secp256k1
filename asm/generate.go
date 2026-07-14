// Package asm owns the isolated avo generators for the opt-in AMD64 backend.
package asm

//go:generate go run ./cmd/cpuid -out ../internal/cpufeat/cpuid_amd64.s -stubs ../internal/cpufeat/cpuid_amd64.go -pkg cpufeat
//go:generate go run ./cmd/field -out ../internal/field/montgomery_amd64.s -stubs ../internal/field/montgomery_amd64_stub.go -pkg field
//go:generate go run ./cmd/scalar -out ../internal/scalar/montgomery_amd64.s -stubs ../internal/scalar/montgomery_amd64_stub.go -pkg scalar
//go:generate go run ./cmd/w6 -out ../scalar_select_amd64.s -stubs ../scalar_select_amd64_stub.go -pkg secp256k1
