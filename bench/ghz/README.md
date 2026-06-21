# ghz 压测配置

运行所有配置。`run.sh ghz` 会先读取当前 `DATA_DIR`，通过 `bench/tools/benchghzgen` 生成一份 fixture-aware 临时配置，再调用 ghz：

```bash
bash ./scripts/bench/run.sh ghz
```

默认目标：

- `feed-*`：`127.0.0.1:5001`
- `content-*`：`127.0.0.1:5001`
- `like-*`：`127.0.0.1:5002`
- `comment-*`、`follow-*`：`127.0.0.1:5002`
- `user-*`：`127.0.0.1:5003`
- `count-*`：`127.0.0.1:5004`
- `search-*`：`127.0.0.1:5006`

通过以下方式覆盖目标：

```bash
GHZ_CONTENT_TARGET=127.0.0.1:5001 \
GHZ_INTERACTION_TARGET=127.0.0.1:5002 \
GHZ_USER_TARGET=127.0.0.1:5003 \
GHZ_COUNT_TARGET=127.0.0.1:5004 \
GHZ_SEARCH_TARGET=127.0.0.1:5006 \
bash ./scripts/bench/run.sh ghz
```

只生成配置、不执行 ghz：

```bash
bash ./scripts/bench/run.sh ghz-config /tmp/zfeed-ghz-configs
```

源配置中的 `data` 字段只保留合成默认值。正式运行时以 `DATA_DIR` 下的 `users.csv`、`content_ids.csv`、`follow_edges.csv`、`search_terms.csv` 为准。
