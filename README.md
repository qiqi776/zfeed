# zfeed

`zfeed` 是一个基于 Go、go-zero 和 Docker Compose 的内容社区后端与交付栈。仓库当前包含 6 个后端服务边界：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`，并配套 MySQL、Redis、Kafka、Canal、etcd 和本地可观测性组件。

## 项目概览

当前包含 6 个后端服务边界：

- `front-api`：HTTP 入口，做参数校验、鉴权、聚合和下游 RPC 调用
- `user-rpc`：用户注册、登录、登出、资料查询、会话管理
- `content-rpc`：内容发布、内容详情、发布流索引、follow inbox 回填
- `interaction-rpc`：点赞、评论、收藏、关注等交互关系写链路
- `count-rpc`：计数写链、读链、批量查询和用户资料聚合
- `search-rpc`：用户 / 内容搜索，以及搜索结果的关系状态补充

## 当前能力状态

| 模块         | 状态     | 说明                                                                             |
| ------------ | -------- | -------------------------------------------------------------------------------- |
| 用户与登录态 | 可用     | 注册、登录、登出、个人信息查询，登录态保存在 Redis                               |
| 内容         | 可用     | 文章/视频发布、内容详情、用户发布流索引                                          |
| 互动         | 可用     | 点赞、评论、收藏、关注                                                           |
| 计数         | 基础可用 | 写链消费、读链回填、批量查询、用户资料聚合                                       |
| Feed         | 部分可用 | 用户发布流、收藏流、follow inbox 回填已接通，完整 follow feed 和 miss 重建仍在补 |
| 推荐/热榜    | 基础可用 | 热榜快照读取已接通，完整推荐策略仍在补                                           |
| 搜索         | 可用     | `search-rpc` 已接通，支持基础用户 / 内容搜索和 viewer 关系补充                   |
| 可观测性     | 可用     | Prometheus、结构化 DB 日志、Jaeger trace 已接通；日志采集链按需开启              |

## 技术栈

- Go `1.25.5`
- go-zero
- gRPC / Protocol Buffers
- GORM
- MySQL
- Redis
- Kafka
- Jaeger
- Canal
- etcd
- Docker Compose

## 目录结构

```text
zfeed/
├── app/
│   ├── front/                    # HTTP API / BFF
│   └── rpc/
│       ├── user/                 # 用户服务
│       ├── content/              # 内容服务
│       ├── interaction/          # 点赞/评论/收藏/关注
│       ├── count/                # 计数服务
│       └── search/               # 搜索服务
├── deploy/                       # Docker Compose、网关与观测配置
├── e2e/                          # 显式触发的全栈端到端测试
├── pkg/                          # 通用组件与工具
├── script/                       # 启停脚本、SQL bootstrap
```

## 本地开发

### 全量 Docker 模式

```bash
bash ./script/start.sh
```

停止：

```bash
bash ./script/stop.sh
```

这个入口会拉起完整 Docker 栈：

- 基础设施：`etcd`、`redis`、`mysql`、`kafka`、`canal`、`xxl-job-admin`
- 后端服务：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`
- 网关入口：`nginx`
- 默认观测：`prometheus`
- 可选观测：`otel-collector`、`jaeger`、`logstash`、`filebeat`

启动成功后默认访问：

- Gateway `/v1/*`：`http://127.0.0.1:18080`
- API 直连：`http://127.0.0.1:5000`
- Prometheus：`http://127.0.0.1:19090`
- Jaeger：`http://127.0.0.1:16686`

### 日志

后端容器日志会落到宿主机 `logs/` 目录：

- `logs/front-api`
- `logs/user-rpc`
- `logs/content-rpc`
- `logs/interaction-rpc`
- `logs/count-rpc`
- `logs/search-rpc`

开启 `ENABLE_LOG_PIPELINE=1` 后，`filebeat` 会采集这些日志并写入 `logs/collected/`。

## 测试

### 常规测试

```bash
GOCACHE=/tmp/go-build go test ./...
```

### 定向验证

```bash
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestObservabilityE2E -count=1
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestCountChainE2E -count=1
GOCACHE=/tmp/go-build go test -tags=e2e ./e2e -run TestRecommendHotSnapshotE2E -count=1
```

这些 `e2e` 测试依赖本地完整 Docker 栈已启动，并且会修改本地开发用 MySQL / Redis / Kafka 状态；默认 `go test ./...` 不会执行它们。
