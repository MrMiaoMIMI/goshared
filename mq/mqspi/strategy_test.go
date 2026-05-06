package mqspi

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConsumerFailurePolicyRetriesWithBackoff(t *testing.T) {
	policy := DefaultConsumerFailurePolicy()

	decision := policy.Decide(context.Background(), &ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 3,
	})

	if decision.Action != ConsumerFailureActionRetry {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 4*time.Second {
		t.Fatalf("unexpected backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyFinalAction(t *testing.T) {
	policy := &ConsumerFailurePolicy{
		MaxAttempts: 2,
		FinalAction: ConsumerFailureActionSkip,
	}

	decision := policy.Decide(context.Background(), &ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 2,
	})

	if decision.Action != ConsumerFailureActionSkip {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 0 {
		t.Fatalf("unexpected final backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyFinalRetryKeepsBackoff(t *testing.T) {
	policy := &ConsumerFailurePolicy{
		MaxAttempts:       2,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2,
		FinalAction:       ConsumerFailureActionRetry,
	}

	decision := policy.Decide(context.Background(), &ConsumerFailure{
		Err:     errors.New("process failed"),
		Attempt: 2,
	})

	if decision.Action != ConsumerFailureActionRetry {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
	if decision.Backoff != 200*time.Millisecond {
		t.Fatalf("unexpected backoff: %s", decision.Backoff)
	}
}

func TestConsumerFailurePolicyHandler(t *testing.T) {
	called := false
	policy := &ConsumerFailurePolicy{
		Handler: func(_ context.Context, failure *ConsumerFailure) {
			called = true
			if failure.Attempt != 1 {
				t.Fatalf("unexpected attempt: %d", failure.Attempt)
			}
		},
	}

	policy.Decide(context.Background(), &ConsumerFailure{Attempt: 1})

	if !called {
		t.Fatalf("handler was not called")
	}
}
