# zfeed

![Go](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go&logoColor=white)
![go-zero](https://img.shields.io/badge/go--zero-service%20framework-2f6fed)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=111)
![Vite](https://img.shields.io/badge/Vite-7-646CFF?logo=vite&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker%20Compose-local%20stack-2496ED?logo=docker&logoColor=white)

`zfeed` 是一个内容社区系统，仓库内同时包含后端服务和前端应用。

后端基于 Go、go-zero 和 Docker Compose，按用户、内容、互动、计数、推荐流、搜索拆分为多个服务，并配套 MySQL、Redis、Kafka、Canal、etcd、Prometheus、Jaeger 等基础设施。前端位于 [zfeed-front](./zfeed-front)，使用 React、TypeScript、Vite 和 Tailwind CSS，覆盖注册登录、Feed 浏览、内容发布、点赞收藏评论、搜索、个人资料和管理后台。

## 快速开始

### 环境要求

- Docker 和 Docker Compose，用于启动本地后端依赖和服务
- Go 1.26.2+
- Node.js 20.19+ 或 22.12+，用于运行 Vite 7 前端
- npm，建议使用随 Node.js 安装的版本

### 启动后端

在仓库根目录执行：

```bash
bash ./scripts/start.sh
```

脚本会读取 `deploy/.env`，检查端口冲突，补齐缺失镜像，并通过 [deploy/docker-compose.yml](./deploy/docker-compose.yml) 拉起基础设施、6 个后端服务、nginx 网关和 Prometheus。

常用入口：

- HTTP 网关：`http://127.0.0.1:5000`
- nginx 网关：以 [deploy/README.md](./deploy/README.md) 和 `deploy/.env` 为准
- Prometheus：以 [deploy/prometheus/prometheus.yml](./deploy/prometheus/prometheus.yml) 配置为准

停止本地栈：

```bash
bash ./scripts/stop.sh
```

### 启动前端

前端代码在 [zfeed-front](./zfeed-front)：

```bash
cd zfeed-front
npm install --cache .npm-cache
npm run dev
```

`npm install --cache .npm-cache` 会把 npm 下载缓存放在前端项目目录内，避免写入用户全局缓存目录。`.npm-cache/` 和 `node_modules/` 都已被 `.gitignore` 忽略。

开发服务默认运行在：

```text
http://127.0.0.1:5173/
```

前端通过 Vite 代理访问后端 API，默认目标是 `http://127.0.0.1:5000`。如需覆盖接口地址，可以设置：

```bash
VITE_API_BASE_URL=http://127.0.0.1:5000 npm run dev
```

### 生产构建

开发时使用 `npm run dev`，它不会生成 `dist/`。需要部署静态资源时再执行：

```bash
cd zfeed-front
npm run build
```

构建产物会生成到 `zfeed-front/dist/`，可交给 nginx、Docker 镜像或其他静态文件服务部署。`dist/` 是构建产物，默认不提交到 git。

本地预览生产构建：

```bash
npm run preview
```

## 服务边界

| 服务              |   端口 | 职责                                              |
| ----------------- | -----: | ------------------------------------------------- |
| `front-api`       | `5000` | HTTP 入口、参数校验、鉴权、RPC 聚合               |
| `content-rpc`     | `5001` | 内容发布、详情读取、推荐流、关注流、冷热内容快照  |
| `interaction-rpc` | `5002` | 点赞、评论、收藏、关注关系及互动事件              |
| `user-rpc`        | `5003` | 注册、登录、会话、用户资料、用户统计聚合          |
| `count-rpc`       | `5004` | 高频计数写链、读链、批量查询和延迟失效            |
| `search-rpc`      | `5006` | 用户与内容搜索、Redis snapshot 稳定分页、搜索观测 |

## 主要能力

- 用户：注册、登录、登出、个人资料、头像上传、粉丝列表、关注列表、用户主页统计。
- 内容：文章和视频发布、编辑、删除、详情聚合、上传凭证。
- Feed：推荐流、关注流、用户发布列表、用户收藏列表。
- 互动：点赞、取消点赞、评论、回复、收藏、取消收藏、关注和取消关注。
- 计数：点赞数、评论数、收藏数、用户获赞和获收藏等高频计数。
- 搜索：用户搜索、内容搜索、`latest` cursor 分页、`relevance/hybrid` page token 稳定分页。
- 前端：用户端 SPA、管理后台、玻璃拟态 UI、会话恢复、API 代理和 OSS 上传。
- 观测：Prometheus 指标、结构化日志、慢查询日志、可选 OTEL + Jaeger 链路追踪。
- 压测：k6 HTTP 场景、ghz gRPC 场景、Go benchmark、结果归档。

## 前端应用

前端目录：

```text
zfeed-front/
├── src/                 React 页面、路由、运行时 API 封装和全局样式
├── e2e/                 Playwright e2e 测试
├── package.json         npm 脚本和依赖
├── vite.config.ts       Vite 配置
└── dist/                生产构建产物，执行 npm run build 后生成
```

常用脚本：

| 命令               | 说明                      |
| ------------------ | ------------------------- |
| `npm run dev`      | 启动 Vite 开发服务器      |
| `npm run build`    | TypeScript 检查并生产构建 |
| `npm run preview`  | 本地预览 `dist/`          |
| `npm test`         | 运行 Vitest 单元测试      |
| `npm run lint`     | 运行 ESLint               |
| `npm run test:e2e` | 运行 Playwright e2e 测试  |

主要路由：

| 路由           | 说明                 |
| -------------- | -------------------- |
| `/home`        | 推荐 Feed            |
| `/following`   | 关注 Feed            |
| `/me`          | 当前用户主页         |
| `/user/:id`    | 其他用户主页         |
| `/content/:id` | 内容详情、评论和互动 |
| `/compose`     | 发布文章或视频       |
| `/search`      | 搜索内容和作者       |
| `/settings`    | 账号设置             |

更完整的前端说明见 [zfeed-front/README.md](./zfeed-front/README.md)。

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

## 测试

后端单元测试和包级集成测试：

```bash
go test ./...
```

完整 Docker 栈启动后，可以运行 `tests/e2e`。这些测试会写入和修改本地开发数据，仅建议在当前仓库自己的 Docker 栈上执行：

```bash
GOCACHE=/tmp/go-build go test -tags=e2e ./tests/e2e -count=1
```

前端验证：

```bash
cd zfeed-front
npm test
npm run lint
npm run build
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

## 可观测性

服务日志默认写入 `logs/`，例如 `logs/front-api`、`logs/content-rpc`、`logs/search-rpc`。go-zero 会输出 `access.log`、`error.log`、`slow.log`、`stat.log` 等文件。开启 `ENABLE_LOG_PIPELINE=1` 后，filebeat 会采集日志并写入 `logs/collected/`。

Prometheus 配置位于 [deploy/prometheus/prometheus.yml](./deploy/prometheus/prometheus.yml)。Tracing 配置位于 [deploy/otel/otel-collector.yaml](./deploy/otel/otel-collector.yaml)，启用后可在 Jaeger UI 查看跨服务调用链。

## 项目结构

```text
app/front/          HTTP API 网关，handlers，中间件，请求/响应类型定义
app/rpc/            go-zero zrpc 服务：用户、内容、互动、计数、搜索
zfeed-front/        React + Vite 前端应用
build/              应用服务 Dockerfile
deploy/             Docker Compose 栈，环境变量，nginx，MySQL，Prometheus，OTEL
tests/              黑盒正确性测试：e2e、integration、共享 fixtures
orm/                共享 GORM 初始化及指标插件
pkg/                共享包：errorx, grpcx, hotrank, mobilex, xxljob
scripts/            启动/停止脚本，SQL 初始化脚本，压测运行器
bench/              k6, ghz, Go benchmark 测试数据与结果归档
```

## 文档

- [zfeed-front/README.md](./zfeed-front/README.md)：前端路由、脚本、OSS 上传和验证命令。
- [deploy/README.md](./deploy/README.md)：本地 Docker 栈入口、网关路由和 e2e 命令。
- [docs/ports.md](./docs/ports.md)：默认端口占用和冲突排查。
- [docs/System.md](./docs/System.md)：Timeline Feed 系统设计说明。
- [docs/benchmark/README.md](./docs/benchmark/README.md)：压测目标、场景和结果解读。
- [docs/diagrams/README.md](./docs/diagrams/README.md)：架构图维护说明。
