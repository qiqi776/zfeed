# ghz 压测配置

运行所有配置：

```bash
bash ./scripts/bench/run.sh ghz
```

默认目标：

- `feed-*`：`127.0.0.1:5001`
- `like-*`：`127.0.0.1:5002`
- `search-*`：`127.0.0.1:5006`

通过以下方式覆盖目标：

```bash
GHZ_CONTENT_TARGET=127.0.0.1:5001 \
GHZ_INTERACTION_TARGET=127.0.0.1:5002 \
GHZ_SEARCH_TARGET=127.0.0.1:5006 \
bash ./scripts/bench/run.sh ghz
```

数据值是合成的默认值。在将结果作为容量证据之前，请将 ID 替换为当前压测数据集中的预置数据 ID。
