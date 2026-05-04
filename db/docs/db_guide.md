# DB Module Guide

本指南介绍 `db` 模块的用户侧用法。`db` 模块以 `DatabaseConfig -> Manager -> TableStore` 为主线，提供普通 CRUD、软删除、公共字段自动填充、事务、Raw SQL 逃生口，以及分库分表能力。

核心流程：

1. 定义 Entity，并按需组合公共字段。
2. 编写 `DatabaseConfig`。
3. 通过 `dbhelper.NewManager` 初始化 `Manager`。
4. 通过 `dbhelper.NewTableStore` 或 `dbhelper.NewSoftDeleteTableStore` 获取 TableStore。
5. 使用 TableStore 执行 CRUD、查询、事务或 sharding 操作。

---

## 目录

- [1. 包定位](#1-包定位)
- [2. 快速开始](#2-快速开始)
- [3. Entity 与公共字段](#3-entity-与公共字段)
- [4. DatabaseGroupConfig 配置模式](#4-databasegroupconfig-配置模式)
  - [4.1 配置模式总览](#41-配置模式总览)
  - [4.2 单库](#42-单库)
  - [4.3 单 server 多分库](#43-单-server-多分库)
  - [4.4 多 server 多分库](#44-多-server-多分库)
  - [4.5 只 table sharding](#45-只-table-sharding)
  - [4.6 db + table sharding](#46-db--table-sharding)
  - [4.7 复合分片键（不同列路由库和表）](#47-复合分片键不同列路由库和表)
  - [4.8 Entity 级别覆写](#48-entity-级别覆写)
  - [4.9 混合配置：多库组](#49-混合配置多库组)
  - [4.10 连接池配置](#410-连接池配置)
- [5. Manager 与 TableStore](#5-manager-与-tablestore)
- [6. 事务](#6-事务)
- [7. Sharding 路由与查询模式](#7-sharding-路由与查询模式)
  - [7.1 ShardingKey 三种模式](#71-shardingkey-三种模式)
  - [7.2 多值场景：同表放行 vs 跨表拒绝](#72-多值场景同表放行-vs-跨表拒绝)
  - [7.3 Scatter-Gather（全分片查询）](#73-scatter-gather全分片查询)
- [8. 表达式语法速查](#8-表达式语法速查)
- [9. 完整示例](#9-完整示例)
- [10. 注意事项](#10-注意事项)

---

## 1. 包定位

`db` 模块的用户侧 API 分布在两个 package：

| Package | 定位 | 常用内容 |
|---------|------|----------|
| `dbspi` | 稳定契约和配置类型 | `DatabaseConfig`、`TableStore`、`SoftDeleteTableStore`、`CommonFields`、`ShardingKey` |
| `dbhelper` | 工厂和构造 helper | `NewManager`、`NewTableStore`、`Transaction`、`NewField`、`Q`、`NewUpdater` |

一般业务代码只需要：

- 用 `dbspi` 定义模型、配置和接口类型。
- 用 `dbhelper` 初始化 Manager、创建 TableStore、构造 Query/Updater。

---

## 2. 快速开始

```go
cfg := dbspi.DatabaseConfig{
    DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
        dbspi.DefaultDatabaseGroupKey: {
            Host:         "127.0.0.1",
            Port:         3306,
            User:         "root",
            Password:     "secret",
            DatabaseName: "my_app_db",
        },
    },
}

mgr, err := dbhelper.NewManager(cfg)
if err != nil {
    return err
}

userStore := dbhelper.NewTableStore(&User{}, dbhelper.WithManager(mgr))

ctx := context.Background()
users, err := userStore.Find(ctx, nil, nil)
```

`NewManager` 会校验配置并初始化数据库连接；配置非法时直接返回 error。

---

## 3. Entity 与公共字段

每个 Entity 需要实现 `TableName()` 接口。分片 Entity 还需通过 `DatabaseGroupKey()` 声明所属数据库组。

```go
// 非分片 Entity — 使用 dbspi.DefaultDatabaseGroupKey 数据库
type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}

func (*User) TableName() string   { return "user_tab" }
func (*User) IdFieldName() string { return dbspi.DefaultIdFieldName }

// 分片 Entity — 路由到 "order_dbs" 数据库组
type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64 `gorm:"column:amount"`
}

func (*Order) TableName() string   { return "order_tab" }
func (*Order) DatabaseGroupKey() string { return "order_dbs" }
func (*Order) IdFieldName() string { return dbspi.DefaultIdFieldName }
```

**接口一览**：

| 接口 | 必须 | 说明 |
|------|------|------|
| `TableName() string` | 是 | 逻辑表名 |
| `DatabaseGroupKey() string` | 否 | 所属库组 key（不实现则走 `dbspi.DefaultDatabaseGroupKey`） |
| `IdFieldName() string` | 否 | ID 列名（用于 `GetById`/`UpdateById` 等方法） |

**Auto ShardingKey 对 Entity 的要求**：Entity 的 struct field 上必须有 `gorm:"column:xxx"` tag 与配置中的 `@{xxx}` 列名对应，否则 auto 提取无法从 Entity 中读取分片字段值。

### 公共字段

`dbspi` 提供可嵌入的公共字段组合：

| 类型 | 字段 | 说明 |
|------|------|------|
| `IdField` | `id` | 标准主键字段 |
| `TimeFields` | `ctime`、`mtime` | 创建/更新时间 |
| `OperatorFields` | `creator`、`updater` | 创建/更新人 |
| `SoftDeleteField` | `deleted` | 软删除字段 |
| `CommonFields` | 上述全部字段 | 完整公共字段组合 |

示例：

```go
type User struct {
    dbspi.CommonFields
    Name string `gorm:"column:name"`
}

func (*User) TableName() string { return "user_tab" }
```

默认情况下，`ctime/mtime` 使用 Unix milliseconds，`creator/updater` 从 ctx 中读取：

```go
ctx := dbspi.WithOperator(context.Background(), "system")
err := userStore.Create(ctx, &User{Name: "Alice"})
```

可以在 Manager 或 TableStore 级别调整公共字段自动填充：

```go
mgr, err := dbhelper.NewManager(
    cfg,
    dbhelper.WithCommonFieldTimeProvider(func(ctx context.Context) uint64 {
        return uint64(time.Now().Unix())
    }),
)

userStore := dbhelper.NewTableStore(
    &User{},
    dbhelper.WithManager(mgr),
    dbhelper.WithCommonFieldAutoFill(false),
)
```

---

## 4. DatabaseGroupConfig 配置模式

所有配置以 `database_groups` 为根节点，每个 key 是一个数据库组名称。一个 `DatabaseGroupConfig` 同时描述连接目标和可选的分库/分表规则：

- `database_sharding`：数据库级路由规则，决定访问哪个 database target。
- `servers`：多个显式 database target，`key` 必须能被 `database_sharding.name_expr` 计算出来。
- `table_sharding`：表级路由规则，决定访问哪个物理表。
- `table_rules`：按逻辑表名覆写默认表级路由规则。

### 4.1 配置模式总览

| 模式 | 需要配置 | 不要配置 | 适用场景 |
|------|----------|----------|----------|
| 单库 | `database_name` 或 `dsn` | `database_sharding`、`servers`、`table_sharding` | 普通表，所有操作落到一个库的一张逻辑表 |
| 单 server 多分库 | `host/port/user/password` + `database_sharding` | `database_name`、`dsn`、`servers` | 同一个 MySQL server 上有多个 schema |
| 多 server 多分库 | `servers` + `database_sharding` | 顶层 `host/port/database_name` | 不同分库在不同 server，或每个分库需要不同 DSN |
| 只 table sharding | `database_name` 或 `dsn` + `table_sharding` | `database_sharding`、`servers` | 一个库内多张同构分表 |
| db + table sharding | `database_sharding` + `table_sharding` | 单 server 场景不要配 `database_name`/`dsn` | 同时按库和表分片 |

> `servers` 单独配置但没有 `database_sharding` 语义不清晰，不推荐使用。需要多分库时应同时配置 `database_sharding`。

### 4.2 单库

单库模式不配置任何 sharding 规则。没有 `DatabaseGroupKey()` 的 Entity 会默认使用 `default` 库组。

```yaml
database_groups:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_name: my_app_db
```

DSN 模式也可以：

```yaml
database_groups:
  default:
    dsn: "root:secret@tcp(10.0.0.1:3306)/my_app_db?charset=utf8mb4&parseTime=True&loc=Local"
```

### 4.3 单 server 多分库

多个 schema 位于同一个 MySQL server。配置顶层连接信息和 `database_sharding`，不要配置 `database_name`，库名由 `database_sharding.name_expr` 枚举和路由。

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = @{shop_id} % 4"
```

**效果**：启动时初始化 `order_db_0` ~ `order_db_3` 四个 database target。

**路由**：`shop_id % 4` → database index。

### 4.4 多 server 多分库

当不同分库位于不同 MySQL server，或者每个分库需要独立 DSN 时，使用 `servers` 明确列出 database targets，并用 `database_sharding` 计算 target key。

```yaml
database_groups:
  order_dbs:
    servers:
      - key: "order_db_0"
        host: 10.0.0.1
        port: 3306
        user: root
        password: secret
        database_name: order_db_0
      - key: "order_db_1"
        host: 10.0.0.2
        port: 3306
        user: root
        password: secret
        database_name: order_db_1
    database_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 2)"
        - "${idx} = @{shop_id} % 2"
```

`servers[].key` 必须与 `database_sharding.name_expr` 的计算结果匹配。上例中 `shop_id % 2 == 0` 路由到 `order_db_0`，`shop_id % 2 == 1` 路由到 `order_db_1`。

### 4.5 只 table sharding

一个 database 内按 `shop_id` 取模分 10 张表：

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_name: order_db
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

**效果**：`order_tab_00000000` ~ `order_tab_00000009`

**路由**：`shop_id % 10` → 表索引

### 4.6 db + table sharding

同一个 database group 可以同时配置库级和表级分片。下面示例中，同一个 MySQL server 上有 4 个 schema，每个 schema 下有 10 张分表：

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_sharding:
      name_expr: "order_db_${db_idx}"
      expand_exprs:
        - "${db_idx} := range(0, 4)"
        - "${db_idx} = @{shop_id} % 4"
    table_sharding:
      name_expr: "order_tab_${table_idx}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${table_idx} = fill(${idx}, 8)"
```

> 单 server `db + table sharding` 场景不要填 `database_name`，也不要用 `dsn`。库名由 `database_sharding` 生成。

如果分库分布在多个 server，只需要在这个模式上增加 `servers`，并保证 `servers[].key` 与 `database_sharding.name_expr` 的计算结果一致。

### 4.7 复合分片键（不同列路由库和表）

DB 按 `region` 枚举分，Table 按 `shop_id` 取模分：

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_sharding:
      name_expr: "order_${region}_db"
      expand_exprs:
        - "${region} := enum(SG, TH, ID)"
        - "${region} = @{region}"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

此配置下 Auto ShardingKey 需要 Entity 同时包含 `region` 和 `shop_id` 两个列。

### 4.8 Entity 级别覆写

同一库组中，不同 Entity 可以使用不同的分表规则。

#### `${table}` 内置变量

`name_expr` 支持 `${table}` 内置变量，自动替换为 Entity 的 `TableName()` 返回值。
这使得一套 `name_expr` 可以复用于多个 Entity：

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = @{shop_id} % 4"
    table_sharding:
      # ${table} 在运行时自动替换为 Entity.TableName()
      # Order -> "order_tab_00000005", OrderDetail -> "order_detail_tab_00000005"
      name_expr: "${table}_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
    table_rules:
      - tables: ["order_detail_tab"]
        table_sharding:
          # name_expr 为空时自动继承全局的 "${table}_${index}"
          # 仅需覆写 expand_exprs 即可改变分片数
          expand_exprs:
            - "${idx} := range(0, 20)"
            - "${idx} = @{shop_id} % 20"
            - "${index} = fill(${idx}, 8)"
```

**规则：**
- `${table}` 在 `ResolveTable` 和 `ShardName` 时自动绑定为 `entity.TableName()`
- table_rules 中的 `name_expr` 如果省略（空字符串），自动继承全局 `table_sharding.name_expr`
- 如果需要完全不同的命名模式，可以在 table_rules 中显式指定 `name_expr`

#### 不使用 `${table}` 的传统写法

如果不同 Entity 需要完全不同的命名模式，也可以显式指定 `name_expr`：

```yaml
    table_rules:
      - tables: ["order_detail_tab"]
        table_sharding:
          name_expr: "order_detail_tab_${index}"
          expand_exprs:
            - "${idx} := range(0, 20)"
            - "${idx} = @{shop_id} % 20"
            - "${index} = fill(${idx}, 8)"
```

### 4.9 混合配置：多库组

```yaml
database_groups:
  # 非分片库
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_name: my_app_db

  # 单 server db + table sharding
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = @{shop_id} % 4"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
    max_concurrency: 5
```

### 4.10 连接池配置

```yaml
database_groups:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_name: order_db
    max_open_conns: 200
    max_idle_conns: 20
    conn_max_lifetime_seconds: 1800
    debug: true
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

---

## 5. Manager 与 TableStore

```go
import (
    "gopkg.in/yaml.v3"
    "github.com/MrMiaoMIMI/goshared/db/dbhelper"
    "github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// 加载 YAML 配置
var cfg dbspi.DatabaseConfig
yaml.Unmarshal(configBytes, &cfg)

// 创建 Manager
mgr, err := dbhelper.NewManager(cfg)
if err != nil {
    return err
}

// 获取 TableStore
userStore := dbhelper.NewTableStore(&User{}, dbhelper.WithManager(mgr))     // → dbspi.DefaultDatabaseGroupKey 库
orderStore := dbhelper.NewTableStore(&Order{}, dbhelper.WithManager(mgr))   // → "order_dbs" 库组（根据 DatabaseGroupKey()）
```

### 基础 CRUD

```go
ctx := context.Background()

err := userStore.Create(ctx, &User{Name: "Alice"})
user, err := userStore.GetById(ctx, uint64(1))
exists, user, err := userStore.ExistsById(ctx, uint64(1))
err = userStore.Save(ctx, user)
err = userStore.DeleteById(ctx, uint64(1))
```

### Query / Pagination / Order

```go
nameField := dbhelper.NewField[string]("name")
ageField := dbhelper.NewField[int]("age")
name := "Alice"
minAge := 18
limit := 20
offset := 0

pagination := dbhelper.NewPagination().
    WithLimit(&limit).
    WithOffset(&offset).
    AppendOrder(dbhelper.Desc(ageField))

users, err := userStore.Find(ctx, dbhelper.Q(
    nameField.Eq(&name),
    ageField.GtEq(&minAge),
), pagination)
```

### UpdateByQuery

```go
statusField := dbhelper.NewField[string]("status")
status := "active"

updater := dbhelper.NewUpdater().
    Set(statusField, status)

err := userStore.UpdateByQuery(ctx, dbhelper.Q(nameField.Eq(&name)), updater)
```

### SoftDeleteTableStore

如果 Entity 包含 `dbspi.SoftDeleteField` 或实现了 soft-delete accessor，可以使用软删除 TableStore：

```go
userStore := dbhelper.NewSoftDeleteTableStore(&User{}, dbhelper.WithManager(mgr))

err := userStore.SoftDeleteById(ctx, uint64(1))
err = userStore.RestoreById(ctx, uint64(1))
users, err := userStore.FindNotDeleted(ctx, nil, nil)
```

### Raw / Exec

Raw SQL 是高级逃生口，不在基础 `TableStore` 接口上。需要通过 `AsSQLTableStore` 显式获取：

```go
sqlStore, ok := dbhelper.AsSQLTableStore(userStore)
if !ok {
    return errors.New("raw SQL is not supported")
}

rows, err := sqlStore.Raw(ctx, "SELECT * FROM user_tab WHERE status = ?", "active")
err = sqlStore.Exec(ctx, "UPDATE user_tab SET status = ? WHERE id = ?", "disabled", 1)
```

对于 sharding 表，Raw/Exec 无法自动提取 sharding key，需要先用 `Shard(key)` 或在 ctx 中设置 `dbspi.WithShardingKey(ctx, key)`。

---

## 6. 事务

`Transaction` 在一个物理 database transaction 中执行回调。事务内可以操作多张表，但必须属于同一个 database group，并且 db-sharding 场景下必须绑定到同一个物理 database target。

```go
err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
    txUserStore := dbhelper.NewTableStore(&User{}, dbhelper.WithTx(tx))
    txShopStore := dbhelper.NewTableStore(&Shop{}, dbhelper.WithTx(tx))

    if err := txUserStore.Create(ctx, &User{Name: "Alice"}); err != nil {
        return err
    }
    return txShopStore.Create(ctx, &Shop{Name: "Alice Shop"})
}, dbhelper.WithManager(mgr))
```

db-sharding database group 需要显式指定事务所在的 database group 和 database shard：

```go
key := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))

err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
    txOrderStore := dbhelper.NewTableStore(&Order{}, dbhelper.WithTx(tx))
    return txOrderStore.Create(ctx, &Order{ShopID: 12345, Amount: 100})
},
    dbhelper.WithManager(mgr),
    dbhelper.WithTransactionDatabaseGroupKey("order_dbs"),
    dbhelper.WithTransactionShardingKey(key),
)
```

---

## 7. Sharding 路由与查询模式

### 7.1 ShardingKey 三种模式

分片 TableStore 在执行 CRUD 时，需要确定目标分片（哪个库、哪张表）。ShardingKey 的值可以来自三个来源，系统会**聚合所有来源的值**并校验它们是否指向同一个分片目标。

#### 7.1.1 Auto 模式：从 CRUD 参数自动提取

**无需手动设置 ShardingKey**，系统自动从 CRUD 参数中提取分片列的值。

#### 从 Entity 提取（Create / Save / Update / Delete）

```go
ctx := context.Background()

// shop_id=12345 自动从 Entity struct 中读取
err := orderStore.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// → 路由到 order_tab_00000005（12345 % 10 = 5）
```

#### 从 Query 提取（Find / Count / Exists / UpdateByQuery / DeleteByQuery）

```go
shopIdField := dbhelper.NewField[int64]("shop_id")

ctx := context.Background()
shopId := int64(12345)

// shop_id=12345 从 Eq 条件中提取
orders, err := orderStore.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
```

支持从以下条件类型中提取值：

| 条件类型 | 示例 | 提取行为 |
|----------|------|----------|
| `Eq` | `shopIdField.Eq(&val)` | 提取单个值 |
| `In` | `shopIdField.In([]int64{1, 2, 3})` | 提取所有值 |
| `OR` 内的 `Eq`/`In` | `Or(shopIdField.Eq(&v1), shopIdField.Eq(&v2))` | 提取所有值 |
| `Gt` / `Lt` / `Like` 等 | `shopIdField.Gt(&val)` | **不提取**（范围条件无法确定分片） |

#### 从 ID 提取（GetById / UpdateById / DeleteById）

当分片键就是 ID 列时，自动从 `id` 参数提取：

```go
// 配置：${idx} = @{id} % 10
order, err := orderStore.GetById(ctx, int64(1001))
// → id=1001 自动提取，路由到对应分片
```

#### 从 Entity + Query 聚合提取（FirstOrCreate）

```go
shopId := int64(12345)
result, err := orderStore.FirstOrCreate(ctx,
    &Order{ShopID: 12345, Amount: 100},
    dbhelper.Q(shopIdField.Eq(&shopId)),
)
// Entity 和 Query 中的 shop_id 值都会被收集并校验
```

#### 7.1.2 Manual 模式：手动设置 ShardingKey

通过 Context 注入或 `Shard()` 方法手动指定分片键。

#### 通过 Context 注入

```go
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

orders, err := orderStore.Find(ctx, nil, nil)
```

适用场景：在中间件/拦截器中统一设置，后续所有操作自动路由。

#### 通过 Shard() 方法

```go
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))
shardStore, err := orderStore.Shard(sk)

// 在同一分片上执行多次操作
order, _ := shardStore.GetById(ctx, 1001)
orders, _ := shardStore.Find(ctx, query, nil)
```

适用场景：同一分片上需要执行多次操作。

#### Raw/Exec（仅支持手动）

Raw SQL 和 Exec 无法自动提取分片键，必须手动设置：

```go
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

sqlStore, ok := dbhelper.AsSQLTableStore(orderStore)
if !ok {
    return errors.New("raw SQL is not supported")
}
rows, err := sqlStore.Raw(ctx, "SELECT * FROM order_tab WHERE amount > ?", 100)
```

#### 7.1.3 Mix 模式：自动 + 手动聚合校验

当 Context 中有手动 ShardingKey，同时 CRUD 参数中也能自动提取到分片值时，系统会**聚合所有来源**，统一校验是否指向同一个分片目标。

```go
// 手动设置 shop_id=12345
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

shopId := int64(12345)

// Query 中也有 shop_id=12345 → 两个来源值相同 → 正常路由
orders, err := orderStore.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
// ✅ OK
```

**跨分片检测**：如果手动 key 和自动提取的值路由到不同分片，会报错：

```go
// 手动：shop_id=99999 (99999 % 10 = 9)
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(99999))
ctx := dbspi.WithShardingKey(context.Background(), sk)

// 自动：entity shop_id=12345 (12345 % 10 = 5)
err := orderStore.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// ❌ Error: cross-shard query not allowed: column "shop_id" values route to different targets
```

**同表放行**：即使值不同，只要路由到同一张物理表，就允许：

```go
// 手动：shop_id=22345 (22345 % 10 = 5)
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(22345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

// 自动：entity shop_id=12345 (12345 % 10 = 5) → 同一张表
err := orderStore.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// ✅ OK: 两个值都路由到 order_tab_00000005
```

---

### 7.2 多值场景：同表放行 vs 跨表拒绝

当同一个分片列出现多个值时（来自重复 Eq、OR、IN 或多个来源的合并），系统会：

1. **去重**：相同的值合并
2. **校验**：所有去重后的值是否路由到同一个物理目标（db + table）
3. 同目标 → **放行**，取第一个值路由
4. 不同目标 → **拒绝**，返回 `cross-shard query not allowed` 错误

#### 7.2.1 重复 Eq 值

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(21111) // 21111 % 10 = 1 → 同表

// AND(shop_id=11111, shop_id=21111) → 同表 → OK
query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderStore.Find(ctx, query, nil) // ✅ OK
```

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(22222) // 22222 % 10 = 2 → 不同表

// AND(shop_id=11111, shop_id=22222) → 跨表 → Error
query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderStore.Find(ctx, query, nil) // ❌ cross-shard error
```

#### 7.2.2 OR 表达式

OR 子句中的 Eq/In 值也会被提取并校验：

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(21111) // 21111 % 10 = 1 → 同表

// OR(shop_id=11111, shop_id=21111) → 同表 → OK
orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderStore.Find(ctx, orQuery, nil) // ✅ OK
```

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(22222) // 22222 % 10 = 2 → 不同表

// OR(shop_id=11111, shop_id=22222) → 跨表 → Error
orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderStore.Find(ctx, orQuery, nil) // ❌ cross-shard error
```

#### 7.2.3 IN 表达式

IN 的所有值都会被提取并校验：

```go
// 11111 % 10 = 1, 21111 % 10 = 1, 31111 % 10 = 1 → 同表
inQuery := dbhelper.Q(shopIdField.In([]int64{11111, 21111, 31111}))
orders, err := orderStore.Find(ctx, inQuery, nil) // ✅ OK
```

```go
// 11111 % 10 = 1, 22222 % 10 = 2 → 不同表
inQuery := dbhelper.Q(shopIdField.In([]int64{11111, 22222}))
orders, err := orderStore.Find(ctx, inQuery, nil) // ❌ cross-shard error
```

#### 7.2.4 Entity + Query 跨源

FirstOrCreate 同时接收 Entity 和 Query，两个来源的分片值会被聚合校验：

```go
queryShopId := int64(22345) // 22345 % 10 = 5
// Entity shop_id=12345 (% 10 = 5) + Query shop_id=22345 (% 10 = 5) → 同表
result, err := orderStore.FirstOrCreate(ctx,
    &Order{ShopID: 12345, Amount: 100},
    dbhelper.Q(shopIdField.Eq(&queryShopId)),
) // ✅ OK
```

#### 7.2.5 Context + Auto 跨源

手动 Context key 和自动提取的值聚合后统一校验：

```go
// 手动：shop_id=22345 (% 10 = 5)
sk := dbspi.NewShardingKey().SetValue("shop_id", int64(22345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

shopId := int64(12345) // % 10 = 5 → 同表
orders, err := orderStore.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
// ✅ OK: 两个值都路由到 order_tab_00000005
```

### 综合场景

```go
// 混合 AND + OR + 非分片条件
shopId := int64(12345)
status1 := 1
status2 := 2

// AND(OR(status=1, status=2), shop_id=12345)
// shop_id 有 1 个值 → 正常路由
// status 有 2 个值，但 status 不是分片列 → 忽略
query := dbhelper.Q(
    dbhelper.Or(statusField.Eq(&status1), statusField.Eq(&status2)),
    shopIdField.Eq(&shopId),
)
orders, err := orderStore.Find(ctx, query, nil) // ✅ OK
```

---

### 7.3 Scatter-Gather（全分片查询）

跨所有分片查询，不需要 ShardingKey：

```go
ctx := context.Background()

// 查询所有分片，每分片批量 100 条
allOrders, err := orderStore.FindAll(ctx, query, 100)

// 统计所有分片总数
totalCount, err := orderStore.CountAll(ctx, query)
```

`max_concurrency` 控制并发 goroutine 数，推荐对大分片数场景设置合理值：

```yaml
database_groups:
  order_dbs:
    max_concurrency: 10
    # ...
```

---

## 8. 表达式语法速查

### name_expr（名称模板）

仅支持 `${var}` 引用，所有计算逻辑放在 `expand_exprs` 中。

| 模板 | 变量值 | 结果 |
|------|--------|------|
| `order_${region}_db` | region="SG" | `order_SG_db` |
| `order_tab_${index}` | index="00000005" | `order_tab_00000005` |

### expand_exprs（变量声明与计算）

**声明 `:=`** — 启动时枚举所有可能值：

```yaml
- "${region} := enum(SG, TH, ID)"    # 字符串枚举
- "${idx}    := range(0, 10)"        # 整数范围 [0, 10)
```

**计算 `=`** — 运行时根据 ShardingKey 计算：

```yaml
- "${region} = @{region}"                  # 直接传递列值
- "${idx}    = @{shop_id} % 10"            # 算术运算
- "${idx}    = hash(@{shop_id}) % 1000"    # 函数 + 运算
- "${index}  = fill(${idx}, 8)"            # 格式化
```

### 内建函数

| 函数 | 说明 | 示例 | 结果 |
|------|------|------|------|
| `fill(value, width)` | 零填充 | `fill(5, 8)` | `"00000005"` |
| `hash(value)` | FNV-1a 哈希 | `hash(@{shop_id})` | int64 |
| `str(value)` | 转字符串 | `str(42)` | `"42"` |
| `mod(a, b)` | 取模 | `mod(100, 7)` | `2` |
| `div(a, b)` | 除法 | `div(100, 7)` | `14` |
| `lower(value)` | 小写 | `lower(SG)` | `"sg"` |
| `upper(value)` | 大写 | `upper(sg)` | `"SG"` |
| `concat(a, b, ...)` | 拼接 | `concat(a, _, b)` | `"a_b"` |

### 引用语法

| 语法 | 位置 | 含义 |
|------|------|------|
| `${var}` | name_expr + expand_exprs | 扩展变量 |
| `@{col}` | expand_exprs 内 | 列引用（从 ShardingKey 读取） |

### `${table}` 内置变量

`${table}` 是一个特殊的内置变量，在运行时自动绑定为 `entity.TableName()` 的返回值。
在 `name_expr` 中使用 `${table}` 可以让同一套分表规则复用于多个 Entity：

```yaml
table_sharding:
  name_expr: "${table}_${index}"  # Order → "order_tab_XXXX", OrderDetail → "order_detail_tab_XXXX"
  expand_exprs:
    - "${idx} := range(0, 10)"
    - "${idx} = @{shop_id} % 10"
    - "${index} = fill(${idx}, 8)"
```

配合 table_rules 的 `name_expr` 继承：如果 table_rule 中不指定 `name_expr`，自动继承全局 `table_sharding.name_expr`，仅需覆写 `expand_exprs`。

---

## 9. 完整示例

### config.yaml

```yaml
database_groups:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    database_name: my_app_db

  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    max_open_conns: 200
    max_idle_conns: 20
    max_concurrency: 10
    database_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = @{shop_id} % 4"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

### main.go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "gopkg.in/yaml.v3"
    "github.com/MrMiaoMIMI/goshared/db/dbhelper"
    "github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}
func (*User) TableName() string   { return "user_tab" }
func (*User) IdFieldName() string { return dbspi.DefaultIdFieldName }

type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64 `gorm:"column:amount"`
}
func (*Order) TableName() string   { return "order_tab" }
func (*Order) DatabaseGroupKey() string { return "order_dbs" }
func (*Order) IdFieldName() string { return dbspi.DefaultIdFieldName }

func main() {
    // 1. 加载配置
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    var cfg dbspi.DatabaseConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        log.Fatal(err)
    }

    // 2. 初始化 Manager
    mgr, err := dbhelper.NewManager(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 获取 TableStore
    userStore := dbhelper.NewTableStore(&User{}, dbhelper.WithManager(mgr))
    orderStore := dbhelper.NewTableStore(&Order{}, dbhelper.WithManager(mgr))
    shopIdField := dbhelper.NewField[int64]("shop_id")

    ctx := context.Background()

    // ===== 非分片操作 =====
    users, _ := userStore.Find(ctx, nil, nil)
    fmt.Printf("Users: %d\n", len(users))

    // ===== Auto 模式：从 Entity 自动提取 =====
    _ = orderStore.Create(ctx, &Order{ShopID: 12345, Amount: 100})

    // ===== Auto 模式：从 Query 自动提取 =====
    shopId := int64(12345)
    orders, _ := orderStore.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
    fmt.Printf("Orders: %d\n", len(orders))

    // ===== Manual 模式：手动设置 =====
    sk := dbspi.NewShardingKey().SetValue("shop_id", int64(12345))
    manualCtx := dbspi.WithShardingKey(ctx, sk)
    orders, _ = orderStore.Find(manualCtx, nil, nil)

    // ===== Mix 模式：手动 + 自动聚合校验 =====
    // 手动 key 和 query 都指向 shop_id % 10 = 5 → OK
    mixCtx := dbspi.WithShardingKey(ctx,
        dbspi.NewShardingKey().SetValue("shop_id", int64(22345)))
    orders, _ = orderStore.Find(mixCtx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)

    // ===== Scatter-Gather =====
    totalCount, _ := orderStore.CountAll(ctx, nil)
    fmt.Printf("Total orders: %d\n", totalCount)

    allOrders, _ := orderStore.FindAll(ctx, nil, 100)
    fmt.Printf("All orders: %d\n", len(allOrders))
}
```

---

## 10. 注意事项

### Auto 提取的来源优先级

| 方法类型 | 值来源 |
|----------|--------|
| Create / Save / Update / Delete / BatchCreate / BatchSave | ctx key + entity struct fields |
| Find / Count / Exists / UpdateByQuery / DeleteByQuery | ctx key + query conditions (Eq/In/OR) |
| GetById / ExistsById / UpdateById / DeleteById | ctx key + id parameter |
| FirstOrCreate | ctx key + entity + query |
| Raw / Exec | ctx key only（无 auto 提取） |

所有来源的值会被**聚合到一起**进行同分片校验。

### 哪些条件类型不会被提取

以下条件**不会**作为 ShardingKey 值被提取：

- `Like` / `NotLike` / `StartsWith` / `EndsWith` / `Contains`（模糊匹配）
- `IsNull` / `IsNotNull`（空值判断）
- `NotEq` / `NotIn`（否定条件）
- `NOT(...)` 子句内的所有条件

### 范围条件的主动检测

范围条件 `Gt` / `GtEq` / `Lt` / `LtEq` / `Between` 不会被提取为值，但系统会**主动检测**它们是否作用于分片列。如果分片列**只有**范围条件而没有 `Eq`/`In` 值，系统会给出专门的错误提示：

```
sharding columns [shop_id] have range conditions (Gt/Lt/Between) which cannot determine
a single shard; range conditions may cause cross-shard operations. Use Eq/In for sharding
columns, set WithShardingKey(ctx, key), or use FindAll/CountAll for cross-shard queries
```

示例：

```go
shopIdField := dbhelper.NewField[int64]("shop_id")
amountField := dbhelper.NewField[int64]("amount")
orderStore := dbhelper.NewTableStore(&Order{}, dbhelper.WithManager(mgr))

// ❌ 分片列 shop_id 只有范围条件 → 主动报错
min := int64(10000)
orderStore.Find(ctx, dbhelper.Q(shopIdField.Gt(&min)), nil)
// → error: sharding columns [shop_id] have range conditions...

// ❌ Between 同样会被检测
min, max := int64(10000), int64(99999)
orderStore.Find(ctx, dbhelper.Q(shopIdField.Between(&min, &max)), nil)
// → error: sharding columns [shop_id] have range conditions...

// ✅ 分片列有 Eq，非分片列的范围条件无影响
shopId := int64(12345)
minAmount := int64(100)
orderStore.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId), amountField.Gt(&minAmount)), nil)
// → OK: shop_id 通过 Eq 确定分片，amount 的 Gt 仅作为过滤条件
```

### DSN 与 database_sharding 不兼容

单服务器场景下，`dsn` 不能与 `database_sharding` 同时使用（DSN 中包含数据库名，无法动态切换）。分库场景请使用 host/port/user/password 字段或 `servers` 列表。

### database_sharding 时不要填 database_name

使用 `database_sharding` 时，库名由表达式自动生成，`database_name` 字段无意义。

### Entity gorm tag 要求

Auto ShardingKey 依赖 gorm tag 中的 `column:xxx` 来定位 struct field。确保分片列的 gorm tag 与配置中的 `@{xxx}` 一致：

```go
// ✅ 正确：gorm tag column 名 = 配置中 @{shop_id}
ShopID int64 `gorm:"column:shop_id"`

// ❌ 缺少 gorm tag，auto 将按 snake_case 推断为 "shop_i_d"（不一定正确）
ShopID int64
```

### 跨表查询需使用 Scatter-Gather

当需要跨多个分片查询时，不要尝试在 Query 中放入路由到不同表的值（会被 cross-shard 校验拒绝）。应使用 `FindAll` / `CountAll` 进行全分片查询。
