//go:build amd64 && secp256k1_asm && secp256k1_amd64_bench

// Package amd64bench owns the environment-controlled selector used only by
// the dedicated AMD64 performance workflow. Production builds do not import
// this package and cannot observe the selector.
package amd64bench

import (
	"fmt"
	"os"
	"strings"
)

const Environment = "SECP256K1_AMD64_V2_BENCH_KERNEL"

var enabled = parse(os.Getenv(Environment))

// Enabled reports whether a v2 kernel is selected. An empty selection and
// "all" model the production backend; "accepted" disables every v2 kernel.
func Enabled(kernel string) bool {
	if enabled["all"] {
		return true
	}
	if enabled["scalar"] && strings.HasPrefix(kernel, "scalar-") {
		return true
	}
	return enabled[kernel]
}

// Selection returns the raw benchmark selection for diagnostics.
func Selection() string {
	selection := os.Getenv(Environment)
	if selection == "" {
		return "all"
	}
	return selection
}

func parse(selection string) map[string]bool {
	if selection == "" || selection == "all" {
		return map[string]bool{"all": true}
	}
	if selection == "accepted" {
		return map[string]bool{}
	}

	allowed := map[string]bool{
		"scalar":         true,
		"scalar-mul":     true,
		"scalar-square":  true,
		"scalar-squaren": true,
		"invvartime":     true,
		"field-add":      true,
		"field-sub":      true,
	}
	result := make(map[string]bool)
	for _, kernel := range strings.Split(selection, ",") {
		kernel = strings.TrimSpace(kernel)
		if !allowed[kernel] {
			panic(fmt.Sprintf("unknown %s value %q", Environment, selection))
		}
		result[kernel] = true
	}
	return result
}
