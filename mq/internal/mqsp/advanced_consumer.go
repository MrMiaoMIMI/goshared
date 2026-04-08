package mqsp

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var _ mqspi.AdvancedConsumer = (*SaramaAdvancedConsumer)(nil)

const retryCountHeader = "x-retry-count"

type SaramaAdvancedConsumer struct {
	consumerGroup  sarama.ConsumerGroup
	topics         []string
	brokers        []string
	credentials    mqspi.Credentials
	processor      mqspi.MessageProcessor
	batchProcessor mqspi.BatchMessageProcessor
	maxRetries     int
	dlqTopic       string
	syncProducer   sarama.SyncProducer

	ctx      context.Context
	cancel   context.CancelFunc
	closed   atomic.Bool
	errDone  chan struct{}
}

func NewAdvancedConsumer(config mqspi.AdvancedConsumerConfig, processor mqspi.MessageProcessor) (mqspi.AdvancedConsumer, error) {
	return newAdvancedConsumer(config, processor, nil)
}

func NewAdvancedBatchConsumer(config mqspi.AdvancedConsumerConfig, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
	return newAdvancedConsumer(config, nil, batchProcessor)
}

func newAdvancedConsumer(config mqspi.AdvancedConsumerConfig, processor mqspi.MessageProcessor, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	applySASL(saramaConfig, config.Credentials())

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers(), config.GroupID(), saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("mq: failed to create consumer group: %w", err)
	}

	var syncProducer sarama.SyncProducer
	if config.DLQTopic() != "" || config.MaxRetries() > 0 {
		producerConfig := sarama.NewConfig()
		producerConfig.Producer.Return.Successes = true
		producerConfig.Producer.RequiredAcks = sarama.WaitForAll
		applySASL(producerConfig, config.Credentials())
		syncProducer, err = sarama.NewSyncProducer(config.Brokers(), producerConfig)
		if err != nil {
			consumerGroup.Close()
			return nil, fmt.Errorf("mq: failed to create internal producer for retry/dlq: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	ac := &SaramaAdvancedConsumer{
		consumerGroup:  consumerGroup,
		topics:         config.Topics(),
		brokers:        config.Brokers(),
		credentials:    config.Credentials(),
		processor:      processor,
		batchProcessor: batchProcessor,
		maxRetries:     config.MaxRetries(),
		dlqTopic:       config.DLQTopic(),
		syncProducer:   syncProducer,
		ctx:            ctx,
		cancel:         cancel,
		errDone:        make(chan struct{}),
	}

	go ac.drainErrors()

	return ac, nil
}

func (c *SaramaAdvancedConsumer) drainErrors() {
	defer close(c.errDone)
	for range c.consumerGroup.Errors() {
	}
}

func (c *SaramaAdvancedConsumer) Run() error {
	handler := &advancedConsumerHandler{
		processor:      c.processor,
		batchProcessor: c.batchProcessor,
		maxRetries:     c.maxRetries,
		dlqTopic:       c.dlqTopic,
		syncProducer:   c.syncProducer,
	}

	for {
		if c.ctx.Err() != nil {
			return nil
		}
		if err := c.consumerGroup.Consume(c.ctx, c.topics, handler); err != nil {
			if c.ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("mq: consumer group error: %w", err)
		}
	}
}

func (c *SaramaAdvancedConsumer) Close(_ context.Context) error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()

	var errs []error
	if err := c.consumerGroup.Close(); err != nil {
		errs = append(errs, err)
	}
	<-c.errDone
	if c.syncProducer != nil {
		if err := c.syncProducer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("mq: errors closing advanced consumer: %v", errs)
	}
	return nil
}

