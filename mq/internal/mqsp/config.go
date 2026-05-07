package mqsp

import (
	"fmt"
	"strings"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

func validateProducerConfig(config *mqspi.ProducerConfig) error {
	if config == nil {
		return fmt.Errorf("%w: producer config is nil", mqspi.ErrInvalidConfig)
	}
	if len(cleanStrings(config.Brokers)) == 0 {
		return fmt.Errorf("%w: producer brokers is empty", mqspi.ErrInvalidConfig)
	}
	if err := validateCredentials(config.Credentials); err != nil {
		return err
	}
	if config.RetryMax != nil && *config.RetryMax < 0 {
		return fmt.Errorf("%w: producer retry_max must be >= 0", mqspi.ErrInvalidConfig)
	}
	if config.RetryBackoff < 0 {
		return fmt.Errorf("%w: producer retry_backoff must be >= 0", mqspi.ErrInvalidConfig)
	}
	if err := validateProducerCompression(config.Compression); err != nil {
		return err
	}
	return nil
}

func validateConsumerConfig(config *mqspi.ConsumerConfig) error {
	if config == nil {
		return fmt.Errorf("%w: consumer config is nil", mqspi.ErrInvalidConfig)
	}
	if len(cleanStrings(config.Brokers)) == 0 {
		return fmt.Errorf("%w: consumer brokers is empty", mqspi.ErrInvalidConfig)
	}
	if strings.TrimSpace(config.GroupID) == "" {
		return fmt.Errorf("%w: consumer group id is empty", mqspi.ErrInvalidConfig)
	}
	if len(consumerTopics(config)) == 0 {
		return fmt.Errorf("%w: consumer topic is empty", mqspi.ErrInvalidConfig)
	}
	if err := validateCredentials(config.Credentials); err != nil {
		return err
	}
	if err := validateFailurePolicy(config.FailurePolicy); err != nil {
		return err
	}
	if config.BatchSize < 0 {
		return fmt.Errorf("%w: consumer batch_size must be >= 0", mqspi.ErrInvalidConfig)
	}
	if config.BatchWait < 0 {
		return fmt.Errorf("%w: consumer batch_wait must be >= 0", mqspi.ErrInvalidConfig)
	}
	if config.BufferSize < 0 {
		return fmt.Errorf("%w: consumer buffer_size must be >= 0", mqspi.ErrInvalidConfig)
	}
	return nil
}

func validateProducerCompression(compression mqspi.ProducerCompression) error {
	switch normalizedProducerCompression(compression) {
	case mqspi.ProducerCompressionNone,
		mqspi.ProducerCompressionGZIP,
		mqspi.ProducerCompressionSnappy,
		mqspi.ProducerCompressionLZ4,
		mqspi.ProducerCompressionZSTD:
		return nil
	default:
		return fmt.Errorf("%w: unsupported producer compression %q", mqspi.ErrInvalidConfig, compression)
	}
}

func validateFailurePolicy(policy *mqspi.ConsumerFailurePolicy) error {
	if policy == nil {
		return nil
	}
	switch policy.FinalAction {
	case "", mqspi.ConsumerFailureActionRetry, mqspi.ConsumerFailureActionSkip, mqspi.ConsumerFailureActionStop:
		return nil
	default:
		return fmt.Errorf("%w: unsupported consumer failure final action %q", mqspi.ErrInvalidConfig, policy.FinalAction)
	}
}

func validateCredentials(credentials *mqspi.Credentials) error {
	if credentials == nil || credentials.Mechanism == "" {
		return nil
	}
	switch credentials.Mechanism {
	case mqspi.SASLMechanismPlain, mqspi.SASLMechanismSCRAMSHA256, mqspi.SASLMechanismSCRAMSHA512:
		return nil
	default:
		return fmt.Errorf("%w: unsupported sasl mechanism %q", mqspi.ErrInvalidConfig, credentials.Mechanism)
	}
}

func consumerTopics(config *mqspi.ConsumerConfig) []string {
	if config == nil {
		return nil
	}
	if topics := cleanStrings(config.Topics); len(topics) > 0 {
		return topics
	}
	if topic := strings.TrimSpace(config.Topic); topic != "" {
		return []string{topic}
	}
	return nil
}

func producerRetryMax(config *mqspi.ProducerConfig) int {
	if config == nil || config.RetryMax == nil {
		return mqspi.DefaultProducerRetryMax
	}
	return *config.RetryMax
}

func producerRetryBackoff(config *mqspi.ProducerConfig) time.Duration {
	if config == nil || config.RetryBackoff <= 0 {
		return mqspi.DefaultProducerRetryBackoff
	}
	return config.RetryBackoff
}

func normalizedProducerCompression(compression mqspi.ProducerCompression) mqspi.ProducerCompression {
	switch strings.ToLower(strings.TrimSpace(string(compression))) {
	case "", string(mqspi.ProducerCompressionNone):
		return mqspi.ProducerCompressionNone
	case string(mqspi.ProducerCompressionGZIP):
		return mqspi.ProducerCompressionGZIP
	case string(mqspi.ProducerCompressionSnappy):
		return mqspi.ProducerCompressionSnappy
	case string(mqspi.ProducerCompressionLZ4):
		return mqspi.ProducerCompressionLZ4
	case string(mqspi.ProducerCompressionZSTD):
		return mqspi.ProducerCompressionZSTD
	default:
		return compression
	}
}

func consumerBatchSize(config *mqspi.ConsumerConfig) int {
	if config == nil || config.BatchSize <= 0 {
		return mqspi.DefaultConsumerBatchSize
	}
	return config.BatchSize
}

func consumerBatchWait(config *mqspi.ConsumerConfig) time.Duration {
	if config == nil || config.BatchWait <= 0 {
		return mqspi.DefaultConsumerBatchWait
	}
	return config.BatchWait
}

func consumerBufferSize(config *mqspi.ConsumerConfig) int {
	if config == nil || config.BufferSize <= 0 {
		return mqspi.DefaultConsumerBufferSize
	}
	return config.BufferSize
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
