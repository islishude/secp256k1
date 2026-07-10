package main

import (
	"strings"
	"testing"
)

func TestParseBenchmarksAndMedian(t *testing.T) {
	input := strings.NewReader(`
goos: darwin
BenchmarkVerifyHotPublicKey-16  100  30.5 ns/op  0 B/op  0 allocs/op
BenchmarkVerifyHotPublicKey-16  100  29.5 ns/op  0 B/op  0 allocs/op
BenchmarkSignRecoverable-8      100  24 ns/op    0 B/op  0 allocs/op
`)
	results, err := parseBenchmarks(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := median(results["BenchmarkVerifyHotPublicKey"]); got != 30 {
		t.Fatalf("verify median = %v, want 30", got)
	}
	if got := median(results["BenchmarkSignRecoverable"]); got != 24 {
		t.Fatalf("sign median = %v, want 24", got)
	}
}

func TestCanonicalBenchmarkName(t *testing.T) {
	if got := canonicalBenchmarkName("BenchmarkFoo/bar-16"); got != "BenchmarkFoo/bar" {
		t.Fatalf("canonical name = %q", got)
	}
	if got := canonicalBenchmarkName("BenchmarkFoo-arm64"); got != "BenchmarkFoo-arm64" {
		t.Fatalf("non-CPU suffix changed to %q", got)
	}
}
