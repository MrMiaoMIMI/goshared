package mqsp

import "github.com/MrMiaoMIMI/goshared/mq/mqspi"

var (
	_ mqspi.ProducerConfig         = (*ProducerConfig)(nil)
	_ mqspi.ConsumerConfig         = (*ConsumerConfig)(nil)
	_ mqspi.AdvancedConsumerConfig = (*AdvancedConsumerConfig)(nil)
	_ mqspi.Credentials            = (*Credentials)(nil)
)

// ==================== Credentials ====================

type Credentials struct {
	username  string
	password  string
	mechanism string
}

func NewCredentials(username, password, mechanism string) mqspi.Credentials {
	return &Credentials{username: username, password: password, mechanism: mechanism}
}

func (c *Credentials) Username() string  { return c.username }
func (c *Credentials) Password() string  { return c.password }
func (c *Credentials) Mechanism() string { return c.mechanism }

// ==================== ProducerConfig ====================

type ProducerConfig struct {
	brokers     []string
	topic       string
	credentials mqspi.Credentials
}

func NewProducerConfig(brokers []string, topic string, credentials mqspi.Credentials) mqspi.ProducerConfig {
	return &ProducerConfig{brokers: brokers, topic: topic, credentials: credentials}
}

func (c *ProducerConfig) Brokers() []string          { return c.brokers }
func (c *ProducerConfig) Topic() string               { return c.topic }
func (c *ProducerConfig) Credentials() mqspi.Credentials { return c.credentials }

// ==================== ConsumerConfig ====================

type ConsumerConfig struct {
	brokers     []string
	topic       string
	topics      []string
	groupID     string
	credentials mqspi.Credentials
}

func NewConsumerConfig(brokers []string, topic string, groupID string, credentials mqspi.Credentials) mqspi.ConsumerConfig {
	return &ConsumerConfig{brokers: brokers, topic: topic, groupID: groupID, credentials: credentials}
}

func NewConsumerConfigWithTopics(brokers []string, topic string, topics []string, groupID string, credentials mqspi.Credentials) mqspi.ConsumerConfig {
	return &ConsumerConfig{brokers: brokers, topic: topic, topics: topics, groupID: groupID, credentials: credentials}
}

func (c *ConsumerConfig) Brokers() []string              { return c.brokers }
func (c *ConsumerConfig) Topic() string                   { return c.topic }
func (c *ConsumerConfig) GroupID() string                  { return c.groupID }
func (c *ConsumerConfig) Credentials() mqspi.Credentials { return c.credentials }

func (c *ConsumerConfig) Topics() []string {
	if len(c.topics) > 0 {
		return c.topics
	}
	if c.topic != "" {
		return []string{c.topic}
	}
	return nil
}

// ==================== AdvancedConsumerConfig ====================

type AdvancedConsumerConfig struct {
	ConsumerConfig
	maxRetries int
	dlqTopic   string
}

func NewAdvancedConsumerConfig(brokers []string, topic string, groupID string, maxRetries int, dlqTopic string, credentials mqspi.Credentials) mqspi.AdvancedConsumerConfig {
	return &AdvancedConsumerConfig{
		ConsumerConfig: ConsumerConfig{brokers: brokers, topic: topic, groupID: groupID, credentials: credentials},
		maxRetries:     maxRetries,
		dlqTopic:       dlqTopic,
	}
}

func NewAdvancedConsumerConfigWithTopics(brokers []string, topic string, topics []string, groupID string, maxRetries int, dlqTopic string, credentials mqspi.Credentials) mqspi.AdvancedConsumerConfig {
	return &AdvancedConsumerConfig{
		ConsumerConfig: ConsumerConfig{brokers: brokers, topic: topic, topics: topics, groupID: groupID, credentials: credentials},
		maxRetries:     maxRetries,
		dlqTopic:       dlqTopic,
	}
}

func (c *AdvancedConsumerConfig) MaxRetries() int  { return c.maxRetries }
func (c *AdvancedConsumerConfig) DLQTopic() string { return c.dlqTopic }
