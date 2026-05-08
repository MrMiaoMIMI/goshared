# Logger Module Guide

logger 模块用于统一输出结构化日志，并通过 `context.Context` 自动携带请求
`trace_id`。

使用方通常只需要关注四件事：

- 启动时配置全局 logger
- 业务代码中输出日志
- 请求入口设置 trace_id
- 按组件或任务创建带固定字段的 logger

## 配置全局 Logger

服务启动时调用 `Configure`：

```go
err := logger.Configure(logger.Config{
    Level:       logger.LevelInfo,
    Encoding:    logger.EncodingJSON,
    OutputPaths: []string{logger.OutputStdout, "/var/log/app.log"},
})
if err != nil {
    return err
}
defer func() { _ = logger.Sync() }()
```

配置文件示例：

```yaml
logger:
  level: info
  development: false
  encoding: json
  output_paths:
    - stdout
    - /var/log/app.log
```

配置项说明：

- `level`: 最低输出级别，支持 `debug`、`info`、`warn`、`error`、`dpanic`、`panic`、`fatal`；为空时使用 `info`
- `development`: 开发模式；开启后 `DPanic` 会触发 panic
- `encoding`: 输出格式，支持 `json` 和 `console`；为空时使用 `json`
- `output_paths`: 输出目标，支持 `stdout`、`stderr` 或文件路径；为空时使用 `stdout`

如果启动时希望配置错误直接 panic，可以使用 `Init(cfg)`。普通服务更推荐
`Configure(cfg)`，由启动流程处理返回的 error。

## 输出日志

普通日志调用直接使用包级方法：

```go
logger.Info(ctx, "order created",
    logger.Int64("order_id", orderID),
    logger.String("region", region),
)
```

错误日志使用 `Err`：

```go
if err != nil {
    logger.Error(ctx, "create order failed", logger.Err(err))
    return err
}
```

优先使用 `String`、`Int64`、`Bool`、`Duration` 等具体类型字段。
只有在没有合适字段构造函数时，再使用 `Any`。

## 设置 Trace ID

请求入口设置一次 trace_id：

```go
ctx = logger.SetTraceID(ctx, requestID)
```

后续所有使用这个 `ctx` 的日志都会自动带上 `trace_id`：

```go
logger.Info(ctx, "request received", logger.String("path", "/api/orders"))
```

## 创建带固定字段的 Logger

如果某个组件、任务或 worker 的日志都需要携带相同字段，可以创建一个
`Logger` 复用：

```go
workerLog := logger.WithFields(logger.String("component", "worker"))

workerLog.Info(ctx, "job started", logger.String("job", jobName))
workerLog.Info(ctx, "job finished", logger.String("job", jobName))
```

`WithFields` 创建的 logger 仍然会读取 `ctx` 中的 `trace_id`。

## 格式化日志

需要使用格式化字符串时，可以通过 `WithContext` 创建格式化 logger：

```go
log := logger.WithContext(ctx)
log.Infof("job %s finished", jobName)
```

结构化日志更利于检索和聚合。除本地调试外，优先使用 `Info(ctx, msg, fields...)`
这类结构化方法。
