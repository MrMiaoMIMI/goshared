package mqsp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

func TestConsumerFailurePolicyStrategyRetriesWithBackoff(t *testing.T) {
	strategy := newConsumerFailurePolicyStrategy(nil)

	decision := strategy.Decide(context.Background(), &mqspi.ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 3,
	})

	if decision.Action != mqspi.ConsumerFailureActionRetry {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 4*time.Second {
		t.Fatalf("unexpected backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyStrategyFinalAction(t *testing.T) {
	strategy := newConsumerFailurePolicyStrategy(&mqspi.ConsumerFailurePolicy{
		MaxAttempts: 2,
		FinalAction: mqspi.ConsumerFailureActionSkip,
	})

	decision := strategy.Decide(context.Background(), &mqspi.ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 2,
	})

	if decision.Action != mqspi.ConsumerFailureActionSkip {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 0 {
		t.Fatalf("unexpected final backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyStrategyFinalRetryKeepsBackoff(t *testing.T) {
	strategy := newConsumerFailurePolicyStrategy(&mqspi.ConsumerFailurePolicy{
		MaxAttempts:       2,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2,
		FinalAction:       mqspi.ConsumerFailureActionRetry,
	})

	decision := strategy.Decide(context.Background(), &mqspi.ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 2,
	})

	if decision.Action != mqspi.ConsumerFailureActionRetry {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 200*time.Millisecond {
		t.Fatalf("unexpected backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyStrategyHandler(t *testing.T) {
	called := false
	strategy := newConsumerFailurePolicyStrategy(&mqspi.ConsumerFailurePolicy{
		Handler: func(_ context.Context, failure *mqspi.ConsumerFailure) {
			called = true
			if failure.Attempt != 1 {
				t.Fatalf("unexpected attempt: %d", failure.Attempt)
			}
		},
	})

	strategy.Decide(context.Background(), &mqspi.ConsumerFailure{Attempt: 1})

	if !called {
		t.Fatalf("handler was not called")
	}
}
