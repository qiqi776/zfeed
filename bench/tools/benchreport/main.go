package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	k6MaxFailedRate = 0.005
	k6MinChecksRate = 0.995
	k6MaxP95MS      = 500.0
	k6MaxP99MS      = 1500.0
)

type Report struct {
	ResultDir      string
	BaselineDir    string
	Env            map[string]string
	K6             *K6Summary
	GHZ            []GHZSummary
	GoBench        []GoBenchSummary
	Evidence       []EvidencePath
	Benchstat      string
	BenchstatError string
	Verdict        string
	Reasons        []string
}

type K6Summary struct {
	RequestCount   int64
	RequestRate    float64
	HTTPFailedRate float64
	ChecksRate     float64
	ChecksPasses   int64
	ChecksFails    int64
	DurationAvgMS  float64
	DurationP95MS  float64
	DurationP99MS  float64
	HasDurationP99 bool
}

type GHZSummary struct {
	Name        string
	Count       int64
	AverageMS   float64
	P95MS       float64
	P99MS       float64
	RequestsSec float64
	OKResponses int64
	Errors      int64
}

type GoBenchSummary struct {
	Package    string
	Benchmarks int
	OKLine     string
}

type EvidencePath struct {
	Name        string
	Status      string
	Path        string
	Description string
}

func main() {
	baseline := flag.String("baseline", "", "optional baseline result directory for benchstat comparison")
	output := flag.String("output", "", "output report path, default is <result-dir>/report.md")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "用法：benchreport [--baseline <result-dir>] [--output <path>] <result-dir>")
		os.Exit(2)
	}

	resultDir := flag.Arg(0)
	report, err := collectReport(resultDir, *baseline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成报告失败：%v\n", err)
		os.Exit(1)
	}

	body := renderReport(report)
	outputPath := *output
	if outputPath == "" {
		outputPath = filepath.Join(resultDir, "report.md")
	}
	if err := os.WriteFile(outputPath, []byte(body), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入报告失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("报告已生成：%s\n", outputPath)
}

