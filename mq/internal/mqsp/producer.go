package mqsp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var _ mqspi.Producer = (*SaramaProducer)(nil)

type asyncMeta struct {
	originalMsg *mqspi.ProducerMessage
	callback    mqspi.AsyncProduceCallback
	ctx         context.Context
}

type SaramaProducer struct {
	defaultTopic  string
	syncProducer  sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
	mu            sync.RWMutex
	wg            sync.WaitGroup
	closed        atomic.Bool
	done          chan struct{}
}

func NewProducer(config *mqspi.ProducerConfig) (mqspi.Producer, error) {
	if err := validateProducerConfig(config); err != nil {
		return nil, err
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3
	applySASL(saramaConfig, config.Credentials)

	brokers := cleanStrings(config.Brokers)
	syncProducer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("mq: failed to create sync producer: %w", err)
	}

	asyncProducer, err := sarama.NewAsyncProducer(brokers, saramaConfig)
	if err != nil {
		syncProducer.Close()
		return nil, fmt.Errorf("mq: failed to create async producer: %w", err)
	}

	p := &SaramaProducer{
		defaultTopic:  config.Topic,
		syncProducer:  syncProducer,
		asyncProducer: asyncProducer,
		done:          make(chan struct{}),
	}

	p.wg.Add(2)
	go p.handleAsyncSuccesses()
	go p.handleAsyncErrors()

	return p, nil
}

func (p *SaramaProducer) resolveTopic(msgTopic string) (string, error) {
	if msgTopic != "" {
		return msgTopic, nil
	}
	if p.defaultTopic != "" {
		return p.defaultTopic, nil
	}
	return "", fmt.Errorf("%w: producer topic is empty", mqspi.ErrInvalidConfig)
}

func (p *SaramaProducer) Produce(ctx context.Context, msg *mqspi.ProducerMessage) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.closed.Load() {
		return mqspi.ErrProducerClosed
	}
	if msg == nil {
		return fmt.Errorf("%w: producer message is nil", mqspi.ErrInvalidConfig)
	}

	topic, err := p.resolveTopic(msg.Topic)
	if err != nil {
		return err
	}
	msg.Topic = topic
	saramaMsg := toSaramaProducerMessage(msg, nil)
	partition, offset, err := p.syncProducer.SendMessage(saramaMsg)
	if err != nil {
		return err
	}
	msg.Partition = partition
	msg.Offset = offset
	msg.Timestamp = resolvedTimestamp(msg.Timestamp, saramaMsg.Timestamp)
	return nil
}

func (p *SaramaProducer) BatchProduce(ctx context.Context, msgs []*mqspi.ProducerMessage) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.closed.Load() {
		return mqspi.ErrProducerClosed
	}

	saramaMsgs := make([]*sarama.ProducerMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			return fmt.Errorf("%w: producer message is nil", mqspi.ErrInvalidConfig)
		}
		topic, err := p.resolveTopic(msg.Topic)
		if err != nil {
			return err
		}
		msg.Topic = topic
		saramaMsgs = append(saramaMsgs, toSaramaProducerMessage(msg, msg))
	}
	if len(saramaMsgs) == 0 {
		return nil
	}

	if err := p.syncProducer.SendMessages(saramaMsgs); err != nil {
		return err
	}
	for _, saramaMsg := range saramaMsgs {
		if originalMsg, ok := saramaMsg.Metadata.(*mqspi.ProducerMessage); ok {
			originalMsg.Partition = saramaMsg.Partition
			originalMsg.Offset = saramaMsg.Offset
			originalMsg.Timestamp = resolvedTimestamp(originalMsg.Timestamp, saramaMsg.Timestamp)
		}
	}
	return nil
}

func (p *SaramaProducer) AsyncProduce(ctx context.Context, msg *mqspi.ProducerMessage, callback mqspi.AsyncProduceCallback) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("%w: producer message is nil", mqspi.ErrInvalidConfig)
	}

	topic, err := p.resolveTopic(msg.Topic)
	if err != nil {
		return err
	}
	msg.Topic = topic
	saramaMsg := toSaramaProducerMessage(msg, &asyncMeta{
		originalMsg: msg,
		callback:    callback,
		ctx:         ctx,
	})

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed.Load() {
		return mqspi.ErrProducerClosed
	}

	select {
	case p.asyncProducer.Input() <- saramaMsg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return mqspi.ErrProducerClosed
	}
}

func (p *SaramaProducer) Close(_ context.Context) error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(p.done)
	// Wait for in-flight AsyncProduce calls to finish sending to Input()
	p.mu.Lock()
	p.mu.Unlock()

	var errs []error
	if err := p.asyncProducer.Close(); err != nil {
		errs = append(errs, err)
	}
	p.wg.Wait()
	if err := p.syncProducer.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("mq: errors closing producer: %v", errs)
	}
	return nil
}

func (p *SaramaProducer) handleAsyncSuccesses() {
	defer p.wg.Done()
	for msg := range p.asyncProducer.Successes() {
		if meta, ok := msg.Metadata.(*asyncMeta); ok && meta.callback != nil {
			meta.originalMsg.Partition = msg.Partition
			meta.originalMsg.Offset = msg.Offset
			meta.originalMsg.Timestamp = resolvedTimestamp(meta.originalMsg.Timestamp, msg.Timestamp)
			meta.callback(meta.ctx, meta.originalMsg, nil)
		}
	}
}

func (p *SaramaProducer) handleAsyncErrors() {
	defer p.wg.Done()
	for pErr := range p.asyncProducer.Errors() {
		if meta, ok := pErr.Msg.Metadata.(*asyncMeta); ok && meta.callback != nil {
			meta.callback(meta.ctx, meta.originalMsg, pErr.Err)
		}
	}
}

func toSaramaProducerMessage(msg *mqspi.ProducerMessage, metadata any) *sarama.ProducerMessage {
	saramaMsg := &sarama.ProducerMessage{
		Topic:    msg.Topic,
		Metadata: metadata,
	}
	if msg.Key != nil {
		saramaMsg.Key = sarama.ByteEncoder(msg.Key)
	}
	if msg.Value != nil {
		saramaMsg.Value = sarama.ByteEncoder(msg.Value)
	}
	if len(msg.Headers) > 0 {
		headers := make([]sarama.RecordHeader, len(msg.Headers))
		for i, h := range msg.Headers {
			headers[i] = sarama.RecordHeader{Key: h.Key, Value: h.Value}
		}
		saramaMsg.Headers = headers
	}
	if !msg.Timestamp.IsZero() {
		saramaMsg.Timestamp = msg.Timestamp
	}
	return saramaMsg
}

func resolvedTimestamp(original, delivered time.Time) time.Time {
	if !delivered.IsZero() {
		return delivered
	}
	return original
}
