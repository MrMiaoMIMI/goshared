package mqspi

import (
	"context"
	"time"
)

const (
	// ConsumerFailureActionRetry keeps the failed message uncommitted and retries it later.
	ConsumerFailureActionRetry ConsumerFailureAction = "retry"
	// ConsumerFailureActionSkip commits the failed message or batch and continues consuming.
	// Use it only when losing or bypassing that message is acceptable.
	ConsumerFailureActionSkip ConsumerFailureAction = "skip"
	// ConsumerFailureActionStop stops AdvancedConsumer.Run and returns the processor error.
	ConsumerFailureActionStop ConsumerFailureAction = "stop"
)

const (
	// DefaultFailureInitialBackoff is the first retry delay used by ConsumerFailurePolicy.
	DefaultFailureInitialBackoff = time.Second
	// DefaultFailureMaxBackoff is the upper bound for retry delay used by ConsumerFailurePolicy.
	DefaultFailureMaxBackoff = 30 * time.Second
	// DefaultFailureBackoffMultiple is the exponential backoff multiplier used by ConsumerFailurePolicy.
	DefaultFailureBackoffMultiple = 2.0
)

// ConsumerFailureAction describes how AdvancedConsumer should handle a failed
// processor call.
type ConsumerFailureAction string

// ConsumerFailure describes one processor failure passed to a strategy.
// Message is the first failed message. Messages contains the whole failed batch.
type ConsumerFailure struct {
	Err      error
	Message  *ConsumerMessage
	Messages []*ConsumerMessage
	Attempt  int
}

// ConsumerFailureDecision is returned by ConsumerFailureStrategy.
type ConsumerFailureDecision struct {
	Action  ConsumerFailureAction
	Backoff time.Duration
}

// ConsumerFailureHandler observes processor failures without deciding the final action.
type ConsumerFailureHandler func(ctx context.Context, failure *ConsumerFailure)

// ConsumerFailureStrategy decides what AdvancedConsumer should do after a
// processor failure.
type ConsumerFailureStrategy interface {
	Decide(ctx context.Context, failure *ConsumerFailure) ConsumerFailureDecision
}

// ConsumerFailurePolicy is the default configurable failure strategy for AdvancedConsumer.
// MaxAttempts <= 0 means retry forever. FinalAction is used after MaxAttempts is reached.
// The policy never sends messages to DLQ. It is a plain config struct; Handler
// is intentionally excluded from JSON and YAML.
type ConsumerFailurePolicy struct {
	// MaxAttempts is the maximum attempt count before FinalAction is applied.
	// Values <= 0 mean retry forever.
	MaxAttempts int `json:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	// InitialBackoff is used for the first retry.
	InitialBackoff time.Duration `json:"initial_backoff,omitempty" yaml:"initial_backoff,omitempty"`
	// MaxBackoff caps exponential retry backoff.
	MaxBackoff time.Duration `json:"max_backoff,omitempty" yaml:"max_backoff,omitempty"`
	// BackoffMultiplier grows backoff between attempts. Values <= 1 use the default.
	BackoffMultiplier float64 `json:"backoff_multiplier,omitempty" yaml:"backoff_multiplier,omitempty"`
	// FinalAction is applied after MaxAttempts is reached.
	FinalAction ConsumerFailureAction `json:"final_action,omitempty" yaml:"final_action,omitempty"`
	// Handler is called on every failure before the decision is returned.
	Handler ConsumerFailureHandler `json:"-" yaml:"-"`
}

// DefaultConsumerFailurePolicy returns the default advanced-consumer failure policy:
// retry forever with exponential backoff from 1s to 30s.
func DefaultConsumerFailurePolicy() *ConsumerFailurePolicy {
	return &ConsumerFailurePolicy{
		InitialBackoff:    DefaultFailureInitialBackoff,
		MaxBackoff:        DefaultFailureMaxBackoff,
		BackoffMultiplier: DefaultFailureBackoffMultiple,
		FinalAction:       ConsumerFailureActionStop,
	}
}

// Decide implements ConsumerFailureStrategy.
func (p *ConsumerFailurePolicy) Decide(ctx context.Context, failure *ConsumerFailure) ConsumerFailureDecision {
	policy := normalizeConsumerFailurePolicy(p)
	if policy.Handler != nil {
		policy.Handler(ctx, failure)
	}

	if policy.MaxAttempts > 0 && failure != nil && failure.Attempt >= policy.MaxAttempts {
		if policy.FinalAction == ConsumerFailureActionRetry {
			return ConsumerFailureDecision{
				Action:  ConsumerFailureActionRetry,
				Backoff: policy.backoff(failureAttempt(failure)),
			}
		}
		return ConsumerFailureDecision{Action: policy.FinalAction}
	}

	return ConsumerFailureDecision{
		Action:  ConsumerFailureActionRetry,
		Backoff: policy.backoff(failureAttempt(failure)),
	}
}

func normalizeConsumerFailurePolicy(policy *ConsumerFailurePolicy) ConsumerFailurePolicy {
	if policy == nil {
		return *DefaultConsumerFailurePolicy()
	}

	out := *policy
	if out.InitialBackoff <= 0 {
		out.InitialBackoff = DefaultFailureInitialBackoff
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = DefaultFailureMaxBackoff
	}
	if out.BackoffMultiplier <= 1 {
		out.BackoffMultiplier = DefaultFailureBackoffMultiple
	}
	if out.FinalAction == "" {
		out.FinalAction = ConsumerFailureActionStop
	}
	return out
}

func (p ConsumerFailurePolicy) backoff(attempt int) time.Duration {
	if attempt <= 1 {
		return p.InitialBackoff
	}

	backoff := p.InitialBackoff
	for i := 1; i < attempt; i++ {
		next := time.Duration(float64(backoff) * p.BackoffMultiplier)
		if next <= 0 || next >= p.MaxBackoff {
			return p.MaxBackoff
		}
		backoff = next
	}
	return backoff
}

func failureAttempt(failure *ConsumerFailure) int {
	if failure == nil || failure.Attempt <= 0 {
		return 1
	}
	return failure.Attempt
}