func collectReport(resultDir string, baselineDir string) (Report, error) {
	info, err := os.Stat(resultDir)
	if err != nil {
		return Report{}, err
	}
	if !info.IsDir() {
		return Report{}, fmt.Errorf("%s is not a directory", resultDir)
	}

	report := Report{
		ResultDir:   resultDir,
		BaselineDir: baselineDir,
		Env:         map[string]string{},
	}

	env, err := parseEnv(filepath.Join(resultDir, "env.md"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Report{}, err
	}
	report.Env = env

	k6, err := parseK6Summary(filepath.Join(resultDir, "k6-summary.json"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Report{}, err
	}
	report.K6 = k6

	ghz, err := parseGHZSummaries(resultDir)
	if err != nil {
		return Report{}, err
	}
	report.GHZ = ghz

	goBench, err := parseGoBench(filepath.Join(resultDir, "go-bench.txt"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Report{}, err
	}
	report.GoBench = goBench
	report.Evidence = collectEvidence(resultDir)

	if baselineDir != "" {
		report.Benchstat, report.BenchstatError = runBenchstat(
			filepath.Join(baselineDir, "go-bench.txt"),
			filepath.Join(resultDir, "go-bench.txt"),
		)
	}

	report.Verdict, report.Reasons = evaluate(report)
	return report, nil
}

func parseEnv(path string) (map[string]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		if !strings.Contains(line, "：") {
			continue
		}
		parts := strings.SplitN(line, "：", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			env[key] = value
		}
	}
	return env, nil
}

func parseK6Summary(path string) (*K6Summary, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type metricValues struct {
		Avg    float64  `json:"avg"`
		P95    float64  `json:"p(95)"`
		P99    *float64 `json:"p(99)"`
		Count  int64    `json:"count"`
		Rate   float64  `json:"rate"`
		Value  float64  `json:"value"`
		Passes int64    `json:"passes"`
		Fails  int64    `json:"fails"`
	}
	type metric struct {
		metricValues
		Values *metricValues `json:"values"`
	}
	var raw struct {
		Metrics map[string]metric `json:"metrics"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	values := func(metric metric) metricValues {
		if metric.Values != nil {
			return *metric.Values
		}
		return metric.metricValues
	}

	summary := &K6Summary{}
	if metric, ok := raw.Metrics["http_req_duration"]; ok {
		metric := values(metric)
		summary.DurationAvgMS = metric.Avg
		summary.DurationP95MS = metric.P95
		if metric.P99 != nil {
			summary.DurationP99MS = *metric.P99
			summary.HasDurationP99 = true
		}
	}
	if metric, ok := raw.Metrics["http_reqs"]; ok {
		metric := values(metric)
		summary.RequestCount = metric.Count
		summary.RequestRate = metric.Rate
	}
	if metric, ok := raw.Metrics["http_req_failed"]; ok {
		metric := values(metric)
		summary.HTTPFailedRate = metric.Value
		if total := metric.Passes + metric.Fails; total > 0 {
			summary.HTTPFailedRate = float64(metric.Passes) / float64(total)
		}
	}
	if metric, ok := raw.Metrics["checks"]; ok {
		metric := values(metric)
		summary.ChecksRate = metric.Value
		summary.ChecksPasses = metric.Passes
		summary.ChecksFails = metric.Fails
		if total := metric.Passes + metric.Fails; total > 0 {
			summary.ChecksRate = float64(metric.Passes) / float64(total)
		}
	}
	return summary, nil
}

func parseGHZSummaries(dir string) ([]GHZSummary, error) {
	var summaries []GHZSummary
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".txt" {
			return nil
		}
		name := filepath.Base(path)
		if name == "k6-output.txt" || name == "go-bench.txt" || name == "pprof-top.txt" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !isGHZSummaryText(string(body)) {
			return nil
		}
		summary := parseGHZText(string(body))
		if summary.Name == "" {
			summary.Name = strings.TrimSuffix(name, ".txt")
		}
		summaries = append(summaries, summary)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries, nil
}

func isGHZSummaryText(body string) bool {
	markers := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*Count:\s*[0-9.]+`),
		regexp.MustCompile(`(?m)^\s*Average:\s*[0-9.]+\s*ms`),
		regexp.MustCompile(`(?m)^\s*Requests/sec:\s*[0-9.]+`),
		regexp.MustCompile(`(?m)^\s*Latency distribution:`),
		regexp.MustCompile(`(?m)^\s*Status code distribution:`),
	}
	for _, marker := range markers {
		if marker.MatchString(body) {
			return true
		}
	}
	return false
}

func collectEvidence(resultDir string) []EvidencePath {
	specs := []struct {
		name        string
		files       []string
		description string
	}{
		{
			name:        "Prometheus 查询",
			files:       []string{"promql-snapshots.md"},
			description: "promql-snapshots.md",
		},
		{
			name:        "Jaeger trace",
			files:       []string{"jaeger-traces.md"},
			description: "jaeger-traces.md",
		},
		{
			name:        "慢日志",
			files:       []string{"slow-logs.ndjson", "slow-logs.md"},
			description: "slow-logs.ndjson 或 slow-logs.md",
		},
		{
			name:        "pprof top",
			files:       []string{"pprof-top.txt"},
			description: "pprof-top.txt",
		},
	}

	evidence := make([]EvidencePath, 0, len(specs))
	for _, spec := range specs {
		item := EvidencePath{
			Name:        spec.name,
			Status:      "未采集",
			Path:        spec.description,
			Description: spec.description,
		}
		for _, file := range spec.files {
			path := filepath.Join(resultDir, file)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				item.Status = "已采集"
				item.Path = path
				break
			}
		}
		evidence = append(evidence, item)
	}
	return evidence
}

