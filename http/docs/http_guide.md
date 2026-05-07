# HTTP Module Guide

本指南介绍 `http` 模块的用户侧用法。`http` 模块以 `httpspi.ClientConfig -> httpspi.Client` 为主线，面向业务服务提供配置化 HTTP client 创建能力；每次请求再通过 `Client.New()` 设置 path、query、header、body 和 response receiver。

核心流程：

1. 在配置文件中维护 `httpspi.ClientConfig` 或 `httpspi.ClientConfigs`。
2. 通过 `httphelper.NewClient` 或 `httphelper.NewClients` 初始化 `httpspi.Client`。
3. 业务代码依赖 `httpspi.Client` 接口，不关心底层 `net/http` 实现。
4. client 创建策略优先写入配置文件；`httphelper` 不提供 `WithXxx` option function。

---

## 目录

- [1. 包定位](#1-包定位)
- [2. 快速开始](#2-快速开始)
- [3. 配置说明](#3-配置说明)
- [4. 发送请求](#4-发送请求)
- [5. 错误与响应](#5-错误与响应)
- [6. 使用建议](#6-使用建议)

---

## 1. 包定位

`http` 模块的用户侧 API 分布在两个 package：

| Package | 定位 | 常用内容 |
|---------|------|----------|
| `httpspi` | 稳定契约和配置类型 | `Client`、`Clients`、`ClientConfig`、`ClientConfigs`、`RetryConfig`、`Response`、`StatusError`、`ResponseDecoder` |
| `httphelper` | 工厂方法 | `NewClient`、`NewClients` |

一般业务代码只需要：

- 用 `httpspi` 声明配置和依赖接口。
- 用 `httphelper.NewClient` 根据配置创建单个 client。
- 用 `httphelper.NewClients` 根据 `httpspi.ClientConfigs` 创建多个命名 client。
- 每次发送请求前调用 `client.New()`，避免请求状态在多次调用之间串用。

---

## 2. 快速开始

推荐业务服务使用统一配置入口：

```yaml
http:
  clients:
    user_service:
      base_url: https://user.example.com
      default_timeout: 5s
      default_headers:
        X-App: order-service
      retry:
        max_retries: 2
        delay: 200ms
    payment_service:
      base_url: https://payment.example.com
      default_timeout: 3s
```

```go
type AppConfig struct {
    HTTP struct {
        Clients httpspi.ClientConfigs `yaml:"clients" json:"clients"`
    } `yaml:"http" json:"http"`
}

var cfg AppConfig
// 使用项目已有配置加载逻辑填充 cfg，例如 yaml.Unmarshal。

clients, err := httphelper.NewClients(cfg.HTTP.Clients)
if err != nil {
    return err
}

userClient := clients["user_service"]

var user User
resp, err := userClient.New().
    Get("/api/v1/users/10001").
    QueryParam("region", "SG").
    Request(ctx, &user, nil)
if err != nil {
    return err
}
_ = resp.StatusCode
```

如果应用只需要一个 HTTP client，也可以直接创建：

```go
client, err := httphelper.NewClient(httpspi.ClientConfig{
    BaseURL:        "https://user.example.com",
    DefaultTimeout: 5 * time.Second,
})
```

---

## 3. 配置说明

### 3.1 ClientConfig

```yaml
base_url: https://user.example.com
default_timeout: 5s
default_headers:
  X-App: order-service
retry:
  max_retries: 2
  delay: 200ms
```

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `base_url` | 空 | 请求 path 的基础 URL；如果每次请求都使用绝对 URL，可以不配置 |
| `default_timeout` | `30s` | `Client.Request` 使用的默认超时时间；`Receive` 可单次传入 timeout |
| `default_headers` | 空 | 每次请求自动携带的 header |
| `retry.max_retries` | `0` | 网络错误和 5xx 响应的重试次数；0 表示不重试 |
| `retry.delay` | `100ms` | 重试间隔；启用重试但不配置时使用默认值 |

`base_url` 如果配置，必须包含 scheme 和 host，例如 `https://user.example.com`。

---

## 4. 发送请求

### 4.1 GET + Query

```go
var out SearchUserResponse
_, err := client.New().
    Get("/api/v1/users").
    QueryParam("keyword", "alice").
    QueryParam("page", "1").
    Request(ctx, &out, nil)
```

也可以用 struct tag 生成 query：

```go
type SearchUserRequest struct {
    Keyword string `url:"keyword"`
    Page    int    `url:"page"`
}

_, err := client.New().
    Get("/api/v1/users").
    QueryStruct(&SearchUserRequest{Keyword: "alice", Page: 1}).
    Request(ctx, &out, nil)
```

### 4.2 POST JSON

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var out CreateUserResponse
_, err := client.New().
    Post("/api/v1/users").
    BodyJSON(&CreateUserRequest{Name: "Alice", Email: "alice@example.com"}).
    Request(ctx, &out, nil)
```

### 4.3 Header 和鉴权

默认 header 放在配置文件中。单次请求的动态 header 在请求链路中设置：

```go
_, err := client.New().
    Get("/api/v1/profile").
    BearerToken(accessToken).
    Set("X-Request-ID", requestID).
    Request(ctx, &out, &apiErr)
```

### 4.4 RequestStruct

`RequestStruct` 可以同时从一个 struct 中提取 query、header 和 JSON body：

```go
type CreateOrderRequest struct {
    ShopID  int64  `url:"shop_id"`
    TraceID string `header:"X-Trace-ID"`
    ItemID  int64  `json:"item_id"`
    Count   int    `json:"count"`
}

_, err := client.New().
    Post("/api/v1/orders").
    RequestStruct(&CreateOrderRequest{
        ShopID:  10001,
        TraceID: traceID,
        ItemID:  20002,
        Count:   1,
    }).
    Request(ctx, &out, &apiErr)
```

---

## 5. 错误与响应

`Request` 会自动读取并关闭 response body，并返回轻量的 `httpspi.Response`：

```go
var out UserResponse
var apiErr APIError

resp, err := client.New().Get("/api/v1/users/10001").Request(ctx, &out, &apiErr)
if err != nil {
    var statusErr httpspi.StatusError
    if errors.As(err, &statusErr) {
        // 非 2xx 响应。apiErr 已按 JSON 尝试解码。
        return fmt.Errorf("upstream status=%d message=%s", statusErr.StatusCode, apiErr.Message)
    }
    return err
}

_ = resp.Header.Get("X-Request-ID")
```

如果调用方需要直接读取原始 `*http.Response` 或自定义单次 timeout，可以使用 `Receive`。当 successV/failureV 为 nil 时，调用方需要自行读取并关闭 response body。

---

## 6. 使用建议

- client 创建只使用 `httpspi.ClientConfig`，不要在业务代码中堆叠 option function。
- 一个上游服务对应一个共享 client；每次请求必须从 `client.New()` 开始。
- 固定的 base URL、默认 timeout、默认 header、retry 策略放入配置文件。
- 动态 token、trace id、请求参数和 body 放在单次请求链路中。
- 业务服务依赖 `httpspi.Client` 接口，初始化层依赖 `httphelper`。
