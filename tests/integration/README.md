# integration 测试规则

`tests/integration/` 放跨服务、跨依赖或真实基础设施边界的测试。这里的测试必须显式使用 build tag，不允许被 `go test ./...` 默认跑到。

## 文件规则

每个 Go 测试文件顶部都要写：

```go
//go:build integration
```

包名按测试范围命名。测试只验证公开行为和边界契约，不访问业务包里的未导出函数。需要测未导出逻辑时，把测试放回源码同目录。

推荐按依赖或业务边界建子目录：

```text
tests/integration/
  mysql/
  redis/
  kafka/
  search/
```

目录不够用时再新增，不提前拆空目录。

## 运行方式

```bash
go test -tags=integration ./tests/integration/... -count=1
```

如果测试需要本地 Docker 栈，文档和测试名要写清楚。例如：

```text
TestMySQLLikeRepositoryIntegration
TestKafkaCountEventIntegration
```

## 依赖选择

优先级从高到低：

1. SQLite、miniredis、fake producer/client。
2. 本地 Docker 栈中的 MySQL、Redis、Kafka。
3. 专用 CI service 容器。

禁止默认连接远程环境、staging 环境或生产环境。测试需要真实基础设施时，必须从 `deploy/.env`、`.env.local` 或当前进程环境读取地址，不能把私有地址写死在测试里。

## 数据规则

- 测试数据必须使用 `integration_` 前缀或唯一时间戳。
- 测试创建的数据要能清理，或能安全重复写入。
- 不提交真实手机号、邮箱、头像 URL、token、cookie、trace id。
- 大型 fixture 放到 `tests/fixtures/`，小型请求体可以直接写在测试文件里。

## 何时不要写 integration 测试

- 只验证纯函数、参数校验、错误映射时，写普通单元测试。
- 只验证单个包里的未导出 helper 时，写同包测试。
- 只验证性能指标时，放到 `bench/` 或包内 benchmark。
- 需要完整用户旅程、Prometheus、Jaeger、Kafka 链路时，放到 `tests/e2e/`。
