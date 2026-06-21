package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareBenchmarksFailsOnNsOpRegression(t *testing.T) {
	baseline := parseBenchmarksFromText(`goos: linux
goarch: amd64
pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  100.0 ns/op  0 B/op  0 allocs/op
PASS
`)
	current := parseBenchmarksFromText(`goos: linux
goarch: amd64
pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  116.0 ns/op  0 B/op  0 allocs/op
PASS
`)

	report := compareBenchmarks(baseline, current, Thresholds{Time: 0.10, Bytes: 0.10, Allocs: 0.10})
	if report.Passed {
		t.Fatalf("expected regression to fail gate: %+v", report)
	}
	if len(report.Failures) != 1 || report.Failures[0].Metric != "ns/op" {
		t.Fatalf("failures = %+v, want one ns/op failure", report.Failures)
	}
}

func TestCompareBenchmarksPassesWithinThreshold(t *testing.T) {
	baseline := parseBenchmarksFromText(`pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  100.0 ns/op  100 B/op  10 allocs/op
`)
	current := parseBenchmarksFromText(`pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  108.0 ns/op  109 B/op  10 allocs/op
`)

	report := compareBenchmarks(baseline, current, Thresholds{Time: 0.10, Bytes: 0.10, Allocs: 0.10})
	if !report.Passed {
		t.Fatalf("expected gate to pass within threshold: %+v", report)
	}
}

func TestCompareBenchmarksFailsWhenCurrentBenchmarkIsMissing(t *testing.T) {
	baseline := parseBenchmarksFromText(`pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  100.0 ns/op  100 B/op  10 allocs/op
`)
	current := map[string]BenchmarkMetric{}

	report := compareBenchmarks(baseline, current, Thresholds{Time: 0.10, Bytes: 0.10, Allocs: 0.10})
	if report.Passed {
		t.Fatalf("expected missing benchmark to fail gate: %+v", report)
	}
	if len(report.Failures) != 1 || report.Failures[0].Metric != "missing" {
		t.Fatalf("failures = %+v, want missing benchmark failure", report.Failures)
	}
}

func TestRunGateParsesFiles(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.txt")
	currentPath := filepath.Join(dir, "current.txt")
	writeBenchGateFile(t, baselinePath, `pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  100.0 ns/op  100 B/op  10 allocs/op
`)
	writeBenchGateFile(t, currentPath, `pkg: zfeed/pkg/hotrank
BenchmarkFormulaScore-8  1000000  101.0 ns/op  100 B/op  10 allocs/op
`)

	report, err := runGate(baselinePath, currentPath, Thresholds{Time: 0.10, Bytes: 0.10, Allocs: 0.10})
	if err != nil {
		t.Fatalf("runGate returned error: %v", err)
	}
	if !report.Passed || report.Compared != 1 {
		t.Fatalf("report = %+v, want pass with one comparison", report)
	}
}

func writeBenchGateFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
