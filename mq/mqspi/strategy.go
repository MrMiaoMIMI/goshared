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

// ConsumerFailurePolicy configures the default failure strategy for AdvancedConsumer.
// MaxAttempts <= 0 means retry forever. FinalAction is used after MaxAttempts is reached.
// The policy never sends messages to DLQ. Handler is code-only and is intentionally
// excluded from JSON and YAML.
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
