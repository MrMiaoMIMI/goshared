# Expression-Based Sharding Guide

本指南详细介绍如何使用基于表达式的分库分表系统。所有分片规则通过 **表达式** 描述，配置仅需两个字段：`name_expr`（名称模板）和 `expand_exprs`（变量声明与计算）。

**核心工作流**：用户维护 YAML/JSON 配置 → 代码中通过 `dbhelper.For()` 获取 Executor → 直接 CRUD。

---

## 目录

- [1. 快速开始](#1-快速开始)
- [2. 表达式语法](#2-表达式语法)
  - [2.1 两种引用](#21-两种引用)
  - [2.2 name_expr（名称模板）](#22-name_expr名称模板)
  - [2.3 expand_exprs（变量声明与计算）](#23-expand_exprs变量声明与计算)
  - [2.4 内建函数](#24-内建函数)
  - [2.5 函数参数规则](#25-函数参数规则)
- [3. 配置场景](#3-配置场景)
  - [3.1 无分库分表（单库单表）](#31-无分库分表单库单表)
  - [3.2 单库分表（Hash-Mod）](#32-单库分表hash-mod)
  - [3.3 单库分表（直接取模）](#33-单库分表直接取模)
  - [3.4 分库分表（Hash-Mod）](#34-分库分表hash-mod)
  - [3.5 分库分表（不同列路由）](#35-分库分表不同列路由)
  - [3.6 按区域分库（枚举）](#36-按区域分库枚举)
  - [3.7 按区域分库（简写）](#37-按区域分库简写)
  - [3.8 多 DB Server 分库分表](#38-多-db-server-分库分表)
  - [3.9 Entity 级别覆写](#39-entity-级别覆写)
  - [3.10 连接池配置](#310-连接池配置)
- [4. 代码使用](#4-代码使用)
  - [4.1 Entity 定义](#41-entity-定义)
  - [4.2 初始化 DbManager](#42-初始化-dbmanager)
  - [4.3 获取 Executor](#43-获取-executor)
  - [4.4 ShardingKey 使用](#44-shardingkey-使用)
  - [4.5 Scatter-Gather（全分片查询）](#45-scatter-gather全分片查询)
- [5. 表达式进阶](#5-表达式进阶)
  - [5.1 多级计算（链式依赖）](#51-多级计算链式依赖)
  - [5.2 字符串函数](#52-字符串函数)
  - [5.3 自定义函数](#53-自定义函数)
- [6. 完整示例](#6-完整示例)

---

## 1. 快速开始

### 第一步：编写 YAML 配置

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
    db_sharding:
      name_expr: "order_${region}_db"
      expand_exprs:
        - "${region} := enum(SG, TH, ID)"
        - "${region} = @{region}"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"
```

### 第二步：定义 Entity

```go
type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}

func (*User) TableName() string { return "user_tab" }

type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64
}

func (*Order) TableName() string  { return "order_tab" }
func (*Order) DbKey() string      { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }
```

### 第三步：初始化并使用

```go
// 从 YAML 加载配置
var cfg dbhelper.DatabaseConfig
yaml.Unmarshal(configBytes, &cfg)

// 创建 DbManager 并设为全局默认
mgr := dbhelper.NewDbManager(cfg)
dbhelper.SetDefault(mgr)

// 获取 Executor
userExec := dbhelper.For(&User{})
orderExec := dbhelper.For(&Order{})

// 非分片：直接 CRUD
users, _ := userExec.Find(ctx, nil, nil)

// 分片：设置 ShardingKey 后 CRUD
sk := dbspi.NewShardingKey().
    SetVal("region", "SG").
    SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, _ := orderExec.Find(ctx, nil, nil)
```

---

## 2. 表达式语法

### 2.1 两种引用

`name_expr` 和 `expand_exprs` 使用不同的语法范围：

**name_expr 中**（名称模板）— 仅支持 `${var}` 变量引用：

| 语法 | 含义 | 示例 |
|------|------|------|
| `${var}` | **扩展变量** — 由 expand_exprs 计算 | `${idx}`, `${region}`, `${index}` |

**expand_exprs 中**（变量声明与计算）— 支持全部引用和函数：

| 语法 | 含义 | 示例 |
|------|------|------|
| `@{col}` | **列变量** — 从 ShardingKey 读取 | `@{shop_id}`, `@{region}` |
| `${var}` | **扩展变量** — 引用其他变量 | `${idx}`, `${idx2}` |
| `func(args)` | **函数调用** — 直接使用函数名 | `fill(${idx}, 8)`, `hash(@{shop_id})` |

支持的运算符：`+`, `-`, `*`, `/`, `%`（取模），以及括号 `()` 控制优先级。

### 2.2 name_expr（名称模板）

用于生成最终的数据库名或表名。**仅支持 `${var}` 引用**，所有计算逻辑放在 `expand_exprs` 中。

| 模板 | 变量值 | 结果 |
|------|--------|------|
| `order_${region}_db` | region="SG" | `order_SG_db` |
| `order_tab_${index}` | index="00000123" | `order_tab_00000123` |
| `order_db_${idx}` | idx=2 | `order_db_2` |

### 2.3 expand_exprs（变量声明与计算）

**声明 (`:=`)** — 启动时声明变量的所有可能值：

```yaml
expand_exprs:
  - "${region} := enum(SG, TH, ID)"    # 字符串枚举
  - "${idx}    := range(0, 1000)"      # 整数范围 [0, 1000)
```

用途：
- DB 分片：枚举所有数据库名，创建连接
- Table 分片：枚举所有表名，支持 scatter-gather

**计算 (`=`)** — 运行时根据 ShardingKey 计算变量值：

```yaml
expand_exprs:
  - "${region} = @{region}"                    # 直接传递列值
  - "${idx}    = @{shop_id} % 1000"            # 算术运算
  - "${idx}    = hash(@{shop_id}) % 1000"      # 函数调用
  - "${index}  = fill(${idx}, 8)"              # 格式化输出
```

同一个变量可同时有 `:=` 声明和 `=` 计算。多个计算表达式之间的依赖会**自动拓扑排序**。

### 2.4 内建函数

在 `expand_exprs` 中直接使用函数名调用：

| 函数 | 说明 | 示例 | 结果 |
|------|------|------|------|
| `fill(value, width)` | 零填充整数 | `fill(5, 8)` | `"00000005"` |
| `str(value)` | 转为字符串 | `str(42)` | `"42"` |
| `hash(value)` | FNV-1a 哈希 | `hash(@{shop_id})` | 确定性 int64 |
| `mod(a, b)` | 取模（= `a % b`）| `mod(100, 7)` | `2` |
| `div(a, b)` | 除法（= `a / b`）| `div(100, 7)` | `14` |
| `lower(value)` | 小写 | `lower(SG)` | `"sg"` |
| `upper(value)` | 大写 | `upper(sg)` | `"SG"` |
| `concat(a, b, ...)` | 拼接字符串 | `concat(hello, _, world)` | `"hello_world"` |

**枚举声明函数**（仅在 `:=` 中使用）：

| 函数 | 说明 | 示例 |
|------|------|------|
| `enum(v1, v2, ...)` | 字符串枚举 | `${region} := enum(SG, TH, ID)` |
| `range(start, end)` | 整数范围 [start, end) | `${idx} := range(0, 1000)` |

### 2.5 函数参数规则

- 裸标识符自动视为**字符串**：`lower(SG)` 中的 `SG` 等同于 `"SG"`
- 数字保持为**数字类型**：`fill(${idx}, 8)` 中的 `8` 是数字
- `@{col}` 和 `${var}` 保持原始类型
- 函数内部自动类型转换：需要数字时从字符串解析，解析失败则报错

---

## 3. 配置场景

### 3.1 无分库分表（单库单表）

```yaml
databases:
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db
```

```go
userExec := dbhelper.For(&User{})
users, _ := userExec.Find(ctx, nil, nil)
```

### 3.2 单库分表（Hash-Mod）

**场景**：order_tab 按 shop_id 哈希取模分 1000 张表，表名格式 `order_tab_00000000` 到 `order_tab_00000999`。

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
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"
```

```go
orderExec := dbhelper.For(&Order{})

sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, _ := orderExec.Find(ctx, nil, nil)
// → 查询 order_db.order_tab_XXXXXXXX
```

### 3.3 单库分表（直接取模）

不使用 hash，直接按 shop_id 取模：

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
        - "${idx} := range(0, 1000)"
        - "${idx} = @{shop_id} % 1000"
        - "${index} = fill(${idx}, 8)"
```

### 3.4 分库分表（Hash-Mod）

**场景**：4 个数据库（order_db_0 到 order_db_3），每库 1000 张表。

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
        - "${idx} = hash(@{shop_id}) % 4"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"
```

```go
orderExec := dbhelper.For(&Order{})

sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, _ := orderExec.Find(ctx, nil, nil)
// → 路由到 order_db_X.order_tab_XXXXXXXX
```

### 3.5 分库分表（不同列路由）

**场景**：DB 按 region 分，Table 按 shop_id 分。

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
        - "${idx} := range(0, 1000)"
        - "${idx2} = @{shop_id} / 1000"
        - "${idx} = ${idx2} % 1000"
        - "${index} = fill(${idx}, 8)"
```

```go
orderExec := dbhelper.For(&Order{})

sk := dbspi.NewShardingKey().
    SetVal("region", "SG").
    SetVal("shop_id", int64(123456789))
ctx := dbspi.WithShardingKey(context.Background(), sk)
orders, _ := orderExec.Find(ctx, nil, nil)
// → 路由到 order_SG_db.order_tab_00000456
// 因为 123456789 / 1000 = 123456, 123456 % 1000 = 456
```

### 3.6 按区域分库（枚举）

**场景**：按 region 名称拆分数据库，每个区域一个库。

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
        - "${region} := enum(SG, TH, ID, MY, PH, VN)"
        - "${region} = @{region}"
```

启动时自动创建 6 个数据库连接：`order_SG_db`, `order_TH_db`, `order_ID_db`, `order_MY_db`, `order_PH_db`, `order_VN_db`。

### 3.7 按区域分库（简写）

当 `:=` 声明的变量名与 `name_expr` 中的 `${var}` 同名，且该变量需要直接从同名列传递时，可省略 `= @{col}` 计算（系统自动推断）：

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
        - "${region} := enum(SG, TH, ID, MY, PH, VN)"
        # ${region} = @{region} 自动推断，无需手动编写
```

效果与 3.6 完全相同。

### 3.8 多 DB Server 分库分表

**场景**：不同数据库分片位于不同物理服务器。

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
      name_expr: "${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = hash(@{shop_id}) % 4"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"
```

> **注意**：使用 `servers` 时，`db_sharding.name_expr` 计算出的结果与 `servers[].key` 匹配进行路由。

### 3.9 Entity 级别覆写

**场景**：同一个数据库组中，不同的 Entity 使用不同的表分片规则。

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
        - "${idx} = hash(@{shop_id}) % 4"
    # 默认表分片：10 张表
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 10)"
        - "${idx} = hash(@{shop_id}) % 10"
        - "${index} = fill(${idx}, 8)"
    # 特定 Entity 覆写
    entity_rules:
      - tables: ["order_detail_tab"]
        table_sharding:
          name_expr: "order_detail_tab_${index}"
          expand_exprs:
            - "${idx} := range(0, 20)"
            - "${idx} = hash(@{shop_id}) % 20"
            - "${index} = fill(${idx}, 8)"
      - tables: ["order_log_tab"]
        table_sharding:
          name_expr: "order_log_tab_${index}"
          expand_exprs:
            - "${idx} := range(0, 50)"
            - "${idx} = hash(@{shop_id}) % 50"
            - "${index} = fill(${idx}, 4)"
```

```go
// Order 使用默认规则：4 库 × 10 表
orderExec := dbhelper.For(&Order{})

// OrderDetail 使用覆写规则：4 库 × 20 表
detailExec := dbhelper.For(&OrderDetail{})

// OrderLog 使用覆写规则：4 库 × 50 表
logExec := dbhelper.For(&OrderLog{})
```

### 3.10 连接池配置

```yaml
databases:
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    max_open_conns: 200
    max_idle_conns: 20
    conn_max_lifetime_seconds: 1800
    debug: true
    max_concurrency: 10                # scatter-gather 并发限制
    db_sharding:
      name_expr: "order_db_${idx}"
      expand_exprs:
        - "${idx} := range(0, 4)"
        - "${idx} = hash(@{shop_id}) % 4"
    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"
```

也可使用 DSN：

```yaml
databases:
  default:
    dsn: "root:pass@tcp(10.0.0.1:3306)/my_app_db?charset=utf8mb4&parseTime=True&loc=Local"
    max_open_conns: 200
```

> **注意**：DSN 模式不能与 `db_sharding` 同时使用（因为 DSN 包含数据库名）。分库场景请使用 Host/Port/User/Password 字段或 `servers` 列表。

---

## 4. 代码使用

### 4.1 Entity 定义

每个 Entity 需实现 `TableName()` 方法。分片 Entity 还需实现 `DbKey()` 声明所属数据库组：

```go
// 非分片 Entity — 不实现 DbKey()，使用 "default" 数据库
type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}

func (*User) TableName() string  { return "user_tab" }
func (*User) IdFiledName() string { return "id" }

// 分片 Entity — 实现 DbKey() 路由到指定数据库组
type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64
}

func (*Order) TableName() string  { return "order_tab" }
func (*Order) DbKey() string      { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }
```

### 4.2 初始化 DbManager

```go
// 从 YAML 配置文件加载
var cfg dbhelper.DatabaseConfig
if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
    panic(err)
}

mgr := dbhelper.NewDbManager(cfg)

// 设为全局默认（可选，推荐）
dbhelper.SetDefault(mgr)
```

### 4.3 获取 Executor

```go
// 方式一：使用全局默认 DbManager
userExec := dbhelper.For(&User{})
orderExec := dbhelper.For(&Order{})

// 方式二：指定 DbManager
userExec := dbhelper.For(&User{}, mgr)
orderExec := dbhelper.For(&Order{}, mgr)
```

Entity 通过 `DbKey()` 自动路由到对应的数据库组。不实现 `DbKey()` 的 Entity 使用 `"default"` 组。

### 4.4 ShardingKey 使用

**方式一：通过 Context 注入**（推荐，适合在中间件/拦截器中统一设置）

```go
sk := dbspi.NewShardingKey().
    SetVal("region", "SG").
    SetVal("shop_id", int64(12345))
ctx := dbspi.WithShardingKey(context.Background(), sk)

// 后续操作自动路由
orders, _ := orderExec.Find(ctx, nil, nil)
order, _ := orderExec.GetById(ctx, 1001)
err := orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
```

**方式二：通过 Shard() 方法**（适合在同一分片上执行多次操作）

```go
sk := dbspi.NewShardingKey().SetVal("shop_id", int64(12345))
shardExec, err := orderExec.Shard(sk)
if err != nil {
    return err
}

// 在同一个分片上执行多次操作
order, _ := shardExec.GetById(ctx, 1001)
orders, _ := shardExec.Find(ctx, query, nil)
err = shardExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
```

**使用 Column 引用**（类型安全）

```go
var ShopID = dbhelper.NewColumn("shop_id")
var Region = dbhelper.NewColumn("region")

sk := dbspi.NewShardingKey().
    Set(Region, "SG").
    Set(ShopID, int64(12345))
```

支持的值类型：`int64`, `int`, `int32`, `uint64`, `uint32`, `string`。

### 4.5 Scatter-Gather（全分片查询）

跨所有分片查询，不需要 ShardingKey：

```go
ctx := context.Background()

// 查询所有分片的所有数据
allOrders, _ := orderExec.FindAll(ctx, query, 0)

// 带批次控制（每个分片每次查 100 条，防止单次返回过多数据）
allOrders, _ := orderExec.FindAll(ctx, query, 100)

// 统计所有分片的总数
totalCount, _ := orderExec.CountAll(ctx, query)
```

> Scatter-Gather 依赖 `:= range()` 或 `:= enum()` 声明来枚举所有可能的分片。配置了 `max_concurrency` 可限制并发数。

---

## 5. 表达式进阶

### 5.1 多级计算（链式依赖）

表达式之间的依赖会自动拓扑排序，书写顺序不影响执行顺序：

```yaml
table_sharding:
  name_expr: "order_tab_${index}"
  expand_exprs:
    - "${idx} := range(0, 1000)"
    - "${idx} = ${idx2} % 1000"       # 依赖 idx2
    - "${idx2} = @{shop_id} / 1000"   # 先计算 idx2
    - "${index} = fill(${idx}, 8)"    # 格式化输出
```

执行顺序（自动排序）：
1. `${idx2} = @{shop_id} / 1000` → idx2 = 123456
2. `${idx} = ${idx2} % 1000` → idx = 456
3. `${index} = fill(${idx}, 8)` → index = "00000456"

结果：`order_tab_00000456`

### 5.2 字符串函数

```yaml
# 转小写区域名
db_sharding:
  name_expr: "order_${tag}_db"
  expand_exprs:
    - "${region} := enum(SG, TH, ID)"
    - "${tag} = lower(@{region})"

# region="SG" → order_sg_db
```

### 5.3 自定义函数

注册自定义函数扩展表达式能力：

```go
import "github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"

func init() {
    expr.RegisterFunc("crc32", func(args []expr.Value) (expr.Value, error) {
        if len(args) != 1 {
            return expr.Value{}, fmt.Errorf("crc32() expects 1 argument")
        }
        h := crc32.ChecksumIEEE([]byte(args[0].String()))
        return expr.IntValue(int64(h)), nil
    })
}
```

然后在 YAML 中使用：

```yaml
table_sharding:
  name_expr: "order_tab_${index}"
  expand_exprs:
    - "${idx} := range(0, 1000)"
    - "${idx} = crc32(@{shop_id}) % 1000"
    - "${index} = fill(${idx}, 8)"
```

---

## 6. 完整示例

### YAML 配置

```yaml
databases:
  # 非分片数据库
  default:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    db_name: my_app_db
    max_open_conns: 100
    max_idle_conns: 10

  # 分库分表数据库组
  order_dbs:
    host: 10.0.0.1
    port: 3306
    user: root
    password: secret
    max_open_conns: 200
    max_idle_conns: 20
    conn_max_lifetime_seconds: 1800
    max_concurrency: 10

    db_sharding:
      name_expr: "order_${region}_db"
      expand_exprs:
        - "${region} := enum(SG, TH, ID)"
        - "${region} = @{region}"

    table_sharding:
      name_expr: "order_tab_${index}"
      expand_exprs:
        - "${idx} := range(0, 1000)"
        - "${idx} = hash(@{shop_id}) % 1000"
        - "${index} = fill(${idx}, 8)"

    entity_rules:
      - tables: ["order_detail_tab"]
        table_sharding:
          name_expr: "order_detail_tab_${index}"
          expand_exprs:
            - "${idx} := range(0, 100)"
            - "${idx} = hash(@{shop_id}) % 100"
            - "${index} = fill(${idx}, 8)"
```

### Go 代码

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

// ========== Entity 定义 ==========

type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}
func (*User) TableName() string  { return "user_tab" }
func (*User) IdFiledName() string { return "id" }

type Order struct {
    ID     int64 `gorm:"primaryKey"`
    ShopID int64 `gorm:"column:shop_id"`
    Amount int64
}
func (*Order) TableName() string  { return "order_tab" }
func (*Order) DbKey() string      { return "order_dbs" }
func (*Order) IdFiledName() string { return "id" }

type OrderDetail struct {
    ID      int64 `gorm:"primaryKey"`
    OrderID int64 `gorm:"column:order_id"`
    ShopID  int64 `gorm:"column:shop_id"`
}
func (*OrderDetail) TableName() string  { return "order_detail_tab" }
func (*OrderDetail) DbKey() string      { return "order_dbs" }
func (*OrderDetail) IdFiledName() string { return "id" }

// ========== 初始化 ==========

func main() {
    // 加载配置
    data, _ := os.ReadFile("config.yaml")
    var cfg dbhelper.DatabaseConfig
    yaml.Unmarshal(data, &cfg)

    // 初始化 DbManager
    mgr := dbhelper.NewDbManager(cfg)
    dbhelper.SetDefault(mgr)

    ctx := context.Background()

    // ========== 非分片操作 ==========

    userExec := dbhelper.For(&User{})
    users, _ := userExec.Find(ctx, nil, nil)
    fmt.Printf("Users: %d\n", len(users))

    // ========== 分片操作 ==========

    orderExec := dbhelper.For(&Order{})
    detailExec := dbhelper.For(&OrderDetail{})

    // 设置 ShardingKey
    sk := dbspi.NewShardingKey().
        SetVal("region", "SG").
        SetVal("shop_id", int64(12345))
    shardCtx := dbspi.WithShardingKey(ctx, sk)

    // 路由到 order_SG_db.order_tab_XXXXXXXX
    orders, _ := orderExec.Find(shardCtx, nil, nil)
    fmt.Printf("Orders: %d\n", len(orders))

    // 路由到 order_SG_db.order_detail_tab_XXXXXXXX（Entity 覆写规则）
    details, _ := detailExec.Find(shardCtx, nil, nil)
    fmt.Printf("Details: %d\n", len(details))

    // 创建订单
    _ = orderExec.Create(shardCtx, &Order{ShopID: 12345, Amount: 100})

    // ========== Scatter-Gather ==========

    totalCount, _ := orderExec.CountAll(ctx, nil)
    fmt.Printf("Total orders across all shards: %d\n", totalCount)

    allOrders, _ := orderExec.FindAll(ctx, nil, 100)
    fmt.Printf("All orders: %d\n", len(allOrders))
}
```