func parseGHZText(body string) GHZSummary {
	return GHZSummary{
		Name:        firstStringSubmatch(body, `(?m)^\s*Name:\s*(\S+)`),
		Count:       int64(firstFloatSubmatch(body, `(?m)^\s*Count:\s*([0-9.]+)`)),
		AverageMS:   firstFloatSubmatch(body, `(?m)^\s*Average:\s*([0-9.]+)\s*ms`),
		RequestsSec: firstFloatSubmatch(body, `(?m)^\s*Requests/sec:\s*([0-9.]+)`),
		P95MS:       firstFloatSubmatch(body, `(?m)^\s*95\s*%\s*in\s*([0-9.]+)\s*ms`),
		P99MS:       firstFloatSubmatch(body, `(?m)^\s*99\s*%\s*in\s*([0-9.]+)\s*ms`),
		OKResponses: int64(firstFloatSubmatch(body, `(?m)^\s*\[OK\]\s*([0-9]+)\s*responses`)),
		Errors:      int64(firstFloatSubmatch(body, `(?m)^\s*\[[A-Z_]+\]\s*([0-9]+)\s*errors`)),
	}
}

func parseGoBench(path string) ([]GoBenchSummary, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var summaries []GoBenchSummary
	var current *GoBenchSummary
	for _, line := range strings.Split(string(body), "\n") {
		switch {
		case strings.HasPrefix(line, "pkg: "):
			if current != nil {
				summaries = append(summaries, *current)
			}
			current = &GoBenchSummary{Package: strings.TrimSpace(strings.TrimPrefix(line, "pkg: "))}
		case current != nil && strings.HasPrefix(line, "Benchmark"):
			current.Benchmarks++
		case current != nil && strings.HasPrefix(line, "ok  "):
			current.OKLine = strings.TrimSpace(line)
		}
	}
	if current != nil {
		summaries = append(summaries, *current)
	}
	return summaries, nil
}

func runBenchstat(before string, after string) (string, string) {
	if _, err := os.Stat(before); err != nil {
		return "", fmt.Sprintf("baseline go-bench.txt 不可读：%v", err)
	}
	if _, err := os.Stat(after); err != nil {
		return "", fmt.Sprintf("当前 go-bench.txt 不可读：%v", err)
	}
	if _, err := exec.LookPath("benchstat"); err != nil {
		return "", "未找到 benchstat，可执行 `go install golang.org/x/perf/cmd/benchstat@latest` 后重试"
	}

	cmd := exec.Command("benchstat", before, after)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", strings.TrimSpace(string(out)) + "\n" + err.Error()
	}
	return strings.TrimSpace(string(out)), ""
}

func evaluate(report Report) (string, []string) {
	var warnings []string
	var failures []string

	if report.K6 != nil {
		if report.K6.HTTPFailedRate > k6MaxFailedRate {
			failures = append(failures, fmt.Sprintf("HTTP 错误率 %.4f%% 超过 0.5%%", report.K6.HTTPFailedRate*100))
		}
		if report.K6.ChecksRate > 0 && report.K6.ChecksRate < k6MinChecksRate {
			failures = append(failures, fmt.Sprintf("k6 checks 通过率 %.4f%% 低于 99.5%%", report.K6.ChecksRate*100))
		}
		if report.K6.DurationP95MS > k6MaxP95MS {
			warnings = append(warnings, fmt.Sprintf("HTTP P95 %.2fms 超过 %.0fms", report.K6.DurationP95MS, k6MaxP95MS))
		}
		if !report.K6.HasDurationP99 {
			warnings = append(warnings, "k6 summary 未包含 HTTP P99，请确认 summaryTrendStats 已包含 p(99)")
		} else if report.K6.DurationP99MS > k6MaxP99MS {
			warnings = append(warnings, fmt.Sprintf("HTTP P99 %.2fms 超过 %.0fms", report.K6.DurationP99MS, k6MaxP99MS))
		}
	}
	for _, item := range report.GHZ {
		if item.Errors > 0 {
			failures = append(failures, fmt.Sprintf("ghz 场景 %s 出现 %d 个错误", item.Name, item.Errors))
		}
	}
	if report.BenchstatError != "" {
		warnings = append(warnings, report.BenchstatError)
	}
	if hasMissingEvidence(report.Evidence) {
		warnings = append(warnings, "诊断证据未采集完整，不能直接作为容量结论")
	}

	if len(failures) > 0 {
		return "FAIL", failures
	}
	if len(warnings) > 0 {
		return "WARN", warnings
	}
	return "PASS", []string{"所有已采集指标均满足当前阈值"}
}

