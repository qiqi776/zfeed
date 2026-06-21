# 压测结果

`bash ./scripts/bench/run.sh` 会按以下结构将压测结果输出到此目录：

```text
YYYYMMDD-HHMMSS-<commit>-<scenario>/
  README.md
  env.md
  report.md
  k6-summary.json
  k6-output.txt
  go-bench.txt
  <ghz-scenario>.txt
  promql-snapshots.md
  jaeger-traces.md
  slow-logs.ndjson
  pprof-top.txt
```

Go 压测结果会写入 `go-bench.txt`；ghz 运行会为每个 ghz 配置生成一个文本文件。报告生成器只会把包含 ghz 摘要标记的 `.txt` 纳入 RPC 指标，普通备注或附件文本不会污染 ghz 汇总。
`report.md` 会自动汇总已存在的证据路径；如果 `promql-snapshots.md`、`jaeger-traces.md`、`slow-logs.ndjson` 或 `pprof-top.txt` 不存在，报告会标记为未采集，避免把缺失证据误当作通过依据。

从当前版本开始，脚本会在真正发起 HTTP/gRPC/pprof 请求前先做目标可达性检查。如果目标服务尚未启动，将直接报错退出，不再生成只有 `env.md` 的“半成品结果目录”。

仅在 git 中保留有代表性的报告。大的原始输出、日志和长时间运行的浸泡测试产物应存放在仓库外。

## 报告摘要模板

```markdown
# zfeed 压测报告

## 基本信息

- 判定：
- 结果目录：
- 场景：
- 提交：
- 基础 URL：
- 数据目录：
- 数据规模：
- 环境类型：
- 镜像：
- 机器规格：
- Go 版本：
- GOMAXPROCS：
- 开始时间：
- 结束时间：

## 结论

- PASS/WARN/FAIL 原因：

## 容量和瓶颈结论

- 最大稳定吞吐：
- 容量拐点：
- 主要瓶颈：
- 是否通过阈值：

## 关键指标汇总

| 指标 | 结果 | 阈值 | 结论 |
| --- | ---: | ---: | --- |
| HTTP 成功率 | | >= 99.5% | |
| HTTP P95 | | < 500ms | |
| HTTP P99 | | < 1500ms | |
| RPC P95 | | 场景阈值 | |
| DB P95 | 未采集 | 100ms | WARN |
| 搜索慢请求 | 未采集 | 0 sustained | WARN |
| Go runtime goroutines | 未采集 | 不持续增长 | WARN |
| 计数一致性延迟 | 未采集 | 5s | WARN |
| Go benchmark 包数 | | >= 1 | |

## k6 HTTP 指标

## ghz RPC 指标

## Go benchmark 指标

## benchstat 对比

## Top 慢接口

| 接口 | P95 | P99 | 错误率 |
| --- | ---: | ---: | ---: |
| HTTP overall | | | |

## Top 慢 RPC

| 场景 | P95 | P99 | 错误率 |
| --- | ---: | ---: | ---: |
| ghz scenario | | | |

## Top 慢 DB

| 服务 | 表 | 操作 | P95 | slow rate |
| --- | --- | --- | ---: | ---: |
| 未采集 DB 慢查询摘要 | - | - | - | - |

## 证据路径

| 类型 | 状态 | 路径 |
| --- | --- | --- |
| Prometheus 查询 | 未采集 | promql-snapshots.md |
| Jaeger 追踪 | 未采集 | jaeger-traces.md |
| 慢日志 | 未采集 | slow-logs.ndjson |
| pprof top | 未采集 | pprof-top.txt |

## 后续动作

- 本轮结果可作为风险样本，建议补充 Prometheus、Jaeger 或日志证据后复测。

### 优化项

-

### 风险

-

## 下次复测条件

-
```
