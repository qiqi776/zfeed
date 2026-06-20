# zfeed 测试目录

`tests/` 只放跨包、黑盒或需要明确隔离的正确性测试。Go 单元测试和包级测试继续和源码放在一起，例如 `app/.../*_test.go`、`pkg/.../*_test.go`、`orm/*_test.go`。

## 目录约定

```text
tests/
  e2e/          Docker 栈端到端测试，使用 //go:build e2e 隔离
  integration/  跨服务或真实依赖集成测试，新增时使用 //go:build integration
  fixtures/     跨包共享测试数据
```

## 准入规则

只有下面几类测试可以进入 `tests/`：

- 跨多个服务或多个包的黑盒行为测试。
- 需要真实或半真实基础设施的测试，比如 MySQL、Redis、Kafka、Prometheus、Jaeger。
- 需要从 HTTP、RPC、消息或外部进程边界验证系统行为的测试。
- 多个包共享的测试数据、样例请求、事件 payload。

下面几类不要放进 `tests/`：

- 测单个函数、单个结构体、单个逻辑分支的单元测试。
- 需要访问未导出函数或未导出类型的同包测试。
- Go benchmark、k6 场景、ghz 配置和压测结果。
- 项目启动、停止、SQL 初始化、压测编排脚本。

## 运行方式

普通单元测试和包级集成测试：

```bash
go test ./...
```

完整 Docker 栈启动后运行 e2e：

```bash
GOCACHE=/tmp/go-build go test -tags=e2e ./tests/e2e -count=1
```

`tests/e2e` 会写入和修改本地 Docker 栈中的 MySQL、Redis、Kafka 等开发数据，只适合在当前仓库自己的本地 Docker 栈或专用 CI 环境中执行。

未来新增 integration 测试时，统一用下面的命令运行：

```bash
go test -tags=integration ./tests/integration/... -count=1
```

integration 测试默认也不进入 `go test ./...`。如果某个测试可以无副作用运行，优先放回被测包旁边，或者用 SQLite、miniredis、fake producer/client 做包级测试。

## 命名规则

- e2e 测试文件按业务链路命名，例如 `count_chain_test.go`、`recommend_hot_snapshot_test.go`。
- integration 测试文件按依赖和行为命名，例如 `mysql_like_repository_test.go`、`redis_session_test.go`。
- fixture 文件名要说明业务对象和用途，例如 `users_small.json`、`content_public_article.json`、`kafka_like_event.json`。
- 新增测试必须使用带前缀的合成数据，例如 `e2e_`、`integration_`、`bench_`，不要写真实手机号、邮箱、头像地址或 token。

## 副作用边界

- `tests/e2e` 可以写本地 Docker 栈数据，但测试必须自带唯一前缀或时间戳，便于清理。
- `tests/integration` 默认使用可控替身。必须连接真实 MySQL、Redis 或 Kafka 时，要在测试文件头部、README 或测试名中写清楚依赖和写入范围。
- 任何测试都不能默认访问远程环境、staging 环境或生产环境。
- 需要真实外部系统的测试必须单独说明运行前提，不能挂在默认命令里。

## 不放在这里的内容

- 性能压测继续放在 `bench/`，入口仍是 `scripts/bench/run.sh`。
- 项目启动、停止、SQL 初始化等脚本继续放在根目录 `scripts/`。
- 贴近源码的 Go 单元测试继续放在被测包旁边。