func renderReport(report Report) string {
	if report.Verdict == "" {
		report.Verdict, report.Reasons = evaluate(report)
	}

	var out bytes.Buffer
	fmt.Fprintln(&out, "# zfeed 压测报告")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "## 基本信息")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "- 判定：%s\n", report.Verdict)
	fmt.Fprintf(&out, "- 结果目录：%s\n", valueOr(report.ResultDir, "-"))
	for _, key := range []string{
		"场景",
		"提交",
		"基础 URL",
		"数据目录",
		"数据规模",
		"环境类型",
		"镜像",
		"机器规格",
		"Go 版本",
		"GOMAXPROCS",
		"开始时间",
		"结束时间",
	} {
		fmt.Fprintf(&out, "- %s：%s\n", key, valueOr(report.Env[key], "-"))
	}
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## 结论")
	fmt.Fprintln(&out)
	for _, reason := range report.Reasons {
		fmt.Fprintf(&out, "- %s\n", reason)
	}
	fmt.Fprintln(&out)
	renderCapacityAndBottleneck(&out, report)
	renderKeyMetrics(&out, report)

	if report.K6 != nil {
		fmt.Fprintln(&out, "## k6 HTTP 指标")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "| 指标 | 结果 | 阈值 |")
		fmt.Fprintln(&out, "| --- | ---: | ---: |")
		fmt.Fprintf(&out, "| HTTP 请求数 | %d | - |\n", report.K6.RequestCount)
		fmt.Fprintf(&out, "| HTTP RPS | %.2f | - |\n", report.K6.RequestRate)
		fmt.Fprintf(&out, "| HTTP 错误率 | %.4f%% | < 0.5%% |\n", report.K6.HTTPFailedRate*100)
		fmt.Fprintf(&out, "| checks 通过率 | %.4f%% | >= 99.5%% |\n", report.K6.ChecksRate*100)
		fmt.Fprintf(&out, "| HTTP 平均耗时 | %.2fms | - |\n", report.K6.DurationAvgMS)
		fmt.Fprintf(&out, "| HTTP P95 | %.2fms | < 500ms |\n", report.K6.DurationP95MS)
		fmt.Fprintf(&out, "| HTTP P99 | %s | < 1500ms |\n", k6P99Value(report.K6))
		fmt.Fprintln(&out)
	}

	if len(report.GHZ) > 0 {
		fmt.Fprintln(&out, "## ghz RPC 指标")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "| 场景 | 请求数 | RPS | 平均耗时 | P95 | P99 | OK | Errors |")
		fmt.Fprintln(&out, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
		for _, item := range report.GHZ {
			fmt.Fprintf(&out, "| %s | %d | %.2f | %.2fms | %.2fms | %.2fms | %d | %d |\n",
				item.Name, item.Count, item.RequestsSec, item.AverageMS, item.P95MS, item.P99MS, item.OKResponses, item.Errors)
		}
		fmt.Fprintln(&out)
	}

	if len(report.GoBench) > 0 {
		fmt.Fprintln(&out, "## Go benchmark 指标")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "| 包 | benchmark 条目 | 状态 |")
		fmt.Fprintln(&out, "| --- | ---: | --- |")
		for _, item := range report.GoBench {
			fmt.Fprintf(&out, "| %s | %d | %s |\n", item.Package, item.Benchmarks, valueOr(item.OKLine, "-"))
		}
		fmt.Fprintln(&out)
	}

	if report.Benchstat != "" || report.BenchstatError != "" {
		fmt.Fprintln(&out, "## benchstat 对比")
		fmt.Fprintln(&out)
		if report.BenchstatError != "" {
			fmt.Fprintf(&out, "- %s\n\n", report.BenchstatError)
		}
		if report.Benchstat != "" {
			fmt.Fprintln(&out, "```text")
			fmt.Fprintln(&out, report.Benchstat)
			fmt.Fprintln(&out, "```")
			fmt.Fprintln(&out)
		}
	}

	renderTopSlowSections(&out, report)

	if len(report.Evidence) > 0 {
		fmt.Fprintln(&out, "## 证据路径")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "| 类型 | 状态 | 路径 |")
		fmt.Fprintln(&out, "| --- | --- | --- |")
		for _, item := range report.Evidence {
			fmt.Fprintf(&out, "| %s | %s | %s |\n", item.Name, item.Status, item.Path)
		}
		fmt.Fprintln(&out)
	}

	fmt.Fprintln(&out, "## 后续动作")
	fmt.Fprintln(&out)
	switch report.Verdict {
	case "FAIL":
		fmt.Fprintln(&out, "- 本轮结果不能作为通过证据，先定位失败项再复测。")
	case "WARN":
		fmt.Fprintln(&out, "- 本轮结果可作为风险样本，建议补充 Prometheus、Jaeger 或日志证据后复测。")
	default:
		fmt.Fprintln(&out, "- 本轮结果可作为当前环境下的基线样本。")
	}
	fmt.Fprintln(&out)
	renderFollowUps(&out, report)
	renderRetestConditions(&out, report)

	return out.String()
}

