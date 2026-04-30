# Sharding Guide

本指南以 **YAML 配置驱动** 的方式，从零搭建分库分表服务。核心流程：

1. 编写 YAML 配置（`DatabaseConfig`）
2. 初始化 `DbManager`
3. 通过 `For(&Entity{})` 获取 Executor
4. CRUD 时自动/手动/混合提供 ShardingKey

---

## 目录

- [1. Entity 定义](#1-entity-定义)
- [2. YAML 配置](#2-yaml-配置)
  - [2.1 非分片单库](#21-非分片单库)
  - [2.2 单库分表](#22-单库分表)
  - [2.3 分库分表](#23-分库分表)
  - [2.4 复合分片键（不同列路由库和表）](#24-复合分片键不同列路由库和表)
  - [2.5 Entity 级别覆写](#25-entity-级别覆写)
  - [2.6 混合配置：多库组](#26-混合配置多库组)
  - [2.7 多服务器分库](#27-多服务器分库)
  - [2.8 连接池配置](#28-连接池配置)
- [3. 初始化 DbManager](#3-初始化-dbmanager)
- [4. ShardingKey 三种模式](#4-shardingkey-三种模式)
  - [4.1 Auto 模式：从 CRUD 参数自动提取](#41-auto-模式从-crud-参数自动提取)
  - [4.2 Manual 模式：手动设置 ShardingKey](#42-manual-模式手动设置-shardingkey)
  - [4.3 Mix 模式：自动 + 手动聚合校验](#43-mix-模式自动--手动聚合校验)
- [5. 多值场景：同表放行 vs 跨表拒绝](#5-多值场景同表放行-vs-跨表拒绝)
  - [5.1 重复 Eq 值](#51-重复-eq-值)
  - [5.2 OR 表达式](#52-or-表达式)
  - [5.3 IN 表达式](#53-in-表达式)
  - [5.4 Entity + Query 跨源](#54-entity--query-跨源)
  - [5.5 Context + Auto 跨源](#55-context--auto-跨源)
- [6. Scatter-Gather（全分片查询）](#6-scatter-gather全分片查询)
- [7. 表达式语法速查](#7-表达式语法速查)
  - [7.x ${table} 内置变量](#7x-table-内置变量)
- [8. 完整示例](#8-完整示例)
- [9. 注意事项](#9-注意事项)

---

## 1. Entity 定义

每个 Entity 需要实现 `TableName()` 接口。分片 Entity 还需通过 `DbKey()` 声明所属数据库组。

```go
// 非分片 Entity — 使用 "default" 数据库
type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}

func (*User) TableName() string   { return "user_tab" }
func (*User) IdFiledName() string { return "id" }

// 分片 Entity — 路由到 "order_dbs" 数据库组
type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64 `gorm:"column:amount"`
}

func (*Order) TableName() string   { return "order_tab" }
func (*Order) DbKey() string       { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }
```

**接口一览**：

| 接口 | 必须 | 说明 |
|------|------|------|
| `TableName() string` | 是 | 逻辑表名 |
| `DbKey() string` | 否 | 所属库组 key（不实现则走 `"default"`） |
| `IdFiledName() string` | 否 | ID 列名（用于 `GetById`/`UpdateById` 等方法） |

**Auto ShardingKey 对 Entity 的要求**：Entity 的 struct field 上必须有 `gorm:"column:xxx"` tag 与配置中的 `@{xxx}` 列名对应，否则 auto 提取无法从 Entity 中读取分片字段值。

---

## 2. YAML 配置

所有配置以 `databases` 为根节点，每个 key 是一个数据库组名称。分片规则使用 **表达式语法**（`name_expr` + `expand_exprs`）描述。

### 2.1 非分片单库

```yaml
databases:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db
```

### 2.2 单库分表

按 `shop_id` 取模分 10 张表：

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: order_db
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

**效果**：`order_tab_00000000` ~ `order_tab_00000009`

**路由**：`shop_id % 10` → 表索引

### 2.3 分库分表

4 个库，每库 10 张表：

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
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

> **注意**：使用 `db_sharding` 时不要填 `db_name`（库名由表达式自动生成）。

### 2.4 复合分片键（不同列路由库和表）

DB 按 `region` 枚举分，Table 按 `shop_id` 取模分：

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
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

### 2.5 Entity 级别覆写

同一库组中，不同 Entity 可以使用不同的分表规则。

#### `${table}` 内置变量

`name_expr` 支持 `${table}` 内置变量，自动替换为 Entity 的 `TableName()` 返回值。
这使得一套 `name_expr` 可以复用于多个 Entity：

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = @{shop_id} % 4"
    table_sharding:
      # ${table} 在运行时自动替换为 Entity.TableName()
      # Order → "order_tab_00000005", OrderDetail → "order_detail_tab_00000005"
      name_expr: "${table}_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
    entity_rules:
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
- entity_rules 中的 `name_expr` 如果省略（空字符串），自动继承全局 `table_sharding.name_expr`
- 如果需要完全不同的命名模式，可以在 entity_rules 中显式指定 `name_expr`

#### 不使用 `${table}` 的传统写法

如果不同 Entity 需要完全不同的命名模式，也可以显式指定 `name_expr`：

```yaml
    entity_rules:
      - tables: ["order_detail_tab"]
        table_sharding:
          name_expr: "order_detail_tab_${index}"
          expand_exprs:
            - "${idx} := range(0, 20)"
            - "${idx} = @{shop_id} % 20"
            - "${index} = fill(${idx}, 8)"
```

### 2.6 混合配置：多库组

```yaml
databases:
  # 非分片库
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db

  # 分库分表
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_sharding:
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

### 2.7 多服务器分库

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
    db_sharding:
      name_expr: "${idx}"
      expand_exprs:
        - "${idx} := range(0, 2)"
        - "${idx} = @{shop_id} % 2"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = @{shop_id} % 10"
        - "${index} = fill(${idx}, 8)"
```

> `servers[].key` 必须与 `db_sharding.name_expr` 的计算结果匹配。

### 2.8 连接池配置

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: order_db
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

## 3. 初始化 DbManager

```go
import (
    "gopkg.in/yaml.v3"
    "github.com/MrMiaoMIMI/goshared/db/dbhelper"
    "github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// 加载 YAML 配置
var cfg dbspi.DatabaseConfig
yaml.Unmarshal(configBytes, &cfg)

// 创建 DbManager 并设为全局默认
mgr := dbhelper.NewDbManager(cfg)
dbhelper.SetDefault(mgr)

// 获取 Executor
userExec := dbhelper.For(&User{})     // → "default" 库
orderExec := dbhelper.For(&Order{})   // → "order_dbs" 库组（根据 DbKey()）
```

---

## 4. ShardingKey 三种模式

分片 Executor 在执行 CRUD 时，需要确定目标分片（哪个库、哪张表）。ShardingKey 的值可以来自三个来源，系统会**聚合所有来源的值**并校验它们是否指向同一个分片目标。

### 4.1 Auto 模式：从 CRUD 参数自动提取

**无需手动设置 ShardingKey**，系统自动从 CRUD 参数中提取分片列的值。

#### 从 Entity 提取（Create / Save / Update / Delete）

```go
ctx := context.Background()

// shop_id=12345 自动从 Entity struct 中读取
err := orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// → 路由到 order_tab_00000005（12345 % 10 = 5）
```

#### 从 Query 提取（Find / Count / Exists / UpdateByQuery / DeleteByQuery）

```go
shopIdField := dbhelper.NewField[int64]("shop_id")

ctx := context.Background()
shopId := int64(12345)

// shop_id=12345 从 Eq 条件中提取
orders, err := orderExec.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
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
order, err := orderExec.GetById(ctx, int64(1001))
// → id=1001 自动提取，路由到对应分片
```

#### 从 Entity + Query 聚合提取（FirstOrCreate）

```go
shopId := int64(12345)
result, err := orderExec.FirstOrCreate(ctx,
    &Order{ShopID: 12345, Amount: 100},
    dbhelper.Q(shopIdField.Eq(&shopId)),
)
// Entity 和 Query 中的 shop_id 值都会被收集并校验
```

### 4.2 Manual 模式：手动设置 ShardingKey

通过 Context 注入或 `Shard()` 方法手动指定分片键。

#### 通过 Context 注入

```go
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

orders, err := orderExec.Find(ctx, nil, nil)
```

适用场景：在中间件/拦截器中统一设置，后续所有操作自动路由。

#### 通过 Shard() 方法

```go
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
shardExec, err := orderExec.Shard(sk)

// 在同一分片上执行多次操作
order, _ := shardExec.GetById(ctx, 1001)
orders, _ := shardExec.Find(ctx, query, nil)
```

适用场景：同一分片上需要执行多次操作。

#### Raw/Exec（仅支持手动）

Raw SQL 和 Exec 无法自动提取分片键，必须手动设置：

```go
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

rows, err := orderExec.Raw(ctx, "SELECT * FROM order_tab WHERE amount > ?", 100)
```

### 4.3 Mix 模式：自动 + 手动聚合校验

当 Context 中有手动 ShardingKey，同时 CRUD 参数中也能自动提取到分片值时，系统会**聚合所有来源**，统一校验是否指向同一个分片目标。

```go
// 手动设置 shop_id=12345
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

shopId := int64(12345)

// Query 中也有 shop_id=12345 → 两个来源值相同 → 正常路由
orders, err := orderExec.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
// ✅ OK
```

**跨分片检测**：如果手动 key 和自动提取的值路由到不同分片，会报错：

```go
// 手动：shop_id=99999 (99999 % 10 = 9)
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(99999))
ctx := dbspi.WithShardingKey(context.Background(), sk)

// 自动：entity shop_id=12345 (12345 % 10 = 5)
err := orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// ❌ Error: cross-shard query not allowed: column "shop_id" values route to different targets
```

**同表放行**：即使值不同，只要路由到同一张物理表，就允许：

```go
// 手动：shop_id=22345 (22345 % 10 = 5)
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(22345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

// 自动：entity shop_id=12345 (12345 % 10 = 5) → 同一张表
err := orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
// ✅ OK: 两个值都路由到 order_tab_00000005
```

---

## 5. 多值场景：同表放行 vs 跨表拒绝

当同一个分片列出现多个值时（来自重复 Eq、OR、IN 或多个来源的合并），系统会：

1. **去重**：相同的值合并
2. **校验**：所有去重后的值是否路由到同一个物理目标（db + table）
3. 同目标 → **放行**，取第一个值路由
4. 不同目标 → **拒绝**，返回 `cross-shard query not allowed` 错误

### 5.1 重复 Eq 值

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(21111) // 21111 % 10 = 1 → 同表

// AND(shop_id=11111, shop_id=21111) → 同表 → OK
query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderExec.Find(ctx, query, nil) // ✅ OK
```

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(22222) // 22222 % 10 = 2 → 不同表

// AND(shop_id=11111, shop_id=22222) → 跨表 → Error
query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderExec.Find(ctx, query, nil) // ❌ cross-shard error
```

### 5.2 OR 表达式

OR 子句中的 Eq/In 值也会被提取并校验：

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(21111) // 21111 % 10 = 1 → 同表

// OR(shop_id=11111, shop_id=21111) → 同表 → OK
orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderExec.Find(ctx, orQuery, nil) // ✅ OK
```

```go
shopId1 := int64(11111) // 11111 % 10 = 1
shopId2 := int64(22222) // 22222 % 10 = 2 → 不同表

// OR(shop_id=11111, shop_id=22222) → 跨表 → Error
orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
orders, err := orderExec.Find(ctx, orQuery, nil) // ❌ cross-shard error
```

### 5.3 IN 表达式

IN 的所有值都会被提取并校验：

```go
// 11111 % 10 = 1, 21111 % 10 = 1, 31111 % 10 = 1 → 同表
inQuery := dbhelper.Q(shopIdField.In([]int64{11111, 21111, 31111}))
orders, err := orderExec.Find(ctx, inQuery, nil) // ✅ OK
```

```go
// 11111 % 10 = 1, 22222 % 10 = 2 → 不同表
inQuery := dbhelper.Q(shopIdField.In([]int64{11111, 22222}))
orders, err := orderExec.Find(ctx, inQuery, nil) // ❌ cross-shard error
```

### 5.4 Entity + Query 跨源

FirstOrCreate 同时接收 Entity 和 Query，两个来源的分片值会被聚合校验：

```go
queryShopId := int64(22345) // 22345 % 10 = 5
// Entity shop_id=12345 (% 10 = 5) + Query shop_id=22345 (% 10 = 5) → 同表
result, err := orderExec.FirstOrCreate(ctx,
    &Order{ShopID: 12345, Amount: 100},
    dbhelper.Q(shopIdField.Eq(&queryShopId)),
) // ✅ OK
```

### 5.5 Context + Auto 跨源

手动 Context key 和自动提取的值聚合后统一校验：

```go
// 手动：shop_id=22345 (% 10 = 5)
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(22345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

shopId := int64(12345) // % 10 = 5 → 同表
orders, err := orderExec.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
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
orders, err := orderExec.Find(ctx, query, nil) // ✅ OK
```

---

## 6. Scatter-Gather（全分片查询）

跨所有分片查询，不需要 ShardingKey：

```go
ctx := context.Background()

// 查询所有分片，每分片批量 100 条
allOrders, err := orderExec.FindAll(ctx, query, 100)

// 统计所有分片总数
totalCount, err := orderExec.CountAll(ctx, query)
```

`max_concurrency` 控制并发 goroutine 数，推荐对大分片数场景设置合理值：

```yaml
databases:
  order_dbs:
    max_concurrency: 10
    # ...
```

---

## 7. 表达式语法速查

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

配合 entity_rules 的 `name_expr` 继承：如果 entity_rule 中不指定 `name_expr`，自动继承全局 `table_sharding.name_expr`，仅需覆写 `expand_exprs`。

---

## 8. 完整示例

### config.yaml

```yaml
databases:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db

  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    max_open_conns: 200
    max_idle_conns: 20
    max_concurrency: 10
    db_sharding:
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
func (*User) IdFiledName() string { return "id" }

type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64 `gorm:"column:amount"`
}
func (*Order) TableName() string   { return "order_tab" }
func (*Order) DbKey() string       { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }

func main() {
    // 1. 加载配置
    data, _ := os.ReadFile("config.yaml")
    var cfg dbspi.DatabaseConfig
    yaml.Unmarshal(data, &cfg)

    // 2. 初始化 DbManager
    mgr := dbhelper.NewDbManager(cfg)
    dbhelper.SetDefault(mgr)

    // 3. 获取 Executor
    userExec := dbhelper.For(&User{})
    orderExec := dbhelper.For(&Order{})
    shopIdField := dbhelper.NewField[int64]("shop_id")

    ctx := context.Background()

    // ===== 非分片操作 =====
    users, _ := userExec.Find(ctx, nil, nil)
    fmt.Printf("Users: %d\n", len(users))

    // ===== Auto 模式：从 Entity 自动提取 =====
    _ = orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})

    // ===== Auto 模式：从 Query 自动提取 =====
    shopId := int64(12345)
    orders, _ := orderExec.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
    fmt.Printf("Orders: %d\n", len(orders))

    // ===== Manual 模式：手动设置 =====
    sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
    manualCtx := dbspi.WithShardingKey(ctx, sk)
    orders, _ = orderExec.Find(manualCtx, nil, nil)

    // ===== Mix 模式：手动 + 自动聚合校验 =====
    // 手动 key 和 query 都指向 shop_id % 10 = 5 → OK
    mixCtx := dbspi.WithShardingKey(ctx,
        dbspi.NewShardingKey().SetVal("shop_id", int64(22345)))
    orders, _ = orderExec.Find(mixCtx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)

    // ===== Scatter-Gather =====
    totalCount, _ := orderExec.CountAll(ctx, nil)
    fmt.Printf("Total orders: %d\n", totalCount)

    allOrders, _ := orderExec.FindAll(ctx, nil, 100)
    fmt.Printf("All orders: %d\n", len(allOrders))
}
```

---

## 9. 注意事项

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

// ❌ 分片列 shop_id 只有范围条件 → 主动报错
min := int64(10000)
executor.Find(ctx, dbhelper.Q(shopIdField.Gt(&min)), nil)
// → error: sharding columns [shop_id] have range conditions...

// ❌ Between 同样会被检测
min, max := int64(10000), int64(99999)
executor.Find(ctx, dbhelper.Q(shopIdField.Between(&min, &max)), nil)
// → error: sharding columns [shop_id] have range conditions...

// ✅ 分片列有 Eq，非分片列的范围条件无影响
shopId := int64(12345)
minAmount := int64(100)
executor.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId), amountField.Gt(&minAmount)), nil)
// → OK: shop_id 通过 Eq 确定分片，amount 的 Gt 仅作为过滤条件
```

### DSN 与 db_sharding 不兼容

单服务器场景下，`dsn` 不能与 `db_sharding` 同时使用（DSN 中包含数据库名，无法动态切换）。分库场景请使用 host/port/user/password 字段或 `servers` 列表。

### db_sharding 时不要填 db_name

使用 `db_sharding` 时，库名由表达式自动生成，`db_name` 字段无意义。

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
