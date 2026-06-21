# zfeed 压测数据

此目录包含安全的压测预置数据。此目录树中的所有值都必须是合成的，并易于删除。

## 布局

```text
bench/data/
  small|medium|large/
    users.csv
    tokens.json
    content_ids.csv
    follow_edges.csv
    search_terms.csv
    summary.md
```

k6 冒烟测试场景会通过带时间戳的手机号注册用户，并自助创建一篇文章。CSV 文件为更大规模场景提供稳定的数据结构和后备值。

## 生成

```bash
bash scripts/bench/data.sh generate small --reset
bash scripts/bench/data.sh generate medium --append
```

也可以通过压测入口转发：

```bash
bash scripts/bench/run.sh data small --reset
```

默认生成到 `bench/data/<scale>`。如需写到临时目录，可覆盖 `DATA_ROOT`：

```bash
DATA_ROOT=/tmp/zfeed-bench-data bash scripts/bench/data.sh generate small --reset
```

这些 fixture 只写文件，不写数据库。本地 smoke 默认使用运行时注册和发布出来的真实测试 ID；当 perf 环境已经导入对应 fixture 数据时，可以设置 `BENCH_USE_FIXTURE_IDS=1` 让 k6 从 `content_ids.csv`、`users.csv` 和 `follow_edges.csv` 中采样 ID。

## 清理

数据库清理入口默认只打印 SQL，不会写数据库：

```bash
bash scripts/bench/data.sh cleanup-db --dry-run
```

真实执行需要显式确认，并只删除 `bench_` 前缀用户、内容和关联计数数据：

```bash
BENCH_DB_HOST=127.0.0.1 \
BENCH_DB_USER=zfeed \
BENCH_DB_PASSWORD=zfeed \
BENCH_DB_NAME=zfeed \
BENCH_CLEANUP_CONFIRM=delete-bench-data \
bash scripts/bench/data.sh cleanup-db run
```

## 规则

- 绝不在 `tokens.json` 中放入生产环境令牌。
- 使用 `bench_` 前缀让压测数据明显是合成的。
- 建议在环境准备阶段从压测用户重新生成令牌。
- 将写入密集型运行隔离到本地或性能测试数据库。