func renderCapacityAndBottleneck(out *bytes.Buffer, report Report) {
	fmt.Fprintln(out, "## 容量和瓶颈结论")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "- 最大稳定吞吐：%s\n", stableThroughput(report))
	fmt.Fprintf(out, "- 容量拐点：%s\n", capacityKnee(report))
	fmt.Fprintf(out, "- 主要瓶颈：%s\n", bottleneckSummary(report))
	fmt.Fprintf(out, "- 是否通过阈值：%s\n", valueOr(report.Verdict, "-"))
	fmt.Fprintln(out)
}

func renderKeyMetrics(out *bytes.Buffer, report Report) {
	fmt.Fprintln(out, "## 关键指标汇总")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| 指标 | 结果 | 阈值 | 结论 |")
	fmt.Fprintln(out, "| --- | ---: | ---: | --- |")
	if report.K6 != nil {
		fmt.Fprintf(out, "| HTTP 成功率 | %.4f%% | >= 99.5%% | %s |\n",
			(1-report.K6.HTTPFailedRate)*100,
			statusByThreshold(report.K6.HTTPFailedRate <= k6MaxFailedRate),
		)
		fmt.Fprintf(out, "| HTTP P95 | %.2fms | < 500ms | %s |\n",
			report.K6.DurationP95MS,
			statusByThreshold(report.K6.DurationP95MS <= k6MaxP95MS),
		)
		fmt.Fprintf(out, "| HTTP P99 | %s | < 1500ms | %s |\n",
			k6P99Value(report.K6),
			statusByThreshold(report.K6.HasDurationP99 && report.K6.DurationP99MS <= k6MaxP99MS),
		)
	} else {
		fmt.Fprintln(out, "| HTTP 成功率 | 未采集 | >= 99.5% | WARN |")
		fmt.Fprintln(out, "| HTTP P95 | 未采集 | < 500ms | WARN |")
		fmt.Fprintln(out, "| HTTP P99 | 未采集 | < 1500ms | WARN |")
	}
	if slowest, ok := slowestGHZ(report.GHZ); ok {
		fmt.Fprintf(out, "| RPC P95 | %.2fms | 场景阈值 | %s |\n",
			slowest.P95MS,
			statusByThreshold(slowest.Errors == 0),
		)
	} else {
		fmt.Fprintln(out, "| RPC P95 | 未采集 | 场景阈值 | WARN |")
	}
	fmt.Fprintln(out, "| DB P95 | 未采集 | 100ms | WARN |")
	fmt.Fprintln(out, "| 搜索慢请求 | 未采集 | 0 sustained | WARN |")
	fmt.Fprintln(out, "| Go runtime goroutines | 未采集 | 不持续增长 | WARN |")
	fmt.Fprintln(out, "| 计数一致性延迟 | 未采集 | 5s | WARN |")
	fmt.Fprintf(out, "| Go benchmark 包数 | %d | >= 1 | %s |\n",
		len(report.GoBench),
		statusByThreshold(len(report.GoBench) > 0),
	)
	fmt.Fprintln(out)
}

