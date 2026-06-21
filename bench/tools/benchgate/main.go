package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Thresholds struct {
	Time   float64
	Bytes  float64
	Allocs float64
}

type BenchmarkMetric struct {
	Package  string
	Name     string
	NsOp     float64
	BytesOp  float64
	AllocsOp float64
}

type GateFailure struct {
	Benchmark string
	Metric    string
	Baseline  float64
	Current   float64
	Change    float64
	Threshold float64
}

type GateReport struct {
	Passed   bool
	Compared int
	Failures []GateFailure
}

func main() {
	baseline := flag.String("baseline", "", "baseline go-bench.txt")
	current := flag.String("current", "", "current go-bench.txt")
	timeThreshold := flag.Float64("time-threshold", 0.10, "allowed ns/op regression ratio")
	bytesThreshold := flag.Float64("bytes-threshold", 0.10, "allowed B/op regression ratio")
	allocsThreshold := flag.Float64("allocs-threshold", 0.10, "allowed allocs/op regression ratio")
	flag.Parse()

	if *baseline == "" || *current == "" {
		fmt.Fprintln(os.Stderr, "用法：benchgate --baseline <go-bench.txt> --current <go-bench.txt> [--time-threshold 0.10]")
		os.Exit(2)
	}

	report, err := runGate(*baseline, *current, Thresholds{
		Time:   *timeThreshold,
		Bytes:  *bytesThreshold,
		Allocs: *allocsThreshold,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark gate 失败：%v\n", err)
		os.Exit(1)
	}

	renderGateReport(report)
	if !report.Passed {
		os.Exit(1)
	}
}

func runGate(baselinePath string, currentPath string, thresholds Thresholds) (GateReport, error) {
	baselineBody, err := os.ReadFile(baselinePath)
	if err != nil {
		return GateReport{}, err
	}
	currentBody, err := os.ReadFile(currentPath)
	if err != nil {
		return GateReport{}, err
	}
	report := compareBenchmarks(
		parseBenchmarksFromText(string(baselineBody)),
		parseBenchmarksFromText(string(currentBody)),
		thresholds,
	)
	if report.Compared == 0 {
		return report, fmt.Errorf("未找到可比较的 benchmark 条目")
	}
	return report, nil
}

func parseBenchmarksFromText(body string) map[string]BenchmarkMetric {
	result := map[string]BenchmarkMetric{}
	currentPackage := ""
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "pkg: ") {
			currentPackage = strings.TrimSpace(strings.TrimPrefix(line, "pkg: "))
			continue
		}
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		metric := BenchmarkMetric{
			Package: currentPackage,
			Name:    fields[0],
		}
		for i := 2; i+1 < len(fields); i++ {
			value, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				continue
			}
			switch fields[i+1] {
			case "ns/op":
				metric.NsOp = value
			case "B/op":
				metric.BytesOp = value
			case "allocs/op":
				metric.AllocsOp = value
			}
		}
		result[metric.key()] = metric
	}
	return result
}

func compareBenchmarks(baseline map[string]BenchmarkMetric, current map[string]BenchmarkMetric, thresholds Thresholds) GateReport {
	report := GateReport{Passed: true}
	for key, base := range baseline {
		now, ok := current[key]
		if !ok {
			report.addFailure(&GateFailure{
				Benchmark: key,
				Metric:    "missing",
			})
			continue
		}
		report.Compared++
		report.addFailure(compareMetric(key, "ns/op", base.NsOp, now.NsOp, thresholds.Time))
		report.addFailure(compareMetric(key, "B/op", base.BytesOp, now.BytesOp, thresholds.Bytes))
		report.addFailure(compareMetric(key, "allocs/op", base.AllocsOp, now.AllocsOp, thresholds.Allocs))
	}
	if len(report.Failures) > 0 {
		report.Passed = false
	}
	return report
}

func (r *GateReport) addFailure(failure *GateFailure) {
	if failure == nil {
		return
	}
	r.Failures = append(r.Failures, *failure)
}

func compareMetric(name string, metric string, baseline float64, current float64, threshold float64) *GateFailure {
	if baseline <= 0 || current <= 0 {
		return nil
	}
	change := (current - baseline) / baseline
	if change <= threshold {
		return nil
	}
	return &GateFailure{
		Benchmark: name,
		Metric:    metric,
		Baseline:  baseline,
		Current:   current,
		Change:    change,
		Threshold: threshold,
	}
}

func renderGateReport(report GateReport) {
	if report.Passed {
		fmt.Printf("PASS benchmark gate: compared=%d\n", report.Compared)
		return
	}
	fmt.Printf("FAIL benchmark gate: compared=%d failures=%d\n", report.Compared, len(report.Failures))
	for _, failure := range report.Failures {
		if failure.Metric == "missing" {
			fmt.Printf("- %s missing in current benchmark output\n", failure.Benchmark)
			continue
		}
		fmt.Printf(
			"- %s %s %.4g -> %.4g change=%.2f%% threshold=%.2f%%\n",
			failure.Benchmark,
			failure.Metric,
			failure.Baseline,
			failure.Current,
			failure.Change*100,
			failure.Threshold*100,
		)
	}
}

func (m BenchmarkMetric) key() string {
	if m.Package == "" {
		return m.Name
	}
	return m.Package + " " + m.Name
}
