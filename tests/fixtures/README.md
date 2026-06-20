# fixtures 规则

`tests/fixtures/` 放跨包共享的测试数据。只被单个包使用的数据，应放在该包旁边的 `testdata/`。

## 目录建议

按业务对象或外部边界分组：

```text
tests/fixtures/
  users/
  content/
  interaction/
  count/
  search/
  kafka/
  http/
```

没有实际文件前，不需要提前创建这些目录。

## 命名规则

文件名使用小写和下划线，包含对象和用途：

```text
users_small.json
content_public_article.json
kafka_like_event.json
http_login_success.json
search_terms_small.csv
```

数据规模写在文件名里，比如 `small`、`medium`、`large`。默认只提交 `small` 级别数据，大型数据生成脚本或压测数据继续放在 `bench/data/`。

## 数据规则

- 只提交合成数据，不能提交真实用户数据。
- 手机号、邮箱、头像、昵称、token 都要使用明显的测试值。
- 时间字段优先使用固定时间，避免测试随日期变化。
- ID 使用独立号段，避免和本地开发数据混在一起。
- JSON、CSV、YAML 文件要保持稳定排序，减少无意义 diff。

## 使用规则

- 测试读取 fixture 时用相对当前包或仓库根目录的明确路径。
- 测试需要修改 fixture 内容时，先复制到 `t.TempDir()`，不要直接写回仓库文件。
- fixture 变更必须说明会影响哪些测试。
- 如果 fixture 只为 e2e 或 integration 服务，文件名或目录名要带出业务场景。
