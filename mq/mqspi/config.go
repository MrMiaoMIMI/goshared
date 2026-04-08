package mqspi

// Credentials holds SASL authentication information for connecting to MQ servers.
// Nil credentials means no authentication is required.
type Credentials interface {
	Username() string
	Password() string
	// Mechanism returns the SASL mechanism.
	// Supported values: "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512".
	// Empty string means no SASL authentication.
	Mechanism() string
}

type ProducerConfig interface {
	Brokers() []string
	// Topic returns the default topic for producing messages.
	// If ProducerMessage.Topic is empty, this topic will be used.
	Topic() string
	// Credentials returns the SASL credentials for authentication.
	// Nil means no authentication.
	Credentials() Credentials
}

type ConsumerConfig interface {
	Brokers() []string
	// Topic returns the default/primary topic.
	Topic() string
	// Topics returns all subscribed topics.
	// If not explicitly set, returns []string{Topic()}.
	Topics() []string
	GroupID() string
	// Credentials returns the SASL credentials for authentication.
	// Nil means no authentication.
	Credentials() Credentials
}

type AdvancedConsumerConfig interface {
	ConsumerConfig
	// MaxRetries is the maximum number of retries before sending to DLQ.
	// 0 means no retry, failed messages are sent directly to DLQ.
	MaxRetries() int
	// DLQTopic returns the dead letter queue topic name.
	// Empty string means DLQ is disabled.
	DLQTopic() string
}
