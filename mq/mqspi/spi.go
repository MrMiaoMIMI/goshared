package mqspi

import (
	"context"
	"time"
)

// Producer sends messages to MQ.
//
// Produce and BatchProduce are synchronous. AsyncProduce only reports whether
// the message entered the async producer; final delivery result is returned
// through the callback.
type Producer interface {
	// Produce sends one message synchronously.
	// If msg.Topic is empty, the producer default topic is used.
	// On success, Partition, Offset, and Timestamp are written back to msg.
	Produce(ctx context.Context, msg *ProducerMessage) error
	// BatchProduce sends multiple messages synchronously through the underlying
	// batch producer. If a message topic is empty, the producer default topic is used.
	BatchProduce(ctx context.Context, msgs []*ProducerMessage) error
	// AsyncProduce enqueues one message for asynchronous delivery.
	// The callback receives the final broker delivery result.
	AsyncProduce(ctx context.Context, msg *ProducerMessage, callback AsyncProduceCallback) error
	// Close gracefully shuts down the producer and releases all resources.
	Close(ctx context.Context) error
}

// Consumer is a manual consumer. Call Consume in an application-owned loop and
// call Ack only after the message has been processed successfully.
type Consumer interface {
	// Consume tries to fetch a message before ctx is done.
	// Possible errors: ErrConsumerClosed, ErrConsumeContextDone.
	Consume(ctx context.Context) (*ConsumerMessage, error)
	// Ack confirms a message after successful processing and commits its offset.
	Ack(ctx context.Context, msg *ConsumerMessage) error
	// Close gracefully shuts down the consumer and releases all resources.
	Close(ctx context.Context) error
}

// AdvancedConsumer owns the consume loop and delegates message handling to a
// MessageProcessor or BatchMessageProcessor.
type AdvancedConsumer interface {
	// Run starts consuming and blocks until ctx is done, Close is called, the
	// consumer group returns an unrecoverable error, or the failure strategy stops it.
	Run(ctx context.Context) error
	// Close stops the consumer and releases resources.
	Close(ctx context.Context) error
}

// MessageProcessor handles one message for AdvancedConsumer.
// Returning nil commits the message offset. Returning an error delegates the
// message to the configured ConsumerFailureStrategy.
type MessageProcessor interface {
	Process(ctx context.Context, msg *ConsumerMessage) error
}

// BatchMessageProcessor handles a batch of messages for AdvancedConsumer.
// Returning nil commits the whole batch. Returning an error delegates the batch
// to the configured ConsumerFailureStrategy.
type BatchMessageProcessor interface {
	BatchProcess(ctx context.Context, msgs []*ConsumerMessage) error
}

// ProducerMessage is the public message shape used by Producer.
// Metadata is reserved by the implementation and ignored by JSON/YAML encoding.
type ProducerMessage struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   []Header
	Partition int32
	Offset    int64
	Timestamp time.Time

	Metadata Metadata `json:"-" yaml:"-"`
}

// ConsumerMessage is the public message shape returned by Consumer and passed
// to AdvancedConsumer processors.
type ConsumerMessage struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   []Header
	Partition int32
	Offset    int64
	Timestamp time.Time

	// Metadata is reserved for the MQ implementation. User code should not modify it.
	Metadata Metadata `json:"-" yaml:"-"`
}

type Header struct {
	Key   []byte
	Value []byte
}

// Metadata carries implementation-specific context.
// Business code should treat it as read-only.
type Metadata map[interface{}]interface{}

// AsyncProduceCallback receives the broker result for AsyncProduce.
type AsyncProduceCallback func(ctx context.Context, msg *ProducerMessage, err error)
