package example

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqhelper"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

// ==================== Test Models ====================

var testBrokers = []string{"localhost:9092"}

// ==================== AsyncProduceCallback Implementation ====================

type LogCallback struct{}

func (c *LogCallback) Handle(_ context.Context, msg *mqspi.ProducerMessage, err error) {
	if err != nil {
		fmt.Printf("async produce failed: topic=%s, err=%v\n", msg.Topic, err)
		return
	}
	fmt.Printf("async produce succeeded: topic=%s, partition=%d, offset=%d\n",
		msg.Topic, msg.Partition, msg.Offset)
}

// ==================== MessageProcessor Implementation ====================

type PrintProcessor struct{}

func (p *PrintProcessor) Process(_ context.Context, msg *mqspi.ConsumerMessage) error {
	fmt.Printf("processing message: topic=%s, key=%s, value=%s\n",
		msg.Topic, string(msg.Key), string(msg.Value))
	return nil
}

// ==================== BatchMessageProcessor Implementation ====================

type PrintBatchProcessor struct{}

func (p *PrintBatchProcessor) BatchProcess(_ context.Context, msgs []*mqspi.ConsumerMessage) error {
	fmt.Printf("batch processing %d messages\n", len(msgs))
	for i, msg := range msgs {
		fmt.Printf("  [%d] topic=%s, key=%s, value=%s\n",
			i, msg.Topic, string(msg.Key), string(msg.Value))
	}
	return nil
}

// ==================== Producer Examples ====================

func Test_Producer_SyncProduce(t *testing.T) {
	// Create producer with default topic "test-topic" (no auth)
	config := mqhelper.NewProducerConfig(testBrokers, "test-topic", nil)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	ctx := context.Background()

	// Use default topic (topic field left empty)
	msg := mqhelper.NewProducerMessage("", []byte("key-1"), []byte("hello world"))
	err = producer.Produce(ctx, msg)
	t.Logf("Sync produce (default topic): topic=%s, partition=%d, offset=%d, err=%v",
		msg.Topic, msg.Partition, msg.Offset, err)

	// Override topic explicitly
	msg2 := mqhelper.NewProducerMessage("another-topic", []byte("key-2"), []byte("explicit topic"))
	err = producer.Produce(ctx, msg2)
	t.Logf("Sync produce (explicit topic): topic=%s, partition=%d, offset=%d, err=%v",
		msg2.Topic, msg2.Partition, msg2.Offset, err)
}

func Test_Producer_AsyncProduce(t *testing.T) {
	config := mqhelper.NewProducerConfig(testBrokers, "test-topic", nil)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	ctx := context.Background()
	callback := &LogCallback{}

	msg := mqhelper.NewProducerMessage("", []byte("key-2"), []byte("async hello"))
	producer.AsyncProduce(ctx, msg, callback)

	time.Sleep(2 * time.Second)
	t.Logf("Async produce sent")
}

func Test_Producer_WithHeaders(t *testing.T) {
	config := mqhelper.NewProducerConfig(testBrokers, "test-topic", nil)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	ctx := context.Background()

	msg := mqhelper.NewProducerMessage("", []byte("key-3"), []byte("msg with headers"))
	msg.Headers = []mqspi.Header{
		{Key: []byte("trace-id"), Value: []byte("abc-123")},
		{Key: []byte("source"), Value: []byte("test-service")},
	}
	err = producer.Produce(ctx, msg)
	t.Logf("Produce with headers: topic=%s, partition=%d, offset=%d, err=%v",
		msg.Topic, msg.Partition, msg.Offset, err)
}

func Test_Producer_Check(t *testing.T) {
	config := mqhelper.NewProducerConfig(testBrokers, "test-topic", nil)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close(context.Background())

	err = producer.Check(context.Background())
	t.Logf("Producer check: err=%v", err)
}

// ==================== ManualConsumer Examples ====================

