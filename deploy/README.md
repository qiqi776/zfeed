# deploy 目录说明

## 入口

- 业务入口网关 `/v1/*`：`http://127.0.0.1:18080`
- 直连 API：`http://127.0.0.1:5000`
- Prometheus：`http://127.0.0.1:19090`
- Grafana：`http://127.0.0.1:13000`
- Jaeger：`http://127.0.0.1:16686`

## 一键启动

仓库根目录执行：

```bash
bash ./script/start.sh
```

该脚本会通过 `deploy/docker-compose.yml` 拉起：

- 基础设施：`etcd`、`redis`、`mysql`、`kafka`、`canal`、`xxl-job-admin`
- 后端服务：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`
- 网关入口：`nginx`
- 观测组件：`prometheus`
- 可选链路：`jaeger`、`otel-collector`、`logstash`、`filebeat`

停止：

```bash
bash ./script/stop.sh
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
