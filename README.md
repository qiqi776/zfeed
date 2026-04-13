# zfeed

`zfeed` 是一个基于 Go、go-zero 和 Docker Compose 的内容社区后端。仓库当前包含 5 个服务边界：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`，并配套 MySQL、Redis、Kafka、Canal、etcd 和本地可观测性组件。

## 项目概览

当前包含 5 个服务边界：

- `front-api`：HTTP 入口，做参数校验、鉴权、聚合和下游 RPC 调用
- `user-rpc`：用户注册、登录、登出、资料查询、会话管理
- `content-rpc`：内容发布、内容详情、发布流索引、follow inbox 回填
- `interaction-rpc`：点赞、评论、收藏、关注等交互关系写链路
- `count-rpc`：计数写链、读链、批量查询和用户资料聚合

典型调用关系如下：

```text
Client
  |
  v
front-api
  |-- user-rpc
  |-- content-rpc
  |-- interaction-rpc
  `-- count-rpc

Infra:
  - MySQL
  - Redis
  - Kafka
  - Canal
  - etcd
```

## 当前能力状态

| 模块         | 状态     | 说明                                                                             |
| ------------ | -------- | -------------------------------------------------------------------------------- |
| 用户与登录态 | 可用     | 注册、登录、登出、个人信息查询，登录态保存在 Redis                               |
| 内容         | 可用     | 文章/视频发布、内容详情、用户发布流索引                                          |
| 互动         | 可用     | 点赞、评论、收藏、关注                                                           |
| 计数         | 基础可用 | 写链消费、读链回填、批量查询、用户资料聚合                                       |
| Feed         | 部分可用 | 用户发布流、收藏流、follow inbox 回填已接通，完整 follow feed 和 miss 重建仍在补 |
| 推荐/热榜    | 基础可用 | 热榜快照读取已接通，完整推荐策略仍在补                                           |
| 可观测性     | 可用     | Prometheus、结构化 DB 日志、Jaeger trace 已接通；日志采集链按需开启              |

## 技术栈

- Go `1.25.5`
- go-zero
- gRPC / Protocol Buffers
- GORM
- MySQL
- Redis
- Kafka
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
│       └── count/                # 计数服务
├── deploy/                       # Docker Compose 与基础设施配置
├── pkg/                          # 通用组件与工具
└── script/                       # 启停脚本、SQL bootstrap、辅助脚本
```

## 本地开发

### 一键启动

```bash
bash ./script/start.sh
```

停止：

```bash
bash ./script/stop.sh
```

启动成功后默认访问：

- API：
  `http://127.0.0.1:5000`
- Prometheus：
  `http://127.0.0.1:19090`
- Jaeger：
  `http://127.0.0.1:16686`

### 日志

后端容器日志会落到宿主机 `logs/` 目录：

- `logs/front-api`
- `logs/user-rpc`
- `logs/content-rpc`
- `logs/interaction-rpc`
- `logs/count-rpc`

开启 `ENABLE_LOG_PIPELINE=1` 后，`filebeat` 会采集这些日志并写入 `logs/collected/`。

## 测试

### 常规测试

```bash
GOCACHE=/tmp/go-build go test ./...
```

### 定向验证

```bash
bash ./script/test_observability.sh
bash ./script/test_count_chain.sh
bash ./script/test_count_read_path.sh
bash ./script/test_recommend_hot_snapshot.sh
```

## 当前限制

- `follow` 完整读链和 miss 重建还没有做完
- `user favorite feed` 的详情拼装还没有完全补齐
- 推荐流当前以快照读取为主，完整排序策略还在补
- 日志采集链默认关闭，需要手动打开 `ENABLE_LOG_PIPELINE=1`
