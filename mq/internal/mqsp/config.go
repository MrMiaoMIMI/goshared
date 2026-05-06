package mqsp

import (
	"fmt"
	"strings"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

const defaultConsumerBuffer = 256

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
	return nil
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
