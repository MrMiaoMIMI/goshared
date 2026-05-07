package mqsp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var _ mqspi.Consumer = (*SaramaConsumer)(nil)

// metaKey is an unexported type to prevent external key collisions in Metadata.
type metaKey string

const (
	metaKeySession metaKey = "session"
	metaKeyRaw     metaKey = "raw"
)

type wrappedMessage struct {
	raw     *sarama.ConsumerMessage
	session sarama.ConsumerGroupSession
}

type SaramaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string

	msgChan chan *wrappedMessage
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	closed  atomic.Bool
	errDone chan struct{}
}

func NewConsumer(config *mqspi.ConsumerConfig) (mqspi.Consumer, error) {
	if err := validateConsumerConfig(config); err != nil {
		return nil, err
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	applySASL(saramaConfig, config.Credentials)

	brokers := cleanStrings(config.Brokers)
	topics := consumerTopics(config)
	consumerGroup, err := sarama.NewConsumerGroup(brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("mq: failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &SaramaConsumer{
		consumerGroup: consumerGroup,
		topics:        topics,
		msgChan:       make(chan *wrappedMessage, consumerBufferSize(config)),
		ctx:           ctx,
		cancel:        cancel,
	}

	c.errDone = make(chan struct{})
	c.wg.Add(1)
	go c.consumeLoop()
	go c.drainErrors()

	return c, nil
}

func (c *SaramaConsumer) consumeLoop() {
	defer c.wg.Done()
	handler := &manualConsumerHandler{msgChan: c.msgChan}
	for {
		if c.ctx.Err() != nil {
			return
		}
		if err := c.consumerGroup.Consume(c.ctx, c.topics, handler); err != nil {
			if c.ctx.Err() != nil {
				return
			}
		}
	}
}

func (c *SaramaConsumer) drainErrors() {
	defer close(c.errDone)
	for range c.consumerGroup.Errors() {
	}
}

func (c *SaramaConsumer) Consume(ctx context.Context) (*mqspi.ConsumerMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.closed.Load() {
		return nil, mqspi.ErrConsumerClosed
	}

	select {
	case <-ctx.Done():
		return nil, mqspi.ErrConsumeContextDone
	case <-c.ctx.Done():
		return nil, mqspi.ErrConsumerClosed
	case wrapped, ok := <-c.msgChan:
		if !ok {
			return nil, mqspi.ErrConsumerClosed
		}
		msg := fromSaramaConsumerMessage(wrapped.raw)
		msg.Metadata[metaKeySession] = wrapped.session
		msg.Metadata[metaKeyRaw] = wrapped.raw
		return msg, nil
	}
}

func (c *SaramaConsumer) Ack(ctx context.Context, msg *mqspi.ConsumerMessage) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.closed.Load() {
		return mqspi.ErrConsumerClosed
	}
	if msg == nil {
		return fmt.Errorf("%w: consumer message is nil", mqspi.ErrInvalidConfig)
	}
	if msg.Metadata == nil {
		return fmt.Errorf("mq: message metadata is nil, cannot ack")
	}

	session, ok := msg.Metadata[metaKeySession].(sarama.ConsumerGroupSession)
	if !ok {
		return fmt.Errorf("mq: session not found in metadata")
	}
	raw, ok := msg.Metadata[metaKeyRaw].(*sarama.ConsumerMessage)
	if !ok {
		return fmt.Errorf("mq: raw message not found in metadata")
	}
	session.MarkMessage(raw, "")
	session.Commit()
	return nil
}

func (c *SaramaConsumer) Close(_ context.Context) error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()
	c.wg.Wait()

	var errs []error
	if err := c.consumerGroup.Close(); err != nil {
		errs = append(errs, err)
	}
	<-c.errDone
	if len(errs) > 0 {
		return fmt.Errorf("mq: errors closing consumer: %v", errs)
	}
	return nil
}

// manualConsumerHandler implements sarama.ConsumerGroupHandler
type manualConsumerHandler struct {
	msgChan chan *wrappedMessage
}

func (h *manualConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *manualConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *manualConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			select {
			case h.msgChan <- &wrappedMessage{raw: msg, session: session}:
			case <-session.Context().Done():
				return nil
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

func fromSaramaConsumerMessage(raw *sarama.ConsumerMessage) *mqspi.ConsumerMessage {
	headers := make([]mqspi.Header, len(raw.Headers))
	for i, h := range raw.Headers {
		if h != nil {
			headers[i] = mqspi.Header{Key: h.Key, Value: h.Value}
		}
	}
	return &mqspi.ConsumerMessage{
		Topic:     raw.Topic,
		Key:       raw.Key,
		Value:     raw.Value,
		Headers:   headers,
		Partition: raw.Partition,
		Offset:    raw.Offset,
		Timestamp: raw.Timestamp,
		Metadata:  make(mqspi.Metadata),
	}
}
