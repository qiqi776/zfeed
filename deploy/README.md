# deploy 目录说明

## 入口

- 业务入口网关 `/v1/*`：`http://127.0.0.1:18080`
- 直连 API：`http://127.0.0.1:5000`
- Prometheus：`http://127.0.0.1:19090`
- Grafana：`http://127.0.0.1:13000`
- Jaeger：`http://127.0.0.1:16686`
- OpenSearch：`http://127.0.0.1:19200`

## 一键启动

仓库根目录执行：

```bash
bash ./scripts/start.sh
```

该脚本会通过 `deploy/docker-compose.yml` 拉起：

- 基础设施：`etcd`、`redis`、`mysql`、`kafka`、`canal`、`xxl-job-admin`
- 后端服务：`front-api`、`admin-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`
- 网关入口：`nginx`
- 观测组件：`prometheus`
- 搜索索引：`opensearch`、`search-index-bootstrap`、`search-indexer`
- 可选链路：`jaeger`、`otel-collector`、`logstash`、`filebeat`

停止：

```bash
bash ./scripts/stop.sh
```

## E2E 验证

完整栈启动后，可以显式执行 `e2e` 测试：

```bash
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestObservabilityE2E -count=1
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestCountChainE2E -count=1
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestRecommendHotSnapshotE2E -count=1
```

这些测试会修改本地开发数据，只适合在当前仓库自己的 Docker 栈上执行。

## 网关路由

- `/v1/admin/*` -> `admin-api:5001`（宿主机端口 `${ADMIN_API_PORT}`，默认 5011）
- `/v1/*` -> `front-api:5000`
- `/` -> `404`

## 搜索索引

默认仍保持 `SEARCH_BACKEND=mysql`、`SEARCH_ENGINE_TRAFFIC_PERCENT=0`、`SEARCH_INDEX_ENGINE_TYPE=noop`，不会把在线搜索流量切到 OpenSearch，也不会默认写入索引。需要重建/校验时显式覆盖 `SEARCH_INDEX_ENGINE_TYPE=opensearch`。

本地栈会启动 OpenSearch，并通过 `search-index-bootstrap` 创建版本索引和 alias：

- `zfeed_content -> zfeed_content_v1`
- `zfeed_user -> zfeed_user_v1`

全量重建示例：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer rebuild -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -entity all -batch-size 200
```

校验示例：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer verify -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -entity all -top-queries "增长,科技"
```

校验通过后切 alias 示例：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer switch-alias -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -entity all -content-index zfeed_content_v2 -user-index zfeed_user_v2 \
  -content-alias zfeed_content -user-alias zfeed_user
```

## 搜索引擎灰度演示

搜索链路默认处于安全态：

- `SEARCH_BACKEND=mysql`
- `SEARCH_ENGINE_TRAFFIC_PERCENT=0`
- `SEARCH_INDEX_ENGINE_TYPE=noop`

演示目标是完成一次“重建索引 -> 校验 -> 影子比对 -> 小流量灰度 -> 回滚 MySQL”的闭环。客户端 API 不需要调整，仍只感知 `mode`、`page_token`、`snapshot_id`。

1. 启动本地栈：

```bash
bash ./scripts/start.sh
```

2. 重建 OpenSearch 索引：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer rebuild -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -entity all -batch-size 200
```

3. 校验索引质量：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer verify -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -entity all -top-queries "增长,科技"
```

4. 开启影子比对，不承接真实结果：

```bash
SEARCH_BACKEND=engine
SEARCH_ENGINE_TRAFFIC_PERCENT=0
SEARCH_INDEX_COMPARE_ENABLED=true
```

此时主结果仍来自 MySQL，OpenSearch 只做 shadow 查询。重点观察 `zfeed_search_compare_overlap_ratio`、`zfeed_search_engine_fallback_total`、搜索耗时和空结果率。

5. 小流量灰度：

```bash
SEARCH_BACKEND=engine
SEARCH_ENGINE_TRAFFIC_PERCENT=1
SEARCH_INDEX_COMPARE_ENABLED=true
```

观察稳定后再按 `1% -> 10% -> 50% -> 100%` 提升流量。每一档至少看错误率、p95/p99、fallback、empty result、topN overlap。

6. 回滚 MySQL：

```bash
SEARCH_BACKEND=mysql
SEARCH_ENGINE_TRAFFIC_PERCENT=0
```

7. 重放索引失败事件：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm search-indexer \
  env SEARCH_INDEX_ENGINE_TYPE=opensearch \
  /app/bin/search-indexer replay-failed -f /app/app/rpc/search/search-indexer/etc/search-indexer.yaml \
  -file /var/log/zfeed/search-indexer-failures.jsonl -limit 100
```

手动验收清单：

- 同一关键词连续翻 5 页，结果不重复、不丢失。
- OpenSearch 故障时在线搜索可 fallback 到 MySQL。
- `verify` 失败时不执行 `switch-alias`。
- 模拟 OpenSearch 500 后，`/var/log/zfeed/search-indexer-failures.jsonl` 有失败事件，`replay-failed` 可以重放。
