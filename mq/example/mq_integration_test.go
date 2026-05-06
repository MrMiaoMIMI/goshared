//go:build integration

package example

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqhelper"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var testBrokers = []string{"localhost:9092"}

type printProcessor struct{}

func (p *printProcessor) Process(_ context.Context, msg *mqspi.ConsumerMessage) error {
	fmt.Printf("processing message: topic=%s, key=%s, value=%s\n", msg.Topic, string(msg.Key), string(msg.Value))
	return nil
}

type printBatchProcessor struct{}

func (p *printBatchProcessor) BatchProcess(_ context.Context, msgs []*mqspi.ConsumerMessage) error {
	fmt.Printf("batch processing %d messages\n", len(msgs))
	return nil
}

func TestProducerSyncProduce(t *testing.T) {
	config := &mqspi.ProducerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
	}
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	msg := mqhelper.NewProducerMessage("", []byte("key-1"), []byte("hello world"))
	if err := producer.Produce(context.Background(), msg); err != nil {
		t.Fatalf("produce failed: %v", err)
	}
	t.Logf("produced: topic=%s partition=%d offset=%d", msg.Topic, msg.Partition, msg.Offset)
}

func TestProducerBatchProduce(t *testing.T) {
	config := &mqspi.ProducerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
	}
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	msgs := []*mqspi.ProducerMessage{
		mqhelper.NewProducerMessage("", []byte("key-1"), []byte("message-1")),
		mqhelper.NewProducerMessage("", []byte("key-2"), []byte("message-2")),
	}
	if err := producer.BatchProduce(context.Background(), msgs); err != nil {
		t.Fatalf("batch produce failed: %v", err)
	}
}

func TestProducerAsyncProduce(t *testing.T) {
	config := &mqspi.ProducerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
	}
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	done := make(chan error, 1)
	msg := mqhelper.NewProducerMessage("", []byte("key-async"), []byte("async hello"))
	err = producer.AsyncProduce(context.Background(), msg, func(_ context.Context, msg *mqspi.ProducerMessage, err error) {
		if err != nil {
			done <- err
			return
		}
		t.Logf("async produced: topic=%s partition=%d offset=%d", msg.Topic, msg.Partition, msg.Offset)
		done <- nil
	})
	if err != nil {
		t.Fatalf("async enqueue failed: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("async produce failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for async callback")
	}
}

func TestConsumerConsumeAndAck(t *testing.T) {
	config := &mqspi.ConsumerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
		GroupID: "test-group",
	}
	consumer, err := mqhelper.NewConsumer(config)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := consumer.Consume(ctx)
	if err != nil {
		t.Logf("consume error: %v", err)
		return
	}
	if err := consumer.Ack(context.Background(), msg); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
}

func TestAdvancedConsumer(t *testing.T) {
	config := &mqspi.ConsumerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
		GroupID: "test-advanced-group",
		FailurePolicy: &mqspi.ConsumerFailurePolicy{
			MaxAttempts: 3,
			FinalAction: mqspi.ConsumerFailureActionStop,
		},
	}
	consumer, err := mqhelper.NewAdvancedConsumer(config, &printProcessor{})
	if err != nil {
		t.Fatalf("failed to create advanced consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := consumer.Run(ctx); err != nil {
		t.Fatalf("advanced consumer failed: %v", err)
	}
}

func TestAdvancedBatchConsumer(t *testing.T) {
	config := &mqspi.ConsumerConfig{
		Brokers: testBrokers,
		Topic:   "test-topic",
		GroupID: "test-batch-group",
	}
	consumer, err := mqhelper.NewAdvancedBatchConsumer(config, &printBatchProcessor{})
	if err != nil {
		t.Fatalf("failed to create advanced batch consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := consumer.Run(ctx); err != nil {
		t.Fatalf("advanced batch consumer failed: %v", err)
	}
}
