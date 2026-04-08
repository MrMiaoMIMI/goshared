package mqhelper

import (
	"github.com/MrMiaoMIMI/goshared/mq/internal/mqsp"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

// ==================== Credentials Constructor ====================

// NewCredentials creates SASL credentials for MQ server authentication.
//   - mechanism: "PLAIN", "SCRAM-SHA-256", or "SCRAM-SHA-512"
func NewCredentials(username, password, mechanism string) mqspi.Credentials {
	return mqsp.NewCredentials(username, password, mechanism)
}

// ==================== Config Constructors ====================

// NewProducerConfig creates a ProducerConfig.
//   - topic: default topic for producing. If ProducerMessage.Topic is empty, this is used.
//   - credentials: SASL credentials. Pass nil for no authentication.
func NewProducerConfig(brokers []string, topic string, credentials mqspi.Credentials) mqspi.ProducerConfig {
	return mqsp.NewProducerConfig(brokers, topic, credentials)
}

// NewConsumerConfig creates a ConsumerConfig with a single default topic.
//   - credentials: SASL credentials. Pass nil for no authentication.
func NewConsumerConfig(brokers []string, topic string, groupID string, credentials mqspi.Credentials) mqspi.ConsumerConfig {
	return mqsp.NewConsumerConfig(brokers, topic, groupID, credentials)
}

// NewConsumerConfigWithTopics creates a ConsumerConfig with a default topic and multiple subscription topics.
//   - credentials: SASL credentials. Pass nil for no authentication.
func NewConsumerConfigWithTopics(brokers []string, topic string, topics []string, groupID string, credentials mqspi.Credentials) mqspi.ConsumerConfig {
	return mqsp.NewConsumerConfigWithTopics(brokers, topic, topics, groupID, credentials)
}

// NewAdvancedConsumerConfig creates an AdvancedConsumerConfig.
//   - topic: the default/primary topic
//   - maxRetries: max retry count before sending to DLQ. 0 means no retry.
//   - dlqTopic: the dead letter queue topic. Empty string disables DLQ.
//   - credentials: SASL credentials. Pass nil for no authentication.
func NewAdvancedConsumerConfig(brokers []string, topic string, groupID string, maxRetries int, dlqTopic string, credentials mqspi.Credentials) mqspi.AdvancedConsumerConfig {
	return mqsp.NewAdvancedConsumerConfig(brokers, topic, groupID, maxRetries, dlqTopic, credentials)
}

// NewAdvancedConsumerConfigWithTopics creates an AdvancedConsumerConfig with multiple subscription topics.
//   - credentials: SASL credentials. Pass nil for no authentication.
func NewAdvancedConsumerConfigWithTopics(brokers []string, topic string, topics []string, groupID string, maxRetries int, dlqTopic string, credentials mqspi.Credentials) mqspi.AdvancedConsumerConfig {
	return mqsp.NewAdvancedConsumerConfigWithTopics(brokers, topic, topics, groupID, maxRetries, dlqTopic, credentials)
}

// ==================== Factory Functions ====================

// NewProducer creates a new Producer backed by IBM/sarama.
func NewProducer(config mqspi.ProducerConfig) (mqspi.Producer, error) {
	return mqsp.NewProducer(config)
}

// NewManualConsumer creates a new ManualConsumer backed by IBM/sarama.
// The consumer starts consuming immediately upon creation.
func NewManualConsumer(config mqspi.ConsumerConfig) (mqspi.ManualConsumer, error) {
	return mqsp.NewManualConsumer(config)
}

// NewAdvancedConsumer creates a new AdvancedConsumer that processes messages one by one.
// Call Run() to start consuming; it blocks until Close() is called.
func NewAdvancedConsumer(config mqspi.AdvancedConsumerConfig, processor mqspi.MessageProcessor) (mqspi.AdvancedConsumer, error) {
	return mqsp.NewAdvancedConsumer(config, processor)
}

// NewAdvancedBatchConsumer creates a new AdvancedConsumer that processes messages in batches.
// Call Run() to start consuming; it blocks until Close() is called.
func NewAdvancedBatchConsumer(config mqspi.AdvancedConsumerConfig, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
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
