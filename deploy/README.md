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
- 后端服务：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`
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