func renderTopSlowSections(out *bytes.Buffer, report Report) {
	fmt.Fprintln(out, "## Top 慢接口")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| 接口 | P95 | P99 | 错误率 |")
	fmt.Fprintln(out, "| --- | ---: | ---: | ---: |")
	if report.K6 != nil {
		fmt.Fprintf(out, "| HTTP overall | %.2fms | %s | %.4f%% |\n",
			report.K6.DurationP95MS,
			k6P99Value(report.K6),
			report.K6.HTTPFailedRate*100,
		)
	} else {
		fmt.Fprintln(out, "| 未采集 HTTP 分接口摘要 | - | - | - |")
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "## Top 慢 RPC")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| 场景 | P95 | P99 | 错误率 |")
	fmt.Fprintln(out, "| --- | ---: | ---: | ---: |")
	if len(report.GHZ) == 0 {
		fmt.Fprintln(out, "| 未采集 ghz RPC 摘要 | - | - | - |")
	} else {
		items := append([]GHZSummary(nil), report.GHZ...)
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].P95MS == items[j].P95MS {
				return items[i].Name < items[j].Name
			}
			return items[i].P95MS > items[j].P95MS
		})
		for i, item := range items {
			if i >= 5 {
				break
			}
			fmt.Fprintf(out, "| %s | %.2fms | %.2fms | %.4f%% |\n",
				item.Name,
				item.P95MS,
				item.P99MS,
				ghzErrorRate(item)*100,
			)
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "## Top 慢 DB")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| 服务 | 表 | 操作 | P95 | slow rate |")
	fmt.Fprintln(out, "| --- | --- | --- | ---: | ---: |")
	fmt.Fprintln(out, "| 未采集 DB 慢查询摘要 | - | - | - | - |")
	fmt.Fprintln(out)
}

func renderFollowUps(out *bytes.Buffer, report Report) {
	fmt.Fprintln(out, "### 优化项")
	fmt.Fprintln(out)
	optimizations := optimizationReasons(report.Reasons)
	if len(optimizations) == 0 {
		fmt.Fprintln(out, "- 暂无自动识别的优化项。")
	} else {
		for _, reason := range optimizations {
			fmt.Fprintf(out, "- 处理：%s\n", reason)
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "### 风险")
	fmt.Fprintln(out)
	if hasMissingEvidence(report.Evidence) {
		fmt.Fprintln(out, "- 诊断证据未采集完整，不能直接作为容量结论。")
	} else if report.Verdict == "PASS" {
		fmt.Fprintln(out, "- 未发现自动判定风险，仍需保留原始结果用于复查。")
	} else {
		for _, reason := range report.Reasons {
			fmt.Fprintf(out, "- %s\n", reason)
		}
	}
	fmt.Fprintln(out)
}

func optimizationReasons(reasons []string) []string {
	var optimizations []string
	for _, reason := range reasons {
		if strings.Contains(reason, "诊断证据未采集完整") {
			continue
		}
		optimizations = append(optimizations, reason)
	}
	return optimizations
}

func renderRetestConditions(out *bytes.Buffer, report Report) {
	fmt.Fprintln(out, "## 下次复测条件")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "- 使用相同 commit 或明确记录对比 commit。")
	fmt.Fprintln(out, "- 使用相同 `DATA_DIR`、目标地址、Go 版本和 benchmark 参数。")
	if hasMissingEvidence(report.Evidence) {
		fmt.Fprintln(out, "- 补齐缺失的 Prometheus、Jaeger、慢日志或 pprof 证据后复测。")
	}
	if report.Verdict != "PASS" {
		fmt.Fprintln(out, "- 修复 FAIL/WARN 项后重新执行相同场景。")
	}
}

