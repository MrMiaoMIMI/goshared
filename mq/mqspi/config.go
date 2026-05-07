package mqspi

import "time"

const (
	// SASLMechanismPlain enables SASL/PLAIN authentication.
	SASLMechanismPlain = "PLAIN"
	// SASLMechanismSCRAMSHA256 enables SASL/SCRAM-SHA-256 authentication.
	SASLMechanismSCRAMSHA256 = "SCRAM-SHA-256"
	// SASLMechanismSCRAMSHA512 enables SASL/SCRAM-SHA-512 authentication.
	SASLMechanismSCRAMSHA512 = "SCRAM-SHA-512"
)

const (
	// ProducerCompressionNone disables producer message compression.
	ProducerCompressionNone ProducerCompression = "none"
	// ProducerCompressionGZIP uses gzip compression.
	ProducerCompressionGZIP ProducerCompression = "gzip"
	// ProducerCompressionSnappy uses snappy compression.
	ProducerCompressionSnappy ProducerCompression = "snappy"
	// ProducerCompressionLZ4 uses lz4 compression.
	ProducerCompressionLZ4 ProducerCompression = "lz4"
	// ProducerCompressionZSTD uses zstd compression.
	ProducerCompressionZSTD ProducerCompression = "zstd"
)

const (
	// DefaultProducerRetryMax preserves the current producer retry behavior.
	DefaultProducerRetryMax = 3
	// DefaultProducerRetryBackoff is Sarama's default producer retry interval.
	DefaultProducerRetryBackoff = 100 * time.Millisecond
	// DefaultConsumerBatchSize is the default max batch size for AdvancedBatchConsumer.
	DefaultConsumerBatchSize = 100
	// DefaultConsumerBatchWait keeps the current non-blocking batch drain behavior.
	DefaultConsumerBatchWait = 0
	// DefaultConsumerBufferSize is the manual consumer internal channel size.
	DefaultConsumerBufferSize = 256
)

// ProducerCompression is the configured producer compression codec.
type ProducerCompression string

// Credentials holds SASL authentication information for connecting to MQ servers.
// Nil credentials or an empty mechanism means no authentication is required.
type Credentials struct {
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	Mechanism string `json:"mechanism" yaml:"mechanism"`
}

// ProducerConfig is the minimal producer configuration.
// Topic is the default topic used when ProducerMessage.Topic is empty.
type ProducerConfig struct {
	// Brokers is the Kafka broker address list.
	Brokers []string `json:"brokers" yaml:"brokers"`
	// Topic is the default producer topic.
	Topic string `json:"topic" yaml:"topic"`
	// Credentials enables SASL when non-nil and Mechanism is not empty.
	Credentials *Credentials `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	// RetryMax is the maximum number of producer retries.
	// Nil uses DefaultProducerRetryMax. A non-nil value of 0 disables retries.
	RetryMax *int `json:"retry_max,omitempty" yaml:"retry_max,omitempty"`
	// RetryBackoff is the wait between producer retries.
	// Zero uses DefaultProducerRetryBackoff.
	RetryBackoff time.Duration `json:"retry_backoff,omitempty" yaml:"retry_backoff,omitempty"`
	// Compression controls producer message compression.
	// Empty uses ProducerCompressionNone.
	Compression ProducerCompression `json:"compression,omitempty" yaml:"compression,omitempty"`
}

// ConsumerConfig is shared by manual and advanced consumers.
// If Topics is empty, Topic is used as the single subscribed topic.
// FailurePolicy and FailureStrategy are used only by AdvancedConsumer.
// FailurePolicy is configuration-file friendly. FailureStrategy is a code-only
// extension point and has priority over FailurePolicy.
type ConsumerConfig struct {
	// Brokers is the Kafka broker address list.
	Brokers []string `json:"brokers" yaml:"brokers"`
	// Topic is the default or fallback subscribed topic.
	Topic string `json:"topic" yaml:"topic"`
	// Topics is the explicit subscription list. If empty, Topic is used.
	Topics []string `json:"topics,omitempty" yaml:"topics,omitempty"`
	// GroupID is the Kafka consumer group id.
	GroupID string `json:"group_id" yaml:"group_id"`
	// Credentials enables SASL when non-nil and Mechanism is not empty.
	Credentials *Credentials `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	// FailurePolicy configures the built-in advanced-consumer failure strategy.
	FailurePolicy *ConsumerFailurePolicy `json:"failure_policy,omitempty" yaml:"failure_policy,omitempty"`
	// FailureStrategy is a custom advanced-consumer failure strategy. It is code-only.
	FailureStrategy ConsumerFailureStrategy `json:"-" yaml:"-"`
	// BatchSize is the max number of messages passed to BatchMessageProcessor.
	// Zero uses DefaultConsumerBatchSize.
	BatchSize int `json:"batch_size,omitempty" yaml:"batch_size,omitempty"`
	// BatchWait is the max wait after the first batch message for more messages.
	// Zero keeps the current non-blocking drain behavior.
	BatchWait time.Duration `json:"batch_wait,omitempty" yaml:"batch_wait,omitempty"`
	// BufferSize is the internal channel size used by manual Consumer.
	// Zero uses DefaultConsumerBufferSize.
	BufferSize int `json:"buffer_size,omitempty" yaml:"buffer_size,omitempty"`
}