func (c *SaramaAdvancedConsumer) Check(_ context.Context) error {
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

// advancedConsumerHandler implements sarama.ConsumerGroupHandler
type advancedConsumerHandler struct {
	processor      mqspi.MessageProcessor
	batchProcessor mqspi.BatchMessageProcessor
	maxRetries     int
	dlqTopic       string
	syncProducer   sarama.SyncProducer
}

func (h *advancedConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *advancedConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *advancedConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if h.processor != nil {
		return h.consumeClaimSingle(session, claim)
	}
	if h.batchProcessor != nil {
		return h.consumeClaimBatch(session, claim)
	}
	return fmt.Errorf("mq: no processor configured")
}

func (h *advancedConsumerHandler) consumeClaimSingle(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case raw, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			msg := fromSaramaConsumerMessage(raw)
			retryCount := getRetryCount(raw)

			err := h.processor.Process(session.Context(), msg)
			if err == nil {
				session.MarkMessage(raw, "")
				continue
			}

			if retryCount < h.maxRetries {
				if retryErr := h.sendRetry(raw, retryCount+1); retryErr != nil {
					h.sendDLQ(raw)
				}
			} else if h.dlqTopic != "" {
				h.sendDLQ(raw)
			}
			session.MarkMessage(raw, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *advancedConsumerHandler) consumeClaimBatch(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	const batchSize = 100

	for {
		batch := make([]*sarama.ConsumerMessage, 0, batchSize)

		// Block until we get at least one message
		select {
		case raw, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			batch = append(batch, raw)
		case <-session.Context().Done():
			return nil
		}

		// Drain more messages non-blocking, up to batchSize
	drain:
		for len(batch) < batchSize {
			select {
			case raw, ok := <-claim.Messages():
				if !ok {
					break drain
				}
				batch = append(batch, raw)
			default:
				break drain
			}
		}
		msgs := make([]*mqspi.ConsumerMessage, len(batch))
		for i, raw := range batch {
			msgs[i] = fromSaramaConsumerMessage(raw)
		}

		err := h.batchProcessor.BatchProcess(session.Context(), msgs)
		if err != nil {
			for _, raw := range batch {
				retryCount := getRetryCount(raw)
				if retryCount < h.maxRetries {
					if retryErr := h.sendRetry(raw, retryCount+1); retryErr != nil {
						h.sendDLQ(raw)
					}
				} else if h.dlqTopic != "" {
					h.sendDLQ(raw)
				}
			}
		}

		for _, raw := range batch {
			session.MarkMessage(raw, "")
		}
	}
}

func (h *advancedConsumerHandler) sendRetry(raw *sarama.ConsumerMessage, retryCount int) error {
	if h.syncProducer == nil {
		return fmt.Errorf("mq: no producer configured for retry")
	}
	retryMsg := &sarama.ProducerMessage{
		Topic: raw.Topic,
		Key:   sarama.ByteEncoder(raw.Key),
		Value: sarama.ByteEncoder(raw.Value),
		Headers: []sarama.RecordHeader{
			{Key: []byte(retryCountHeader), Value: []byte(strconv.Itoa(retryCount))},
		},
	}
	for _, header := range raw.Headers {
		if header != nil && string(header.Key) != retryCountHeader {
			retryMsg.Headers = append(retryMsg.Headers, sarama.RecordHeader{Key: header.Key, Value: header.Value})
		}
	}
	_, _, err := h.syncProducer.SendMessage(retryMsg)
	return err
}

func (h *advancedConsumerHandler) sendDLQ(raw *sarama.ConsumerMessage) {
	if h.syncProducer == nil || h.dlqTopic == "" {
		return
	}
	dlqMsg := &sarama.ProducerMessage{
		Topic: h.dlqTopic,
		Key:   sarama.ByteEncoder(raw.Key),
		Value: sarama.ByteEncoder(raw.Value),
		Headers: []sarama.RecordHeader{
			{Key: []byte(originalTopicHeader), Value: []byte(raw.Topic)},
		},
	}
	for _, header := range raw.Headers {
		if header != nil {
			dlqMsg.Headers = append(dlqMsg.Headers, sarama.RecordHeader{Key: header.Key, Value: header.Value})
		}
	}
	h.syncProducer.SendMessage(dlqMsg)
}

func getRetryCount(msg *sarama.ConsumerMessage) int {
	for _, h := range msg.Headers {
		if h != nil && string(h.Key) == retryCountHeader {
			count, _ := strconv.Atoi(string(h.Value))
			return count
		}
	}
	return 0
}
