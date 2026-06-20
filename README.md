# zfeed

![Go](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go&logoColor=white)
![go-zero](https://img.shields.io/badge/go--zero-service%20framework-2f6fed)
![Docker Compose](https://img.shields.io/badge/Docker%20Compose-local%20stack-2496ED?logo=docker&logoColor=white)

`zfeed` 是一个基于 Go、go-zero 和 Docker Compose 的内容社区系统。它将用户、内容、互动、计数、推荐流和搜索拆分为独立服务，并配套 MySQL、Redis、Kafka、Canal、etcd、Prometheus、Jaeger 等基础设施。

## 快速开始

### 启动本地环境

```bash
bash ./scripts/start.sh
```

脚本会读取 `deploy/.env`，自动检查端口冲突，补齐缺失镜像，并通过 `deploy/docker-compose.yml` 拉起基础设施、6 个后端服务、nginx 网关和 Prometheus

## 功能

### 服务边界

| 服务              |   端口 | 职责                                              |
| ----------------- | -----: | ------------------------------------------------- |
| `front-api`       | `5000` | HTTP 入口、参数校验、鉴权、RPC 聚合               |
| `user-rpc`        | `5003` | 注册、登录、会话、用户资料、用户统计聚合          |
| `content-rpc`     | `5001` | 内容发布、详情读取、推荐流、关注流、冷热内容快照  |
| `interaction-rpc` | `5002` | 点赞、评论、收藏、关注关系及互动事件              |
| `count-rpc`       | `5004` | 高频计数写链、读链、批量查询和延迟失效            |
| `search-rpc`      | `5006` | 用户与内容搜索、Redis snapshot 稳定分页、搜索观测 |

### 主要能力

- 用户：注册、登录、登出、个人资料、头像上传、粉丝列表、用户主页统计。
- 内容：文章和视频发布、编辑、删除、详情聚合、上传凭证。
- Feed：推荐流、关注流、用户发布列表、用户收藏列表。
- 互动：点赞、取消点赞、评论、回复、收藏、关注关系查询。
- 计数：点赞数、评论数、收藏数、用户获赞和获收藏等高频计数。
- 搜索：用户搜索、内容搜索、`latest` cursor 分页、`relevance/hybrid` page token 稳定分页。
- 观测：Prometheus 指标、结构化日志、慢查询日志、可选 OTEL + Jaeger 链路追踪。
- 压测：k6 HTTP 场景、ghz gRPC 场景、Go benchmark、结果归档。

## API

HTTP API 定义在 [app/front/doc/front.api](./app/front/doc/front.api)，按业务拆分在 `app/front/doc/*/*.api`。当前主要入口如下：

| 分组 | 路径前缀          | 示例                                                             |
| ---- | ----------------- | ---------------------------------------------------------------- |
| 用户 | `/v1`             | `POST /v1/users`、`POST /v1/login`、`GET /v1/users/me`           |
| 内容 | `/v1/content`     | `POST /v1/content/article/publish`、`POST /v1/content/detail`    |
| Feed | `/v1/feed`        | `POST /v1/feed/recommend`、`POST /v1/feed/follow`                |
| 互动 | `/v1/interaction` | `POST /v1/interaction/like`、`POST /v1/interaction/comment/list` |
| 搜索 | `/v1/search`      | `POST /v1/search/users`、`POST /v1/search/contents`              |

RPC 协议定义在各服务的 `app/rpc/*/proto/*.proto`，生成代码位于对应服务目录下的 `client/`、`*service/` 和 protobuf 包。

## 可观测性

服务日志默认写入 `logs/`，例如 `logs/front-api`、`logs/content-rpc`、`logs/search-rpc`。go-zero 会输出 `access.log`、`error.log`、`slow.log`、`stat.log` 等文件。开启 `ENABLE_LOG_PIPELINE=1` 后，filebeat 会采集日志并写入 `logs/collected/`。

Prometheus 配置位于 [deploy/prometheus/prometheus.yml](./deploy/prometheus/prometheus.yml)。Tracing 配置位于 [deploy/otel/otel-collector.yaml](./deploy/otel/otel-collector.yaml)，启用后可在 Jaeger UI 查看跨服务调用链。

## 测试

单元测试和包级集成测试可以直接运行：

```bash
go test ./...
```

完整 Docker 栈启动后，可以运行 `tests/e2e`。注意这些测试会写入和修改本地开发数据，仅建议在当前仓库自己的 Docker 栈上执行：

```bash
GOCACHE=/tmp/go-build go test -tags=e2e ./tests/e2e -count=1
```

## 性能测试

压测入口统一在 [scripts/bench/run.sh](./scripts/bench/run.sh)：

```bash
bash ./scripts/bench/run.sh ports
bash ./scripts/bench/run.sh start-stack
bash ./scripts/bench/run.sh smoke
bash ./scripts/bench/run.sh search
bash ./scripts/bench/run.sh go-bench
bash ./scripts/bench/run.sh ghz
```

测试数据在 [bench/data](./bench/data)，k6 场景在 [bench/k6](./bench/k6)，ghz 配置在 [bench/ghz](./bench/ghz)，结果默认归档到 `bench/results/`。

## 项目结构

```text
app/front/          HTTP API 网关，handlers，中间件，请求/响应类型定义
app/rpc/            go-zero zrpc 服务：用户、内容、互动、计数、搜索
build/              应用服务 Dockerfile
deploy/             Docker Compose 栈，环境变量，nginx，MySQL，Prometheus，OTEL
docs/               架构说明，端口清单，绘图辅助
tests/              黑盒正确性测试：e2e、integration、共享 fixtures
orm/                共享 GORM 初始化及指标插件
pkg/                共享包：errorx, grpcx, hotrank, mobilex, xxljob
scripts/            启动/停止脚本，SQL 初始化脚本，压测运行器
bench/              k6, ghz, Go benchmark 测试数据与结果归档
```

## 文档

- [deploy/README.md](./deploy/README.md)：本地 Docker 栈入口、网关路由和 e2e 命令。
- [docs/ports.md](./docs/ports.md)：默认端口占用和冲突排查。
- [docs/System.md](./docs/System.md)：Timeline Feed 系统设计说明。
- [docs/benchmark/README.md](./docs/benchmark/README.md)：压测目标、场景和结果解读。
- [docs/diagrams/README.md](./docs/diagrams/README.md)：架构图维护说明。
