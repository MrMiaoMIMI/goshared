package mqsp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
	brokers       []string
	defaultTopic  string
	credentials   mqspi.Credentials
	syncProducer  sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
	mu            sync.RWMutex
	wg            sync.WaitGroup
	closed        atomic.Bool
}

func NewProducer(config mqspi.ProducerConfig) (mqspi.Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3
	applySASL(saramaConfig, config.Credentials())

	syncProducer, err := sarama.NewSyncProducer(config.Brokers(), saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("mq: failed to create sync producer: %w", err)
	}

	asyncProducer, err := sarama.NewAsyncProducer(config.Brokers(), saramaConfig)
	if err != nil {
		syncProducer.Close()
		return nil, fmt.Errorf("mq: failed to create async producer: %w", err)
	}

	p := &SaramaProducer{
		brokers:       config.Brokers(),
		defaultTopic:  config.Topic(),
		credentials:   config.Credentials(),
		syncProducer:  syncProducer,
		asyncProducer: asyncProducer,
	}

	p.wg.Add(2)
	go p.handleAsyncSuccesses()
	go p.handleAsyncErrors()

	return p, nil
}

func (p *SaramaProducer) resolveTopic(msgTopic string) string {
	if msgTopic != "" {
		return msgTopic
	}
	return p.defaultTopic
}

func (p *SaramaProducer) Produce(_ context.Context, msg *mqspi.ProducerMessage) error {
	if p.closed.Load() {
		return mqspi.ErrProducerClosed
	}

	msg.Topic = p.resolveTopic(msg.Topic)
	saramaMsg := toSaramaProducerMessage(msg)
	partition, offset, err := p.syncProducer.SendMessage(saramaMsg)
	if err != nil {
		return err
	}
	msg.Partition = partition
	msg.Offset = offset
	return nil
}

func (p *SaramaProducer) BatchProduce(ctx context.Context, msgs []*mqspi.ProducerMessage) error {
	if p.closed.Load() {
		return mqspi.ErrProducerClosed
	}
	for _, msg := range msgs {
		if err := p.Produce(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *SaramaProducer) AsyncProduce(ctx context.Context, msg *mqspi.ProducerMessage, callback mqspi.AsyncProduceCallback) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed.Load() {
		if callback != nil {
			callback.Handle(ctx, msg, mqspi.ErrProducerClosed)
		}
		return
	}

	msg.Topic = p.resolveTopic(msg.Topic)
	saramaMsg := toSaramaProducerMessage(msg)
	saramaMsg.Metadata = &asyncMeta{
		originalMsg: msg,
		callback:    callback,
		ctx:         ctx,
	}
	p.asyncProducer.Input() <- saramaMsg
}

func (p *SaramaProducer) Check(_ context.Context) error {
	cfg := sarama.NewConfig()
	applySASL(cfg, p.credentials)
	client, err := sarama.NewClient(p.brokers, cfg)
	if err != nil {
		return fmt.Errorf("mq: broker connectivity check failed: %w", err)
	}
	defer client.Close()

	if len(client.Brokers()) == 0 {
		return fmt.Errorf("mq: no active brokers found")
	}
	return nil
}

func (p *SaramaProducer) Close(_ context.Context) error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
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
			meta.callback.Handle(meta.ctx, meta.originalMsg, nil)
		}
	}
}

func (p *SaramaProducer) handleAsyncErrors() {
	defer p.wg.Done()
	for pErr := range p.asyncProducer.Errors() {
		if meta, ok := pErr.Msg.Metadata.(*asyncMeta); ok && meta.callback != nil {
			meta.callback.Handle(meta.ctx, meta.originalMsg, pErr.Err)
		}
	}
}

func toSaramaProducerMessage(msg *mqspi.ProducerMessage) *sarama.ProducerMessage {
	saramaMsg := &sarama.ProducerMessage{
		Topic: msg.Topic,
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
