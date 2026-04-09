# zfeed

`zfeed` 是一个基于 Go 和 go-zero 的内容社区后端项目，目标是把用户、内容、互动、Feed 和计数拆成独立服务，并围绕 MySQL、Redis、Kafka、Canal 逐步补齐高并发场景下的主链路设计。

当前仓库已经具备一套可本地启动的微服务基础设施，具备用户登录、内容发布、点赞/评论、收藏、关注等核心写路径。Feed 读路径、计数服务、推荐和热榜仍在持续完善中。

## 项目概览

`zfeed` 当前包含 5 个服务边界：

- `front-api`：HTTP 入口，做参数校验、鉴权、聚合和下游 RPC 调用
- `user-rpc`：用户注册、登录、登出、资料查询、会话管理
- `content-rpc`：内容发布、内容详情、发布流索引、follow inbox 回填
- `interaction-rpc`：点赞、评论、收藏、关注等交互关系写链路
- `count-rpc`：计数服务骨架，读写逻辑仍在补齐

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

| 模块         | 状态       | 说明                                                       |
| ------------ | ---------- | ---------------------------------------------------------- |
| 用户与登录态 | 可用       | 支持注册、登录、登出、个人信息查询，登录态保存在 Redis     |
| 内容发布     | 可用       | 支持文章/视频发布，发布成功后维护用户发布流 ZSet           |
| 内容详情     | 基础可用   | 支持详情查询，互动状态聚合仍会随着 count/feed 能力继续增强 |
| 点赞         | 已接入主链 | 已有接口、缓存和 Kafka 生产链路                            |
| 评论         | 基础可用   | 已支持评论、删除、评论列表与回复列表                       |
| 收藏         | 可用       | 已实现关系写库、关系缓存失效、用户收藏列表缓存增量维护     |
| 关注         | 可用       | 已实现关系写库和 follow 后回填 inbox                       |
| 关注流读取   | 进行中     | inbox 回填已具备，完整读取与 miss 重建仍在完善             |
| 用户收藏流   | 进行中     | 收藏回源能力已补，完整 Feed 拼装仍在完善                   |
| 计数服务     | 骨架       | proto、server、logic 已有，核心读写逻辑待补齐              |
| 推荐/热榜    | 规划中     | API 契约和目录已预留，主实现尚未完成                       |

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
./script/start.sh
```

停止：

```bash
./script/stop.sh
```

### 日志

本地启动后，日志默认会输出到 `logs/` 目录下的服务日志文件。

## 测试

### 常规测试

```bash
GOCACHE=/tmp/go-build go test ./...
```

### 收藏 / 关注真实存储验证

项目包含一条面向真实 MySQL 和 Redis 的定向验证测试，用来确认收藏和关注链路的 DB / Redis 证据链：

```bash
RUN_REAL_STORE=1 GOCACHE=/tmp/go-build go test -run TestRealStoreFavoriteAndFollowFlow -v ./app/rpc/interaction/internal/logic/favoriteservice
```

## 当前限制

作为一个正在持续补齐的项目，当前有几处边界需要明确：

- `count-rpc` 还没有完成真正的计数读写逻辑
- `front-api` 默认会依赖 `count-rpc` client，但本地默认栈并不启动 `count-rpc`
- `follow` 的 inbox 回填已完成，但完整 follow feed 读路径和 miss 重建仍未做完
- `user favorite feed` 的完整读取与详情拼装仍未做完
- 推荐流、热榜、XXL-JOB、Canal 计数主链路仍在后续阶段

## 开发方向

接下来的主要工作会集中在：

- 计数服务主链和读链
- 关注流读取与缓存重建
- 用户发布流 / 收藏流读取
- 推荐流和热榜快照
- 定时任务与最终一致性修正
