package mqsp

import (
	"errors"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
	"github.com/goccy/go-yaml"
)

func TestValidateProducerConfig(t *testing.T) {
	err := validateProducerConfig(nil)
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for nil config, got %v", err)
	}

	err = validateProducerConfig(&mqspi.ProducerConfig{})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for empty brokers, got %v", err)
	}

	err = validateProducerConfig(&mqspi.ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Credentials: &mqspi.Credentials{
			Mechanism: "BAD",
		},
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad credentials, got %v", err)
	}

	retryMax := -1
	err = validateProducerConfig(&mqspi.ProducerConfig{
		Brokers:  []string{"localhost:9092"},
		RetryMax: &retryMax,
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad retry max, got %v", err)
	}

	err = validateProducerConfig(&mqspi.ProducerConfig{
		Brokers:      []string{"localhost:9092"},
		RetryBackoff: -time.Millisecond,
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad retry backoff, got %v", err)
	}

	err = validateProducerConfig(&mqspi.ProducerConfig{
		Brokers:     []string{"localhost:9092"},
		Compression: "bad",
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad compression, got %v", err)
	}

	err = validateProducerConfig(&mqspi.ProducerConfig{Brokers: []string{"localhost:9092"}})
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConsumerConfig(t *testing.T) {
	err := validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "topic-a",
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for empty group id, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "group-a",
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for empty topic, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "topic-a",
		GroupID: "group-a",
		FailurePolicy: &mqspi.ConsumerFailurePolicy{
			FinalAction: "bad",
		},
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad failure action, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     "topic-a",
		GroupID:   "group-a",
		BatchSize: -1,
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad batch size, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     "topic-a",
		GroupID:   "group-a",
		BatchWait: -time.Millisecond,
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad batch wait, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers:    []string{"localhost:9092"},
		Topic:      "topic-a",
		GroupID:    "group-a",
		BufferSize: -1,
	})
	if !errors.Is(err, mqspi.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for bad buffer size, got %v", err)
	}

	err = validateConsumerConfig(&mqspi.ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "topic-a",
		GroupID: "group-a",
	})
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestConsumerTopics(t *testing.T) {
	config := &mqspi.ConsumerConfig{
		Topic:  "primary-topic",
		Topics: []string{" ", "secondary-topic"},
	}
	topics := consumerTopics(config)
	if len(topics) != 1 || topics[0] != "secondary-topic" {
		t.Fatalf("unexpected topics: %#v", topics)
	}

	config.Topics = nil
	topics = consumerTopics(config)
	if len(topics) != 1 || topics[0] != "primary-topic" {
		t.Fatalf("unexpected fallback topics: %#v", topics)
	}
}

func TestConfigDefaults(t *testing.T) {
	producerConfig := &mqspi.ProducerConfig{}
	if producerRetryMax(producerConfig) != mqspi.DefaultProducerRetryMax {
		t.Fatalf("unexpected producer retry max")
	}
	if producerRetryBackoff(producerConfig) != mqspi.DefaultProducerRetryBackoff {
		t.Fatalf("unexpected producer retry backoff")
	}
	if normalizedProducerCompression("") != mqspi.ProducerCompressionNone {
		t.Fatalf("unexpected producer compression")
	}

	retryMax := 0
	producerConfig.RetryMax = &retryMax
	if producerRetryMax(producerConfig) != 0 {
		t.Fatalf("explicit retry max 0 should disable retries")
	}

	consumerConfig := &mqspi.ConsumerConfig{}
	if consumerBatchSize(consumerConfig) != mqspi.DefaultConsumerBatchSize {
		t.Fatalf("unexpected consumer batch size")
	}
	if consumerBatchWait(consumerConfig) != mqspi.DefaultConsumerBatchWait {
		t.Fatalf("unexpected consumer batch wait")
	}
	if consumerBufferSize(consumerConfig) != mqspi.DefaultConsumerBufferSize {
		t.Fatalf("unexpected consumer buffer size")
	}
}

func TestDecodeProducerYAMLConfig(t *testing.T) {
	var config mqspi.ProducerConfig
	err := yaml.Unmarshal([]byte(`
brokers:
  - 127.0.0.1:9092
topic: order-event-topic
retry_max: 0
retry_backoff: 250ms
compression: zstd
credentials:
  username: admin
  password: secret
  mechanism: PLAIN
`), &config)
	if err != nil {
		t.Fatalf("decode producer yaml failed: %v", err)
	}
	if err := validateProducerConfig(&config); err != nil {
		t.Fatalf("expected valid producer config, got %v", err)
	}
	if config.RetryMax == nil || *config.RetryMax != 0 {
		t.Fatalf("unexpected retry max: %#v", config.RetryMax)
	}
	if config.RetryBackoff != 250*time.Millisecond {
		t.Fatalf("unexpected retry backoff: %s", config.RetryBackoff)
	}
	if normalizedProducerCompression(config.Compression) != mqspi.ProducerCompressionZSTD {
		t.Fatalf("unexpected compression: %s", config.Compression)
	}
}

func TestDecodeConsumerYAMLConfig(t *testing.T) {
	var config mqspi.ConsumerConfig
	err := yaml.Unmarshal([]byte(`
brokers:
  - 127.0.0.1:9092
topic: order-event-topic
topics:
  - order-event-topic
  - order-refund-topic
group_id: order-service-group
batch_size: 200
batch_wait: 50ms
buffer_size: 512
`), &config)
	if err != nil {
		t.Fatalf("decode consumer yaml failed: %v", err)
	}
	if err := validateConsumerConfig(&config); err != nil {
		t.Fatalf("expected valid consumer config, got %v", err)
	}
	if consumerBatchSize(&config) != 200 || consumerBatchWait(&config) != 50*time.Millisecond || consumerBufferSize(&config) != 512 {
		t.Fatalf("unexpected consumer runtime config: %#v", config)
	}
}

func TestDecodeAdvancedConsumerFailurePolicyYAMLConfig(t *testing.T) {
	var config mqspi.ConsumerConfig
	err := yaml.Unmarshal([]byte(`
brokers:
  - 127.0.0.1:9092
topic: order-event-topic
group_id: order-service-group
batch_size: 50
batch_wait: 20ms
failure_policy:
  max_attempts: 3
  initial_backoff: 500ms
  max_backoff: 10s
  backoff_multiplier: 1.5
  final_action: skip
`), &config)
	if err != nil {
		t.Fatalf("decode advanced consumer yaml failed: %v", err)
	}
	if err := validateConsumerConfig(&config); err != nil {
		t.Fatalf("expected valid advanced consumer config, got %v", err)
	}
	if config.FailurePolicy == nil {
		t.Fatalf("failure policy was not decoded")
	}
	if config.FailurePolicy.MaxAttempts != 3 || config.FailurePolicy.FinalAction != mqspi.ConsumerFailureActionSkip {
		t.Fatalf("unexpected failure policy: %#v", config.FailurePolicy)
	}
	if config.FailurePolicy.InitialBackoff != 500*time.Millisecond || config.FailurePolicy.MaxBackoff != 10*time.Second {
		t.Fatalf("unexpected failure policy backoff: %#v", config.FailurePolicy)
	}
}
