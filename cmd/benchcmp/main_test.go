package main

import (
	"slices"
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

func TestPairedFinalGateHandlesBimodalRunnerSamples(t *testing.T) {
	baseline := map[string]*samples{}
	candidate := map[string]*samples{}
	all := append([]string{"BenchmarkSignRecoverable", "BenchmarkVerifyHotPublicKey"}, finalNoRegression...)
	for _, name := range all {
		baseline[name] = benchmarkSeries(100, 100, 100, 100, 100, 100, 100, 100, 100, 100)
		candidate[name] = benchmarkSeries(100, 100, 100, 100, 100, 100, 100, 100, 100, 100)
	}
	candidate["BenchmarkSignRecoverable"] = benchmarkSeries(80, 80, 80, 80, 80, 80, 80, 80, 80, 80)
	baseline["BenchmarkVerifyHotPublicKey"] = benchmarkSeries(27739, 23760, 23749, 27720, 23815, 23747, 23738, 27740, 23744, 27301)
	candidate["BenchmarkVerifyHotPublicKey"] = benchmarkSeries(21541, 21967, 24646, 24827, 24832, 21264, 24800, 21254, 21255, 21258)

	if err := checkGate("final", baseline, candidate); err == nil {
		t.Fatal("independent-median gate unexpectedly passed bimodal samples")
	}
	if err := checkGate("final-paired", baseline, candidate); err != nil {
		t.Fatal(err)
	}
	candidate["BenchmarkVerifyHotPublicKey"].nsPerOp = candidate["BenchmarkVerifyHotPublicKey"].nsPerOp[:9]
	if err := checkGate("final-paired", baseline, candidate); err == nil {
		t.Fatal("expected mismatched paired sample count failure")
	}
}

func benchmarkSeries(values ...float64) *samples {
	zeroes := make([]float64, len(values))
	return &samples{
		nsPerOp:     values,
		bytesPerOp:  slices.Clone(zeroes),
		allocsPerOp: zeroes,
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

func TestCanonicalBenchmarkName(t *testing.T) {
	if got := canonicalBenchmarkName("BenchmarkFoo/bar-16"); got != "BenchmarkFoo/bar" {
		t.Fatalf("canonical name = %q", got)
	}
	if got := canonicalBenchmarkName("BenchmarkFoo-arm64"); got != "BenchmarkFoo-arm64" {
		t.Fatalf("non-CPU suffix changed to %q", got)
	}
}
