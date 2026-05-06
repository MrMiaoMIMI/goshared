package mqsp

import (
	"errors"
	"testing"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
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
