package mqhelper

import (
	"encoding/json"
	"fmt"

	"github.com/MrMiaoMIMI/goshared/mq/internal/mqsp"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

// ==================== Factory Functions ====================

// NewProducer creates a Producer from ProducerConfig.
// The config can be built directly or loaded from a config file.
func NewProducer(config *mqspi.ProducerConfig) (mqspi.Producer, error) {
	return mqsp.NewProducer(config)
}

// NewConsumer creates a manual Consumer from ConsumerConfig.
// The consumer starts consuming immediately after creation.
func NewConsumer(config *mqspi.ConsumerConfig) (mqspi.Consumer, error) {
	return mqsp.NewConsumer(config)
}

// NewAdvancedConsumer creates a new AdvancedConsumer that processes messages one by one.
// Call Run(ctx) to start consuming; it blocks until ctx is done, Close is called, or the failure strategy stops it.
func NewAdvancedConsumer(config *mqspi.ConsumerConfig, processor mqspi.MessageProcessor) (mqspi.AdvancedConsumer, error) {
	return mqsp.NewAdvancedConsumer(config, processor)
}

// NewAdvancedBatchConsumer creates a new AdvancedConsumer that processes messages in batches.
// Call Run(ctx) to start consuming; it blocks until ctx is done, Close is called, or the failure strategy stops it.
func NewAdvancedBatchConsumer(config *mqspi.ConsumerConfig, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
	return mqsp.NewAdvancedBatchConsumer(config, batchProcessor)
}

// ==================== Message Constructors ====================

// NewProducerMessage creates a ProducerMessage with the given topic, key, and value.
// If topic is empty, the producer's default topic (from ProducerConfig) will be used.
func NewProducerMessage(topic string, key, value []byte) *mqspi.ProducerMessage {
	return &mqspi.ProducerMessage{
		Topic:    topic,
		Key:      key,
		Value:    value,
		Metadata: make(mqspi.Metadata),
	}
}

// NewProducerMessageWithHeaders creates a ProducerMessage with headers.
func NewProducerMessageWithHeaders(topic string, key, value []byte, headers map[string][]byte) *mqspi.ProducerMessage {
	msg := NewProducerMessage(topic, key, value)
	for k, v := range headers {
		msg.Headers = append(msg.Headers, mqspi.Header{Key: []byte(k), Value: v})
	}
	return msg
}

// NewJSONProducerMessage creates a ProducerMessage with JSON-encoded value.
// Returns an error if JSON marshaling fails.
func NewJSONProducerMessage(topic string, key []byte, v any) (*mqspi.ProducerMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("mqhelper: failed to marshal JSON: %w", err)
	}
	return NewProducerMessage(topic, key, data), nil
}

// GetHeaderValue returns the value of the first header with the given key.
// Returns nil if the header is not found.
func GetHeaderValue(msg *mqspi.ConsumerMessage, key string) []byte {
	for _, h := range msg.Headers {
		if string(h.Key) == key {
			return h.Value
		}
	}
	return nil
}

// GetHeaderString returns the string value of a header.
func GetHeaderString(msg *mqspi.ConsumerMessage, key string) string {
	v := GetHeaderValue(msg, key)
	if v == nil {
		return ""
	}
	return string(v)
}

// UnmarshalValue unmarshals the message value from JSON into dest.
func UnmarshalValue(msg *mqspi.ConsumerMessage, dest any) error {
	return json.Unmarshal(msg.Value, dest)
}
