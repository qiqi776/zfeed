# 压测结果

`bash ./scripts/bench/run.sh` 会按以下结构将压测结果输出到此目录：

```text
YYYYMMDD-HHMMSS-<commit>-<scenario>/
  README.md
  env.md
  k6-summary.json
  k6-output.txt
```

Go 压测结果会写入 `go-bench.txt`；ghz 运行会为每个 ghz 配置生成一个文本文件。

从当前版本开始，脚本会在真正发起 HTTP/gRPC/pprof 请求前先做目标可达性检查。如果目标服务尚未启动，将直接报错退出，不再生成只有 `env.md` 的“半成品结果目录”。

仅在 git 中保留有代表性的报告。大的原始输出、日志和长时间运行的浸泡测试产物应存放在仓库外。

## 报告摘要模板

```markdown
# zfeed 压测报告

## 基本信息

- 提交：
- 场景：
- 数据规模：
- 环境：
- 开始时间：
- 结束时间：
- 目标：

## 结论

- 最大稳定吞吐：
- 容量拐点：
- 主要瓶颈：
- 通过的阈值：

## 关键指标

| 指标        | 结果 |  阈值 | 状态 |
| ----------- | ---: | ----: | ---- |
| HTTP 成功率 |      | 99.5% |      |
| HTTP P95    |      |       |      |
| HTTP P99    |      |       |      |
| RPC P95     |      |       |      |
| DB P95      |      | 100ms |      |

## 证据

- Prometheus 查询：
- Jaeger 追踪：
- 慢日志：
- 后续修复：
```
