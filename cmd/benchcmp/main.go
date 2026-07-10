// Command benchcmp compares median ns/op values from two Go benchmark outputs.
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: benchcmp <baseline.txt> <candidate.txt>")
		os.Exit(2)
	}
	baseline, err := readBenchmarks(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	candidate, err := readBenchmarks(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	names := make([]string, 0, len(baseline))
	for name := range baseline {
		if len(candidate[name]) != 0 {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		log.Fatal("no common benchmarks with ns/op results")
	}

	fmt.Printf("%-52s %12s %12s %9s\n", "Benchmark", "baseline", "candidate", "delta")
	for _, name := range names {
		oldMedian := median(baseline[name])
		newMedian := median(candidate[name])
		delta := (newMedian/oldMedian - 1) * 100
		fmt.Printf("%-52s %12.2f %12.2f %+8.2f%%\n", name, oldMedian, newMedian, delta)
	}
}

func readBenchmarks(path string) (map[string][]float64, error) {
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

func parseBenchmarks(r io.Reader) (map[string][]float64, error) {
	results := make(map[string][]float64)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		for i := 1; i < len(fields); i++ {
			if fields[i] != "ns/op" {
				continue
			}
			value, err := strconv.ParseFloat(fields[i-1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid ns/op value %q", fields[i-1])
			}
			name := canonicalBenchmarkName(fields[0])
			results[name] = append(results[name], value)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
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
	values = slices.Clone(values)
	slices.Sort(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}
