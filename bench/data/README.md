# zfeed 压测数据

此目录包含安全的压测预置数据。此目录树中的所有值都必须是合成的，并易于删除。

## 布局

```text
bench/data/
  small/
    users.csv
    tokens.json
    content_ids.csv
    follow_edges.csv
    search_terms.csv
```

k6 冒烟测试场景可以通过使用带时间戳的手机号注册，来自助创建用户和一篇文章。CSV 文件为更大规模场景提供了稳定的数据结构和后备值。

## 规则

- 绝不在 `tokens.json` 中放入生产环境令牌。
- 使用 `bench_` 前缀让压测数据明显是合成的。
- 建议在环境准备阶段从压测用户重新生成令牌。
- 将写入密集型运行隔离到本地或性能测试数据库。
