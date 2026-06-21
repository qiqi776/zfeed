package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectReportParsesK6GhzGoBenchAndEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "env.md"), `# 压测环境

- 场景：smoke
- 提交：abc1234
- 基础 URL：http://127.0.0.1:18080
- 数据目录：/tmp/zfeed/bench/data/small
- 开始时间：2026-06-21T10:00:00+08:00
`)
	writeFile(t, filepath.Join(dir, "k6-summary.json"), `{
  "metrics": {
    "http_req_duration": {"values": {"avg": 23.5, "p(95)": 120.1, "p(99)": 450.2}},
    "http_reqs": {"values": {"count": 100, "rate": 50.5}},
    "http_req_failed": {"values": {"value": 0.001}},
    "checks": {"values": {"value": 0.998, "passes": 99, "fails": 1}}
  }
}`)
	writeFile(t, filepath.Join(dir, "feed-recommend.txt"), `Summary:
  Name:		feed-recommend
  Count:	1000
  Average:	6.81 ms
  Requests/sec:	2750.42

Latency distribution:
  95 % in 10.67 ms
  99 % in 112.12 ms

Status code distribution:
  [OK]   1000 responses
`)
	writeFile(t, filepath.Join(dir, "go-bench.txt"), `goos: linux
goarch: amd64
pkg: zfeed/pkg/hotrank
cpu: test CPU
BenchmarkFormulaScore-8    1000000    25.35 ns/op    0 B/op    0 allocs/op
PASS
ok  	zfeed/pkg/hotrank	1.234s
`)

	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	if report.Env["场景"] != "smoke" {
		t.Fatalf("scenario = %q, want smoke", report.Env["场景"])
	}
	if report.K6 == nil || report.K6.RequestCount != 100 || report.K6.HTTPFailedRate != 0.001 {
		t.Fatalf("k6 summary = %+v, want parsed request count and failure rate", report.K6)
	}
	if report.K6.DurationP99MS != 450.2 || !report.K6.HasDurationP99 {
		t.Fatalf("k6 p99 = %.2f, has = %t, want 450.2 and true", report.K6.DurationP99MS, report.K6.HasDurationP99)
	}
	if len(report.GHZ) != 1 || report.GHZ[0].Name != "feed-recommend" || report.GHZ[0].P95MS != 10.67 {
		t.Fatalf("ghz summaries = %+v, want feed-recommend p95", report.GHZ)
	}
	if len(report.GoBench) != 1 || report.GoBench[0].Package != "zfeed/pkg/hotrank" || report.GoBench[0].Benchmarks != 1 {
		t.Fatalf("go bench summaries = %+v, want pkg/hotrank count", report.GoBench)
	}
}

func TestCollectReportHandlesLegacyK6SummaryWithoutP99(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "k6-summary.json"), `{
  "metrics": {
    "http_req_duration": {"avg": 23.5, "p(95)": 120.1},
    "http_reqs": {"count": 100, "rate": 50.5},
    "http_req_failed": {"value": 0, "passes": 0, "fails": 100},
    "checks": {"value": 1, "passes": 100, "fails": 0}
  }
}`)

	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	if report.K6 == nil {
		t.Fatalf("k6 summary is nil")
	}
	if report.K6.HTTPFailedRate != 0 {
		t.Fatalf("http failed rate = %.4f, want 0 from failed-rate passes/(passes+fails)", report.K6.HTTPFailedRate)
	}
	if report.K6.ChecksRate != 1 {
		t.Fatalf("checks rate = %.4f, want 1 from passes/(passes+fails)", report.K6.ChecksRate)
	}
	if report.K6.HasDurationP99 {
		t.Fatalf("HasDurationP99 = true, want false")
	}
	if report.Verdict != "WARN" {
		t.Fatalf("verdict = %s, want WARN for missing p99", report.Verdict)
	}
	rendered := renderReport(report)
	if !strings.Contains(rendered, "| HTTP P99 | 未采集 |") {
		t.Fatalf("rendered report missing uncollected p99 marker:\n%s", rendered)
	}
}

func TestCollectReportIgnoresNonGHZTextFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "notes.txt"), "operator notes\nnot a ghz summary\n")

	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	if len(report.GHZ) != 0 {
		t.Fatalf("ghz summaries = %+v, want no summaries from notes.txt", report.GHZ)
	}
	rendered := renderReport(report)
	if strings.Contains(rendered, "notes") {
		t.Fatalf("rendered report includes non-ghz text file:\n%s", rendered)
	}
}

func TestParseK6SummaryUsesRatePassesAsFailedRequests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "k6-summary.json"), `{
  "metrics": {
    "http_req_failed": {"value": 1, "passes": 1, "fails": 12},
    "checks": {"value": 0.923076923, "passes": 12, "fails": 1}
  }
}`)

	summary, err := parseK6Summary(filepath.Join(dir, "k6-summary.json"))
	if err != nil {
		t.Fatalf("parseK6Summary returned error: %v", err)
	}

	if summary.HTTPFailedRate != float64(1)/float64(13) {
		t.Fatalf("http failed rate = %.6f, want 1/13", summary.HTTPFailedRate)
	}
	if summary.ChecksRate != float64(12)/float64(13) {
		t.Fatalf("checks rate = %.6f, want 12/13", summary.ChecksRate)
	}
}

func TestRenderReportMarksFailWhenK6ErrorRateExceedsThreshold(t *testing.T) {
	report := Report{
		ResultDir: "bench/results/example",
		Env: map[string]string{
			"场景": "smoke",
			"提交": "abc1234",
		},
		K6: &K6Summary{
			RequestCount:   100,
			RequestRate:    10,
			HTTPFailedRate: 0.02,
			ChecksRate:     0.99,
			DurationP95MS:  120,
			DurationP99MS:  450,
			HasDurationP99: true,
			DurationAvgMS:  30,
		},
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"# zfeed 压测报告",
		"- 判定：FAIL",
		"HTTP 错误率 2.0000% 超过 0.5%",
		"| HTTP P95 | 120.00ms |",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderReportIncludesReproducibilityEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "env.md"), `# 压测环境

- 场景：go-bench
- 提交：abc1234
- 基础 URL：http://127.0.0.1:18080
- 数据目录：bench/data/small
- 数据规模：small
- 环境类型：local
- 镜像：front-api=zfeed/front-api:abc1234,user-rpc=zfeed/user-rpc:abc1234
- 机器规格：linux amd64 cpu=8 mem=16GiB
- Go 版本：go1.24.1
- GOMAXPROCS：8
- 开始时间：2026-06-21T10:00:00+08:00
- 结束时间：2026-06-21T10:05:00+08:00
`)

	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"- 环境类型：local",
		"- 数据规模：small",
		"- 镜像：front-api=zfeed/front-api:abc1234,user-rpc=zfeed/user-rpc:abc1234",
		"- 机器规格：linux amd64 cpu=8 mem=16GiB",
		"- Go 版本：go1.24.1",
		"- GOMAXPROCS：8",
		"- 结束时间：2026-06-21T10:05:00+08:00",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderReportIncludesSearchAndRuntimeMetricPlaceholders(t *testing.T) {
	report := Report{
		ResultDir: "bench/results/example",
		Env: map[string]string{
			"场景": "mixed",
			"提交": "abc1234",
		},
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"| 搜索慢请求 | 未采集 | 0 sustained | WARN |",
		"| Go runtime goroutines | 未采集 | 不持续增长 | WARN |",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderReportIncludesFollowUpRiskAndOptimizationSections(t *testing.T) {
	report := Report{
		ResultDir: "bench/results/example",
		Env: map[string]string{
			"场景": "mixed",
			"提交": "abc1234",
		},
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"### 优化项",
		"### 风险",
		"## 下次复测条件",
		"- 暂无自动识别的优化项。",
		"- 诊断证据未采集完整，不能直接作为容量结论。",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestEvaluateWarnsWhenEvidenceIsMissing(t *testing.T) {
	report := Report{
		Evidence: []EvidencePath{{
			Name:   "Prometheus 查询",
			Status: "未采集",
			Path:   "promql-snapshots.md",
		}},
	}

	verdict, reasons := evaluate(report)
	if verdict != "WARN" {
		t.Fatalf("verdict = %s, want WARN", verdict)
	}
	if len(reasons) == 0 || !strings.Contains(reasons[0], "诊断证据未采集完整") {
		t.Fatalf("reasons = %+v, want missing evidence warning", reasons)
	}
}

func TestRenderReportIncludesEvidencePaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "promql-snapshots.md"), "# PromQL\n")
	writeFile(t, filepath.Join(dir, "jaeger-traces.md"), "# Jaeger\n")
	writeFile(t, filepath.Join(dir, "slow-logs.ndjson"), "{}\n")
	writeFile(t, filepath.Join(dir, "pprof-top.txt"), "top\n")

	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"## 证据路径",
		"| Prometheus 查询 | 已采集 |",
		"promql-snapshots.md",
		"| Jaeger trace | 已采集 |",
		"jaeger-traces.md",
		"| 慢日志 | 已采集 |",
		"slow-logs.ndjson",
		"| pprof top | 已采集 |",
		"pprof-top.txt",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderReportMarksMissingEvidencePaths(t *testing.T) {
	dir := t.TempDir()
	report, err := collectReport(dir, "")
	if err != nil {
		t.Fatalf("collectReport returned error: %v", err)
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"| Prometheus 查询 | 未采集 | promql-snapshots.md |",
		"| Jaeger trace | 未采集 | jaeger-traces.md |",
		"| 慢日志 | 未采集 | slow-logs.ndjson 或 slow-logs.md |",
		"| pprof top | 未采集 | pprof-top.txt |",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderReportIncludesFixedAnalysisSections(t *testing.T) {
	report := Report{
		ResultDir: "bench/results/example",
		Env: map[string]string{
			"场景":     "mixed",
			"提交":     "abc1234",
			"基础 URL": "http://127.0.0.1:18080",
			"数据目录":   "bench/data/small",
			"开始时间":   "2026-06-21T10:00:00+08:00",
		},
		K6: &K6Summary{
			RequestCount:   100,
			RequestRate:    25,
			HTTPFailedRate: 0,
			ChecksRate:     1,
			DurationAvgMS:  30,
			DurationP95MS:  120,
			DurationP99MS:  450,
			HasDurationP99: true,
		},
		GHZ: []GHZSummary{{
			Name:        "feed-recommend",
			Count:       1000,
			AverageMS:   6,
			P95MS:       11,
			P99MS:       40,
			RequestsSec: 200,
			OKResponses: 1000,
		}},
		GoBench: []GoBenchSummary{{
			Package:    "zfeed/bench/go/feedrank",
			Benchmarks: 2,
			OKLine:     "ok zfeed/bench/go/feedrank 2.414s",
		}},
		Evidence: collectEvidence(t.TempDir()),
	}

	rendered := renderReport(report)
	for _, want := range []string{
		"## 容量和瓶颈结论",
		"- 最大稳定吞吐：",
		"- 容量拐点：",
		"- 主要瓶颈：",
		"## 关键指标汇总",
		"| HTTP 成功率 |",
		"| RPC P95 |",
		"| Go benchmark 包数 |",
		"## Top 慢接口",
		"## Top 慢 RPC",
		"| feed-recommend | 11.00ms | 40.00ms | 0.0000% |",
		"## Top 慢 DB",
		"未采集 DB 慢查询摘要",
		"## 下次复测条件",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered report missing %q:\n%s", want, rendered)
		}
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