func Test_ManualConsumer_ConsumeAndConfirm(t *testing.T) {
	config := mqhelper.NewConsumerConfig(testBrokers, "test-topic", "test-group", nil)
	consumer, err := mqhelper.NewManualConsumer(config)
	if err != nil {
		t.Fatalf("failed to create manual consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := consumer.Consume(ctx)
	if err != nil {
		t.Logf("Consume error (expected if no messages): %v", err)
		return
	}
	t.Logf("Consumed: topic=%s, key=%s, value=%s", msg.Topic, string(msg.Key), string(msg.Value))

	err = consumer.Confirm(msg)
	t.Logf("Confirm: err=%v", err)
}

func Test_ManualConsumer_ConsumeLoop(t *testing.T) {
	config := mqhelper.NewConsumerConfig(testBrokers, "test-topic", "test-group", nil)
	consumer, err := mqhelper.NewManualConsumer(config)
	if err != nil {
		t.Fatalf("failed to create manual consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Typical consume loop pattern
	for {
		select {
		case <-ctx.Done():
			t.Logf("Context done, stopping consume loop")
			return
		default:
			msg, err := consumer.Consume(ctx)
			if err != nil {
				t.Logf("Consume error: %v", err)
				return
			}

			t.Logf("Processing: topic=%s, key=%s, value=%s",
				msg.Topic, string(msg.Key), string(msg.Value))

			if err := consumer.Confirm(msg); err != nil {
				t.Logf("Confirm error: %v", err)
			}
		}
	}
}

func Test_ManualConsumer_ColdRetry(t *testing.T) {
	config := mqhelper.NewConsumerConfig(testBrokers, "test-topic", "test-group", nil)
	consumer, err := mqhelper.NewManualConsumer(config)
	if err != nil {
		t.Fatalf("failed to create manual consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := consumer.Consume(ctx)
	if err != nil {
		t.Logf("Consume error: %v", err)
		return
	}

	// Retry after 60 seconds
	err = consumer.ColdRetry(ctx, msg, 60)
	t.Logf("ColdRetry: err=%v", err)
}

func Test_ManualConsumer_DLQ(t *testing.T) {
	config := mqhelper.NewConsumerConfig(testBrokers, "test-topic", "test-group", nil)
	consumer, err := mqhelper.NewManualConsumer(config)
	if err != nil {
		t.Fatalf("failed to create manual consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := consumer.Consume(ctx)
	if err != nil {
		t.Logf("Consume error: %v", err)
		return
	}

	err = consumer.DLQ(ctx, msg)
	t.Logf("DLQ: err=%v", err)
}

func Test_ManualConsumer_MultipleTopics(t *testing.T) {
	// Subscribe to multiple topics with a primary default topic
	config := mqhelper.NewConsumerConfigWithTopics(
		testBrokers,
		"primary-topic",
		[]string{"primary-topic", "secondary-topic"},
		"multi-topic-group",
		nil,
	)
	consumer, err := mqhelper.NewManualConsumer(config)
	if err != nil {
		t.Fatalf("failed to create manual consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := consumer.Consume(ctx)
	if err != nil {
		t.Logf("Consume error: %v", err)
		return
	}
	t.Logf("Consumed from: topic=%s", msg.Topic)
}

// ==================== Producer with Credentials Example ====================

func Test_Producer_WithCredentials(t *testing.T) {
	// SASL/PLAIN authentication
	creds := mqhelper.NewCredentials("admin", "secret-password", "PLAIN")
	config := mqhelper.NewProducerConfig(testBrokers, "secure-topic", creds)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer with credentials: %v", err)
	}
	defer producer.Close(context.Background())

	msg := mqhelper.NewProducerMessage("", []byte("key"), []byte("authenticated message"))
	err = producer.Produce(context.Background(), msg)
	t.Logf("Produce with SASL/PLAIN: topic=%s, err=%v", msg.Topic, err)
}

func Test_Producer_WithSCRAMCredentials(t *testing.T) {
	// SASL/SCRAM-SHA-256 authentication
	creds := mqhelper.NewCredentials("admin", "secret-password", "SCRAM-SHA-256")
	config := mqhelper.NewProducerConfig(testBrokers, "secure-topic", creds)
	producer, err := mqhelper.NewProducer(config)
	if err != nil {
		t.Fatalf("failed to create producer with SCRAM: %v", err)
	}
	defer producer.Close(context.Background())

	msg := mqhelper.NewProducerMessage("", []byte("key"), []byte("scram message"))
	err = producer.Produce(context.Background(), msg)
	t.Logf("Produce with SCRAM-SHA-256: topic=%s, err=%v", msg.Topic, err)
}

// ==================== AdvancedConsumer Examples ====================

func Test_AdvancedConsumer_SingleProcessor(t *testing.T) {
	config := mqhelper.NewAdvancedConsumerConfig(
		testBrokers,
		"test-topic",         // default topic
		"test-advanced-group",
		3,                    // max 3 retries
		"test-topic-dlq",     // DLQ topic
		nil,                  // no auth
	)
	processor := &PrintProcessor{}

	consumer, err := mqhelper.NewAdvancedConsumer(config, processor)
	if err != nil {
		t.Fatalf("failed to create advanced consumer: %v", err)
	}

	go func() {
		if err := consumer.Run(); err != nil {
			t.Logf("Advanced consumer error: %v", err)
		}
	}()

	time.Sleep(10 * time.Second)

	err = consumer.Close(context.Background())
	t.Logf("Advanced consumer closed: err=%v", err)
}

func Test_AdvancedConsumer_BatchProcessor(t *testing.T) {
	config := mqhelper.NewAdvancedConsumerConfig(
		testBrokers,
		"test-topic",
		"test-batch-group",
		3,
		"test-topic-dlq",
		nil,
	)
	batchProcessor := &PrintBatchProcessor{}

	consumer, err := mqhelper.NewAdvancedBatchConsumer(config, batchProcessor)
	if err != nil {
		t.Fatalf("failed to create advanced batch consumer: %v", err)
	}

	go func() {
		if err := consumer.Run(); err != nil {
			t.Logf("Advanced batch consumer error: %v", err)
		}
	}()

	time.Sleep(10 * time.Second)

	err = consumer.Close(context.Background())
	t.Logf("Advanced batch consumer closed: err=%v", err)
}

func Test_AdvancedConsumer_Check(t *testing.T) {
	config := mqhelper.NewAdvancedConsumerConfig(
		testBrokers,
		"test-topic",
		"test-check-group",
		0,
		"",
		nil,
	)

	consumer, err := mqhelper.NewAdvancedConsumer(config, &PrintProcessor{})
	if err != nil {
		t.Fatalf("failed to create advanced consumer: %v", err)
	}
	defer consumer.Close(context.Background())

	err = consumer.Check(context.Background())
	t.Logf("Advanced consumer check: err=%v", err)
}
