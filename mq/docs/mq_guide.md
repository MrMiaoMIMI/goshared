# MQ Module Guide

本指南介绍 `mq` 模块的用户侧用法。`mq` 模块当前以 Kafka/Sarama 为底层实现，面向业务代码提供 producer、manual consumer、advanced consumer，以及 processor 失败策略。

核心流程：

1. 在配置文件中维护 `ProducerConfig` 或 `ConsumerConfig`，或在测试/小程序中直接写 struct literal。
2. 通过 `mqhelper` 创建 Producer 或 Consumer。
3. Producer 负责同步、批量或异步发送消息。
4. Manual Consumer 由业务主动 `Consume` 并在处理成功后 `Ack`。
5. Advanced Consumer 由业务实现 processor，模块负责消费循环、成功提交和失败策略。

---

## 目录

- [1. 包定位](#1-包定位)
- [2. 快速开始：Producer](#2-快速开始producer)
- [3. Producer 用法](#3-producer-用法)
- [4. Consumer 用法](#4-consumer-用法)
- [5. AdvancedConsumer 用法](#5-advancedconsumer-用法)
- [6. Processor 失败策略](#6-processor-失败策略)
- [7. 配置说明](#7-配置说明)
- [8. Integration Test](#8-integration-test)
- [9. 注意事项](#9-注意事项)

---

## 1. 包定位

`mq` 模块的用户侧 API 分布在两个 package：

| Package | 定位 | 常用内容 |
|---------|------|----------|
| `mqspi` | 稳定契约和配置类型 | `Producer`、`Consumer`、`AdvancedConsumer`、`ProducerConfig`、`ConsumerConfig`、`ConsumerFailurePolicy` |
| `mqhelper` | 运行时工厂和消息 helper | `NewProducer`、`NewConsumer`、`NewAdvancedConsumer`、`NewProducerMessage` |

一般业务代码只需要：

- 用 `mqspi` 声明接口、配置、credentials、processor 和 failure policy。
- 用 `mqhelper` 创建 producer、consumer 和消息。
- 可配置策略优先写入 `ProducerConfig` / `ConsumerConfig`，由配置文件维护；`mqhelper` 不为这些字段额外提供 option function。
- `mqhelper` 不提供 `NewXxxConfig` / `NewCredentials`，避免配置字段同时散落在配置文件和业务代码里。

---

## 2. 快速开始：Producer

```go
cfg := &mqspi.ProducerConfig{
    Brokers: []string{"127.0.0.1:9092"},
    Topic:   "order-event-topic",
}

producer, err := mqhelper.NewProducer(cfg)
if err != nil {
    return err
}
defer producer.Close(context.Background())

msg := mqhelper.NewProducerMessage(
    "",
    []byte("order-10001"),
    []byte(`{"order_id":10001}`),
)

if err := producer.Produce(context.Background(), msg); err != nil {
    return err
}
```

`ProducerConfig.Topic` 是默认 topic。`ProducerMessage.Topic` 为空时使用默认 topic；如果消息上显式设置 topic，则优先使用消息上的 topic。

---

## 3. Producer 用法

### 3.1 同步发送

```go
msg := mqhelper.NewProducerMessage("", []byte("key"), []byte("value"))
err := producer.Produce(ctx, msg)
```

发送成功后，`msg.Partition`、`msg.Offset`、`msg.Timestamp` 会被填充。

### 3.2 批量发送

```go
msgs := []*mqspi.ProducerMessage{
    mqhelper.NewProducerMessage("", []byte("key-1"), []byte("value-1")),
    mqhelper.NewProducerMessage("", []byte("key-2"), []byte("value-2")),
}

err := producer.BatchProduce(ctx, msgs)
```

`BatchProduce` 使用底层 producer 的批量接口。只要返回 error，就表示本批次发送未整体成功，业务应按自己的幂等策略决定是否重试。

### 3.3 异步发送

```go
msg := mqhelper.NewProducerMessage("", []byte("key"), []byte("value"))

err := producer.AsyncProduce(ctx, msg, func(ctx context.Context, msg *mqspi.ProducerMessage, err error) {
    if err != nil {
        // 发送失败，记录日志或做业务补偿。
        return
    }
    // 发送成功，msg.Partition / msg.Offset 已填充。
})
```

`AsyncProduce` 返回的是“入队结果”。真正发送成功或失败会通过 callback 返回。传入的 `ctx` 会控制入队等待，避免 producer 输入队列阻塞时调用方一直卡住。

### 3.4 JSON 消息

```go
msg, err := mqhelper.NewJSONProducerMessage("", []byte("order-10001"), map[string]any{
    "order_id": 10001,
})
if err != nil {
    return err
}
err = producer.Produce(ctx, msg)
```

---

## 4. Consumer 用法

Manual Consumer 适合业务希望自己控制消费循环和 ack 的场景。

```go
cfg := &mqspi.ConsumerConfig{
    Brokers: []string{"127.0.0.1:9092"},
    Topic:   "order-event-topic",
    GroupID: "order-service-group",
}

consumer, err := mqhelper.NewConsumer(cfg)
if err != nil {
    return err
}
defer consumer.Close(context.Background())

for {
    msg, err := consumer.Consume(ctx)
    if err != nil {
        return err
    }

    if err := handleMessage(ctx, msg); err != nil {
        // 不 Ack，消息会保留为未提交状态，后续由 Kafka 重新投递。
        continue
    }

    if err := consumer.Ack(ctx, msg); err != nil {
        return err
    }
}
```

`Ack` 只能用于从当前 consumer `Consume` 出来的消息，因为消息内部携带了实现层保留的 metadata。不要修改 `ConsumerMessage.Metadata`。

---

## 5. AdvancedConsumer 用法

Advanced Consumer 适合把消费循环交给模块，业务只实现 processor 的场景。

### 5.1 单条 processor

```go
type OrderProcessor struct{}

func (p *OrderProcessor) Process(ctx context.Context, msg *mqspi.ConsumerMessage) error {
    return handleOrderEvent(ctx, msg.Value)
}

cfg := &mqspi.ConsumerConfig{
    Brokers: []string{"127.0.0.1:9092"},
    Topic:   "order-event-topic",
    GroupID: "order-service-group",
}

consumer, err := mqhelper.NewAdvancedConsumer(cfg, &OrderProcessor{})
if err != nil {
    return err
}
defer consumer.Close(context.Background())

return consumer.Run(ctx)
```

`Process` 返回 nil 时，模块会提交 offset。`Process` 返回 error 时，模块会进入失败策略，不会直接提交 offset。

### 5.2 批量 processor

```go
type OrderBatchProcessor struct{}

func (p *OrderBatchProcessor) BatchProcess(ctx context.Context, msgs []*mqspi.ConsumerMessage) error {
    return handleOrderEvents(ctx, msgs)
}

consumer, err := mqhelper.NewAdvancedBatchConsumer(cfg, &OrderBatchProcessor{})
if err != nil {
    return err
}
defer consumer.Close(context.Background())

return consumer.Run(ctx)
```

当前 batch size 使用模块默认值。批量处理成功时整批提交；批量处理失败时整批进入失败策略。

---

## 6. Processor 失败策略

`AdvancedConsumer` 支持 consumer failure policy。默认策略是：

- `retry forever`
- 初始退避 `1s`
- 指数退避倍数 `2.0`
- 最大退避 `30s`
- 不提交失败消息 offset
- 不内置 DLQ

这能避免 processor 偶发错误导致整个 consumer 退出，也避免失败消息被静默跳过。

### 6.1 配置最大尝试次数

```go
cfg.FailurePolicy = &mqspi.ConsumerFailurePolicy{
    MaxAttempts: 3,
    FinalAction: mqspi.ConsumerFailureActionStop,
}
```

上例表示：失败后最多尝试 3 次，达到次数后停止 `Run` 并返回错误。

等价 YAML 配置：

```yaml
failure_policy:
  max_attempts: 3
  final_action: stop
```

### 6.2 跳过毒消息

```go
cfg.FailurePolicy = &mqspi.ConsumerFailurePolicy{
    MaxAttempts: 5,
    FinalAction: mqspi.ConsumerFailureActionSkip,
}
```

`skip` 会提交失败消息或失败批次的 offset，然后继续消费后续消息。它适合业务明确接受丢弃毒消息的场景。使用前必须确认不会造成不可接受的数据丢失。

等价 YAML 配置：

```yaml
failure_policy:
  max_attempts: 5
  final_action: skip
```

### 6.3 自定义退避和观测回调

退避参数建议放在配置文件里：

```yaml
failure_policy:
  initial_backoff: 500ms
  max_backoff: 10s
  backoff_multiplier: 1.5
```

如果需要在失败时记录日志、metrics 或告警，可以在加载配置后设置 `Handler`。`Handler` 是代码侧回调，不会被 JSON/YAML 序列化。

```go
if cfg.FailurePolicy == nil {
    cfg.FailurePolicy = mqspi.DefaultConsumerFailurePolicy()
}
cfg.FailurePolicy.Handler = func(ctx context.Context, failure *mqspi.ConsumerFailure) {
    log.Printf("consume failed: attempt=%d err=%v", failure.Attempt, failure.Err)
}
```

`FailureHandler` 只负责观测，例如日志、metrics、告警。它不决定 ack 或 stop；最终动作由 policy 决定。

### 6.4 自定义策略

如果默认 policy 不够用，可以实现 `mqspi.ConsumerFailureStrategy`：

```go
type StopOnValidationErrorStrategy struct{}

func (s *StopOnValidationErrorStrategy) Decide(ctx context.Context, failure *mqspi.ConsumerFailure) mqspi.ConsumerFailureDecision {
    if errors.Is(failure.Err, ErrInvalidPayload) {
        return mqspi.ConsumerFailureDecision{
            Action: mqspi.ConsumerFailureActionSkip,
        }
    }
    return mqspi.ConsumerFailureDecision{
        Action:  mqspi.ConsumerFailureActionRetry,
        Backoff: time.Second,
    }
}

cfg.FailureStrategy = &StopOnValidationErrorStrategy{}
```

`FailureStrategy` 优先级高于 `FailurePolicy`。如果两者都设置，会使用 `FailureStrategy`。

---

## 7. 配置说明

### 7.1 ProducerConfig

```yaml
brokers:
  - 127.0.0.1:9092
topic: order-event-topic
credentials:
  username: admin
  password: secret
  mechanism: PLAIN
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `brokers` | 是 | Kafka broker 地址列表 |
| `topic` | 否 | 默认发送 topic；如果消息设置了 topic，会覆盖该值 |
| `credentials` | 否 | SASL 认证配置 |

### 7.2 ConsumerConfig

```yaml
brokers:
  - 127.0.0.1:9092
topic: order-event-topic
topics:
  - order-event-topic
  - order-refund-topic
group_id: order-service-group
failure_policy:
  max_attempts: 3
  initial_backoff: 1s
  max_backoff: 30s
  backoff_multiplier: 2
  final_action: stop
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `brokers` | 是 | Kafka broker 地址列表 |
| `topic` | 条件必填 | 默认 topic；`topics` 为空时作为唯一订阅 topic |
| `topics` | 条件必填 | 多 topic 订阅列表；为空时使用 `topic` |
| `group_id` | 是 | Kafka consumer group id |
| `credentials` | 否 | SASL 认证配置 |
| `failure_policy` | 否 | 仅 AdvancedConsumer 使用 |

### 7.3 SASL

支持的 mechanism：

- `mqspi.SASLMechanismPlain` / `PLAIN`
- `mqspi.SASLMechanismSCRAMSHA256` / `SCRAM-SHA-256`
- `mqspi.SASLMechanismSCRAMSHA512` / `SCRAM-SHA-512`

```go
cfg := &mqspi.ProducerConfig{
    Brokers: []string{"127.0.0.1:9092"},
    Topic:   "order-event-topic",
    Credentials: &mqspi.Credentials{
        Username:  "admin",
        Password:  "secret",
        Mechanism: mqspi.SASLMechanismPlain,
    },
}
```

---

## 8. Integration Test

`mq/example/mq_integration_test.go` 带有 build tag：

```go
//go:build integration
```

默认 `go test ./mq/...` 不会运行这些测试，因为它们需要真实 Kafka（默认 `localhost:9092`）。

需要显式启用：

```bash
go test -tags=integration ./mq/example -v
```

只检查集成测试能否编译：

```bash
go test -tags=integration ./mq/... -run '^$'
```

VSCode 默认不带 `integration` tag。如果想在 VSCode 里直接运行这些测试，需要配置：

```json
{
  "go.testTags": "integration"
}
```

---

## 9. 注意事项

- 当前模块不内置 DLQ。需要 DLQ 时，业务可以在 processor 或 failure handler 中自行使用 Producer 写入指定 topic。
- `skip` 会提交失败消息 offset，适合明确可以放弃毒消息的场景；不确定时不要使用。
- 默认 failure policy 不提交失败消息 offset，并持续重试，避免数据被静默跳过。
- `Run(ctx)` 返回 nil 通常表示 ctx 取消或 consumer 被关闭；返回 error 通常表示底层 consumer group 错误，或策略选择了 `stop`。
- `ConsumerMessage.Metadata` 是实现层保留字段，不要在业务代码中修改。
- `AsyncProduce` 的返回值只表示消息是否成功进入 async producer；最终发送结果以 callback 为准。
- 所有 producer/consumer 使用完成后都应调用 `Close`。
