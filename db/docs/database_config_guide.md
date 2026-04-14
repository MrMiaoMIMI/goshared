# DatabaseConfig 配置指南

本文档介绍 `DatabaseConfig` 的各种使用场景，以 YAML 格式展示配置，并附带等价的 Go 代码。

---

## 目录

1. [非分片：单数据库](#1-非分片单数据库)
2. [非分片：DSN 连接串](#2-非分片dsn-连接串)
3. [非分片：带连接池](#3-非分片带连接池)
4. [仅表分片](#4-仅表分片)
5. [库+表分片（hash_mod）](#5-库表分片hash_mod)
6. [库分片：named（按地区）](#6-库分片named按地区)
7. [库分片：range（按范围）](#7-库分片range按范围)
8. [仅库分片（无表分片）](#8-仅库分片无表分片)
9. [entity_rules 覆盖](#9-entity_rules-覆盖)
10. [多服务器分库](#10-多服务器分库)
11. [混合配置：多库组](#11-混合配置多库组)
12. [自定义表后缀格式](#12-自定义表后缀格式)
13. [Entity 定义](#13-entity-定义)
14. [ShardingKey 使用](#14-shardingkey-使用)
15. [Go 代码使用](#15-go-代码使用)
16. [注意事项](#16-注意事项)

---

## 1. 非分片：单数据库

最简场景，一个数据库，无分库分表。

```yaml
databases:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db
```

所有未实现 `DbKey()` 的 Entity 自动路由到 `default`。

---

## 2. 非分片：DSN 连接串

使用 DSN 代替 host/port/user/password 等独立字段。DSN 优先级高于独立字段。

```yaml
databases:
  default:
    dsn: "root:secret@tcp(10.0.0.1:3306)/my_app_db?charset=utf8mb4&parseTime=True&loc=Local"
```

> **适用场景**：已有 DSN 连接串、需要传递额外连接参数（charset、parseTime 等）。

---

## 3. 非分片：带连接池

```yaml
databases:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db
    max_open_conns: 200
    max_idle_conns: 20
    conn_max_lifetime_seconds: 1800
    debug: true
```

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `max_open_conns` | 最大打开连接数 | 0（无限制） |
| `max_idle_conns` | 最大空闲连接数 | 0（无限制） |
| `conn_max_lifetime_seconds` | 连接最大生存时间（秒） | 0（不过期） |
| `debug` | 开启 GORM 调试日志 | false |

---

## 4. 仅表分片

单个数据库，表按 hash_mod 分成多张物理表。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: order_db
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

**效果**：`order_tab` → `order_tab_00000000`, `order_tab_00000001`, ..., `order_tab_00000009`

**路由**：`hash(ShardingKey["shop_id"]) % 10` → 物理表索引

---

## 5. 库+表分片（hash_mod）

单服务器上按 hash_mod 分库，每个库内再按 hash_mod 分表。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
      prefix: order_db
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

**效果**：
- 库：`order_db_0`, `order_db_1`, `order_db_2`, `order_db_3`
- 表（每个库内）：`order_tab_00000000` ~ `order_tab_00000009`
- 总共：4 库 × 10 表 = 40 个物理表

**路由**：
- 库：`hash(ShardingKey["shop_id"]) % 4` → 库索引
- 表：`hash(ShardingKey["shop_id"]) % 10` → 表索引

> **注意**：使用 `db_sharding` 时，**不要**填 `db_name`（库名由 prefix + 索引自动生成）。

---

## 6. 库分片：named（按地区）

按地区代码直接映射到对应的数据库。使用复合 ShardingKey（`region` 路由库，`shop_id` 路由表）。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: named
      key_field: region
      prefix: "order_"
      suffix: "_db"
      keys: [SG, TH, ID, MY, VN, PH]
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

**效果**：
- 库：`order_SG_db`, `order_TH_db`, `order_ID_db`, `order_MY_db`, `order_VN_db`, `order_PH_db`
- 命名规则：`{prefix}{key}{suffix}`

**路由**：`ShardingKey["region"]` 的值必须是 keys 中的某个值（如 `"SG"`），直接映射到对应库。

```go
var OrderFields = struct {
    ShopID dbspi.Column
    Region dbspi.Column
}{
    ShopID: dbhelper.NewColumn("shop_id"),
    Region: dbhelper.NewColumn("region"),
}

sk := dbspi.NewShardingKey().
    Set(OrderFields.Region, dbspi.StrVal("SG")).
    Set(OrderFields.ShopID, dbspi.IntVal(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, _ := orderExec.Find(ctx, nil, nil) // → order_SG_db, order_tab_00000005
```

---

## 7. 库分片：range（按范围）

按 sharding key 的数值范围路由到不同的库。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: range
      key_field: shop_id
      prefix: order_db
      boundaries: [10000, 20000, 50000]
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

**效果**：
- 库：`order_db_0`, `order_db_1`, `order_db_2`, `order_db_3`（boundaries 数量 + 1）
- 范围划分：

| ShardingKey["shop_id"] 范围 | 目标库 |
|-------------------|--------|
| key < 10000 | order_db_0 |
| 10000 ≤ key < 20000 | order_db_1 |
| 20000 ≤ key < 50000 | order_db_2 |
| key ≥ 50000 | order_db_3 |

---

## 8. 仅库分片（无表分片）

多个库，每个库内表名不变（不分表）。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
      prefix: order_db
```

**效果**：4 个库（`order_db_0` ~ `order_db_3`），表名保持原样（如 `order_tab`）。

> **注意**：仅库分片时，不需要在 ShardingKey 中提供 table sharding 的 key_field。

---

## 9. entity_rules 覆盖

同一库组中的不同 Entity 可以使用不同的分表规则。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
      prefix: order_db
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
    entity_rules:
      # 规则1：order_detail_tab 和 order_log_tab 使用 20 张表
      - tables: [order_detail_tab, order_log_tab]
        table_sharding:
          rule: hash_mod
          key_field: shop_id
          count: 20
      # 规则2：order_summary_tab 使用 5 张表，且限制并发
      - tables: [order_summary_tab]
        table_sharding:
          rule: hash_mod
          key_field: shop_id
          count: 5
        max_concurrency: 3
```

**效果**：

| Entity (TableName) | 库分片 | 表分片 | 总物理表数 |
|---------------------|--------|--------|-----------|
| `order_tab` | 4 库 hash_mod | 10 表 hash_mod | 4 × 10 = 40 |
| `order_item_tab` | 4 库 hash_mod | 10 表 hash_mod（继承默认） | 4 × 10 = 40 |
| `order_detail_tab` | 4 库 hash_mod | **20** 表 hash_mod（覆盖） | 4 × 20 = 80 |
| `order_log_tab` | 4 库 hash_mod | **20** 表 hash_mod（覆盖） | 4 × 20 = 80 |
| `order_summary_tab` | 4 库 hash_mod | **5** 表 hash_mod（覆盖） | 4 × 5 = 20 |

> **设计说明**：`entity_rules` 中的 `tables` 是一个列表，支持多个表共用同一条 rule，减少配置冗余。

---

## 10. 多服务器分库

数据库分布在不同的物理服务器上。

```yaml
databases:
  order_dbs:
    servers:
      - key: "0"
        host: 10.0.0.1
        port: 3306
        user: root
        password: secret
        db_name: order_db_0
      - key: "1"
        host: 10.0.0.2
        port: 3306
        user: root
        password: secret
        db_name: order_db_1
      - key: "2"
        host: 10.0.0.3
        port: 3306
        user: root
        password: secret
        db_name: order_db_2
      - key: "3"
        host: 10.0.0.4
        port: 3306
        user: root
        password: secret
        db_name: order_db_3
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

**说明**：
- 使用 `servers` 列表代替单个 `host/port` 配置
- 每个 server 的 `key` 必须与 db sharding rule 的路由结果匹配
- `servers` 与顶层的 `host/port/user/password/dsn` 互斥
- 每个 server 支持独立的 `dsn` 配置

**多服务器 + DSN 模式**：

```yaml
databases:
  order_dbs:
    servers:
      - key: "0"
        dsn: "root:secret@tcp(10.0.0.1:3306)/order_db_0?charset=utf8mb4"
      - key: "1"
        dsn: "root:secret@tcp(10.0.0.2:3306)/order_db_1?charset=utf8mb4"
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 2
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
```

---

## 11. 混合配置：多库组

一个 `DatabaseConfig` 中管理多个库组，支持分片和非分片混合。

```yaml
databases:
  # 默认库（非分片）—— 用于 User、Config 等通用表
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db

  # 订单库组（hash_mod 分库分表）
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
      prefix: order_db
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
    max_concurrency: 5
    entity_rules:
      - tables: [order_detail_tab, order_log_tab]
        table_sharding:
          rule: hash_mod
          key_field: shop_id
          count: 20

  # 支付库组（named 分库，按地区 + 复合 key）
  payment_dbs:
    host: 10.0.0.2
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: named
      key_field: region
      prefix: "payment_"
      suffix: "_db"
      keys: [SG, TH, ID]
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 8

  # 日志库组（range 分库，按时间戳）
  log_dbs:
    host: 10.0.0.3
    port: 3306
    user: root
    password: secret
    db_sharding:
      rule: range
      key_field: log_id
      prefix: log_db
      boundaries: [1000000, 2000000, 5000000]
    table_sharding:
      rule: hash_mod
      key_field: log_id
      count: 4
```

**Entity 与库组的映射**通过 `DbKey()` 实现：

```go
type User struct { ... }
func (*User) TableName() string { return "user_tab" }
// 无 DbKey() → 走 "default"

type Order struct { ... }
func (*Order) TableName() string { return "order_tab" }
func (*Order) DbKey() string    { return "order_dbs" }

type Payment struct { ... }
func (*Payment) TableName() string { return "payment_tab" }
func (*Payment) DbKey() string    { return "payment_dbs" }

type AuditLog struct { ... }
func (*AuditLog) TableName() string { return "audit_log_tab" }
func (*AuditLog) DbKey() string    { return "log_dbs" }
```

---

## 12. 自定义表后缀格式

默认表后缀格式为 `_%08d`（8 位零填充），可通过 `format` 自定义。

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: order_db
    table_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 10
      format: "_%02d"     # 2 位后缀
```

**格式对比**：

| format | 物理表名示例 |
|--------|-------------|
| `_%08d`（默认） | `order_tab_00000000`, `order_tab_00000001` |
| `_%02d` | `order_tab_00`, `order_tab_01` |
| `_%d` | `order_tab_0`, `order_tab_1` |
| `_%04d` | `order_tab_0000`, `order_tab_0001` |

---

## 13. Entity 定义

### 必须实现的接口

```go
// Entity 接口 —— 所有表实体必须实现
type Entity interface {
    TableName() string
}
```

### 可选接口

```go
// DbKeyProvider —— 声明所属库组（不实现则走 "default"）
type DbKeyProvider interface {
    DbKey() string
}

// Ider —— 支持 xxById 方法（GetById、UpdateById 等）
type Ider interface {
    IdFiledName() string
}
```

### 完整 Entity 示例

```go
type Order struct {
    ID      int64  `gorm:"primaryKey"`
    ShopID  int64  `gorm:"column:shop_id"`
    Amount  int64  `gorm:"column:amount"`
}

func (*Order) TableName() string  { return "order_tab" }
func (*Order) DbKey() string      { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }
```

---

## 14. ShardingKey 使用

### 基本概念

`ShardingKey` 是一个复合分片键，将 DB 列名映射到 `ShardingValue`。配置中的 `key_field` 指定从 `ShardingKey` 中提取哪个列的值进行路由。

### ShardingValue 类型

`ShardingValue` 支持三种基础类型：

```go
dbspi.IntVal(12345)        // int64 值 → 用于 hash_mod、range 规则
dbspi.UintVal(12345)       // uint64 值 → 用于 hash_mod 规则
dbspi.StrVal("SG")         // string 值 → 用于 named/direct 规则
```

类型转换规则：
- `ToUint64()`: int64 → uint64 直接转换，string → FNV-1a hash
- `ToInt64()`: uint64 → int64 直接转换，string → 报错
- `String()`: int64/uint64 → 十进制字符串，string → 原值

### 定义列引用

使用 `dbhelper.NewColumn` 创建类型安全的列引用：

```go
var OrderFields = struct {
    ShopID dbspi.Column
    Region dbspi.Column
}{
    ShopID: dbhelper.NewColumn("shop_id"),
    Region: dbhelper.NewColumn("region"),
}
```

### 单字段分片（db 和 table 用同一个字段）

```go
// 配置：db_sharding.key_field = "shop_id", table_sharding.key_field = "shop_id"
sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, err := orderExec.Find(ctx, nil, nil)
```

### 复合分片键（db 和 table 用不同字段）

```go
// 配置：db_sharding.key_field = "region", table_sharding.key_field = "shop_id"
sk := dbspi.NewShardingKey().
    Set(OrderFields.Region, dbspi.StrVal("SG")).
    Set(OrderFields.ShopID, dbspi.IntVal(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, err := orderExec.Find(ctx, nil, nil)
// region="SG" → 路由到 order_SG_db
// shop_id=12345 → 路由到 order_tab_00000005
```

### 使用 Shard() 方法直接路由

```go
sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
shardedExec, err := orderExec.Shard(sk)
if err != nil {
    return err
}
orders, err := shardedExec.Find(ctx, nil, nil)
```

### Scatter-gather（全分片查询）

`FindAll` 和 `CountAll` 不需要 ShardingKey，自动遍历所有分片：

```go
ctx := context.Background()
allOrders, err := orderExec.FindAll(ctx, query, 100)   // 批量拉取
totalCount, err := orderExec.CountAll(ctx, query)       // 总计数
```

> **注意**：Scatter-gather 仅支持实现了 `ShardCounter` 接口的 table rule（如 `hash_mod`）。
> 自定义 table rule 如需支持 scatter-gather，需同时实现 `ShardCounter` 接口。

---

## 15. Go 代码使用

### 初始化 DbManager

```go
import "github.com/MrMiaoMIMI/goshared/db/dbhelper"

// 方式 1：从 Go struct 构建
mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
    Databases: map[string]dbhelper.DatabaseEntry{
        "default": {
            Host: "10.0.0.1", Port: 3306, User: "root", Password: "secret",
            DbName: "my_app_db",
        },
        "order_dbs": {
            Host: "10.0.0.1", Port: 3306, User: "root", Password: "secret",
            DbSharding:    &dbhelper.DbShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 4, Prefix: "order_db"},
            TableSharding: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
        },
    },
})

// 方式 2：从 YAML 文件加载（自行解析后传入）
// var cfg dbhelper.DatabaseConfig
// yaml.Unmarshal(data, &cfg)
// mgr := dbhelper.NewDbManager(cfg)
```

### 设置全局默认

```go
dbhelper.SetDefault(mgr)
```

### 获取 Executor

```go
// 使用全局默认 manager（推荐）
userExec := dbhelper.For(&User{})
orderExec := dbhelper.For(&Order{})

// 使用指定 manager
orderExec := dbhelper.For(&Order{}, mgr)
```

### 使用 Executor

```go
ctx := context.Background()

// 非分片 Entity —— 直接使用
users, err := userExec.Find(ctx, nil, nil)

// 分片 Entity —— 需要设置 ShardingKey
sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(shopID))
ctx = dbspi.WithShardingKey(ctx, sk)
orders, err := orderExec.Find(ctx, nil, nil)

// 全分片查询（scatter-gather）—— 无需 ShardingKey
allOrders, err := orderExec.FindAll(ctx, query, 100)
totalCount, err := orderExec.CountAll(ctx, query)
```

---

## 16. 注意事项

### key_field 是必填项

所有 `db_sharding` 和 `table_sharding` 配置都**必须**指定 `key_field`，它是 DB 列名（而非 Go 结构体字段名），用于从 `ShardingKey` 中提取对应的分片值。

```yaml
# ✅ 正确
table_sharding:
  rule: hash_mod
  key_field: shop_id    # DB 列名
  count: 10

# ❌ 错误 —— 缺少 key_field，运行时会报错
table_sharding:
  rule: hash_mod
  count: 10
```

### 复合分片键场景

当 `db_sharding.key_field` 和 `table_sharding.key_field` 不同时，`ShardingKey` 中必须同时包含两个字段的值：

```yaml
db_sharding:
  rule: named
  key_field: region        # 从 ShardingKey 中取 "region" 的值路由库
  keys: [SG, TH, ID]
table_sharding:
  rule: hash_mod
  key_field: shop_id       # 从 ShardingKey 中取 "shop_id" 的值路由表
  count: 10
```

```go
sk := dbspi.NewShardingKey().
    Set(OrderFields.Region, dbspi.StrVal("SG")).     // region 对应 db 路由
    Set(OrderFields.ShopID, dbspi.IntVal(12345))      // shop_id 对应 table 路由
```

如果漏设某个字段，运行时 `Shard()` 或 CRUD 方法会返回 `sharding key field "xxx" not found` 错误。

### DSN 与 db_sharding 不兼容

单服务器场景下，`dsn` 不能与 `db_sharding` 同时使用。因为 DSN 中已包含数据库名，无法动态切换到其他库。

```yaml
# ❌ 错误配置 —— 会 panic
databases:
  order_dbs:
    dsn: "root:secret@tcp(10.0.0.1:3306)/order_db"
    db_sharding:
      rule: hash_mod
      key_field: shop_id
      count: 4
      prefix: order_db
```

**解决方案**：
- 使用 `host/port/user/password` 独立字段
- 或使用 `servers` 列表，每个 server 单独指定 DSN

### Server 与 Servers 互斥

`host/port/user/password`（或 `dsn`）与 `servers` 列表二选一。设置 `servers` 后，顶层连接字段被忽略。

### DbKey 回退机制

Entity 的 `DbKey()` 返回的 key 如果在配置中不存在，会自动回退到 `"default"` 库组。如果 `"default"` 也不存在，则 panic。

### db_sharding 时不要填 db_name

使用 `db_sharding` 时，库名由 `prefix` + 规则自动生成。此时 `db_name` 字段无意义，应省略。

### max_concurrency

`max_concurrency` 控制 `FindAll`/`CountAll` scatter-gather 操作的最大并发 goroutine 数。

- 值为 0 或不设置 → 无限制（所有分片并行查询）
- 推荐对大分片数场景设置合理值，避免数据库连接风暴

### entity_rules 匹配逻辑

`entity_rules` 根据 Entity 的 `TableName()` 返回值进行匹配。确保 `tables` 列表中的名称与 `TableName()` 完全一致。

### Scatter-gather 与自定义 table rule

`FindAll`/`CountAll` 依赖 table rule 实现 `ShardCounter` 接口来枚举所有物理表。内置的 `hash_mod` rule 已实现此接口。自定义 table rule 如不实现 `ShardCounter`，scatter-gather 将仅查询逻辑表名（不分表遍历）。

---

## 字段速查表

### DatabaseEntry

| 字段 | 类型 | 说明 | 与其他字段关系 |
|------|------|------|---------------|
| `dsn` | string | DSN 连接串 | 优先于 host/port/user/password |
| `host` | string | 数据库主机 | - |
| `port` | uint | 端口号 | - |
| `user` | string | 用户名 | - |
| `password` | string | 密码 | - |
| `db_name` | string | 库名（非分片时） | db_sharding 存在时忽略 |
| `debug` | bool | GORM 调试日志 | - |
| `max_open_conns` | int | 最大打开连接数 | - |
| `max_idle_conns` | int | 最大空闲连接数 | - |
| `conn_max_lifetime_seconds` | int | 连接最大生存时间 | - |
| `db_sharding` | DbShardConfig | 库级分片 | 与 dsn 单服务器不兼容 |
| `table_sharding` | TableShardConfig | 默认表级分片 | 可被 entity_rules 覆盖 |
| `entity_rules` | []EntityRule | 分表覆盖规则 | - |
| `servers` | []NamedServerConfig | 多服务器 | 与 host/port/dsn 互斥 |
| `max_concurrency` | int | scatter-gather 并发限制 | 可被 entity_rules 覆盖 |

### DbShardConfig

| 字段 | 类型 | 适用 rule | 说明 |
|------|------|----------|------|
| `rule` | string | 所有 | hash_mod / named / range |
| `key_field` | string | 所有 | **必填**，DB 列名 |
| `count` | int | hash_mod | 分库数量 |
| `prefix` | string | 所有 | 库名前缀 |
| `suffix` | string | named | 库名后缀 |
| `keys` | []string | named | 路由键列表 |
| `boundaries` | []int64 | range | 范围边界 |

### TableShardConfig

| 字段 | 类型 | 说明 |
|------|------|------|
| `rule` | string | 目前仅支持 `hash_mod` |
| `key_field` | string | **必填**，DB 列名 |
| `count` | int | 分表数量 |
| `format` | string | 后缀格式（默认 `_%08d`） |
