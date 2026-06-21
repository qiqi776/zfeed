# zfeed benchmark CI template

`github-actions-benchmark.yml` 是可复制到 `.github/workflows/` 的 benchmark 门禁模板。

当前实现保持在 `bench/ci/` 下，满足本轮文件边界；GitHub Actions 只有真正放入 `.github/workflows/` 后才会由 GitHub 自动触发。

本地 nightly 门禁已通过 `scripts/bench/nightly-gate.sh` 和 `scripts/bench/install-nightly-cron.sh` 提供。它使用 `bench/tools/benchgate` 对比核心 Go benchmark，已有 benchmark 缺失、`ns/op`、`B/op` 或 `allocs/op` 超阈值会失败。
nightly 默认 `BENCH_NIGHTLY_COUNT=10`，避免单次 benchmark 噪声直接触发误报；临时调试可显式降低该值。

预览安装动作：

```bash
bash scripts/bench/install-ci-workflow.sh --dry-run
```

显式安装到 GitHub Actions：

```bash
bash scripts/bench/install-ci-workflow.sh --apply
```

初始化本地 nightly baseline：

```bash
bash scripts/bench/nightly-gate.sh seed
```

执行本地 nightly 门禁：

```bash
bash scripts/bench/nightly-gate.sh run
```

调试时降低重复次数：

```bash
BENCH_NIGHTLY_COUNT=3 bash scripts/bench/nightly-gate.sh run
```

预览本地 cron 安装动作：

```bash
bash scripts/bench/install-nightly-cron.sh --dry-run
```

显式安装本地 cron：

```bash
bash scripts/bench/install-nightly-cron.sh --apply
```

本地验证：

```bash
BENCH_COUNT=3 RESULTS_ROOT=/tmp/zfeed-bench-ci bash scripts/bench/ci-gate.sh
```

带 baseline 验证：

```bash
BENCH_BASELINE=bench/results/<baseline-go-bench-dir> \
BENCH_COUNT=5 \
RESULTS_ROOT=/tmp/zfeed-bench-ci \
bash scripts/bench/ci-gate.sh
```

直接从 git ref 生成 baseline：

```bash
BENCH_BASELINE_REF=origin/main \
BENCH_COUNT=5 \
RESULTS_ROOT=/tmp/zfeed-bench-ci \
bash scripts/bench/ci-gate.sh
```

模板中的 PR 任务会用 `github.base_ref` 作为 `BENCH_BASELINE_REF`。新增 benchmark 不阻塞；已有 benchmark 缺失、`ns/op`、`B/op` 或 `allocs/op` 超阈值会失败。
