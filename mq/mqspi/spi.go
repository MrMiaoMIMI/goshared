package mqspi

import (
	"context"
	"time"
)

type Producer interface {
	Produce(ctx context.Context, msg *ProducerMessage) error
	AsyncProduce(ctx context.Context, msg *ProducerMessage, callback AsyncProduceCallback)
	// Check checks whether there are active brokers and whether configured topics exist
	Check(ctx context.Context) error
	// Close gracefully shuts down the producer and releases all resources.
	Close(ctx context.Context) error
}

type ManualConsumer interface {
	// Consume tries to fetch a message before ctx is done.
	// Normally users need to put it in a loop, for example:
	//
	// // continuously fetch messages until ctx is done
	// for {
	// 	case <-ctx.Done();
	// 		return
	// 	default:
	// 		msg, err := c.Consume(ctx)
	// 		if err!=nil {
	// 			ulog.Error("xxxxxx")
	// 			continue
	// 		}
	//
	// 		// message processing logic
	// 		xxx
	//
	// 		err = c.Confirm(msg)
	// 		if err!=nil {
	// 			ulog.Error("xxxxxx")
	// 		}
	// }
	//
	// Possible errors: ErrConsumerClosed, ErrConsumerUpdating, ErrConsumeContextDone.
	Consume(ctx context.Context) (*ConsumerMessage, error)

	// Confirm a message after processing.
	Confirm(msg *ConsumerMessage) error

	// ColdRetry reproduces the given message to the delayed topic(s) on Kafka clusters.
	// And messages will be re-consumed by one of instances of this consumer group after "AT LEAST seconds".
	ColdRetry(ctx context.Context, msg *ConsumerMessage, seconds int64) error

	// DLQ sends a message to DLQ.
	DLQ(ctx context.Context, msg *ConsumerMessage) error

	// Check checks whether there are active brokers and whether configured topics exist
	Check(ctx context.Context) error
	// Close gracefully shuts down the consumer and releases all resources.
	Close(ctx context.Context) error
}

type AdvancedConsumer interface {
	// Run is used to run this consumer.
	// Note that this method will block until Close() are called.
	Run() error
	// Close is used to close this consumer.
	// Note: Currently, the ctx parameter is not used internally.
	Close(ctx context.Context) error
	// Check checks whether there are active brokers and whether configured topics exist
	Check(ctx context.Context) error
}

// MessageProcessor is the interface for consuming a single message
// Use with AdvancedConsumer
type MessageProcessor interface {
	Process(ctx context.Context, msg *ConsumerMessage) error
}

// BatchMessageProcessor is the interface for consuming a batch of messages
// Use with AdvancedConsumer
type BatchMessageProcessor interface {
	BatchProcess(ctx context.Context, msgs []*ConsumerMessage) error
}

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

type ConsumerMessage struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   []Header
	Partition int32
	Offset    int64
	Timestamp time.Time

	Metadata Metadata `json:"-" yaml:"-"`
}

type Header struct {
	Key   []byte
	Value []byte
}

// Metadata is sent with every message to provide extra context of the specifics MQ implementation.
type Metadata map[interface{}]interface{}

type AsyncProduceCallback interface {
	Handle(ctx context.Context, msg *ProducerMessage, err error)
}