func stableThroughput(report Report) string {
	if report.K6 != nil && report.K6.RequestRate > 0 {
		return fmt.Sprintf("%.2f HTTP RPS", report.K6.RequestRate)
	}
	if slowest, ok := slowestGHZ(report.GHZ); ok && slowest.RequestsSec > 0 {
		return fmt.Sprintf("%.2f RPC RPS (%s)", slowest.RequestsSec, slowest.Name)
	}
	return "未采集，需要 k6 或 ghz 结果"
}

func capacityKnee(report Report) string {
	if report.K6 != nil {
		if report.K6.DurationP95MS > k6MaxP95MS || (report.K6.HasDurationP99 && report.K6.DurationP99MS > k6MaxP99MS) {
			return "本轮已触及 HTTP 延迟阈值"
		}
		return "本轮未观察到明确拐点"
	}
	if len(report.GHZ) > 0 {
		return "本轮仅有 RPC 定点数据，需结合阶梯压测判断"
	}
	return "未采集"
}

func bottleneckSummary(report Report) string {
	if report.K6 != nil && report.K6.HTTPFailedRate > k6MaxFailedRate {
		return "HTTP 错误率超过阈值，优先检查入口和下游错误日志"
	}
	if report.K6 != nil && report.K6.DurationP95MS > k6MaxP95MS {
		return "HTTP P95 超过阈值，优先检查慢接口、RPC 和 DB 证据"
	}
	if slowest, ok := slowestGHZ(report.GHZ); ok && slowest.Errors > 0 {
		return fmt.Sprintf("ghz 场景 %s 出现错误，优先检查对应 RPC 服务", slowest.Name)
	}
	if hasMissingEvidence(report.Evidence) {
		return "自动指标未发现失败项，但诊断证据未采集完整"
	}
	return "未发现明确瓶颈"
}

func slowestGHZ(items []GHZSummary) (GHZSummary, bool) {
	if len(items) == 0 {
		return GHZSummary{}, false
	}
	slowest := items[0]
	for _, item := range items[1:] {
		if item.P95MS > slowest.P95MS {
			slowest = item
		}
	}
	return slowest, true
}

func ghzErrorRate(item GHZSummary) float64 {
	total := item.OKResponses + item.Errors
	if total <= 0 {
		return 0
	}
	return float64(item.Errors) / float64(total)
}

func hasMissingEvidence(items []EvidencePath) bool {
	if len(items) == 0 {
		return true
	}
	for _, item := range items {
		if item.Status != "已采集" {
			return true
		}
	}
	return false
}

func statusByThreshold(ok bool) string {
	if ok {
		return "PASS"
	}
	return "WARN"
}

func firstStringSubmatch(body string, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func firstFloatSubmatch(body string, pattern string) float64 {
	value := firstStringSubmatch(body, pattern)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func k6P99Value(summary *K6Summary) string {
	if summary == nil || !summary.HasDurationP99 {
		return "未采集"
	}
	return fmt.Sprintf("%.2fms", summary.DurationP99MS)
}
