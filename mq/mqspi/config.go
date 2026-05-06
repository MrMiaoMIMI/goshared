package mqspi

const (
	// SASLMechanismPlain enables SASL/PLAIN authentication.
	SASLMechanismPlain = "PLAIN"
	// SASLMechanismSCRAMSHA256 enables SASL/SCRAM-SHA-256 authentication.
	SASLMechanismSCRAMSHA256 = "SCRAM-SHA-256"
	// SASLMechanismSCRAMSHA512 enables SASL/SCRAM-SHA-512 authentication.
	SASLMechanismSCRAMSHA512 = "SCRAM-SHA-512"
)

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
	// FailurePolicy is the built-in advanced-consumer failure strategy.
	FailurePolicy *ConsumerFailurePolicy `json:"failure_policy,omitempty" yaml:"failure_policy,omitempty"`
	// FailureStrategy is a custom advanced-consumer failure strategy. It is code-only.
	FailureStrategy ConsumerFailureStrategy `json:"-" yaml:"-"`
}
