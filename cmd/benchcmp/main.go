// Command benchcmp compares median ns/op values from two Go benchmark outputs.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type samples struct {
	nsPerOp     []float64
	bytesPerOp  []float64
	allocsPerOp []float64
}

func main() {
	gate := flag.String("gate", "", "optional acceptance gate: field, kernel, w6, final, v2-scalar-micro, v2-scalar-e2e, v2-invvartime, or v2-final")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: benchcmp [-gate=field|kernel|w6|final|v2-scalar-micro|v2-scalar-e2e|v2-invvartime|v2-final] <baseline.txt> <candidate.txt>")
		os.Exit(2)
	}
	baseline, err := readBenchmarks(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	candidate, err := readBenchmarks(flag.Arg(1))
	if err != nil {
		log.Fatal(err)
	}

	names := make([]string, 0, len(baseline))
	for name := range baseline {
		if candidate[name] != nil && len(baseline[name].nsPerOp) != 0 && len(candidate[name].nsPerOp) != 0 {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		log.Fatal("no common benchmarks with ns/op results")
	}

	fmt.Printf("%-52s %12s %12s %9s %10s %11s\n", "Benchmark", "baseline", "candidate", "delta", "B/op", "allocs/op")
	for _, name := range names {
		oldMedian := median(baseline[name].nsPerOp)
		newMedian := median(candidate[name].nsPerOp)
		delta := (newMedian/oldMedian - 1) * 100
		fmt.Printf("%-52s %12.2f %12.2f %+8.2f%% %10.0f %11.0f\n",
			name, oldMedian, newMedian, delta,
			median(candidate[name].bytesPerOp), median(candidate[name].allocsPerOp),
		)
	}

	if err := checkGate(*gate, baseline, candidate); err != nil {
		log.Fatal(err)
	}
}

func readBenchmarks(path string) (map[string]*samples, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	results, err := parseBenchmarks(f)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return results, nil
}

func parseBenchmarks(r io.Reader) (map[string]*samples, error) {
	results := make(map[string]*samples)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		name := canonicalBenchmarkName(fields[0])
		result := results[name]
		if result == nil {
			result = new(samples)
			results[name] = result
		}
		for i := 2; i < len(fields); i++ {
			var dst *[]float64
			switch fields[i] {
			case "ns/op":
				dst = &result.nsPerOp
			case "B/op":
				dst = &result.bytesPerOp
			case "allocs/op":
				dst = &result.allocsPerOp
			default:
				continue
			}
			value, err := strconv.ParseFloat(fields[i-1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid %s value %q", fields[i], fields[i-1])
			}
			*dst = append(*dst, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func checkGate(gate string, baseline, candidate map[string]*samples) error {
	if gate == "" {
		return nil
	}
	var improvements map[string]float64
	var noRegression []string
	switch gate {
	case "field":
		improvements = map[string]float64{
			"BenchmarkFieldMul":    15,
			"BenchmarkFieldSquare": 15,
		}
	case "kernel":
		return checkKernelGate(baseline, candidate)
	case "w6":
		improvements = map[string]float64{
			"BenchmarkScalarBaseMultProjective": 5,
			"BenchmarkSignRecoverable":          3,
		}
		noRegression = []string{"BenchmarkVerifyHotPublicKey"}
	case "final":
		improvements = map[string]float64{
			"BenchmarkSignRecoverable":    10,
			"BenchmarkVerifyHotPublicKey": 10,
		}
		noRegression = []string{
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
	case "v2-scalar-micro":
		improvements = map[string]float64{
			"BenchmarkScalarMul":     15,
			"BenchmarkScalarSquare":  15,
			"BenchmarkScalarSquareN": 15,
			"BenchmarkScalarInv":     10,
		}
	case "v2-scalar-e2e":
		improvements = map[string]float64{
			"BenchmarkSignRecoverable": 3,
		}
		noRegression = []string{"BenchmarkVerifyHotPublicKey"}
	case "v2-invvartime":
		improvements = map[string]float64{
			"BenchmarkScalarInvVartime":   15,
			"BenchmarkVerifyHotPublicKey": 3,
		}
		noRegression = []string{
			"BenchmarkSignRecoverable",
			"BenchmarkRecoverDigest",
			"BenchmarkVerifyParseCompressedCold",
			"BenchmarkVerifyParseUncompressedCold",
		}
	case "v2-final":
		improvements = map[string]float64{
			"BenchmarkSignRecoverable":    3,
			"BenchmarkVerifyHotPublicKey": 3,
		}
		noRegression = []string{
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
	default:
		return fmt.Errorf("unknown gate %q", gate)
	}

	for name, minimum := range improvements {
		old, next, err := commonMedians(name, baseline, candidate)
		if err != nil {
			return err
		}
		improvement := (1 - next/old) * 100
		if improvement+1e-9 < minimum {
			return fmt.Errorf("%s improved %.2f%%, requires at least %.2f%%", name, improvement, minimum)
		}
		if median(baseline[name].bytesPerOp) != 0 || median(candidate[name].bytesPerOp) != 0 ||
			median(baseline[name].allocsPerOp) != 0 || median(candidate[name].allocsPerOp) != 0 {
			return fmt.Errorf("%s must remain 0 B/op and 0 allocs/op", name)
		}
	}
	for _, name := range noRegression {
		old, next, err := commonMedians(name, baseline, candidate)
		if err != nil {
			return err
		}
		regression := (next/old - 1) * 100
		if regression > 1+1e-9 {
			return fmt.Errorf("%s regressed %.2f%%, maximum is 1.00%%", name, regression)
		}
	}
	return nil
}

func checkKernelGate(baseline, candidate map[string]*samples) error {
	bestImprovement := -100.0
	for _, name := range []string{"BenchmarkSignRecoverable", "BenchmarkVerifyHotPublicKey"} {
		old, next, err := commonMedians(name, baseline, candidate)
		if err != nil {
			return err
		}
		improvement := (1 - next/old) * 100
		bestImprovement = max(bestImprovement, improvement)
		if improvement < -1-1e-9 {
			return fmt.Errorf("%s regressed %.2f%%, maximum is 1.00%%", name, -improvement)
		}
		if median(baseline[name].bytesPerOp) != 0 || median(candidate[name].bytesPerOp) != 0 ||
			median(baseline[name].allocsPerOp) != 0 || median(candidate[name].allocsPerOp) != 0 {
			return fmt.Errorf("%s must remain 0 B/op and 0 allocs/op", name)
		}
	}
	if bestImprovement+1e-9 < 1 {
		return fmt.Errorf("best signing or verification improvement is %.2f%%, requires at least 1.00%%", bestImprovement)
	}
	return nil
}

func commonMedians(name string, baseline, candidate map[string]*samples) (float64, float64, error) {
	old := baseline[name]
	next := candidate[name]
	if old == nil || next == nil || len(old.nsPerOp) == 0 || len(next.nsPerOp) == 0 {
		return 0, 0, fmt.Errorf("required benchmark %s is missing", name)
	}
	return median(old.nsPerOp), median(next.nsPerOp), nil
}

func canonicalBenchmarkName(name string) string {
	dash := strings.LastIndexByte(name, '-')
	if dash < 0 || dash == len(name)-1 {
		return name
	}
	for _, r := range name[dash+1:] {
		if !unicode.IsDigit(r) {
			return name
		}
	}
	return name[:dash]
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	values = slices.Clone(values)
	slices.Sort(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}
