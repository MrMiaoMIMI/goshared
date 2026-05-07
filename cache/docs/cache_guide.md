# Cache Module Guide

本指南介绍 `cache` 模块的用户侧用法。`cache` 模块以 `CacheConfig -> Cache` 为主线，支持本地内存缓存和 Redis 缓存，并提供统一的 `Get`、`Set`、`SetNX`、批量读写、删除、loader 回源等能力。

核心流程：

1. 在配置文件中维护 `cachespi.CacheConfig`。
2. 通过 `cachehelper.NewCache` 初始化 `cachespi.Cache`。
3. 业务代码依赖 `cachespi.Cache` 接口，不关心底层是 in-memory 还是 Redis。
4. 缓存策略优先写入配置文件；`cachehelper` 不提供 `WithXxx` option function。

---

## 目录

- [1. 包定位](#1-包定位)
- [2. 快速开始](#2-快速开始)
- [3. 配置说明](#3-配置说明)
- [4. 基础操作](#4-基础操作)
- [5. Loader 回源](#5-loader-回源)
- [6. TTL 规则](#6-ttl-规则)
- [7. In-Memory 与 Redis 差异](#7-in-memory-与-redis-差异)
- [8. 注意事项](#8-注意事项)

---

## 1. 包定位

`cache` 模块的用户侧 API 分布在两个 package：

| Package | 定位 | 常用内容 |
|---------|------|----------|
| `cachespi` | 稳定契约和配置类型 | `Cache`、`CacheConfig`、`InMemoryConfig`、`RedisConfig`、`DataLoader`、`DefaultExpiration`、`NoExpiration` |
| `cachehelper` | 工厂方法 | `NewCache`、`NewInMemCache`、`NewRedisCache` |

一般业务代码只需要：

- 用 `cachespi` 声明配置、接口和 TTL 常量。
- 用 `cachehelper.NewCache` 根据配置创建缓存对象。
- 在配置文件里选择 backend、连接信息、默认 TTL、codec 等策略。

---

## 2. 快速开始

推荐业务服务使用统一配置入口：

```yaml
cache:
  backend: redis
  redis:
    addr: 127.0.0.1:6379
    db: 0
    default_ttl: 10m
    codec: json
```

```go
type AppConfig struct {
    Cache cachespi.CacheConfig `yaml:"cache" json:"cache"`
}

var cfg AppConfig
// 使用项目已有配置加载逻辑填充 cfg，例如 yaml.Unmarshal。

cache, err := cachehelper.NewCache(cfg.Cache)
if err != nil {
    return err
}
defer cache.Close(context.Background())

err = cache.Set(ctx, "product:10001", Product{ID: 10001, Name: "Phone"}, cachespi.DefaultExpiration)
if err != nil {
    return err
}

var product Product
err = cache.Get(ctx, "product:10001", &product)
```

测试、小工具或明确只使用某个 backend 的代码，也可以直接调用：

```go
cache, err := cachehelper.NewInMemCache(cachespi.InMemoryConfig{
    DefaultTTL: 5 * time.Minute,
    Codec:      cachespi.CodecJSON,
})
```

---

## 3. 配置说明

### 3.1 统一配置

```yaml
cache:
  backend: in_memory
  in_memory:
    default_ttl: 5m
    num_counters: 1000000
    max_cost: 268435456
    buffer_items: 64
    codec: json
```

`backend` 可选值：

| 值 | 说明 |
|----|------|
| `in_memory` | 使用当前进程内的 Ristretto 缓存 |
| `redis` | 使用 Redis 缓存 |

`backend` 必须显式配置，避免不同环境误用默认 backend。

### 3.2 InMemoryConfig

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `default_ttl` | `5m` | 调用方传 `cachespi.DefaultExpiration` 时使用的默认 TTL |
| `num_counters` | `1e7` | Ristretto admission policy 使用的 counter 数量，通常约等于预期 key 数量的 10 倍 |
| `max_cost` | `1 GiB` | 缓存总 cost 上限；当前每个 key 写入 cost 为 1 |
| `buffer_items` | `64` | Ristretto ring buffer 大小 |
| `codec` | 空 | 空表示直接缓存 Go 值引用；`json` 表示用 JSON 序列化后存储 |

### 3.3 RedisConfig

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `addr` | `localhost:6379` | Redis 地址 |
| `password` | 空 | Redis 密码 |
| `db` | `0` | Redis database index |
| `pool_size` | `10` | 连接池最大连接数 |
| `min_idle_conns` | `2` | 最小空闲连接数 |
| `dial_timeout` | `5s` | 建连超时 |
| `read_timeout` | `3s` | 读超时 |
| `write_timeout` | `3s` | 写超时 |
| `default_ttl` | `5m` | 调用方传 `cachespi.DefaultExpiration` 时使用的默认 TTL |
| `codec` | `json` | Redis 存储 bytes，当前配置化入口支持 `json` |

---

## 4. 基础操作

### 4.1 Get / Set

```go
err := cache.Set(ctx, "user:1", user, cachespi.DefaultExpiration)
if err != nil {
    return err
}

var got User
err = cache.Get(ctx, "user:1", &got)
if errors.Is(err, cachespi.ErrCacheMiss) {
    // 缓存不存在，按业务需要回源。
}
```

`receiver` 必须是非 nil 指针。

### 4.2 SetNX

```go
ok, err := cache.SetNX(ctx, "job:lock:10001", "locked", 30*time.Second)
if err != nil {
    return err
}
if !ok {
    return nil
}
```

`SetNX` 只在 key 不存在时写入。内存缓存的原子性只在当前进程内有效；Redis 缓存在同一个 Redis 部署内对多客户端有效。

### 4.3 批量读写

```go
err := cache.SetMany(ctx, map[string]any{
    "user:1": user1,
    "user:2": user2,
}, cachespi.DefaultExpiration)
if err != nil {
    return err
}

var got1, got2 User
receivers := map[string]any{
    "user:1": &got1,
    "user:2": &got2,
}
err = cache.GetMany(ctx, receivers)
```

`GetMany` 成功返回时，会从 `receivers` 中删除 miss 的 key。只在 error 为 nil 时依赖批量结果。

### 4.4 删除

```go
err := cache.Delete(ctx, "user:1")
if errors.Is(err, cachespi.ErrCacheMiss) {
    return nil
}

err = cache.DeleteMany(ctx, []string{"user:1", "user:2"})
```

`Delete` 删除不存在的 key 会返回 `ErrCacheMiss`；`DeleteMany` 会忽略不存在的 key。

---

## 5. Loader 回源

`Load` 和 `LoadMany` 适合把“读缓存，不存在再查下游并写回缓存”的逻辑收敛到 cache 模块。

```go
loader := func(ctx context.Context, keys []string) ([]any, error) {
    results := make([]any, len(keys))
    for i, key := range keys {
        user, err := userStore.GetByCacheKey(ctx, key)
        if err != nil {
            return nil, err
        }
        if user != nil {
            results[i] = user
        }
    }
    return results, nil
}

var user User
err := cache.Load(ctx, loader, "user:1", &user, cachespi.DefaultExpiration)
```

loader 返回值必须与入参 keys 按下标对齐。某个位置返回 nil 表示该 key 下游也不存在，不会写入缓存。

---

## 6. TTL 规则

每次写入都需要传入 `expire`：

| 值 | 说明 |
|----|------|
| `cachespi.DefaultExpiration` | 使用配置里的 `default_ttl` |
| `cachespi.NoExpiration` | 不主动过期，只能被删除或被 backend 淘汰 |
| 正数 duration | 使用本次写入指定的 TTL |

除 `cachespi.NoExpiration` 之外的负数 duration 会返回 `cachespi.ErrInvalidExpiration`。

---

## 7. In-Memory 与 Redis 差异

| 维度 | In-Memory | Redis |
|------|-----------|-------|
| 存储位置 | 当前进程内 | Redis 服务 |
| 默认 codec | 空，直接保存 Go 值引用 | `json` |
| 引用语义 | 默认存在；修改原对象、map、slice 或返回的指针可能影响缓存值 | 不存在；跨网络存储 bytes |
| 适用场景 | 单进程本地缓存、测试、轻量热数据 | 多实例共享缓存、分布式锁语义、跨进程数据 |

如果希望 in-memory 缓存和 Redis 一样具备 copy semantics，配置：

```yaml
cache:
  backend: in_memory
  in_memory:
    codec: json
```

---

## 8. 注意事项

- 业务服务优先使用 `cachehelper.NewCache(cfg)`，不要在代码里分散维护 backend 选择和 TTL 策略。
- `Cache` 对象应在服务启动时创建并复用，不要在每次请求中重复创建。
- Redis backend 会创建并持有 Redis client，服务退出时调用 `Close(ctx)`。
- `Load` / `LoadMany` 遇到缓存连接错误、反序列化错误时不会回源，避免把真实缓存故障伪装成 miss。
- 缓存 key 命名建议包含业务对象和 ID，例如 `product:10001`、`user:123:profile`，避免不同业务复用同一个 key 空间。
