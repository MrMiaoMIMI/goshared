package mqsp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

var _ mqspi.AdvancedConsumer = (*SaramaAdvancedConsumer)(nil)

const defaultBatchSize = 100

type SaramaAdvancedConsumer struct {
	consumerGroup  sarama.ConsumerGroup
	topics         []string
	processor      mqspi.MessageProcessor
	batchProcessor mqspi.BatchMessageProcessor
	strategy       mqspi.ConsumerFailureStrategy
	failures       *failureTracker

	ctx     context.Context
	cancel  context.CancelFunc
	closed  atomic.Bool
	errDone chan struct{}
}

func NewAdvancedConsumer(config *mqspi.ConsumerConfig, processor mqspi.MessageProcessor) (mqspi.AdvancedConsumer, error) {
	return newAdvancedConsumer(config, processor, nil)
}

func NewAdvancedBatchConsumer(config *mqspi.ConsumerConfig, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
	return newAdvancedConsumer(config, nil, batchProcessor)
}

func newAdvancedConsumer(config *mqspi.ConsumerConfig, processor mqspi.MessageProcessor, batchProcessor mqspi.BatchMessageProcessor) (mqspi.AdvancedConsumer, error) {
	if err := validateConsumerConfig(config); err != nil {
		return nil, err
	}
	if processor == nil && batchProcessor == nil {
		return nil, fmt.Errorf("%w: processor is nil", mqspi.ErrInvalidConfig)
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

	ac := &SaramaAdvancedConsumer{
		consumerGroup:  consumerGroup,
		topics:         topics,
		processor:      processor,
		batchProcessor: batchProcessor,
		strategy:       consumerFailureStrategy(config),
		failures:       newFailureTracker(),
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

func (c *SaramaAdvancedConsumer) Run(ctx context.Context) error {
	runCtx, stop := mergeContexts(ctx, c.ctx)
	defer stop()

	handler := &advancedConsumerHandler{
		processor:      c.processor,
		batchProcessor: c.batchProcessor,
		strategy:       c.strategy,
		failures:       c.failures,
	}

	for {
		if runCtx.Err() != nil {
			return nil
		}
		if err := c.consumerGroup.Consume(runCtx, c.topics, handler); err != nil {
			if runCtx.Err() != nil {
				return nil
			}
			var failureErr *processFailureError
			if errors.As(err, &failureErr) {
				if failureErr.decision.Action == mqspi.ConsumerFailureActionRetry {
					if waitErr := waitFailureBackoff(runCtx, failureErr.decision.Backoff); waitErr != nil {
						return nil
					}
					continue
				}
				return fmt.Errorf("mq: consumer stopped by failure strategy: %w", failureErr)
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
	if len(errs) > 0 {
		return fmt.Errorf("mq: errors closing advanced consumer: %v", errs)
	}
	return nil
}

// advancedConsumerHandler implements sarama.ConsumerGroupHandler
type advancedConsumerHandler struct {
	processor      mqspi.MessageProcessor
	batchProcessor mqspi.BatchMessageProcessor
	strategy       mqspi.ConsumerFailureStrategy
	failures       *failureTracker
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

			if err := h.processor.Process(session.Context(), msg); err != nil {
				if failureErr := h.handleFailure(session, []*sarama.ConsumerMessage{raw}, []*mqspi.ConsumerMessage{msg}, err); failureErr != nil {
					return failureErr
				}
				continue
			}
			session.MarkMessage(raw, "")
			session.Commit()
			h.failures.reset([]*sarama.ConsumerMessage{raw})

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *advancedConsumerHandler) consumeClaimBatch(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		batch := make([]*sarama.ConsumerMessage, 0, defaultBatchSize)

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
		for len(batch) < defaultBatchSize {
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

		if err := h.batchProcessor.BatchProcess(session.Context(), msgs); err != nil {
			if failureErr := h.handleFailure(session, batch, msgs, err); failureErr != nil {
				return failureErr
			}
			continue
		}

		for _, raw := range batch {
			session.MarkMessage(raw, "")
		}
		session.Commit()
		h.failures.reset(batch)
	}
}

func (h *advancedConsumerHandler) handleFailure(session sarama.ConsumerGroupSession, raws []*sarama.ConsumerMessage, msgs []*mqspi.ConsumerMessage, err error) error {
	attempt := h.failures.increment(raws)
	failure := &mqspi.ConsumerFailure{
		Err:      err,
		Attempt:  attempt,
		Messages: msgs,
	}
	if len(msgs) > 0 {
		failure.Message = msgs[0]
	}

	strategy := h.strategy
	if strategy == nil {
		strategy = mqspi.DefaultConsumerFailurePolicy()
	}
	decision := normalizeFailureDecision(strategy.Decide(session.Context(), failure))
	switch decision.Action {
	case mqspi.ConsumerFailureActionSkip:
		for _, raw := range raws {
			session.MarkMessage(raw, "")
		}
		session.Commit()
		h.failures.reset(raws)
		return nil
	case mqspi.ConsumerFailureActionStop:
		h.failures.reset(raws)
		return &processFailureError{failure: failure, decision: decision}
	case mqspi.ConsumerFailureActionRetry:
		return &processFailureError{failure: failure, decision: decision}
	default:
		decision.Action = mqspi.ConsumerFailureActionRetry
		return &processFailureError{failure: failure, decision: decision}
	}
}

func mergeContexts(ctx context.Context, internal context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-internal.Done():
			cancel()
		case <-runCtx.Done():
		}
	}()
	return runCtx, cancel
}

func consumerFailureStrategy(config *mqspi.ConsumerConfig) mqspi.ConsumerFailureStrategy {
	if config != nil && config.FailureStrategy != nil {
		return config.FailureStrategy
	}
	if config != nil && config.FailurePolicy != nil {
		return config.FailurePolicy
	}
	return mqspi.DefaultConsumerFailurePolicy()
}

type processFailureError struct {
	failure  *mqspi.ConsumerFailure
	decision mqspi.ConsumerFailureDecision
}

func (e *processFailureError) Error() string {
	if e == nil || e.failure == nil || e.failure.Err == nil {
		return "mq: consumer processor failed"
	}
	return fmt.Sprintf("mq: consumer processor failed after attempt %d: %v", e.failure.Attempt, e.failure.Err)
}

func (e *processFailureError) Unwrap() error {
	if e == nil || e.failure == nil {
		return nil
	}
	return e.failure.Err
}

func normalizeFailureDecision(decision mqspi.ConsumerFailureDecision) mqspi.ConsumerFailureDecision {
	switch decision.Action {
	case mqspi.ConsumerFailureActionRetry, mqspi.ConsumerFailureActionSkip, mqspi.ConsumerFailureActionStop:
	case "":
		decision.Action = mqspi.ConsumerFailureActionRetry
	default:
		decision.Action = mqspi.ConsumerFailureActionRetry
	}
	if decision.Backoff < 0 {
		decision.Backoff = 0
	}
	return decision
}

func waitFailureBackoff(ctx context.Context, backoff time.Duration) error {
	if backoff <= 0 {
		return nil
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type failureTracker struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newFailureTracker() *failureTracker {
	return &failureTracker{attempts: make(map[string]int)}
}

func (t *failureTracker) increment(raws []*sarama.ConsumerMessage) int {
	key := failureKey(raws)
	if key == "" {
		return 1
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	t.attempts[key]++
	return t.attempts[key]
}

func (t *failureTracker) reset(raws []*sarama.ConsumerMessage) {
	key := failureKey(raws)
	if key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.attempts, key)
}

func failureKey(raws []*sarama.ConsumerMessage) string {
	if len(raws) == 0 || raws[0] == nil {
		return ""
	}
	first := raws[0]
	if len(raws) == 1 {
		return fmt.Sprintf("%s:%d:%d", first.Topic, first.Partition, first.Offset)
	}
	last := raws[len(raws)-1]
	return fmt.Sprintf("%s:%d:%d-%d", first.Topic, first.Partition, first.Offset, last.Offset)
}
