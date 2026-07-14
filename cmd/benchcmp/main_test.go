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
		"BenchmarkScalarBaseMultProjective",
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

func TestKernelGate(t *testing.T) {
	baseline := map[string]*samples{
		"BenchmarkSignRecoverable":    {nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkVerifyHotPublicKey": {nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
	}
	candidate := map[string]*samples{
		"BenchmarkSignRecoverable":    {nsPerOp: []float64{98}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkVerifyHotPublicKey": {nsPerOp: []float64{100.5}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
	}
	if err := checkGate("kernel", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkSignRecoverable"].nsPerOp[0] = 99.5
	if err := checkGate("kernel", baseline, candidate); err == nil {
		t.Fatal("expected insufficient contribution failure")
	}
	candidate["BenchmarkVerifyHotPublicKey"].nsPerOp[0] = 101.5
	if err := checkGate("kernel", baseline, candidate); err == nil {
		t.Fatal("expected regression failure")
	}
}

func TestW6Gate(t *testing.T) {
	baseline := map[string]*samples{
		"BenchmarkScalarBaseMultProjective": {nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkSignRecoverable":          {nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkVerifyHotPublicKey":       {nsPerOp: []float64{100}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
	}
	candidate := map[string]*samples{
		"BenchmarkScalarBaseMultProjective": {nsPerOp: []float64{95}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkSignRecoverable":          {nsPerOp: []float64{97}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
		"BenchmarkVerifyHotPublicKey":       {nsPerOp: []float64{101}, bytesPerOp: []float64{0}, allocsPerOp: []float64{0}},
	}
	if err := checkGate("w6", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkSignRecoverable"].nsPerOp[0] = 98
	if err := checkGate("w6", baseline, candidate); err == nil {
		t.Fatal("expected insufficient W6 signing contribution failure")
	}
}

func TestV2ScalarGates(t *testing.T) {
	baseline := benchmarkSamples(map[string]float64{
		"BenchmarkScalarMul":          100,
		"BenchmarkScalarSquare":       100,
		"BenchmarkScalarSquareN":      100,
		"BenchmarkScalarInv":          100,
		"BenchmarkSignRecoverable":    100,
		"BenchmarkVerifyHotPublicKey": 100,
	})
	candidate := benchmarkSamples(map[string]float64{
		"BenchmarkScalarMul":          85,
		"BenchmarkScalarSquare":       84,
		"BenchmarkScalarSquareN":      80,
		"BenchmarkScalarInv":          89,
		"BenchmarkSignRecoverable":    97,
		"BenchmarkVerifyHotPublicKey": 101,
	})
	if err := checkGate("v2-scalar-micro", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	if err := checkGate("v2-scalar-e2e", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkScalarSquareN"].nsPerOp[0] = 86
	if err := checkGate("v2-scalar-micro", baseline, candidate); err == nil {
		t.Fatal("expected scalar microbenchmark failure")
	}
	candidate["BenchmarkScalarSquareN"].nsPerOp[0] = 80
	candidate["BenchmarkSignRecoverable"].nsPerOp[0] = 98
	if err := checkGate("v2-scalar-e2e", baseline, candidate); err == nil {
		t.Fatal("expected scalar end-to-end failure")
	}
}

func TestV2InvVartimeAndFinalGates(t *testing.T) {
	baseline := benchmarkSamples(map[string]float64{
		"BenchmarkScalarInvVartime":            100,
		"BenchmarkSignRecoverable":             100,
		"BenchmarkVerifyHotPublicKey":          100,
		"BenchmarkScalarBaseMultProjective":    100,
		"BenchmarkSignDigest":                  100,
		"BenchmarkSignRecoverableDigest":       100,
		"BenchmarkVerifyDigest":                100,
		"BenchmarkVerifyParseCompressedCold":   100,
		"BenchmarkVerifyParseUncompressedCold": 100,
		"BenchmarkRecoverDigest":               100,
		"BenchmarkSignCompact":                 100,
		"BenchmarkPublicKeyDerive":             100,
	})
	candidate := benchmarkSamples(map[string]float64{
		"BenchmarkScalarInvVartime":            84,
		"BenchmarkSignRecoverable":             97,
		"BenchmarkVerifyHotPublicKey":          96,
		"BenchmarkScalarBaseMultProjective":    100,
		"BenchmarkSignDigest":                  100,
		"BenchmarkSignRecoverableDigest":       100,
		"BenchmarkVerifyDigest":                100,
		"BenchmarkVerifyParseCompressedCold":   100,
		"BenchmarkVerifyParseUncompressedCold": 100,
		"BenchmarkRecoverDigest":               100,
		"BenchmarkSignCompact":                 100,
		"BenchmarkPublicKeyDerive":             100,
	})
	if err := checkGate("v2-invvartime", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	if err := checkGate("v2-final", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkVerifyHotPublicKey"].nsPerOp[0] = 98
	if err := checkGate("v2-invvartime", baseline, candidate); err == nil {
		t.Fatal("expected InvVartime end-to-end failure")
	}
	if err := checkGate("v2-final", baseline, candidate); err == nil {
		t.Fatal("expected v2 final failure")
	}
}

func benchmarkSamples(values map[string]float64) map[string]*samples {
	result := make(map[string]*samples, len(values))
	for name, value := range values {
		result[name] = &samples{
			nsPerOp:     []float64{value},
			bytesPerOp:  []float64{0},
			allocsPerOp: []float64{0},
		}
	}
	return result
}

func TestCanonicalBenchmarkName(t *testing.T) {
	if got := canonicalBenchmarkName("BenchmarkFoo/bar-16"); got != "BenchmarkFoo/bar" {
		t.Fatalf("canonical name = %q", got)
	}
	if got := canonicalBenchmarkName("BenchmarkFoo-arm64"); got != "BenchmarkFoo-arm64" {
		t.Fatalf("non-CPU suffix changed to %q", got)
	}
}
