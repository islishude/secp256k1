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
	if got := median(results["BenchmarkVerifyHotPublicKey"].nsPerOp); got != 30 {
		t.Fatalf("verify median = %v, want 30", got)
	}
	if got := median(results["BenchmarkSignRecoverable"].nsPerOp); got != 24 {
		t.Fatalf("sign median = %v, want 24", got)
	}
	if got := median(results["BenchmarkVerifyHotPublicKey"].bytesPerOp); got != 0 {
		t.Fatalf("verify B/op median = %v, want 0", got)
	}
	if got := median(results["BenchmarkVerifyHotPublicKey"].allocsPerOp); got != 0 {
		t.Fatalf("verify allocs/op median = %v, want 0", got)
	}
}

func TestFinalGate(t *testing.T) {
	baseline := map[string]*samples{}
	candidate := map[string]*samples{}
	all := []string{
		"BenchmarkSignRecoverable",
		"BenchmarkVerifyHotPublicKey",
		"BenchmarkSignDigest",
		"BenchmarkSignRecoverableDigest",
		"BenchmarkVerifyDigest",
		"BenchmarkVerifyParseCompressedCold",
		"BenchmarkVerifyParseUncompressedCold",
		"BenchmarkRecoverDigest",
		"BenchmarkSignCompact",
		"BenchmarkPublicKeyDerive",
	}
	for _, name := range all {
		baseline[name] = &samples{nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}}
		candidate[name] = &samples{nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}}
	}
	candidate["BenchmarkSignRecoverable"].nsPerOp[0] = 90
	candidate["BenchmarkVerifyHotPublicKey"].nsPerOp[0] = 89
	if err := checkGate("final", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkPublicKeyDerive"].nsPerOp[0] = 102
	if err := checkGate("final", baseline, candidate); err == nil {
		t.Fatal("expected regression failure")
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
