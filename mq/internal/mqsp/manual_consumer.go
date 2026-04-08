package mqsp

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var _ mqspi.ManualConsumer = (*SaramaManualConsumer)(nil)

// metaKey is an unexported type to prevent external key collisions in Metadata.
type metaKey string

const (
	metaKeySession       metaKey = "session"
	metaKeyRaw           metaKey = "raw"
	retryDelayHeader             = "x-retry-delay-seconds"
	originalTopicHeader          = "x-original-topic"
)

type wrappedMessage struct {
	raw     *sarama.ConsumerMessage
	session sarama.ConsumerGroupSession
}

type SaramaManualConsumer struct {
	consumerGroup sarama.ConsumerGroup
	defaultTopic  string
	topics        []string
	brokers       []string
	credentials   mqspi.Credentials
	syncProducer  sarama.SyncProducer

	msgChan chan *wrappedMessage
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	closed  atomic.Bool
	errDone chan struct{}
}

func NewManualConsumer(config mqspi.ConsumerConfig) (mqspi.ManualConsumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	applySASL(saramaConfig, config.Credentials())

	topics := config.Topics()
	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers(), config.GroupID(), saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("mq: failed to create consumer group: %w", err)
	}

	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	applySASL(producerConfig, config.Credentials())

	syncProducer, err := sarama.NewSyncProducer(config.Brokers(), producerConfig)
	if err != nil {
		consumerGroup.Close()
		return nil, fmt.Errorf("mq: failed to create internal producer for retry/dlq: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &SaramaManualConsumer{
		consumerGroup: consumerGroup,
		defaultTopic:  config.Topic(),
		topics:        topics,
		brokers:       config.Brokers(),
		credentials:   config.Credentials(),
		syncProducer:  syncProducer,
		msgChan:       make(chan *wrappedMessage, 256),
		ctx:           ctx,
		cancel:        cancel,
	}

	c.errDone = make(chan struct{})
	c.wg.Add(1)
	go c.consumeLoop()
	go c.drainErrors()

	return c, nil
}

func (c *SaramaManualConsumer) consumeLoop() {
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

func (c *SaramaManualConsumer) drainErrors() {
	defer close(c.errDone)
	for range c.consumerGroup.Errors() {
	}
}

func (c *SaramaManualConsumer) Consume(ctx context.Context) (*mqspi.ConsumerMessage, error) {
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

func (c *SaramaManualConsumer) Confirm(msg *mqspi.ConsumerMessage) error {
	if msg.Metadata == nil {
		return fmt.Errorf("mq: message metadata is nil, cannot confirm")
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

func (c *SaramaManualConsumer) ColdRetry(_ context.Context, msg *mqspi.ConsumerMessage, seconds int64) error {
	sourceTopic := msg.Topic
	if sourceTopic == "" {
		sourceTopic = c.defaultTopic
	}
	retryTopic := sourceTopic + "_retry"

	retryMsg := &sarama.ProducerMessage{
		Topic: retryTopic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
		Headers: []sarama.RecordHeader{
			{Key: []byte(retryDelayHeader), Value: []byte(strconv.FormatInt(seconds, 10))},
			{Key: []byte(originalTopicHeader), Value: []byte(sourceTopic)},
		},
	}
	for _, h := range msg.Headers {
		retryMsg.Headers = append(retryMsg.Headers, sarama.RecordHeader{Key: h.Key, Value: h.Value})
	}
	_, _, err := c.syncProducer.SendMessage(retryMsg)
	return err
}

func (c *SaramaManualConsumer) DLQ(_ context.Context, msg *mqspi.ConsumerMessage) error {
	sourceTopic := msg.Topic
	if sourceTopic == "" {
		sourceTopic = c.defaultTopic
	}
	dlqTopic := sourceTopic + "_dlq"

	dlqMsg := &sarama.ProducerMessage{
		Topic: dlqTopic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
		Headers: []sarama.RecordHeader{
			{Key: []byte(originalTopicHeader), Value: []byte(sourceTopic)},
		},
	}
	for _, h := range msg.Headers {
		dlqMsg.Headers = append(dlqMsg.Headers, sarama.RecordHeader{Key: h.Key, Value: h.Value})
	}
	_, _, err := c.syncProducer.SendMessage(dlqMsg)
	return err
}

func (c *SaramaManualConsumer) Check(_ context.Context) error {
	cfg := sarama.NewConfig()
	applySASL(cfg, c.credentials)
	client, err := sarama.NewClient(c.brokers, cfg)
	if err != nil {
		return fmt.Errorf("mq: broker connectivity check failed: %w", err)
	}
	defer client.Close()

	if len(client.Brokers()) == 0 {
		return fmt.Errorf("mq: no active brokers found")
	}
	for _, topic := range c.topics {
		partitions, err := client.Partitions(topic)
		if err != nil {
			return fmt.Errorf("mq: topic %q check failed: %w", topic, err)
		}
		if len(partitions) == 0 {
			return fmt.Errorf("mq: topic %q has no partitions", topic)
		}
	}
	return nil
}

func (c *SaramaManualConsumer) Close(_ context.Context) error {
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
	if err := c.syncProducer.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("mq: errors closing manual consumer: %v", errs)
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
